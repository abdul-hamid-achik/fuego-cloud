package token

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type CreateTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in"`
}

type TokenResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	claims, err := auth.ValidateToken(auth.ExtractBearerToken(c.Header("Authorization")), cfg.JWTSecret)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req CreateTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		req.Name = "API Token"
	}

	token, err := auth.GenerateAPIToken()
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate token"})
	}

	hashedToken, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to hash token"})
	}

	var expiresAt pgtype.Timestamptz
	var expiresAtPtr *time.Time
	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		expiresAt = pgtype.Timestamptz{Time: exp, Valid: true}
		expiresAtPtr = &exp
	}

	queries := db.New(pool)
	apiToken, err := queries.CreateAPIToken(context.Background(), db.CreateAPITokenParams{
		UserID:    claims.UserID,
		Name:      req.Name,
		TokenHash: string(hashedToken),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create token"})
	}

	return c.JSON(201, TokenResponse{
		ID:        apiToken.ID.String(),
		Name:      apiToken.Name,
		Token:     token,
		CreatedAt: apiToken.CreatedAt,
		ExpiresAt: expiresAtPtr,
	})
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	claims, err := auth.ValidateToken(auth.ExtractBearerToken(c.Header("Authorization")), cfg.JWTSecret)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	tokens, err := queries.ListAPITokensByUser(context.Background(), claims.UserID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list tokens"})
	}

	response := make([]TokenResponse, len(tokens))
	for i, t := range tokens {
		var expiresAt *time.Time
		if t.ExpiresAt.Valid {
			expiresAt = &t.ExpiresAt.Time
		}
		response[i] = TokenResponse{
			ID:        t.ID.String(),
			Name:      t.Name,
			CreatedAt: t.CreatedAt,
			ExpiresAt: expiresAt,
		}
	}

	return c.JSON(200, response)
}
