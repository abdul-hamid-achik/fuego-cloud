package verify

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VerifyResponse struct {
	Domain     string     `json:"domain"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	Message    string     `json:"message,omitempty"`
}

func Post(c *fuego.Context) error {
	cfg := c.Get("config").(*config.Config)
	pool := c.Get("db").(*pgxpool.Pool)
	appName := c.Param("name")
	domainName := c.Param("domain")

	userID, err := getUserID(c, cfg)
	if err != nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	queries := db.New(pool)
	app, err := queries.GetAppByName(context.Background(), db.GetAppByNameParams{
		UserID: userID,
		Name:   appName,
	})
	if err != nil {
		return c.JSON(404, map[string]string{"error": "app not found"})
	}

	domain, err := queries.GetDomainByName(context.Background(), domainName)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	if domain.AppID != app.ID {
		return c.JSON(404, map[string]string{"error": "domain not found"})
	}

	if domain.Verified {
		return c.JSON(200, VerifyResponse{
			Domain:     domain.Domain,
			Verified:   true,
			VerifiedAt: &domain.VerifiedAt.Time,
			Message:    "domain already verified",
		})
	}

	verified, err := verifyDNS(domainName, cfg.AppsDomainSuffix)
	if err != nil || !verified {
		return c.JSON(200, VerifyResponse{
			Domain:   domain.Domain,
			Verified: false,
			Message:  "DNS verification failed. Please ensure CNAME record points to " + cfg.AppsDomainSuffix,
		})
	}

	updatedDomain, err := queries.UpdateDomainVerified(context.Background(), domain.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update domain verification status"})
	}

	verifiedAt := updatedDomain.VerifiedAt.Time
	return c.JSON(200, VerifyResponse{
		Domain:     updatedDomain.Domain,
		Verified:   true,
		VerifiedAt: &verifiedAt,
		Message:    "domain verified successfully",
	})
}

func verifyDNS(domainName, expectedTarget string) (bool, error) {
	cname, err := net.LookupCNAME(domainName)
	if err != nil {
		return false, err
	}

	cname = strings.TrimSuffix(cname, ".")
	expectedTarget = strings.TrimSuffix(expectedTarget, ".")

	return strings.EqualFold(cname, expectedTarget), nil
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
