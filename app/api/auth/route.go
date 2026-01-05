package auth

import (
	"context"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoginRequest struct {
	RedirectURI      string `json:"redirect_uri" query:"redirect_uri"`
	CLITokenExchange bool   `json:"cli_token_exchange" query:"cli"`
}

type LoginResponse struct {
	RedirectURL string `json:"redirect_url"`
}

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/"
	}
	cliTokenExchange := c.Query("cli") == "true"

	state, err := auth.GenerateState()
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to generate state"})
	}

	queries := db.New(pool)
	expiresAt := time.Now().Add(10 * time.Minute)

	_, err = queries.CreateOAuthState(context.Background(), db.CreateOAuthStateParams{
		State:            state,
		RedirectUri:      &redirectURI,
		CliTokenExchange: &cliTokenExchange,
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create oauth state"})
	}

	ghClient := auth.NewGitHubClient(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubCallbackURL)
	authURL := ghClient.GetAuthURL(state)

	return c.Redirect(authURL, 302)
}
