package domains

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

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

type CreateDomainRequest struct {
	Domain string `json:"domain"`
}

type DomainResponse struct {
	ID         string     `json:"id"`
	Domain     string     `json:"domain"`
	Verified   bool       `json:"verified"`
	SSLStatus  string     `json:"ssl_status"`
	CreatedAt  time.Time  `json:"created_at"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
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

	domains, err := queries.ListDomainsByApp(context.Background(), app.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list domains"})
	}

	response := make([]DomainResponse, len(domains))
	for i, d := range domains {
		response[i] = toDomainResponse(d)
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

	var req CreateDomainRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Domain == "" {
		return c.JSON(400, map[string]string{"error": "domain is required"})
	}

	if !domainRegex.MatchString(req.Domain) {
		return c.JSON(400, map[string]string{"error": "invalid domain format"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	_, err = queries.GetDomainByName(context.Background(), req.Domain)
	if err == nil {
		return c.JSON(409, map[string]string{"error": "domain already exists"})
	}

	domain, err := queries.CreateDomain(context.Background(), db.CreateDomainParams{
		AppID:  app.ID,
		Domain: req.Domain,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create domain"})
	}

	return c.JSON(201, toDomainResponse(domain))
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

func toDomainResponse(d db.Domain) DomainResponse {
	resp := DomainResponse{
		ID:        d.ID.String(),
		Domain:    d.Domain,
		Verified:  d.Verified,
		SSLStatus: d.SslStatus,
		CreatedAt: d.CreatedAt,
	}

	if d.VerifiedAt.Valid {
		resp.VerifiedAt = &d.VerifiedAt.Time
	}

	return resp
}
