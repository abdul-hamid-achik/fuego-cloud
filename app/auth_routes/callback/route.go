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
		return c.Redirect("/login?error="+url.QueryEscape(errorParam), 302)
	}

	if code == "" || state == "" {
		return c.Redirect("/login?error=missing_params", 302)
	}

	queries := db.New(pool)

	oauthState, err := queries.GetOAuthState(context.Background(), state)
	if err != nil {
		return c.Redirect("/login?error=invalid_state", 302)
	}

	if time.Now().After(oauthState.ExpiresAt) {
		_ = queries.DeleteOAuthState(context.Background(), state)
		return c.Redirect("/login?error=state_expired", 302)
	}

	_ = queries.DeleteOAuthState(context.Background(), state)

	ghClient := auth.NewGitHubClient(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubCallbackURL)

	token, err := ghClient.Exchange(context.Background(), code)
	if err != nil {
		return c.Redirect("/login?error=exchange_failed", 302)
	}

	ghUser, err := ghClient.GetUser(context.Background(), token)
	if err != nil {
		return c.Redirect("/login?error=github_error", 302)
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
			return c.Redirect("/login?error=create_user_failed", 302)
		}
	} else {
		user, err = queries.UpdateUser(context.Background(), db.UpdateUserParams{
			ID:        user.ID,
			Username:  ghUser.Login,
			Email:     ghUser.Email,
			AvatarUrl: &ghUser.AvatarURL,
		})
		if err != nil {
			return c.Redirect("/login?error=update_user_failed", 302)
		}
	}

	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Username, cfg.JWTSecret)
	if err != nil {
		return c.Redirect("/login?error=token_generation_failed", 302)
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

	redirectURI := "/dashboard"
	if oauthState.RedirectUri != nil && *oauthState.RedirectUri != "" {
		redirectURI = *oauthState.RedirectUri
	}

	parsedURL, err := url.Parse(redirectURI)
	if err != nil || (parsedURL.Host != "" && parsedURL.Host != c.Request.Host) {
		redirectURI = "/dashboard"
	}

	return c.Redirect(redirectURI, 302)
}
