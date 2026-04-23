package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewayModelsAccountRepoStub struct {
	service.AccountRepository
	accountsByGroup map[int64][]service.Account
}

func (s *gatewayModelsAccountRepoStub) ListSchedulable(ctx context.Context) ([]service.Account, error) {
	return nil, nil
}

func (s *gatewayModelsAccountRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]service.Account, error) {
	return append([]service.Account(nil), s.accountsByGroup[groupID]...), nil
}

func newGatewayModelsHandlerForTest(repo service.AccountRepository) *GatewayHandler {
	svc := service.NewGatewayService(
		repo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	return &GatewayHandler{gatewayService: svc}
}

func TestGatewayHandlerModels_OpenAIImageMappingsUseOpenAIShape(t *testing.T) {
	groupID := int64(42)
	repo := &gatewayModelsAccountRepoStub{
		accountsByGroup: map[int64][]service.Account{
			groupID: {
				{
					ID:       1,
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"gpt-image-2": "gpt-image-2",
							"gpt-image-1": "gpt-image-1",
						},
					},
				},
			},
		},
	}

	h := newGatewayModelsHandlerForTest(repo)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Object string         `json:"object"`
		Data   []openai.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "list", resp.Object)
	require.Len(t, resp.Data, 2)
	require.Equal(t, "gpt-image-1", resp.Data[0].ID)
	require.Equal(t, "model", resp.Data[0].Object)
	require.Equal(t, "openai", resp.Data[0].OwnedBy)
	require.Equal(t, "gpt-image-2", resp.Data[1].ID)
}

func TestGatewayHandlerModels_OpenAIFallbackDoesNotAdvertiseUnconfiguredImageModels(t *testing.T) {
	groupID := int64(43)
	repo := &gatewayModelsAccountRepoStub{
		accountsByGroup: map[int64][]service.Account{
			groupID: {
				{
					ID:       1,
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
				},
			},
		},
	}

	h := newGatewayModelsHandlerForTest(repo)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []openai.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, openai.DefaultModels, resp.Data)
	for _, model := range resp.Data {
		require.NotEqual(t, "gpt-image-1", model.ID)
		require.NotEqual(t, "gpt-image-2", model.ID)
	}
}

func TestGatewayHandlerModels_OpenAIImageMappingsHiddenWhenGroupDisallowsImages(t *testing.T) {
	groupID := int64(44)
	repo := &gatewayModelsAccountRepoStub{
		accountsByGroup: map[int64][]service.Account{
			groupID: {
				{
					ID:       1,
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"gpt-image-2": "gpt-image-2",
							"gpt-image-1": "gpt-image-1",
						},
					},
				},
			},
		},
	}

	h := newGatewayModelsHandlerForTest(repo)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []openai.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, openai.DefaultModels, resp.Data)
}
