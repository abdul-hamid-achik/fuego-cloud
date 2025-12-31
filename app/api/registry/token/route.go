package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in"`
}

type TokenResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

type TokenListResponse struct {
	Tokens []TokenResponse `json:"tokens"`
	Count  int             `json:"count"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	tokens, err := queries.ListAPITokensByUser(context.Background(), userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list tokens"})
	}

	response := make([]TokenResponse, len(tokens))
	for i, t := range tokens {
		response[i] = toTokenResponse(t, "")
	}

	return c.JSON(200, TokenListResponse{
		Tokens: response,
		Count:  len(response),
	})
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	var req CreateTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		req.Name = "Registry Token"
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate token"})
	}
	tokenStr := "fgc_" + hex.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(tokenStr))
	tokenHash := hex.EncodeToString(hash[:])

	var expiresAt pgtype.Timestamptz
	if req.ExpiresIn > 0 {
		expTime := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		expiresAt = pgtype.Timestamptz{Time: expTime, Valid: true}
	}

	queries := db.New(pool)
	token, err := queries.CreateAPIToken(context.Background(), db.CreateAPITokenParams{
		UserID:    userID,
		Name:      req.Name,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create token"})
	}

	return c.JSON(201, toTokenResponse(token, tokenStr))
}

func Delete(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	tokenID := c.Query("id")
	if tokenID == "" {
		return c.JSON(400, map[string]string{"error": "token id required"})
	}

	id, err := uuid.Parse(tokenID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid token id"})
	}

	queries := db.New(pool)
	token, err := queries.GetAPITokenByID(context.Background(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "token not found"})
	}

	if token.UserID != userID {
		return c.JSON(404, map[string]string{"error": "token not found"})
	}

	err = queries.DeleteAPIToken(context.Background(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete token"})
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

func toTokenResponse(t db.ApiToken, plainToken string) TokenResponse {
	resp := TokenResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Token:     plainToken,
		CreatedAt: t.CreatedAt,
	}

	if t.ExpiresAt.Valid {
		resp.ExpiresAt = &t.ExpiresAt.Time
	}

	if t.LastUsedAt.Valid {
		resp.LastUsed = &t.LastUsedAt.Time
	}

	return resp
}
