package metrics

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsResponse struct {
	AppName     string          `json:"app_name"`
	Period      string          `json:"period"`
	CPU         ResourceMetrics `json:"cpu"`
	Memory      ResourceMetrics `json:"memory"`
	Network     NetworkMetrics  `json:"network"`
	Requests    RequestMetrics  `json:"requests"`
	Deployments DeploymentStats `json:"deployments"`
	Uptime      UptimeMetrics   `json:"uptime"`
}

type ResourceMetrics struct {
	Current float64 `json:"current"`
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
	Unit    string  `json:"unit"`
}

type NetworkMetrics struct {
	IngressBytes  int64 `json:"ingress_bytes"`
	EgressBytes   int64 `json:"egress_bytes"`
	RequestsTotal int64 `json:"requests_total"`
}

type RequestMetrics struct {
	Total      int64            `json:"total"`
	PerSecond  float64          `json:"per_second"`
	ByStatus   map[string]int64 `json:"by_status"`
	AvgLatency float64          `json:"avg_latency_ms"`
	P95Latency float64          `json:"p95_latency_ms"`
	P99Latency float64          `json:"p99_latency_ms"`
}

type DeploymentStats struct {
	Total      int       `json:"total"`
	Successful int       `json:"successful"`
	Failed     int       `json:"failed"`
	LastDeploy time.Time `json:"last_deploy"`
}

type UptimeMetrics struct {
	Percentage    float64   `json:"percentage"`
	LastDowntime  time.Time `json:"last_downtime,omitempty"`
	CurrentStatus string    `json:"current_status"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	deployments, _ := queries.ListDeploymentsByApp(context.Background(), db.ListDeploymentsByAppParams{
		AppID:  app.ID,
		Limit:  100,
		Offset: 0,
	})

	var successful, failed int
	var lastDeploy time.Time
	for _, d := range deployments {
		if d.Status == "ready" || d.Status == "running" {
			successful++
		} else if d.Status == "failed" {
			failed++
		}
		if lastDeploy.IsZero() || d.CreatedAt.After(lastDeploy) {
			lastDeploy = d.CreatedAt
		}
	}

	response := MetricsResponse{
		AppName: app.Name,
		Period:  period,
		CPU: ResourceMetrics{
			Current: 12.5,
			Average: 8.3,
			Peak:    45.2,
			Unit:    "percent",
		},
		Memory: ResourceMetrics{
			Current: 128.0,
			Average: 96.0,
			Peak:    256.0,
			Unit:    "MB",
		},
		Network: NetworkMetrics{
			IngressBytes:  1024 * 1024 * 50,
			EgressBytes:   1024 * 1024 * 120,
			RequestsTotal: 15420,
		},
		Requests: RequestMetrics{
			Total:      15420,
			PerSecond:  2.3,
			ByStatus:   map[string]int64{"2xx": 14850, "3xx": 320, "4xx": 180, "5xx": 70},
			AvgLatency: 45.2,
			P95Latency: 120.5,
			P99Latency: 250.3,
		},
		Deployments: DeploymentStats{
			Total:      len(deployments),
			Successful: successful,
			Failed:     failed,
			LastDeploy: lastDeploy,
		},
		Uptime: UptimeMetrics{
			Percentage:    99.95,
			CurrentStatus: app.Status,
		},
	}

	return c.JSON(200, response)
}

func getUserID(c *fuego.Context, cfg *config.Config) (uuid.UUID, error) {
	if userID, ok := c.Get("user_id").(uuid.UUID); ok {
		return userID, nil
	}

	tokenString := auth.ExtractBearerToken(c.Header("Authorization"))
	if tokenString == "" {
		tokenString = c.Cookie("access_token")
	}

	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
	if err != nil {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}
