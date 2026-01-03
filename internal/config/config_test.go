package config

import (
	"os"
	"testing"
)

// Helper to clear all environment variables used by config
func clearConfigEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"PORT", "HOST", "ENVIRONMENT", "DATABASE_URL",
		"NEON_API_KEY", "NEON_PROJECT_ID", "BRANCH_ID",
		"GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "GITHUB_CALLBACK_URL",
		"JWT_SECRET", "ENCRYPTION_KEY",
		"KUBECONFIG", "K8S_NAMESPACE_PREFIX",
		"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ZONE_ID",
		"GHCR_TOKEN",
		"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET",
		"PLATFORM_DOMAIN", "APPS_DOMAIN_SUFFIX",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	clearConfigEnv(t)

	cfg := Load()

	// Server defaults
	if cfg.Port != 3000 {
		t.Errorf("expected default Port 3000, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected default Host '0.0.0.0', got %q", cfg.Host)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default Environment 'development', got %q", cfg.Environment)
	}

	// Database default
	if cfg.DatabaseURL != "postgres://neondb_owner@localhost:5432/neondb?sslmode=disable" {
		t.Errorf("expected default DatabaseURL, got %q", cfg.DatabaseURL)
	}

	// K8s defaults
	if cfg.K8sNamespacePrefix != "tenant-" {
		t.Errorf("expected default K8sNamespacePrefix 'tenant-', got %q", cfg.K8sNamespacePrefix)
	}

	// Platform defaults
	if cfg.PlatformDomain != "cloud.fuego.build" {
		t.Errorf("expected default PlatformDomain 'cloud.fuego.build', got %q", cfg.PlatformDomain)
	}
	if cfg.AppsDomainSuffix != "fuego.build" {
		t.Errorf("expected default AppsDomainSuffix 'fuego.build', got %q", cfg.AppsDomainSuffix)
	}

	// OAuth default callback
	if cfg.GitHubCallbackURL != "http://localhost:3000/api/auth/callback" {
		t.Errorf("expected default GitHubCallbackURL, got %q", cfg.GitHubCallbackURL)
	}

	// Empty string defaults for optional credentials
	if cfg.NeonAPIKey != "" {
		t.Errorf("expected empty NeonAPIKey, got %q", cfg.NeonAPIKey)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("expected empty JWTSecret, got %q", cfg.JWTSecret)
	}
	if cfg.EncryptionKey != "" {
		t.Errorf("expected empty EncryptionKey, got %q", cfg.EncryptionKey)
	}
	if cfg.Kubeconfig != "" {
		t.Errorf("expected empty Kubeconfig, got %q", cfg.Kubeconfig)
	}
}

func TestLoad_PortFromEnv(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "8080")

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}
}

func TestLoad_PortInvalidFallback(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "not-a-number")

	cfg := Load()

	if cfg.Port != 3000 {
		t.Errorf("expected fallback Port 3000 for invalid value, got %d", cfg.Port)
	}
}

func TestLoad_PortEmptyFallback(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "")

	cfg := Load()

	if cfg.Port != 3000 {
		t.Errorf("expected fallback Port 3000 for empty value, got %d", cfg.Port)
	}
}

func TestLoad_AllEnvVars(t *testing.T) {
	clearConfigEnv(t)

	// Set all env vars
	t.Setenv("PORT", "9000")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@prod:5432/db")
	t.Setenv("NEON_API_KEY", "neon-key")
	t.Setenv("NEON_PROJECT_ID", "neon-project")
	t.Setenv("BRANCH_ID", "branch-123")
	t.Setenv("GITHUB_CLIENT_ID", "gh-client")
	t.Setenv("GITHUB_CLIENT_SECRET", "gh-secret")
	t.Setenv("GITHUB_CALLBACK_URL", "https://prod.com/callback")
	t.Setenv("JWT_SECRET", "jwt-secret-key")
	t.Setenv("ENCRYPTION_KEY", "32-byte-encryption-key-here!!!!!")
	t.Setenv("KUBECONFIG", "/path/to/kubeconfig")
	t.Setenv("K8S_NAMESPACE_PREFIX", "prod-")
	t.Setenv("CLOUDFLARE_API_TOKEN", "cf-token")
	t.Setenv("CLOUDFLARE_ZONE_ID", "cf-zone")
	t.Setenv("GHCR_TOKEN", "ghcr-token")
	t.Setenv("STRIPE_SECRET_KEY", "stripe-key")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "stripe-webhook")
	t.Setenv("PLATFORM_DOMAIN", "cloud.prod.com")
	t.Setenv("APPS_DOMAIN_SUFFIX", "apps.prod.com")

	cfg := Load()

	// Verify all values
	if cfg.Port != 9000 {
		t.Errorf("expected Port 9000, got %d", cfg.Port)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host '127.0.0.1', got %q", cfg.Host)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got %q", cfg.Environment)
	}
	if cfg.DatabaseURL != "postgres://user:pass@prod:5432/db" {
		t.Errorf("expected DatabaseURL from env, got %q", cfg.DatabaseURL)
	}
	if cfg.NeonAPIKey != "neon-key" {
		t.Errorf("expected NeonAPIKey 'neon-key', got %q", cfg.NeonAPIKey)
	}
	if cfg.NeonProjectID != "neon-project" {
		t.Errorf("expected NeonProjectID 'neon-project', got %q", cfg.NeonProjectID)
	}
	if cfg.BranchID != "branch-123" {
		t.Errorf("expected BranchID 'branch-123', got %q", cfg.BranchID)
	}
	if cfg.GitHubClientID != "gh-client" {
		t.Errorf("expected GitHubClientID 'gh-client', got %q", cfg.GitHubClientID)
	}
	if cfg.GitHubClientSecret != "gh-secret" {
		t.Errorf("expected GitHubClientSecret 'gh-secret', got %q", cfg.GitHubClientSecret)
	}
	if cfg.GitHubCallbackURL != "https://prod.com/callback" {
		t.Errorf("expected GitHubCallbackURL 'https://prod.com/callback', got %q", cfg.GitHubCallbackURL)
	}
	if cfg.JWTSecret != "jwt-secret-key" {
		t.Errorf("expected JWTSecret 'jwt-secret-key', got %q", cfg.JWTSecret)
	}
	if cfg.EncryptionKey != "32-byte-encryption-key-here!!!!!" {
		t.Errorf("expected EncryptionKey from env, got %q", cfg.EncryptionKey)
	}
	if cfg.Kubeconfig != "/path/to/kubeconfig" {
		t.Errorf("expected Kubeconfig '/path/to/kubeconfig', got %q", cfg.Kubeconfig)
	}
	if cfg.K8sNamespacePrefix != "prod-" {
		t.Errorf("expected K8sNamespacePrefix 'prod-', got %q", cfg.K8sNamespacePrefix)
	}
	if cfg.CloudflareAPIToken != "cf-token" {
		t.Errorf("expected CloudflareAPIToken 'cf-token', got %q", cfg.CloudflareAPIToken)
	}
	if cfg.CloudflareZoneID != "cf-zone" {
		t.Errorf("expected CloudflareZoneID 'cf-zone', got %q", cfg.CloudflareZoneID)
	}
	if cfg.GHCRToken != "ghcr-token" {
		t.Errorf("expected GHCRToken 'ghcr-token', got %q", cfg.GHCRToken)
	}
	if cfg.StripeSecretKey != "stripe-key" {
		t.Errorf("expected StripeSecretKey 'stripe-key', got %q", cfg.StripeSecretKey)
	}
	if cfg.StripeWebhookSecret != "stripe-webhook" {
		t.Errorf("expected StripeWebhookSecret 'stripe-webhook', got %q", cfg.StripeWebhookSecret)
	}
	if cfg.PlatformDomain != "cloud.prod.com" {
		t.Errorf("expected PlatformDomain 'cloud.prod.com', got %q", cfg.PlatformDomain)
	}
	if cfg.AppsDomainSuffix != "apps.prod.com" {
		t.Errorf("expected AppsDomainSuffix 'apps.prod.com', got %q", cfg.AppsDomainSuffix)
	}
}

func TestIsDevelopment_True(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ENVIRONMENT", "development")

	cfg := Load()

	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return true for 'development'")
	}
}

func TestIsDevelopment_False(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ENVIRONMENT", "production")

	cfg := Load()

	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return false for 'production'")
	}
}

func TestIsDevelopment_DefaultIsTrue(t *testing.T) {
	clearConfigEnv(t)

	cfg := Load()

	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return true by default")
	}
}

func TestIsProduction_True(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ENVIRONMENT", "production")

	cfg := Load()

	if !cfg.IsProduction() {
		t.Error("expected IsProduction() to return true for 'production'")
	}
}

func TestIsProduction_False(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ENVIRONMENT", "development")

	cfg := Load()

	if cfg.IsProduction() {
		t.Error("expected IsProduction() to return false for 'development'")
	}
}

func TestIsProduction_DefaultIsFalse(t *testing.T) {
	clearConfigEnv(t)

	cfg := Load()

	if cfg.IsProduction() {
		t.Error("expected IsProduction() to return false by default")
	}
}

func TestIsProduction_StagingIsFalse(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ENVIRONMENT", "staging")

	cfg := Load()

	if cfg.IsProduction() {
		t.Error("expected IsProduction() to return false for 'staging'")
	}
	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return false for 'staging'")
	}
}

func TestGetEnv_EmptyReturnsDefault(t *testing.T) {
	clearConfigEnv(t)

	result := getEnv("NONEXISTENT_VAR", "default-value")

	if result != "default-value" {
		t.Errorf("expected 'default-value', got %q", result)
	}
}

func TestGetEnv_ValueReturnsValue(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_VAR", "actual-value")

	result := getEnv("TEST_VAR", "default-value")

	if result != "actual-value" {
		t.Errorf("expected 'actual-value', got %q", result)
	}
}

func TestGetEnv_EmptyStringUsesDefault(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_VAR", "")

	result := getEnv("TEST_VAR", "default-value")

	// Empty string should use default
	if result != "default-value" {
		t.Errorf("expected 'default-value' for empty string, got %q", result)
	}
}

func TestGetEnvInt_ValidInt(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", "8080")

	result := getEnvInt("TEST_INT", 3000)

	if result != 8080 {
		t.Errorf("expected 8080, got %d", result)
	}
}

func TestGetEnvInt_InvalidInt(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", "not-a-number")

	result := getEnvInt("TEST_INT", 3000)

	if result != 3000 {
		t.Errorf("expected default 3000 for invalid int, got %d", result)
	}
}

func TestGetEnvInt_EmptyReturnsDefault(t *testing.T) {
	clearConfigEnv(t)

	result := getEnvInt("NONEXISTENT_INT", 3000)

	if result != 3000 {
		t.Errorf("expected default 3000, got %d", result)
	}
}

func TestGetEnvInt_NegativeInt(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", "-1")

	result := getEnvInt("TEST_INT", 3000)

	if result != -1 {
		t.Errorf("expected -1, got %d", result)
	}
}

func TestGetEnvInt_Zero(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", "0")

	result := getEnvInt("TEST_INT", 3000)

	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}

func TestGetEnvInt_Whitespace(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", " 8080 ")

	result := getEnvInt("TEST_INT", 3000)

	// strconv.Atoi doesn't trim whitespace, so this should fail and return default
	if result != 3000 {
		t.Errorf("expected default 3000 for whitespace-padded int, got %d", result)
	}
}

func TestGetEnvInt_Float(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("TEST_INT", "3.14")

	result := getEnvInt("TEST_INT", 3000)

	// Float should fail to parse as int
	if result != 3000 {
		t.Errorf("expected default 3000 for float, got %d", result)
	}
}

func TestLoad_Concurrency(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "5000")
	t.Setenv("ENVIRONMENT", "test")

	// Load config concurrently to check for race conditions
	done := make(chan *Config, 10)
	for i := 0; i < 10; i++ {
		go func() {
			done <- Load()
		}()
	}

	configs := make([]*Config, 10)
	for i := 0; i < 10; i++ {
		configs[i] = <-done
	}

	// All configs should have the same values
	for i := 1; i < len(configs); i++ {
		if configs[i].Port != configs[0].Port {
			t.Errorf("config %d has different Port", i)
		}
		if configs[i].Environment != configs[0].Environment {
			t.Errorf("config %d has different Environment", i)
		}
	}
}
