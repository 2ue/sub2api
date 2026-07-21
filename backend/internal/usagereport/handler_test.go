package usagereport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func testJWTAuthMiddleware(userID int64, role string) servermiddleware.JWTAuthMiddleware {
	return servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		if userID > 0 {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: userID})
			c.Set(string(servermiddleware.ContextKeyUserRole), role)
		}
		c.Next()
	})
}

func newReportTestRouter(t *testing.T, h *Handler, auth servermiddleware.JWTAuthMiddleware) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h.RegisterRoutes(router, auth, nil)
	return router
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) responseEnvelope {
	t.Helper()

	var envelope responseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return envelope
}

type responseEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Reason  string          `json:"reason,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func TestReportPageRoutes(t *testing.T) {
	handler, err := NewHandler(NewService(nil, &stubReportUserRepo{}))
	require.NoError(t, err)

	router := newReportTestRouter(t, handler, testJWTAuthMiddleware(1, "user"))

	t.Run("root serves the page", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
		require.Contains(t, rec.Body.String(), "Usage Report")
	})

	t.Run("report mount redirects to trailing slash", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/report", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusMovedPermanently, rec.Code)
		require.Equal(t, "/report/", rec.Header().Get("Location"))
	})

	t.Run("report mount serves the page", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/report/", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
		require.Contains(t, rec.Body.String(), "Usage Report")
	})

	t.Run("static assets are served from the embedded bundle", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/report/styles.css", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "text/css; charset=utf-8", rec.Header().Get("Content-Type"))
		require.True(t, strings.Contains(rec.Body.String(), ".app-shell"))
	})
}

func TestBootstrapRequiresAuth(t *testing.T) {
	repo := &stubReportUserRepo{
		getByID: func(context.Context, int64) (*service.User, error) {
			return newActiveUser(1, service.RoleUser), nil
		},
	}
	handler, err := NewHandler(NewService(nil, repo))
	require.NoError(t, err)

	router := newReportTestRouter(t, handler, servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		c.Next()
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/report/bootstrap", nil)

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "User not authenticated")
}

func TestBootstrapAndSearchUsers(t *testing.T) {
	repo := &stubReportUserRepo{
		getByID: func(context.Context, int64) (*service.User, error) {
			return &service.User{
				ID:       7,
				Email:    "admin@example.com",
				Username: "admin",
				Role:     service.RoleAdmin,
				Status:   service.StatusActive,
			}, nil
		},
		listWithFilters: func(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
			require.Equal(t, 1, params.Page)
			require.Equal(t, 20, params.PageSize)
			require.Equal(t, "email", params.SortBy)
			require.Equal(t, "asc", params.SortOrder)
			require.Equal(t, "ali", filters.Search)
			require.True(t, filters.IncludeDeleted)

			return []service.User{
				{
					ID:       9,
					Email:    "alice@example.com",
					Username: "alice",
					Role:     service.RoleUser,
					Status:   service.StatusActive,
				},
			}, &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 20, Pages: 1}, nil
		},
	}
	handler, err := NewHandler(NewService(nil, repo))
	require.NoError(t, err)

	router := newReportTestRouter(t, handler, testJWTAuthMiddleware(7, service.RoleAdmin))

	t.Run("bootstrap returns the current user and permissions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/report/bootstrap", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		envelope := decodeEnvelope(t, rec)
		var payload BootstrapResponse
		require.NoError(t, json.Unmarshal(envelope.Data, &payload))
		require.Equal(t, int64(7), payload.CurrentUser.ID)
		require.Equal(t, service.RoleAdmin, payload.CurrentUser.Role)
		require.True(t, payload.CanSearchUsers)
		require.NotEmpty(t, payload.Timezone)
	})

	t.Run("search returns minimal user options", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/report/users?q=ali", nil)

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		envelope := decodeEnvelope(t, rec)
		var payload []UserOption
		require.NoError(t, json.Unmarshal(envelope.Data, &payload))
		require.Len(t, payload, 1)
		require.Equal(t, int64(9), payload[0].ID)
		require.Equal(t, "alice@example.com", payload[0].Email)
		require.Equal(t, "alice", payload[0].DisplayName)
		require.False(t, payload[0].Deleted)
	})
}

func TestSearchUsersForbiddenForNonAdmin(t *testing.T) {
	repo := &stubReportUserRepo{
		getByID: func(context.Context, int64) (*service.User, error) {
			return newActiveUser(7, service.RoleUser), nil
		},
		listWithFilters: func(context.Context, pagination.PaginationParams, service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
			t.Fatal("ListWithFilters should not be called for non-admin callers")
			return nil, nil, nil
		},
	}
	handler, err := NewHandler(NewService(nil, repo))
	require.NoError(t, err)

	router := newReportTestRouter(t, handler, testJWTAuthMiddleware(7, service.RoleUser))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/report/users?q=ali", nil)

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "cross-user access requires admin")
}
