package name

import (
	"context"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/auth"
	"github.com/abdul-hamid-achik/nexo-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Get renders the app detail page
// GET /apps/{name}
func Get(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")
	activeTab := c.Query("tab")

	userID, userName, err := getUserInfo(c, cfg)
	if err != nil {
		return c.Redirect("/login", 302)
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.Redirect("/dashboard/apps", 302)
	}

	// Get deployments
	deployments, _ := queries.ListDeploymentsByApp(context.Background(), db.ListDeploymentsByAppParams{
		AppID:  app.ID,
		Limit:  50,
		Offset: 0,
	})

	// Get domains
	domains, _ := queries.ListDomainsByApp(context.Background(), app.ID)

	// Convert to template data
	appData := AppData{
		ID:              app.ID.String(),
		Name:            app.Name,
		Status:          app.Status,
		Region:          app.Region,
		Size:            app.Size,
		DeploymentCount: int(app.DeploymentCount),
		URL:             "https://" + app.Name + "." + cfg.AppsDomainSuffix,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}

	deploymentData := make([]DeploymentData, len(deployments))
	for i, d := range deployments {
		dd := DeploymentData{
			ID:        d.ID.String(),
			Version:   int(d.Version),
			Image:     d.Image,
			Status:    d.Status,
			CreatedAt: d.CreatedAt,
		}
		if d.Message != nil {
			dd.Message = *d.Message
		}
		if d.Error != nil {
			dd.Error = *d.Error
		}
		if d.StartedAt.Valid {
			dd.StartedAt = &d.StartedAt.Time
		}
		if d.ReadyAt.Valid {
			dd.ReadyAt = &d.ReadyAt.Time
		}
		deploymentData[i] = dd
	}

	domainData := make([]DomainData, len(domains))
	for i, d := range domains {
		dd := DomainData{
			ID:        d.ID.String(),
			Domain:    d.Domain,
			Verified:  d.Verified,
			SSLStatus: d.SslStatus,
			CreatedAt: d.CreatedAt,
		}
		if d.VerifiedAt.Valid {
			dd.VerifiedAt = &d.VerifiedAt.Time
		}
		domainData[i] = dd
	}

	data := AppDetailData{
		UserName:    userName,
		App:         appData,
		Deployments: deploymentData,
		Domains:     domainData,
		ActiveTab:   activeTab,
	}

	// Check if this is an HTMX request for just the tab content
	if c.Header("HX-Request") == "true" && c.Header("HX-Target") == "tab-content" {
		switch activeTab {
		case "deployments":
			return fuego.TemplComponent(c, 200, DeploymentsTab(appData.Name, deploymentData))
		case "domains":
			return fuego.TemplComponent(c, 200, DomainsTab(appData.Name, domainData))
		case "settings":
			return fuego.TemplComponent(c, 200, SettingsTab(appData))
		default:
			return fuego.TemplComponent(c, 200, OverviewTab(appData, deploymentData))
		}
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
