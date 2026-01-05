// Package health provides health check endpoints.
package health

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/internal/k8s"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status     string `json:"status"`
	Database   string `json:"database"`
	Kubernetes string `json:"kubernetes"`
	Version    string `json:"version"`
}

// Get handles health check requests.
func Get(c *fuego.Context) error {
	response := HealthResponse{
		Status:  "ok",
		Version: "0.1.0",
	}

	// Check database
	pool, ok := c.Get("db").(*pgxpool.Pool)
	if !ok || pool == nil {
		response.Database = "disconnected"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			response.Database = "unhealthy"
			response.Status = "degraded"
		} else {
			response.Database = "healthy"
		}
	}

	// Check Kubernetes
	k8sClient, ok := c.Get("k8s").(*k8s.Client)
	if !ok || k8sClient == nil {
		response.Kubernetes = "disconnected"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := k8sClient.Clientset().Discovery().ServerVersion()
		if err != nil {
			response.Kubernetes = "unhealthy"
		} else {
			response.Kubernetes = "healthy"
		}
		_ = ctx // use context for potential future timeout
	}

	statusCode := 200
	if response.Status != "ok" {
		statusCode = 503
	}

	return c.JSON(statusCode, response)
}
