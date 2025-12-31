package me

import (
	"context"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
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

func getUserIDFromContext(c *fuego.Context) (uuid.UUID, bool) {
	userID, ok := c.Get("user_id").(uuid.UUID)
	return userID, ok
}
