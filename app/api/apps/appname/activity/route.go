package activity

import (
	"context"
	"strconv"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityResponse struct {
	Activities []ActivityEntry `json:"activities"`
	Total      int64           `json:"total"`
	Limit      int32           `json:"limit"`
	Offset     int32           `json:"offset"`
}

type ActivityEntry struct {
	ID        uuid.UUID              `json:"id"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

// Get returns activity logs for an app
// GET /api/apps/{name}/activity
// Query params:
//   - limit: number of entries (default 50, max 100)
//   - offset: pagination offset (default 0)
func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	// Parse query parameters
	limit := int32(50)
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil && parsed > 0 && parsed <= 100 {
			limit = int32(parsed)
		}
	}

	offset := int32(0)
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 32); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
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

	// Convert UUID to pgtype.UUID
	appUUID := pgtype.UUID{Bytes: app.ID, Valid: true}

	// Get activity logs
	logs, err := queries.ListActivityLogsByApp(context.Background(), db.ListActivityLogsByAppParams{
		AppID:  appUUID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to get activity logs"})
	}

	// Get total count
	total, err := queries.CountActivityLogsByApp(context.Background(), appUUID)
	if err != nil {
		total = 0
	}

	// Convert to response format
	activities := make([]ActivityEntry, 0, len(logs))
	for _, log := range logs {
		entry := ActivityEntry{
			ID:        log.ID,
			Action:    log.Action,
			CreatedAt: log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Details is JSONB stored as []byte, needs to be parsed
		// For now, we'll leave it as nil if parsing fails
		_ = log.Details

		if log.IpAddress != nil {
			entry.IPAddress = log.IpAddress.String()
		}

		activities = append(activities, entry)
	}

	return c.JSON(200, ActivityResponse{
		Activities: activities,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	})
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
