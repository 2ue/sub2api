package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const openAIImageMaxN = 10

// ImageGenerations handles OpenAI image generation requests.
func (h *OpenAIGatewayHandler) ImageGenerations(c *gin.Context) {
	body, req, ok := h.parseOpenAIImageGenerationRequest(c)
	if !ok {
		return
	}
	h.handleOpenAIImageRequest(c, body, req, "handler.openai_gateway.image_generations")
}

// ImageEdits handles OpenAI image edit requests.
func (h *OpenAIGatewayHandler) ImageEdits(c *gin.Context) {
	body, req, ok := h.parseOpenAIImageEditRequest(c)
	if !ok {
		return
	}
	h.handleOpenAIImageRequest(c, body, req, "handler.openai_gateway.image_edits")
}

func (h *OpenAIGatewayHandler) parseOpenAIImageGenerationRequest(c *gin.Context) ([]byte, *service.OpenAIImageRequest, bool) {
	body, ok := h.readOpenAIImageRequestBody(c)
	if !ok {
		return nil, nil, false
	}
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return nil, nil, false
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || strings.TrimSpace(modelResult.String()) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, nil, false
	}
	model := strings.TrimSpace(modelResult.String())
	promptResult := gjson.GetBytes(body, "prompt")
	if !promptResult.Exists() || promptResult.Type != gjson.String || strings.TrimSpace(promptResult.String()) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "prompt is required")
		return nil, nil, false
	}
	prompt := strings.TrimSpace(promptResult.String())

	n, ok := parseOpenAIImageNJSON(body)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("n must be an integer between 1 and %d", openAIImageMaxN))
		return nil, nil, false
	}

	imageSize, err := normalizeOpenAIImageSizeTier(strings.TrimSpace(gjson.GetBytes(body, "size").String()))
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, nil, false
	}
	if !isOpenAIImageResponseFormatValid(strings.TrimSpace(gjson.GetBytes(body, "response_format").String())) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "response_format must be one of: url, b64_json")
		return nil, nil, false
	}

	return body, &service.OpenAIImageRequest{
		Body:         body,
		Model:        model,
		Prompt:       prompt,
		N:            n,
		ImageSize:    imageSize,
		UpstreamPath: service.OpenAIImagesGenerationsPath,
	}, true
}

func (h *OpenAIGatewayHandler) parseOpenAIImageEditRequest(c *gin.Context) ([]byte, *service.OpenAIImageRequest, bool) {
	body, ok := h.readOpenAIImageRequestBody(c)
	if !ok {
		return nil, nil, false
	}

	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Content-Type must be multipart/form-data")
		return nil, nil, false
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "multipart boundary is required")
		return nil, nil, false
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	parts := make([]service.OpenAIImageMultipartPart, 0, 8)
	model := ""
	prompt := ""
	sizeRaw := ""
	responseFormat := ""
	n := 1
	imageCount := 0

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse multipart request")
			return nil, nil, false
		}

		partData, err := io.ReadAll(part)
		_ = part.Close()
		if err != nil {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read multipart part")
			return nil, nil, false
		}

		fieldName := part.FormName()
		fileName := part.FileName()
		contentType := part.Header.Get("Content-Type")

		if fileName != "" {
			parts = append(parts, service.OpenAIImageMultipartPart{
				FieldName:   fieldName,
				FileName:    fileName,
				ContentType: contentType,
				Data:        partData,
				IsFile:      true,
			})
			if isOpenAIImagePrimaryField(fieldName) && len(partData) > 0 {
				imageCount++
			}
			continue
		}

		value := string(partData)
		parts = append(parts, service.OpenAIImageMultipartPart{
			FieldName: fieldName,
			Value:     value,
		})

		trimmed := strings.TrimSpace(value)
		switch fieldName {
		case "model":
			model = trimmed
		case "prompt":
			prompt = trimmed
		case "size":
			sizeRaw = trimmed
		case "response_format":
			responseFormat = trimmed
		case "n":
			parsedN, valid := parseOpenAIImageNForm(trimmed)
			if !valid {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("n must be an integer between 1 and %d", openAIImageMaxN))
				return nil, nil, false
			}
			n = parsedN
		}
	}

	if model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, nil, false
	}
	if prompt == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "prompt is required")
		return nil, nil, false
	}
	if imageCount == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "at least one image file is required")
		return nil, nil, false
	}

	imageSize, err := normalizeOpenAIImageSizeTier(sizeRaw)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, nil, false
	}
	if !isOpenAIImageResponseFormatValid(responseFormat) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "response_format must be one of: url, b64_json")
		return nil, nil, false
	}

	return body, &service.OpenAIImageRequest{
		Model:          model,
		Prompt:         prompt,
		N:              n,
		ImageSize:      imageSize,
		UpstreamPath:   service.OpenAIImagesEditsPath,
		MultipartParts: parts,
	}, true
}

func (h *OpenAIGatewayHandler) readOpenAIImageRequestBody(c *gin.Context) ([]byte, bool) {
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return nil, false
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return nil, false
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return nil, false
	}
	return body, true
}

func (h *OpenAIGatewayHandler) handleOpenAIImageRequest(
	c *gin.Context,
	body []byte,
	req *service.OpenAIImageRequest,
	logName string,
) {
	setOpenAIClientTransportHTTP(c)
	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if !h.ensureOpenAIImageCapabilityEnabled(c, apiKey) {
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(
		c,
		logName,
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.String("model", req.Model),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	setOpsRequestContext(c, req.Model, false, body)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, req.Model)
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	streamStarted := false
	maxWait := service.CalculateMaxWait(subject.Concurrency)
	canWait, err := h.concurrencyHelper.IncrementWaitCount(c.Request.Context(), subject.UserID, maxWait)
	waitCounted := false
	if err != nil {
		reqLog.Warn("openai_image.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return
	}
	if err == nil && canWait {
		waitCounted = true
	}
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), subject.UserID)
		}
	}()

	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		reqLog.Warn("openai_image.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", false)
		return
	}
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), subject.UserID)
		waitCounted = false
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai_image.billing_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.errorResponse(c, status, code, message)
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHashWithFallback(c, body, req.Model+"\n"+req.Prompt)
	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	unsupportedAccountSeen := false
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		selection, _, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			sessionHash,
			req.Model,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
		)
		if err != nil {
			if len(failedAccountIDs) == 0 {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error())
				return
			}
			if unsupportedAccountSeen && lastFailoverErr == nil {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No image-capable OpenAI API key accounts")
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, false)
			} else {
				h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, false)
			}
			return
		}

		account := selection.Account
		if account == nil {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts")
			return
		}
		if !isOpenAIImageCapableAccount(account) {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			failedAccountIDs[account.ID] = struct{}{}
			unsupportedAccountSeen = true
			reqLog.Warn("openai_image.account_not_supported",
				zap.Int64("account_id", account.ID),
				zap.String("account_type", account.Type),
			)
			continue
		}

		setOpsSelectedAccount(c, account.ID, account.Platform)
		accountReleaseFunc := selection.ReleaseFunc
		if !selection.Acquired {
			if selection.WaitPlan == nil {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts")
				return
			}
			accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
				c,
				account.ID,
				selection.WaitPlan.MaxConcurrency,
				selection.WaitPlan.Timeout,
				false,
				&streamStarted,
			)
			if err != nil {
				reqLog.Warn("openai_image.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				h.handleConcurrencyError(c, err, "account", false)
				return
			}
		}
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		result, err := h.gatewayService.ForwardImageRequest(c.Request.Context(), c, account, req, channelMapping.MappedModel)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)

		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, false)
					return
				}
				switchCount++
				reqLog.Warn("openai_image.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, false)
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarn(c, wroteFallback) {
				reqLog.Warn("openai_image.forward_failed", fields...)
				return
			}
			reqLog.Error("openai_image.forward_failed", fields...)
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		requestPayloadHash := service.HashUsageRequestPayload(body)

		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpoint(c),
				UpstreamEndpoint:   GetUpstreamEndpoint(c, account.Platform),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(req.Model, result.UpstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", logName),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", req.Model),
					zap.Int64("account_id", account.ID),
				).Error("openai_image.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai_image.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
			zap.Int("image_count", result.ImageCount),
			zap.String("image_size", result.ImageSize),
		)
		return
	}
}

func parseOpenAIImageNJSON(body []byte) (int, bool) {
	value := gjson.GetBytes(body, "n")
	if !value.Exists() {
		return 1, true
	}
	raw := strings.TrimSpace(value.Raw)
	if raw == "" {
		return 1, false
	}
	n, err := strconv.Atoi(strings.Trim(raw, `"`))
	if err != nil || n < 1 || n > openAIImageMaxN {
		return 1, false
	}
	return n, true
}

func parseOpenAIImageNForm(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 1, true
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil || n < 1 || n > openAIImageMaxN {
		return 1, false
	}
	return n, true
}

func isOpenAIImageResponseFormatValid(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == "" || trimmed == "url" || trimmed == "b64_json"
}

func normalizeOpenAIImageSizeTier(raw string) (string, error) {
	size := strings.ToLower(strings.TrimSpace(raw))
	switch size {
	case "", "1k", "1024x1024", "1024x1536", "1536x1024":
		return "1K", nil
	case "2k", "2048x2048", "2048x3072", "3072x2048", "1536x1536":
		return "2K", nil
	case "4k", "4096x4096", "4096x6144", "6144x4096":
		return "4K", nil
	default:
		return "", fmt.Errorf("unsupported size: %s", raw)
	}
}

func isOpenAIImagePrimaryField(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "image", "image[]", "images", "images[]":
		return true
	default:
		return strings.HasPrefix(normalized, "image_") || strings.HasPrefix(normalized, "images_")
	}
}

func isOpenAIImageCapableAccount(account *service.Account) bool {
	return account != nil && account.IsOpenAIApiKey()
}
