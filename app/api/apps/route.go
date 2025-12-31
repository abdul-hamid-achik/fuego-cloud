package apps

import (
	"context"
	"regexp"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var appNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

type CreateAppRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Size   string `json:"size"`
}

type AppResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Region          string    `json:"region"`
	Size            string    `json:"size"`
	Status          string    `json:"status"`
	DeploymentCount int       `json:"deployment_count"`
	URL             string    `json:"url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	apps, err := queries.ListAppsByUser(context.Background(), db.ListAppsByUserParams{
		UserID: userID,
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list apps"})
	}

	response := make([]AppResponse, len(apps))
	for i, app := range apps {
		response[i] = toAppResponse(app, cfg.AppsDomainSuffix)
	}

	return c.JSON(200, response)
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req CreateAppRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name is required"})
	}

	if len(req.Name) < 3 || len(req.Name) > 63 {
		return c.JSON(400, map[string]string{"error": "name must be between 3 and 63 characters"})
	}

	if !appNameRegex.MatchString(req.Name) {
		return c.JSON(400, map[string]string{"error": "name must start with a letter, end with a letter or number, and contain only lowercase letters, numbers, and hyphens"})
	}

	if req.Region == "" {
		req.Region = "gdl"
	}

	if req.Size == "" {
		req.Size = "starter"
	}

	validRegions := map[string]bool{"gdl": true, "mex": true, "qro": true}
	if !validRegions[req.Region] {
		return c.JSON(400, map[string]string{"error": "invalid region"})
	}

	validSizes := map[string]bool{"starter": true, "pro": true, "enterprise": true}
	if !validSizes[req.Size] {
		return c.JSON(400, map[string]string{"error": "invalid size"})
	}

	queries := db.New(pool)

	_, err = queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   req.Name,
	})
	if err == nil {
		return c.JSON(409, map[string]string{"error": "app with this name already exists"})
	}

	app, err := queries.CreateApp(context.Background(), db.CreateAppParams{
		UserID: userID,
		Name:   req.Name,
		Region: req.Region,
		Size:   req.Size,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create app"})
	}

	return c.JSON(201, toAppResponse(app, cfg.AppsDomainSuffix))
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

func toAppResponse(app db.App, domainSuffix string) AppResponse {
	return AppResponse{
		ID:              app.ID.String(),
		Name:            app.Name,
		Region:          app.Region,
		Size:            app.Size,
		Status:          app.Status,
		DeploymentCount: int(app.DeploymentCount),
		URL:             "https://" + app.Name + "." + domainSuffix,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}
}
