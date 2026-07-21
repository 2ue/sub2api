package usagereport

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFS embed.FS

type Handler struct {
	service *reportService
	files   fs.FS
}

func NewHandler(service *reportService) (*Handler, error) {
	dist, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	return &Handler{service: service, files: dist}, nil
}

func (h *Handler) RegisterRoutes(r *gin.Engine, auth middleware2.JWTAuthMiddleware, securityHeaders gin.HandlerFunc) {
	if securityHeaders != nil {
		r.Use(securityHeaders)
	}

	h.registerPrefix(r, "", auth)
	h.registerPrefix(r, "/report", auth)

	r.NoRoute(h.noRoute)
}

func (h *Handler) registerPrefix(r *gin.Engine, prefix string, auth middleware2.JWTAuthMiddleware) {
	mount := strings.TrimSuffix(prefix, "/")
	if mount == "" {
		r.GET("/", h.serveIndex)
		r.GET("/index.html", h.serveIndex)
		r.GET("/healthz", h.healthz)

		api := r.Group("/api/v1/report")
		api.Use(gin.HandlerFunc(auth))
		{
			api.GET("/bootstrap", h.bootstrap)
			api.GET("/users", h.searchUsers)
			api.POST("/summary", h.summary)
		}
		return
	}
	if !strings.HasPrefix(mount, "/") {
		mount = "/" + mount
	}
	r.GET(mount, func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, mount+"/")
	})
	r.GET(mount+"/", h.serveIndex)
	r.GET(mount+"/index.html", h.serveIndex)
	r.GET(mount+"/healthz", h.healthz)

	api := r.Group(mount + "/api/v1/report")
	api.Use(gin.HandlerFunc(auth))
	{
		api.GET("/bootstrap", h.bootstrap)
		api.GET("/users", h.searchUsers)
		api.POST("/summary", h.summary)
	}
}

func (h *Handler) healthz(c *gin.Context) {
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) bootstrap(c *gin.Context) {
	current, role, ok := h.currentActor(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	response.Success(c, h.service.Bootstrap(c.Request.Context(), current, strings.TrimSpace(role) == service.RoleAdmin))
}

func (h *Handler) searchUsers(c *gin.Context) {
	current, role, ok := h.currentActor(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	_ = current
	q := c.Query("q")
	users, err := h.service.SearchUsers(c.Request.Context(), role, q, 20)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, users)
}

func (h *Handler) summary(c *gin.Context) {
	current, role, ok := h.currentActor(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req SummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, ErrInvalidDateRange)
		return
	}

	summary, err := h.service.QuerySummary(c.Request.Context(), current, role, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *Handler) currentActor(c *gin.Context) (*service.User, string, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		return nil, "", false
	}
	role, _ := middleware2.GetUserRoleFromContext(c)
	if h.service == nil || h.service.userRepo == nil {
		return nil, role, false
	}
	user, err := h.service.userRepo.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		return nil, role, false
	}
	return user, role, true
}

func (h *Handler) serveIndex(c *gin.Context) {
	if h.files == nil {
		c.Status(http.StatusNotFound)
		return
	}
	data, err := fs.ReadFile(h.files, "index.html")
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (h *Handler) noRoute(c *gin.Context) {
	if h.isAPIPath(c.Request.URL.Path) {
		response.NotFound(c, "Not found")
		return
	}

	clean := h.stripKnownPrefix(strings.TrimPrefix(c.Request.URL.Path, "/"))
	if clean != "" {
		if data, err := fs.ReadFile(h.files, clean); err == nil {
			contentType := contentTypeFor(clean)
			if contentType == "" {
				contentType = http.DetectContentType(data)
			}
			if clean == "app.js" || clean == "styles.css" {
				c.Header("Cache-Control", "no-cache")
			}
			c.Data(http.StatusOK, contentType, data)
			return
		}
	}
	h.serveIndex(c)
}

func (h *Handler) isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/report/api/")
}

func (h *Handler) stripKnownPrefix(clean string) string {
	if strings.HasPrefix(clean, "report/") {
		return strings.TrimPrefix(clean, "report/")
	}
	return clean
}

func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	default:
		return ""
	}
}
