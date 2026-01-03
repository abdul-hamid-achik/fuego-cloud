package domain

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	domainName := c.Param("domain")

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

	domain, err := queries.GetDomainByName(context.Background(), domainName)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	if domain.AppID != app.ID {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	return c.JSON(200, toDomainResponse(domain))
}

func Delete(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")
	domainName := c.Param("domain")

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

	domain, err := queries.GetDomainByName(context.Background(), domainName)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	if domain.AppID != app.ID {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	err = queries.DeleteDomain(context.Background(), domain.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete domain"})
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
