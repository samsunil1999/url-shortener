package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samsunil1999/url-shortener/internal/handler"
	"github.com/samsunil1999/url-shortener/internal/middleware"
	"github.com/samsunil1999/url-shortener/internal/repository"
	"github.com/samsunil1999/url-shortener/internal/service"
	"github.com/samsunil1999/url-shortener/pkg/cache"
	"github.com/samsunil1999/url-shortener/pkg/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Init DB
	db, err := database.NewPostgres(os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Init Redis
	rdb, err := cache.NewRedis(os.Getenv("REDIS_URL"))
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Wire up layers
	repo := repository.NewURLRepository(db)
	analyticRepo := repository.NewAnalyticsRepository(db)
	svc := service.NewURLService(repo, analyticRepo, rdb, logger)
	h := handler.NewHandler(svc, logger)

	// Start expiration worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartExpirationWorker(ctx)

	// Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.RateLimit(rdb))

	// Routes
	r.GET("/health", h.Health)
	r.POST("/api/shorten", h.Shorten)
	r.GET("/:code", h.Redirect)
	r.GET("/api/urls/:code/stats", h.GetStats)
	r.DELETE("/api/urls/:code", h.Delete)

	srv := &http.Server{
		Addr:    ":" + getEnv("PORT", "8080"),
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server starting", "port", getEnv("PORT", "8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
	slog.Info("server stopped")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
