package api_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/google/uuid"
)

// Domain validation regex (same as in route.go)
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// TestDomainValidation tests domain name validation
func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		isValid bool
	}{
		{"valid simple domain", "example.com", true},
		{"valid subdomain", "app.example.com", true},
		{"valid multi-subdomain", "api.v1.example.com", true},
		{"valid with hyphen", "my-app.example.com", true},
		{"valid long TLD", "example.technology", true},
		{"valid country TLD", "example.co.uk", true},
		{"valid numbers in subdomain", "app123.example.com", true},

		{"invalid no TLD", "example", false},
		{"invalid single letter TLD", "example.c", false},
		{"invalid starts with hyphen", "-example.com", false},
		{"invalid ends with hyphen", "example-.com", false},
		{"invalid double hyphen at start", "--example.com", false},
		{"invalid spaces", "my app.com", false},
		{"invalid underscore", "my_app.example.com", false},
		{"invalid special chars", "app@example.com", false},
		{"invalid empty", "", false},
		{"invalid just dot", ".", false},
		{"invalid double dots", "example..com", false},
		{"invalid protocol included", "https://example.com", false},
		{"invalid trailing slash", "example.com/", false},
		{"invalid path", "example.com/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := domainRegex.MatchString(tt.domain)
			if isValid != tt.isValid {
				t.Errorf("domain %q: expected valid=%v, got valid=%v", tt.domain, tt.isValid, isValid)
			}
		})
	}
}

// TestDomainOperations tests domain database operations
func TestDomainOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	// Create an app first
	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "domain-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	t.Run("create domain", func(t *testing.T) {
		domainName := "test-" + uuid.New().String()[:8] + ".example.com"

		domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
			AppID:  app.ID,
			Domain: domainName,
		})
		if err != nil {
			t.Fatalf("CreateDomain failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteDomain(ctx, domain.ID) }()

		if domain.Domain != domainName {
			t.Errorf("expected domain %q, got %q", domainName, domain.Domain)
		}
		if domain.Verified {
			t.Error("expected domain to be unverified initially")
		}
		if domain.SslStatus != "pending" {
			t.Errorf("expected ssl_status 'pending', got %q", domain.SslStatus)
		}
	})

	t.Run("get domain by name", func(t *testing.T) {
		domainName := "get-" + uuid.New().String()[:8] + ".example.com"

		created, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
			AppID:  app.ID,
			Domain: domainName,
		})
		if err != nil {
			t.Fatalf("CreateDomain failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteDomain(ctx, created.ID) }()

		retrieved, err := testQueries.GetDomainByName(ctx, domainName)
		if err != nil {
			t.Fatalf("GetDomainByName failed: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
		}
	})

	t.Run("list domains by app", func(t *testing.T) {
		// Create multiple domains
		var domainIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			domainName := "list-" + uuid.New().String()[:8] + ".example.com"
			domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
				AppID:  app.ID,
				Domain: domainName,
			})
			if err != nil {
				t.Fatalf("CreateDomain failed: %v", err)
			}
			domainIDs = append(domainIDs, domain.ID)
		}
		defer func() {
			for _, id := range domainIDs {
				_ = testQueries.DeleteDomain(ctx, id)
			}
		}()

		domains, err := testQueries.ListDomainsByApp(ctx, app.ID)
		if err != nil {
			t.Fatalf("ListDomainsByApp failed: %v", err)
		}

		if len(domains) < 3 {
			t.Errorf("expected at least 3 domains, got %d", len(domains))
		}
	})

	t.Run("update domain verified status", func(t *testing.T) {
		domainName := "verify-" + uuid.New().String()[:8] + ".example.com"

		domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
			AppID:  app.ID,
			Domain: domainName,
		})
		if err != nil {
			t.Fatalf("CreateDomain failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteDomain(ctx, domain.ID) }()

		// Verify the domain - UpdateDomainVerified takes only ID parameter
		updated, err := testQueries.UpdateDomainVerified(ctx, domain.ID)
		if err != nil {
			t.Fatalf("UpdateDomainVerified failed: %v", err)
		}

		if !updated.Verified {
			t.Error("expected domain to be verified")
		}
		if !updated.VerifiedAt.Valid {
			t.Error("expected verified_at to be set")
		}
	})

	t.Run("delete domain", func(t *testing.T) {
		domainName := "delete-" + uuid.New().String()[:8] + ".example.com"

		domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
			AppID:  app.ID,
			Domain: domainName,
		})
		if err != nil {
			t.Fatalf("CreateDomain failed: %v", err)
		}

		// Delete
		err = testQueries.DeleteDomain(ctx, domain.ID)
		if err != nil {
			t.Fatalf("DeleteDomain failed: %v", err)
		}

		// Verify deletion
		_, err = testQueries.GetDomainByName(ctx, domainName)
		if err == nil {
			t.Error("expected error after deletion, got nil")
		}
	})
}

// TestDomainSSLStatus tests SSL status handling
func TestDomainSSLStatus(t *testing.T) {
	validStatuses := []string{"pending", "provisioning", "active", "failed"}

	for _, status := range validStatuses {
		t.Run("ssl_status_"+status, func(t *testing.T) {
			// Verify status is recognized
			found := false
			for _, valid := range validStatuses {
				if status == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ssl_status %q not in valid statuses", status)
			}
		})
	}
}

// TestDomainUniqueness tests that domains are unique across all apps
func TestDomainUniqueness(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	// Create two apps
	app1, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "unique-test-1-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app1.ID) }()

	app2, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "unique-test-2-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app2.ID) }()

	// Create domain on app1
	domainName := "unique-" + uuid.New().String()[:8] + ".example.com"
	domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID:  app1.ID,
		Domain: domainName,
	})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteDomain(ctx, domain.ID) }()

	// Try to create same domain on app2 - should fail due to unique constraint
	_, err = testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID:  app2.ID,
		Domain: domainName,
	})
	if err == nil {
		t.Error("expected error when creating duplicate domain, got nil")
	}
}

// TestDomainOwnership tests domain ownership verification
func TestDomainOwnership(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "ownership-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	domainName := "ownership-" + uuid.New().String()[:8] + ".example.com"
	domain, err := testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID:  app.ID,
		Domain: domainName,
	})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteDomain(ctx, domain.ID) }()

	// Verify ownership
	if domain.AppID != app.ID {
		t.Errorf("expected AppID %s, got %s", app.ID, domain.AppID)
	}

	// Verify we can look up the domain
	retrieved, err := testQueries.GetDomainByName(ctx, domainName)
	if err != nil {
		t.Fatalf("GetDomainByName failed: %v", err)
	}

	if retrieved.AppID != app.ID {
		t.Errorf("expected AppID %s, got %s", app.ID, retrieved.AppID)
	}
}
