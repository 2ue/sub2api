package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	OpenAIImagesGenerationsPath = "/v1/images/generations"
	OpenAIImagesEditsPath       = "/v1/images/edits"
)

type OpenAIImageMultipartPart struct {
	FieldName   string
	FileName    string
	ContentType string
	Value       string
	Data        []byte
	IsFile      bool
}

type OpenAIImageRequest struct {
	Body           []byte
	Model          string
	Prompt         string
	N              int
	ImageSize      string
	UpstreamPath   string
	MultipartParts []OpenAIImageMultipartPart
}

func buildOpenAIEndpointURL(base, endpoint string) string {
	normalizedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	normalizedEndpoint := "/" + strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	if normalizedBase == "" {
		return normalizedEndpoint
	}
	if strings.HasSuffix(normalizedBase, normalizedEndpoint) {
		return normalizedBase
	}
	if strings.HasSuffix(normalizedBase, "/v1") && strings.HasPrefix(normalizedEndpoint, "/v1/") {
		return normalizedBase + strings.TrimPrefix(normalizedEndpoint, "/v1")
	}
	return normalizedBase + normalizedEndpoint
}

func buildOpenAIImageURL(base, upstreamPath string) string {
	return buildOpenAIEndpointURL(base, upstreamPath)
}

func (s *OpenAIGatewayService) ForwardImageRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	input *OpenAIImageRequest,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if input == nil {
		return nil, fmt.Errorf("image request is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("openai image endpoints require api key accounts")
	}

	startTime := time.Now()
	originalModel := strings.TrimSpace(input.Model)
	upstreamModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, originalModel, defaultMappedModel))

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	upstreamReq, err := s.buildOpenAIImageUpstreamRequest(ctx, c, account, input, token, upstreamModel)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, s.handleOpenAIImageErrorResponse(ctx, resp, c, account)
	}

	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	body = normalizeOpenAIImageSuccessBody(body)

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)

	usage := &OpenAIUsage{}
	if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(body); ok {
		*usage = parsedUsage
	}

	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		Usage:           *usage,
		Model:           originalModel,
		UpstreamModel:   upstreamModel,
		Stream:          false,
		OpenAIWSMode:    false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
		ImageCount:      extractOpenAIImageCountFromBody(body),
		ImageSize:       strings.TrimSpace(input.ImageSize),
	}, nil
}

func (s *OpenAIGatewayService) buildOpenAIImageUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	input *OpenAIImageRequest,
	token string,
	upstreamModel string,
) (*http.Request, error) {
	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	validatedURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	targetURL := buildOpenAIImageURL(validatedURL, input.UpstreamPath)

	body := input.Body
	contentType := "application/json"
	if len(input.MultipartParts) > 0 {
		body, contentType, err = buildOpenAIImageMultipartBody(input.MultipartParts, upstreamModel)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, fmt.Errorf("rewrite image request model: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if !openaiPassthroughAllowedHeaders[lowerKey] || lowerKey == "content-type" {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}

	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Set("authorization", "Bearer "+token)
	req.Header.Set("content-type", contentType)
	if req.Header.Get("accept") == "" {
		req.Header.Set("accept", "application/json")
	}

	if accountUA := account.GetOpenAIUserAgent(); accountUA != "" {
		req.Header.Set("user-agent", accountUA)
	}
	return req, nil
}

func buildOpenAIImageMultipartBody(parts []OpenAIImageMultipartPart, upstreamModel string) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, part := range parts {
		if part.IsFile {
			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeMultipartField(part.FieldName), escapeMultipartField(part.FileName)))
			contentType := strings.TrimSpace(part.ContentType)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			header.Set("Content-Type", contentType)
			w, err := writer.CreatePart(header)
			if err != nil {
				_ = writer.Close()
				return nil, "", err
			}
			if _, err := w.Write(part.Data); err != nil {
				_ = writer.Close()
				return nil, "", err
			}
			continue
		}

		value := part.Value
		if part.FieldName == "model" {
			value = upstreamModel
		}
		if err := writer.WriteField(part.FieldName, value); err != nil {
			_ = writer.Close()
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}

func escapeMultipartField(raw string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(raw)
}

func extractOpenAIImageCountFromBody(body []byte) int {
	if len(body) == 0 {
		return 0
	}
	data := gjson.GetBytes(body, "data")
	if data.IsArray() {
		return len(data.Array())
	}
	return 0
}

func normalizeOpenAIImageSuccessBody(body []byte) []byte {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return body
	}

	data := gjson.GetBytes(body, "data")
	if data.IsArray() {
		return ensureOpenAIImageResponseCreated(body)
	}

	if data.Exists() && data.Type == gjson.JSON {
		trimmed := strings.TrimSpace(data.Raw)
		if strings.HasPrefix(trimmed, "{") {
			normalized, err := sjson.SetRawBytes(body, "data", []byte("["+trimmed+"]"))
			if err == nil {
				body = normalized
			}
		}
	} else {
		items := make([]map[string]any, 0, 4)
		appendItem := func(urlValue, b64Value, revisedPrompt string) {
			item := make(map[string]any, 3)
			if urlValue = strings.TrimSpace(urlValue); urlValue != "" {
				item["url"] = urlValue
			}
			if b64Value = strings.TrimSpace(b64Value); b64Value != "" {
				item["b64_json"] = b64Value
			}
			if revisedPrompt = strings.TrimSpace(revisedPrompt); revisedPrompt != "" {
				item["revised_prompt"] = revisedPrompt
			}
			if len(item) > 0 {
				items = append(items, item)
			}
		}

		images := gjson.GetBytes(body, "images")
		if images.IsArray() {
			for _, item := range images.Array() {
				if item.Type == gjson.String {
					appendItem(item.String(), "", "")
					continue
				}
				appendItem(
					gjson.Get(item.Raw, "url").String(),
					gjson.Get(item.Raw, "b64_json").String(),
					gjson.Get(item.Raw, "revised_prompt").String(),
				)
			}
		} else {
			appendItem(
				gjson.GetBytes(body, "url").String(),
				gjson.GetBytes(body, "b64_json").String(),
				gjson.GetBytes(body, "revised_prompt").String(),
			)
		}

		if len(items) > 0 {
			normalized, err := sjson.SetBytes(body, "data", items)
			if err == nil {
				body = normalized
			}
		}
	}

	if extractOpenAIImageCountFromBody(body) <= 0 {
		return body
	}
	return ensureOpenAIImageResponseCreated(body)
}

func ensureOpenAIImageResponseCreated(body []byte) []byte {
	if gjson.GetBytes(body, "created").Exists() {
		return body
	}
	normalized, err := sjson.SetBytes(body, "created", time.Now().Unix())
	if err != nil {
		return body
	}
	return normalized
}

func (s *OpenAIGatewayService) handleOpenAIImageErrorResponse(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	if s.rateLimitService != nil {
		_ = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:             account.Platform,
		AccountID:            account.ID,
		AccountName:          account.Name,
		UpstreamStatusCode:   resp.StatusCode,
		UpstreamRequestID:    resp.Header.Get("x-request-id"),
		Kind:                 "http_error",
		Message:              upstreamMsg,
		Detail:               upstreamDetail,
		UpstreamResponseBody: upstreamDetail,
	})
	if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, body) {
		return &UpstreamFailoverError{
			StatusCode:      resp.StatusCode,
			ResponseBody:    body,
			ResponseHeaders: resp.Header.Clone(),
		}
	}

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	if gjson.ValidBytes(body) && gjson.GetBytes(body, "error").Exists() {
		c.Data(resp.StatusCode, contentType, body)
	} else {
		message := upstreamMsg
		if message == "" {
			message = "Upstream image request failed"
		}
		c.JSON(resp.StatusCode, gin.H{
			"error": gin.H{
				"type":    "upstream_error",
				"message": message,
			},
		})
	}
	if upstreamMsg == "" {
		return fmt.Errorf("upstream error: %d", resp.StatusCode)
	}
	return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
}
