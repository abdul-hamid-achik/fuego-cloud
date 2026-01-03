package restart

import (
	"context"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/k8s"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RestartResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Post restarts an app
// POST /api/apps/{name}/restart
func Post(c *fuego.Context) error {
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

	// Get K8s client
	k8sClient, err := k8s.NewClient(cfg.Kubeconfig, cfg.K8sNamespacePrefix)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "kubernetes not available"})
	}

	// Restart the app
	if err := k8sClient.RestartApp(context.Background(), app.Name); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, RestartResponse{
		Success: true,
		Message: "restart initiated",
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
