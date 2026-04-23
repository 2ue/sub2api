package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeOpenAIImageSuccessBody(t *testing.T) {
	t.Run("single top-level url becomes data array", func(t *testing.T) {
		normalized := normalizeOpenAIImageSuccessBody([]byte(`{"url":"https://example.com/a.png","revised_prompt":"refined"}`))
		require.Equal(t, 1, extractOpenAIImageCountFromBody(normalized))
		require.Equal(t, "https://example.com/a.png", gjson.GetBytes(normalized, "data.0.url").String())
		require.Equal(t, "refined", gjson.GetBytes(normalized, "data.0.revised_prompt").String())
		require.True(t, gjson.GetBytes(normalized, "created").Int() > 0)
	})

	t.Run("images array becomes openai data array", func(t *testing.T) {
		normalized := normalizeOpenAIImageSuccessBody([]byte(`{"images":[{"b64_json":"abc"},{"url":"https://example.com/b.png"}]}`))
		require.Equal(t, 2, extractOpenAIImageCountFromBody(normalized))
		require.Equal(t, "abc", gjson.GetBytes(normalized, "data.0.b64_json").String())
		require.Equal(t, "https://example.com/b.png", gjson.GetBytes(normalized, "data.1.url").String())
		require.True(t, gjson.GetBytes(normalized, "created").Int() > 0)
	})

	t.Run("existing data array only backfills created", func(t *testing.T) {
		normalized := normalizeOpenAIImageSuccessBody([]byte(`{"data":[{"url":"https://example.com/a.png"}]}`))
		require.Equal(t, 1, extractOpenAIImageCountFromBody(normalized))
		require.True(t, gjson.GetBytes(normalized, "created").Int() > 0)
	})
}

func TestBuildOpenAIImageMultipartBody(t *testing.T) {
	body, contentType, err := buildOpenAIImageMultipartBody([]OpenAIImageMultipartPart{
		{FieldName: "model", Value: "gpt-image-1"},
		{FieldName: "prompt", Value: "draw a fox"},
		{FieldName: "image", FileName: "source.png", ContentType: "image/png", Data: []byte("img-bytes"), IsFile: true},
	}, "mapped-image-model")
	require.NoError(t, err)
	require.Contains(t, contentType, "multipart/form-data")

	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)

	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	fields := map[string]string{}
	files := map[string][]byte{}

	for {
		part, nextErr := reader.NextPart()
		if nextErr == io.EOF {
			break
		}
		require.NoError(t, nextErr)
		data, readErr := io.ReadAll(part)
		require.NoError(t, readErr)
		if part.FileName() != "" {
			files[part.FormName()] = data
		} else {
			fields[part.FormName()] = string(data)
		}
		_ = part.Close()
	}

	require.Equal(t, "mapped-image-model", fields["model"])
	require.Equal(t, "draw a fox", fields["prompt"])
	require.Equal(t, []byte("img-bytes"), files["image"])
}

func TestHandleOpenAIImageErrorResponse_ReturnsFailoverErrorForRetryableStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_retry"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error"}}`)),
	}

	err := svc.handleOpenAIImageErrorResponse(context.Background(), resp, c, &Account{
		ID:       1,
		Name:     "image-account",
		Platform: PlatformOpenAI,
	})

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Equal(t, `{"error":{"message":"rate limit exceeded","type":"rate_limit_error"}}`, string(failoverErr.ResponseBody))
}
