package apps

import (
	"context"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)

	userID, userName, err := getUserInfo(c, cfg)
	if err != nil {
		return c.Redirect("/login", 302)
	}

	queries := db.New(pool)
	apps, err := queries.ListAppsByUser(context.Background(), db.ListAppsByUserParams{
		UserID: userID,
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		apps = []db.App{}
	}

	appList := make([]AppItem, 0, len(apps))
	for _, app := range apps {
		appList = append(appList, AppItem{
			ID:              app.ID.String(),
			Name:            app.Name,
			Status:          app.Status,
			Region:          app.Region,
			Size:            app.Size,
			DeploymentCount: int(app.DeploymentCount),
			URL:             "https://" + app.Name + "." + cfg.AppsDomainSuffix,
		})
	}

	data := AppsPageData{
		UserName: userName,
		Apps:     appList,
	}

	return fuego.TemplComponent(c, 200, Page(data))
}

func getUserInfo(c *fuego.Context, cfg *config.Config) (uuid.UUID, string, error) {
	tokenString := c.Cookie("access_token")
	if tokenString == "" {
		tokenString = auth.ExtractBearerToken(c.Header("Authorization"))
	}

	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
	if err != nil {
		return uuid.Nil, "", err
	}

	return claims.UserID, claims.Username, nil
}
