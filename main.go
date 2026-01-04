package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdul-hamid-achik/fuego-cloud/app/api"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/cloudflare"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/k8s"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Warn("database not available, continuing without database", "error", err)
		pool = nil
	} else {
		slog.Info("connected to database")
	}

	// Initialize Kubernetes client
	var k8sClient *k8s.Client
	if cfg.Kubeconfig != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		k8sClient, err = k8s.NewClient(cfg.Kubeconfig, cfg.K8sNamespacePrefix)
		if err != nil {
			slog.Warn("kubernetes not available", "error", err)
		} else {
			slog.Info("connected to kubernetes")
		}
	}

	// Initialize Cloudflare client
	var cfClient *cloudflare.Client
	if cfg.CloudflareAPIToken != "" && cfg.CloudflareZoneID != "" {
		cfClient = cloudflare.NewClient(cfg.CloudflareAPIToken, cfg.CloudflareZoneID)
		slog.Info("cloudflare client initialized")
	}

	app := fuego.New()

	// Add security middleware stack
	app.Use(api.RecoveryMiddleware())        // Panic recovery (outermost)
	app.Use(api.RequestIDMiddleware())       // Request ID tracking
	app.Use(api.RequestLoggingMiddleware())  // Request logging
	app.Use(api.SecurityHeadersMiddleware()) // Security headers
	app.Use(api.RateLimitMiddleware())       // Rate limiting
	app.Use(api.CORSMiddleware([]string{     // CORS
		"http://localhost:3000",
		"http://localhost:5173",
		"https://cloud.fuego.build",
	}))

	// Inject dependencies
	app.Use(func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			c.Set("db", pool)
			c.Set("config", cfg)
			c.Set("k8s", k8sClient)
			c.Set("cloudflare", cfClient)
			return next(c)
		}
	})

	RegisterRoutes(app)

	app.Static("/static", "static")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		slog.Info("starting server", "host", cfg.Host, "port", cfg.Port)
		if err := app.Listen(addr); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
}
