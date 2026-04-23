package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEnsureOpenAIImageCapabilityEnabled_DeniesGroupWithoutFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	h := &OpenAIGatewayHandler{}
	ok := h.ensureOpenAIImageCapabilityEnabled(c, &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	})

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "permission_error", gjson.Get(rec.Body.String(), "error.type").String())
	require.Equal(t, openAIImageGenerationDisabledMessage, gjson.Get(rec.Body.String(), "error.message").String())
}

func TestEnsureOpenAIImageModelAllowed_AllowsNonImageModelWithoutFlag(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &OpenAIGatewayHandler{}
	ok := h.ensureOpenAIImageModelAllowed(c, &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}, "gpt-5.4")

	require.True(t, ok)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestEnsureOpenAIImageModelAllowed_AllowsImageModelWhenGroupEnabled(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &OpenAIGatewayHandler{}
	ok := h.ensureOpenAIImageModelAllowed(c, &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI, AllowImageGeneration: true},
	}, "gpt-image-1")

	require.True(t, ok)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestOpenAIResolvedImageGenerationAllowed_DeniesChannelMappedImageModelWithoutFlag(t *testing.T) {
	apiKey := &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}

	allowed := openAIResolvedImageGenerationAllowed(apiKey, nil, "gpt-5.4", service.ChannelMappingResult{
		Mapped:      true,
		MappedModel: "gpt-image-1",
	}, "")

	require.False(t, allowed)
}

func TestOpenAIResolvedImageGenerationAllowed_DeniesAccountMappedImageModelWithoutFlag(t *testing.T) {
	apiKey := &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}
	account := &service.Account{
		Platform: service.PlatformOpenAI,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gpt-5.4": "gpt-image-1",
			},
		},
	}

	allowed := openAIResolvedImageGenerationAllowed(apiKey, account, "gpt-5.4", service.ChannelMappingResult{}, "")

	require.False(t, allowed)
}

func TestOpenAIResolvedImageGenerationAllowed_DeniesDefaultMappedImageModelWithoutFlag(t *testing.T) {
	apiKey := &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI},
	}
	account := &service.Account{Platform: service.PlatformOpenAI}

	allowed := openAIResolvedImageGenerationAllowed(apiKey, account, "gpt-5.4", service.ChannelMappingResult{}, "gpt-image-1")

	require.False(t, allowed)
}

func TestOpenAIResolvedImageGenerationAllowed_AllowsMappedImageModelWhenGroupEnabled(t *testing.T) {
	apiKey := &service.APIKey{
		Group: &service.Group{Platform: service.PlatformOpenAI, AllowImageGeneration: true},
	}
	account := &service.Account{
		Platform: service.PlatformOpenAI,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gpt-5.4": "gpt-image-1",
			},
		},
	}

	allowed := openAIResolvedImageGenerationAllowed(apiKey, account, "gpt-5.4", service.ChannelMappingResult{}, "")

	require.True(t, allowed)
}
