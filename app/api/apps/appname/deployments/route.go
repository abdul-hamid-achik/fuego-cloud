package deployments

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateDeploymentRequest struct {
	Image string `json:"image"`
}

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

	deployments, err := queries.ListDeploymentsByApp(context.Background(), db.ListDeploymentsByAppParams{
		AppID:  app.ID,
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list deployments"})
	}

	response := make([]DeploymentResponse, len(deployments))
	for i, d := range deployments {
		response[i] = toDeploymentResponse(d)
	}

	return c.JSON(200, response)
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req CreateDeploymentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Image == "" {
		return c.JSON(400, map[string]string{"error": "image is required"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	latestDeployment, _ := queries.GetLatestDeployment(context.Background(), app.ID)
	nextVersion := int32(1)
	if latestDeployment.ID != uuid.Nil {
		nextVersion = latestDeployment.Version + 1
	}

	deployment, err := queries.CreateDeployment(context.Background(), db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: nextVersion,
		Image:   req.Image,
		Status:  "pending",
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create deployment"})
	}

	_, err = queries.IncrementDeploymentCount(context.Background(), app.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update app"})
	}

	_, err = queries.UpdateAppStatus(context.Background(), db.UpdateAppStatusParams{
		ID:                  app.ID,
		Status:              "deploying",
		CurrentDeploymentID: pgtype.UUID{Bytes: deployment.ID, Valid: true},
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update app status"})
	}

	return c.JSON(201, toDeploymentResponse(deployment))
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
