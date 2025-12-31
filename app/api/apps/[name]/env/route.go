package env

import (
	"context"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/crypto"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EnvVarsResponse struct {
	Variables map[string]string `json:"variables"`
	Count     int               `json:"count"`
}

type UpdateEnvVarsRequest struct {
	Variables map[string]string `json:"variables"`
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

	redacted := c.Query("redacted") != "false"

	if len(app.EnvVarsEncrypted) == 0 {
		return c.JSON(200, EnvVarsResponse{
			Variables: make(map[string]string),
			Count:     0,
		})
	}

	envVars, err := crypto.Decrypt(app.EnvVarsEncrypted, cfg.EncryptionKey)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to decrypt environment variables"})
	}

	if redacted {
		for key := range envVars {
			envVars[key] = "••••••••"
		}
	}

	return c.JSON(200, EnvVarsResponse{
		Variables: envVars,
		Count:     len(envVars),
	})
}

func Put(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req UpdateEnvVarsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	encrypted, err := crypto.Encrypt(req.Variables, cfg.EncryptionKey)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to encrypt environment variables"})
	}

	_, err = queries.UpdateAppEnvVars(context.Background(), db.UpdateAppEnvVarsParams{
		ID:               app.ID,
		EnvVarsEncrypted: encrypted,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update environment variables"})
	}

	redactedVars := make(map[string]string)
	for key := range req.Variables {
		redactedVars[key] = "••••••••"
	}

	return c.JSON(200, EnvVarsResponse{
		Variables: redactedVars,
		Count:     len(req.Variables),
	})
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
