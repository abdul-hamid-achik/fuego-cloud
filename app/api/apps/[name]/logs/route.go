package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/k8s"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogsResponse struct {
	Logs []k8s.LogLine `json:"logs"`
}

// Get returns recent logs for an app
// GET /api/apps/{name}/logs
// Query params:
//   - tail: number of lines (default 100)
//   - follow: stream logs via SSE (default false)
func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	// Verify app ownership
	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	// Parse query parameters
	tailLines := int64(100)
	if t := c.Query("tail"); t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil && parsed > 0 {
			tailLines = parsed
		}
	}

	follow := c.Query("follow") == "true"

	// Get K8s client
	k8sClient, err := k8s.NewClient(cfg.Kubeconfig, cfg.K8sNamespacePrefix)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "kubernetes not available"})
	}

	if follow {
		return streamLogs(c, k8sClient, app.Name, tailLines)
	}

	// Get recent logs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logs, err := k8sClient.GetRecentLogs(ctx, app.Name, tailLines)
	if err != nil {
		return c.JSON(500, map[string]string{"error": fmt.Sprintf("failed to get logs: %v", err)})
	}

	return c.JSON(200, LogsResponse{Logs: logs})
}

// streamLogs streams logs via Server-Sent Events (SSE)
func streamLogs(c *fuego.Context, k8sClient *k8s.Client, appName string, tailLines int64) error {
	// Set SSE headers
	c.Response.Header().Set("Content-Type", "text/event-stream")
	c.Response.Header().Set("Cache-Control", "no-cache")
	c.Response.Header().Set("Connection", "keep-alive")
	c.Response.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := c.Response.(http.Flusher)
	if !ok {
		return c.JSON(500, map[string]string{"error": "streaming not supported"})
	}

	// Create context that cancels when client disconnects
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Channel to receive log lines
	logCh := make(chan k8s.LogLine, 100)

	// Start streaming logs in background
	go func() {
		opts := k8s.LogStreamOptions{
			Follow:     true,
			TailLines:  tailLines,
			Timestamps: true,
		}
		if err := k8sClient.StreamLogs(ctx, appName, opts, logCh); err != nil {
			// Log error but don't panic
			fmt.Printf("log stream error: %v\n", err)
		}
		close(logCh)
	}()

	// Stream logs to client
	for {
		select {
		case <-ctx.Done():
			return nil
		case log, ok := <-logCh:
			if !ok {
				return nil
			}
			data, _ := json.Marshal(log)
			fmt.Fprintf(c.Response, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func getUserID(c *fuego.Context, cfg *config.Config) (uuid.UUID, error) {
	if id, ok := c.Get("user_id").(uuid.UUID); ok {
		return id, nil
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
