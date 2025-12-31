package callback

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		return c.JSON(400, map[string]string{
			"error":       errorParam,
			"description": errorDesc,
		})
	}

	if code == "" || state == "" {
		return c.JSON(400, map[string]string{"error": "missing code or state"})
	}

	queries := db.New(pool)

	oauthState, err := queries.GetOAuthState(context.Background(), state)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid or expired state"})
	}

	if time.Now().After(oauthState.ExpiresAt) {
		_ = queries.DeleteOAuthState(context.Background(), state)
		return c.JSON(400, map[string]string{"error": "state expired"})
	}

	_ = queries.DeleteOAuthState(context.Background(), state)

	ghClient := auth.NewGitHubClient(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubCallbackURL)

	token, err := ghClient.Exchange(context.Background(), code)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to exchange code for token"})
	}

	ghUser, err := ghClient.GetUser(context.Background(), token)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to get user from github"})
	}

	user, err := queries.GetUserByGitHubID(context.Background(), ghUser.ID)
	if err != nil {
		user, err = queries.CreateUser(context.Background(), db.CreateUserParams{
			GithubID:  ghUser.ID,
			Username:  ghUser.Login,
			Email:     ghUser.Email,
			AvatarUrl: &ghUser.AvatarURL,
		})
		if err != nil {
			return c.JSON(500, map[string]string{"error": "failed to create user"})
		}
	} else {
		user, err = queries.UpdateUser(context.Background(), db.UpdateUserParams{
			ID:        user.ID,
			Username:  ghUser.Login,
			Email:     ghUser.Email,
			AvatarUrl: &ghUser.AvatarURL,
		})
		if err != nil {
			return c.JSON(500, map[string]string{"error": "failed to update user"})
		}
	}

	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Username, cfg.JWTSecret)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate tokens"})
	}

	if oauthState.CliTokenExchange != nil && *oauthState.CliTokenExchange {
		return c.JSON(200, map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
			"token_type":    tokenPair.TokenType,
			"user": map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
		})
	}

	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		MaxAge:   int(time.Until(tokenPair.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   !cfg.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
	})

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/",
		MaxAge:   int(7 * 24 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   !cfg.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
	})

	redirectURI := "/"
	if oauthState.RedirectUri != nil && *oauthState.RedirectUri != "" {
		redirectURI = *oauthState.RedirectUri
	}

	parsedURL, err := url.Parse(redirectURI)
	if err != nil || (parsedURL.Host != "" && parsedURL.Host != c.Request.Host) {
		redirectURI = "/"
	}

	return c.Redirect(redirectURI, 302)
}
