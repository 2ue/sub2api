package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/usagereport"
	"github.com/gin-gonic/gin"
)

// Version may be injected with -ldflags "-X main.Version=...".
var Version = ""

func main() {
	logger.InitBootstrap()
	defer logger.Sync()

	addrFlag := flag.String("addr", "", "listen address override")
	flag.Parse()

	cfg, err := config.ProvideConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := logger.Init(logger.OptionsFromConfig(cfg.Log)); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	addr := strings.TrimSpace(*addrFlag)
	if addr == "" {
		if envAddr := strings.TrimSpace(os.Getenv("REPORT_ADDR")); envAddr != "" {
			addr = envAddr
		} else {
			addr = "127.0.0.1:8081"
		}
	}

	entClient, err := repository.ProvideEnt(cfg)
	if err != nil {
		log.Fatalf("failed to initialize ent: %v", err)
	}
	sqlDB, err := repository.ProvideSQLDB(entClient)
	if err != nil {
		log.Fatalf("failed to initialize sql db: %v", err)
	}

	userRepo := repository.NewUserRepository(entClient, sqlDB)
	groupRepo := repository.NewGroupRepository(entClient, sqlDB)
	proxyRepo := repository.NewProxyRepository(entClient, sqlDB)
	settingRepo := repository.NewSettingRepository(entClient)
	settingService := service.ProvideSettingService(settingRepo, groupRepo, proxyRepo, cfg)
	authService := service.NewAuthService(entClient, userRepo, nil, nil, cfg, settingService, nil, nil, nil, nil, nil, nil, nil)
	userService := service.NewUserService(userRepo, settingRepo, nil, nil)

	reportSvc := usagereport.NewService(sqlDB, userRepo)
	reportHandler, err := usagereport.NewHandler(reportSvc)
	if err != nil {
		log.Fatalf("failed to initialize report handler: %v", err)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.SecurityHeaders(config.CSPConfig{Enabled: true, Policy: config.DefaultCSPPolicy}, nil))
	reportHandler.RegisterRoutes(router, middleware.NewJWTAuthMiddleware(authService, userService, settingService, nil), nil)

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("Usage report service started on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("report server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("report server shutdown error: %v", err)
	}
}
