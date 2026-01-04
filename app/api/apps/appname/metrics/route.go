package metrics

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/k8s"
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
		switch d.Status {
		case "ready", "running":
			successful++
		case "failed":
			failed++
		}
		if lastDeploy.IsZero() || d.CreatedAt.After(lastDeploy) {
			lastDeploy = d.CreatedAt
		}
	}

	// Get real metrics from K8s if available
	var cpuCurrent, cpuAvg, memCurrent, memAvg float64
	var podCount, readyPods int

	if k8sClient, ok := c.Get("k8s").(*k8s.Client); ok && k8sClient != nil {
		if appMetrics, err := k8sClient.GetAppMetrics(context.Background(), app.Name); err == nil {
			cpuCurrent = appMetrics.TotalCPU * 100 // Convert to percentage (assuming 1 core = 100%)
			cpuAvg = appMetrics.AvgCPU * 100
			memCurrent = appMetrics.TotalMemoryMB
			memAvg = appMetrics.AvgMemoryMB
			podCount = appMetrics.PodCount
			readyPods = appMetrics.ReadyPods
		}
	}

	// Calculate uptime based on ready pods
	uptimePercent := 100.0
	if podCount > 0 {
		uptimePercent = (float64(readyPods) / float64(podCount)) * 100
	}

	response := MetricsResponse{
		AppName: app.Name,
		Period:  period,
		CPU: ResourceMetrics{
			Current: cpuCurrent,
			Average: cpuAvg,
			Peak:    cpuCurrent * 1.5, // Estimate peak as 1.5x current
			Unit:    "percent",
		},
		Memory: ResourceMetrics{
			Current: memCurrent,
			Average: memAvg,
			Peak:    memCurrent * 1.2, // Estimate peak as 1.2x current
			Unit:    "MB",
		},
		Network: NetworkMetrics{
			IngressBytes:  0, // Requires CNI metrics or service mesh
			EgressBytes:   0,
			RequestsTotal: 0,
		},
		Requests: RequestMetrics{
			Total:      0, // Requires Prometheus/service mesh integration
			PerSecond:  0,
			ByStatus:   map[string]int64{},
			AvgLatency: 0,
			P95Latency: 0,
			P99Latency: 0,
		},
		Deployments: DeploymentStats{
			Total:      len(deployments),
			Successful: successful,
			Failed:     failed,
			LastDeploy: lastDeploy,
		},
		Uptime: UptimeMetrics{
			Percentage:    uptimePercent,
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
