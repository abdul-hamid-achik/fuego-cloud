package dashboard

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, userName, err := getUserInfo(c, cfg)
	if err != nil {
		return c.Redirect("/login", 302)
	}

	queries := db.New(pool)
	apps, _ := queries.ListAppsByUser(context.Background(), db.ListAppsByUserParams{
		UserID: userID,
		Limit:  5,
		Offset: 0,
	})

	var runningCount int
	recentApps := make([]AppSummary, 0, len(apps))
	for _, app := range apps {
		if app.Status == "running" || app.Status == "ready" {
			runningCount++
		}
		recentApps = append(recentApps, AppSummary{
			Name:      app.Name,
			Status:    app.Status,
			Region:    app.Region,
			UpdatedAt: formatTime(app.UpdatedAt),
		})
	}

	totalApps, _ := queries.CountAppsByUser(context.Background(), userID)

	data := DashboardData{
		UserName:    userName,
		TotalApps:   int(totalApps),
		RunningApps: runningCount,
		RecentApps:  recentApps,
	}

	return fuego.TemplComponent(c, 200, Page(data))
}

func getUserInfo(c *fuego.Context, cfg *config.Config) (uuid.UUID, string, error) {
	tokenString := c.Cookie("access_token")
	if tokenString == "" {
		tokenString = auth.ExtractBearerToken(c.Header("Authorization"))
	}

	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
	if err != nil {
		return uuid.Nil, "", err
	}

	return claims.UserID, claims.Username, nil
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return intToString(mins) + " minutes ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return intToString(hours) + " hours ago"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return intToString(days) + " days ago"
	default:
		return t.Format("Jan 2, 2006")
	}
}
