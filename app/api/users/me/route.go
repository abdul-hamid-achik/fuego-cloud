package me

import (
	"context"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
	Plan      string  `json:"plan"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	tokenString := auth.ExtractBearerToken(c.Header("Authorization"))
	if tokenString == "" {
		tokenString = c.Cookie("access_token")
	}

	if tokenString == "" {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "invalid token"})
	}

	queries := db.New(pool)
	user, err := queries.GetUserByID(context.Background(), claims.UserID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "user not found"})
	}

	return c.JSON(200, UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		AvatarURL: user.AvatarUrl,
		Plan:      user.Plan,
	})
}

// UpdateUserRequest represents the update request body
type UpdateUserRequest struct {
	Email *string `json:"email,omitempty"`
}

// Put updates the current user's profile
// PUT /api/users/me
func Put(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	tokenString := auth.ExtractBearerToken(c.Header("Authorization"))
	if tokenString == "" {
		tokenString = c.Cookie("access_token")
	}

	if tokenString == "" {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "invalid token"})
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	queries := db.New(pool)

	// Get current user
	user, err := queries.GetUserByID(context.Background(), claims.UserID)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "user not found"})
	}

	// Update email if provided
	if req.Email != nil && *req.Email != "" {
		err = queries.UpdateUserEmail(context.Background(), db.UpdateUserEmailParams{
			ID:    user.ID,
			Email: *req.Email,
		})
		if err != nil {
			return c.JSON(500, map[string]string{"error": "failed to update email"})
		}
		user.Email = *req.Email
	}

	return c.JSON(200, UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		AvatarURL: user.AvatarUrl,
		Plan:      user.Plan,
	})
}
