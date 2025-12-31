package api

import (
	"context"
	"strings"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func Middleware() fuego.MiddlewareFunc {
	return func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			path := c.Path()

			if auth.IsPublicPath(path) {
				return next(c)
			}

			cfg := c.Get("config").(*config.Config)
			pool := c.Get("db").(*pgxpool.Pool)

			tokenString := auth.ExtractBearerToken(c.Header("Authorization"))
			if tokenString == "" {
				tokenString = c.Cookie("access_token")
			}

			if tokenString == "" {
				return c.JSON(401, map[string]string{"error": "missing authorization"})
			}

			if strings.HasPrefix(tokenString, "fgt_") {
				return handleAPIToken(c, next, pool, tokenString)
			}

			claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "invalid token"})
			}

			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("claims", claims)

			return next(c)
		}
	}
}

func handleAPIToken(c *fuego.Context, next fuego.HandlerFunc, pool *pgxpool.Pool, token string) error {
	queries := db.New(pool)

	apiToken, err := searchAllTokens(pool, token)
	if err != nil || apiToken == nil {
		return c.JSON(401, map[string]string{"error": "invalid api token"})
	}

	if apiToken.ExpiresAt.Valid && apiToken.ExpiresAt.Time.Before(apiToken.CreatedAt) {
		return c.JSON(401, map[string]string{"error": "token expired"})
	}

	_ = queries.UpdateAPITokenLastUsed(context.Background(), apiToken.ID)

	user, err := queries.GetUserByID(context.Background(), apiToken.UserID)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "user not found"})
	}

	c.Set("user_id", user.ID)
	c.Set("username", user.Username)

	return next(c)
}

func searchAllTokens(pool *pgxpool.Pool, token string) (*db.ApiToken, error) {
	queries := db.New(pool)

	rows, err := pool.Query(context.Background(), "SELECT id, user_id, name, token_hash, last_used_at, expires_at, created_at FROM api_tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t db.ApiToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			continue
		}

		if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(token)); err == nil {
			_ = queries.UpdateAPITokenLastUsed(context.Background(), t.ID)
			return &t, nil
		}
	}

	return nil, nil
}
