package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestParseOpenAIImageGenerationRequest_Validation(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader("{"))
		c.Request.Header.Set("Content-Type", "application/json")

		h := &OpenAIGatewayHandler{}
		body, req, ok := h.parseOpenAIImageGenerationRequest(c)

		require.False(t, ok)
		require.Nil(t, body)
		require.Nil(t, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "invalid_request_error", gjson.Get(rec.Body.String(), "error.type").String())
		require.Equal(t, "Failed to parse request body", gjson.Get(rec.Body.String(), "error.message").String())
	})

	t.Run("invalid n", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1","prompt":"cat","n":11}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h := &OpenAIGatewayHandler{}
		body, req, ok := h.parseOpenAIImageGenerationRequest(c)

		require.False(t, ok)
		require.Nil(t, body)
		require.Nil(t, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, gjson.Get(rec.Body.String(), "error.message").String(), "n must be an integer")
	})
}

func TestParseOpenAIImageEditRequest_ValidationAndMultipartCollection(t *testing.T) {
	t.Run("missing image file", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "add a hat"))
		require.NoError(t, writer.Close())

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(buf.Bytes()))
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		h := &OpenAIGatewayHandler{}
		body, req, ok := h.parseOpenAIImageEditRequest(c)

		require.False(t, ok)
		require.Nil(t, body)
		require.Nil(t, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "at least one image file is required", gjson.Get(rec.Body.String(), "error.message").String())
	})

	t.Run("collect multipart parts", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "add a hat"))
		require.NoError(t, writer.WriteField("n", "2"))
		require.NoError(t, writer.WriteField("size", "2048x2048"))
		require.NoError(t, writer.WriteField("response_format", "b64_json"))

		imagePart, err := writer.CreateFormFile("image", "source.png")
		require.NoError(t, err)
		_, err = imagePart.Write([]byte("source-image"))
		require.NoError(t, err)

		maskPart, err := writer.CreateFormFile("mask", "mask.png")
		require.NoError(t, err)
		_, err = maskPart.Write([]byte("mask-image"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(buf.Bytes()))
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		h := &OpenAIGatewayHandler{}
		body, req, ok := h.parseOpenAIImageEditRequest(c)

		require.True(t, ok)
		require.NotEmpty(t, body)
		require.NotNil(t, req)
		require.Equal(t, "gpt-image-1", req.Model)
		require.Equal(t, "add a hat", req.Prompt)
		require.Equal(t, 2, req.N)
		require.Equal(t, "2K", req.ImageSize)
		require.Equal(t, service.OpenAIImagesEditsPath, req.UpstreamPath)
		require.Len(t, req.MultipartParts, 7)

		fileFields := 0
		maskFound := false
		for _, part := range req.MultipartParts {
			if part.IsFile {
				fileFields++
			}
			if part.FieldName == "mask" && part.IsFile {
				maskFound = true
			}
		}
		require.Equal(t, 2, fileFields)
		require.True(t, maskFound)
	})
}

func TestNormalizeOpenAIImageSizeTier(t *testing.T) {
	size, err := normalizeOpenAIImageSizeTier("2048x3072")
	require.NoError(t, err)
	require.Equal(t, "2K", size)

	_, err = normalizeOpenAIImageSizeTier("512x512")
	require.Error(t, err)
}

func TestParseOpenAIImageNHelpers(t *testing.T) {
	n, ok := parseOpenAIImageNJSON([]byte(`{"n":"3"}`))
	require.True(t, ok)
	require.Equal(t, 3, n)

	n, ok = parseOpenAIImageNForm("4")
	require.True(t, ok)
	require.Equal(t, 4, n)

	_, ok = parseOpenAIImageNForm("0")
	require.False(t, ok)
}

func TestIsOpenAIImageResponseFormatValid(t *testing.T) {
	require.True(t, isOpenAIImageResponseFormatValid(""))
	require.True(t, isOpenAIImageResponseFormatValid("url"))
	require.True(t, isOpenAIImageResponseFormatValid("b64_json"))
	require.False(t, isOpenAIImageResponseFormatValid("binary"))
}

func TestParseOpenAIImageGenerationRequest_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1","prompt":"cat","n":2,"size":"1024x1024"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &OpenAIGatewayHandler{}
	body, req, ok := h.parseOpenAIImageGenerationRequest(c)

	require.True(t, ok)
	require.JSONEq(t, `{"model":"gpt-image-1","prompt":"cat","n":2,"size":"1024x1024"}`, string(body))
	require.Equal(t, &service.OpenAIImageRequest{
		Body:         body,
		Model:        "gpt-image-1",
		Prompt:       "cat",
		N:            2,
		ImageSize:    "1K",
		UpstreamPath: service.OpenAIImagesGenerationsPath,
	}, req)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
}
