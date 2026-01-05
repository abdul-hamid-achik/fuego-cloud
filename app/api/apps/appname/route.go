package name

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

type UpdateAppRequest struct {
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
	name := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	return c.JSON(200, toAppResponse(app, cfg.AppsDomainSuffix))
}

func Put(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	name := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req UpdateAppRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	region := app.Region
	if req.Region != "" {
		validRegions := map[string]bool{"gdl": true, "mex": true, "qro": true}
		if !validRegions[req.Region] {
			return c.JSON(400, map[string]string{"error": "invalid region"})
		}
		region = req.Region
	}

	size := app.Size
	if req.Size != "" {
		validSizes := map[string]bool{"starter": true, "pro": true, "enterprise": true}
		if !validSizes[req.Size] {
			return c.JSON(400, map[string]string{"error": "invalid size"})
		}
		size = req.Size
	}

	updatedApp, err := queries.UpdateApp(context.Background(), db.UpdateAppParams{
		ID:     app.ID,
		Name:   app.Name,
		Region: region,
		Size:   size,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update app"})
	}

	return c.JSON(200, toAppResponse(updatedApp, cfg.AppsDomainSuffix))
}

func Delete(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	name := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	err = queries.DeleteApp(context.Background(), app.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete app"})
	}

	return c.NoContent()
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
