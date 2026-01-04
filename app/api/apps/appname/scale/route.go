package scale

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

type ScaleRequest struct {
	Replicas int32 `json:"replicas"`
}

type ScaleResponse struct {
	Success  bool   `json:"success"`
	Replicas int32  `json:"replicas"`
	Message  string `json:"message"`
}

// Post scales an app
// POST /api/apps/{name}/scale
// Body: { "replicas": 3 }
func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	// Parse request body
	var req ScaleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	// Validate replicas
	if req.Replicas < 0 || req.Replicas > 10 {
		return c.JSON(400, map[string]string{"error": "replicas must be between 0 and 10"})
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

	// Scale the app
	if err := k8sClient.ScaleApp(context.Background(), app.Name, req.Replicas); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, ScaleResponse{
		Success:  true,
		Replicas: req.Replicas,
		Message:  "scaling initiated",
	})
}

// Get returns the current scale of an app
// GET /api/apps/{name}/scale
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

	// Get K8s client
	k8sClient, err := k8s.NewClient(cfg.Kubeconfig, cfg.K8sNamespacePrefix)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "kubernetes not available"})
	}

	// Get app status
	status, err := k8sClient.GetAppStatus(context.Background(), app.Name)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, status)
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
