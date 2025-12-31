package id

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeploymentResponse struct {
	ID        string     `json:"id"`
	AppID     string     `json:"app_id"`
	Version   int        `json:"version"`
	Image     string     `json:"image"`
	Status    string     `json:"status"`
	Message   *string    `json:"message,omitempty"`
	Error     *string    `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	ReadyAt   *time.Time `json:"ready_at,omitempty"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")
	deploymentID := c.Param("id")

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

	depID, err := uuid.Parse(deploymentID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid deployment id"})
	}

	deployment, err := queries.GetDeploymentByID(context.Background(), depID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "deployment not found"})
	}

	if deployment.AppID != app.ID {
		return c.JSON(404, map[string]string{"error": "deployment not found"})
	}

	return c.JSON(200, toDeploymentResponse(deployment))
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")
	deploymentID := c.Param("id")

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

	depID, err := uuid.Parse(deploymentID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid deployment id"})
	}

	deployment, err := queries.GetDeploymentByID(context.Background(), depID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "deployment not found"})
	}

	if deployment.AppID != app.ID {
		return c.JSON(404, map[string]string{"error": "deployment not found"})
	}

	newDeployment, err := queries.CreateDeployment(context.Background(), db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: deployment.Version + 1,
		Image:   deployment.Image,
		Status:  "pending",
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create rollback deployment"})
	}

	_, err = queries.IncrementDeploymentCount(context.Background(), app.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update app"})
	}

	_, err = queries.UpdateAppStatus(context.Background(), db.UpdateAppStatusParams{
		ID:                  app.ID,
		Status:              "deploying",
		CurrentDeploymentID: pgtype.UUID{Bytes: newDeployment.ID, Valid: true},
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update app status"})
	}

	return c.JSON(201, toDeploymentResponse(newDeployment))
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

func toDeploymentResponse(d db.Deployment) DeploymentResponse {
	resp := DeploymentResponse{
		ID:        d.ID.String(),
		AppID:     d.AppID.String(),
		Version:   int(d.Version),
		Image:     d.Image,
		Status:    d.Status,
		Message:   d.Message,
		Error:     d.Error,
		CreatedAt: d.CreatedAt,
	}

	if d.StartedAt.Valid {
		resp.StartedAt = &d.StartedAt.Time
	}

	if d.ReadyAt.Valid {
		resp.ReadyAt = &d.ReadyAt.Time
	}

	return resp
}
