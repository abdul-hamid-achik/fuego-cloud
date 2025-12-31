package health

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

func Get(c *fuego.Context) error {
	response := HealthResponse{
		Status:  "ok",
		Version: "0.1.0",
	}

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

	statusCode := 200
	if response.Status != "ok" {
		statusCode = 503
	}

	return c.JSON(statusCode, response)
}
