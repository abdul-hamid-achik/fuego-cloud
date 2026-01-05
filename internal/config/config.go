// Package config handles application configuration.
package config

import (
	"os"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	Port        int
	Host        string
	Environment string

	DatabaseURL string

	NeonAPIKey    string
	NeonProjectID string
	BranchID      string

	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string

	JWTSecret     string
	EncryptionKey string

	Kubeconfig         string
	K8sNamespacePrefix string

	CloudflareAPIToken string
	CloudflareZoneID   string

	GHCRToken string

	StripeSecretKey     string
	StripeWebhookSecret string

	PlatformDomain   string
	AppsDomainSuffix string
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 3000),
		Host:        getEnv("HOST", "0.0.0.0"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://neondb_owner@localhost:5432/neondb?sslmode=disable"),

		NeonAPIKey:    getEnv("NEON_API_KEY", ""),
		NeonProjectID: getEnv("NEON_PROJECT_ID", ""),
		BranchID:      getEnv("BRANCH_ID", ""),

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubCallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:3000/api/auth/callback"),

		JWTSecret:     getEnv("JWT_SECRET", ""),
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),

		Kubeconfig:         getEnv("KUBECONFIG", ""),
		K8sNamespacePrefix: getEnv("K8S_NAMESPACE_PREFIX", "tenant-"),

		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),

		GHCRToken: getEnv("GHCR_TOKEN", ""),

		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		PlatformDomain:   getEnv("PLATFORM_DOMAIN", "cloud.nexo.build"),
		AppsDomainSuffix: getEnv("APPS_DOMAIN_SUFFIX", "nexo.build"),
	}
}

// IsDevelopment checks if the environment is development.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction checks if the environment is production.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
