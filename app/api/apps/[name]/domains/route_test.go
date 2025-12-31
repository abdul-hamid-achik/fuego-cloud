package domains

import (
	"testing"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"valid simple domain", "example.com", true},
		{"valid subdomain", "app.example.com", true},
		{"valid multi-level subdomain", "my.app.example.com", true},
		{"valid with numbers", "app123.example.com", true},
		{"valid with hyphen", "my-app.example.com", true},
		{"valid long TLD", "example.technology", true},

		{"invalid no TLD", "example", false},
		{"invalid starts with dot", ".example.com", false},
		{"invalid ends with dot", "example.com.", false},
		{"invalid double dots", "example..com", false},
		{"invalid starts with hyphen", "-example.com", false},
		{"invalid ends with hyphen", "example-.com", false},
		{"invalid underscore", "my_app.example.com", false},
		{"invalid space", "my app.example.com", false},
		{"invalid special chars", "my@app.example.com", false},
		{"invalid single char TLD", "example.c", false},
		{"empty domain", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := domainRegex.MatchString(tt.domain)
			if valid != tt.valid {
				t.Errorf("domainRegex.MatchString(%q) = %v, want %v", tt.domain, valid, tt.valid)
			}
		})
	}
}

func TestDomainResponseConversion(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()
	verifiedAt := now.Add(-1 * time.Hour)

	domain := db.Domain{
		ID:         id,
		AppID:      appID,
		Domain:     "myapp.example.com",
		Verified:   true,
		SslStatus:  "active",
		CreatedAt:  now,
		VerifiedAt: pgtype.Timestamptz{Time: verifiedAt, Valid: true},
	}

	resp := toDomainResponse(domain)

	if resp.ID != id.String() {
		t.Errorf("expected ID %s, got %s", id.String(), resp.ID)
	}

	if resp.Domain != "myapp.example.com" {
		t.Errorf("expected Domain 'myapp.example.com', got %s", resp.Domain)
	}

	if !resp.Verified {
		t.Error("expected Verified to be true")
	}

	if resp.SSLStatus != "active" {
		t.Errorf("expected SSLStatus 'active', got %s", resp.SSLStatus)
	}

	if resp.VerifiedAt == nil {
		t.Error("expected VerifiedAt to be set")
	}
}

func TestDomainResponseWithUnverified(t *testing.T) {
	domain := db.Domain{
		ID:         uuid.New(),
		AppID:      uuid.New(),
		Domain:     "pending.example.com",
		Verified:   false,
		SslStatus:  "pending",
		CreatedAt:  time.Now(),
		VerifiedAt: pgtype.Timestamptz{Valid: false},
	}

	resp := toDomainResponse(domain)

	if resp.Verified {
		t.Error("expected Verified to be false")
	}

	if resp.SSLStatus != "pending" {
		t.Errorf("expected SSLStatus 'pending', got %s", resp.SSLStatus)
	}

	if resp.VerifiedAt != nil {
		t.Error("expected VerifiedAt to be nil for unverified domain")
	}
}

func TestSSLStatuses(t *testing.T) {
	validStatuses := []string{
		"pending",
		"provisioning",
		"active",
		"error",
		"expired",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			domain := db.Domain{
				ID:        uuid.New(),
				AppID:     uuid.New(),
				Domain:    "test.example.com",
				SslStatus: status,
				CreatedAt: time.Now(),
			}

			resp := toDomainResponse(domain)
			if resp.SSLStatus != status {
				t.Errorf("expected SSLStatus %q, got %q", status, resp.SSLStatus)
			}
		})
	}
}

func TestCreateDomainRequestValidation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"valid domain", "api.myapp.com", true},
		{"empty domain", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateDomainRequest{Domain: tt.domain}
			valid := req.Domain != ""
			if valid != tt.valid {
				t.Errorf("domain %q valid = %v, want %v", tt.domain, valid, tt.valid)
			}
		})
	}
}
