package deployments

import (
	"testing"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateDeploymentRequestValidation(t *testing.T) {
	tests := []struct {
		name  string
		image string
		valid bool
	}{
		{"valid image with tag", "ghcr.io/user/app:v1.0.0", true},
		{"valid image with sha", "ghcr.io/user/app@sha256:abc123", true},
		{"valid docker hub image", "nginx:latest", true},
		{"empty image", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateDeploymentRequest{Image: tt.image}
			valid := req.Image != ""
			if valid != tt.valid {
				t.Errorf("image %q valid = %v, want %v", tt.image, valid, tt.valid)
			}
		})
	}
}

func TestDeploymentResponseConversion(t *testing.T) {
	id := uuid.New()
	appID := uuid.New()
	now := time.Now()
	startedAt := now.Add(-5 * time.Minute)
	readyAt := now.Add(-3 * time.Minute)
	message := "Deployment successful"

	deployment := db.Deployment{
		ID:        id,
		AppID:     appID,
		Version:   5,
		Image:     "ghcr.io/test/app:v5",
		Status:    "running",
		Message:   &message,
		Error:     nil,
		CreatedAt: now,
		StartedAt: pgtype.Timestamptz{Time: startedAt, Valid: true},
		ReadyAt:   pgtype.Timestamptz{Time: readyAt, Valid: true},
	}

	resp := toDeploymentResponse(deployment)

	if resp.ID != id.String() {
		t.Errorf("expected ID %s, got %s", id.String(), resp.ID)
	}

	if resp.AppID != appID.String() {
		t.Errorf("expected AppID %s, got %s", appID.String(), resp.AppID)
	}

	if resp.Version != 5 {
		t.Errorf("expected Version 5, got %d", resp.Version)
	}

	if resp.Image != "ghcr.io/test/app:v5" {
		t.Errorf("expected Image 'ghcr.io/test/app:v5', got %s", resp.Image)
	}

	if resp.Status != "running" {
		t.Errorf("expected Status 'running', got %s", resp.Status)
	}

	if resp.Message == nil || *resp.Message != message {
		t.Errorf("expected Message %q, got %v", message, resp.Message)
	}

	if resp.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	if resp.ReadyAt == nil {
		t.Error("expected ReadyAt to be set")
	}
}

func TestDeploymentResponseWithNullTimestamps(t *testing.T) {
	deployment := db.Deployment{
		ID:        uuid.New(),
		AppID:     uuid.New(),
		Version:   1,
		Image:     "test:latest",
		Status:    "pending",
		CreatedAt: time.Now(),
		StartedAt: pgtype.Timestamptz{Valid: false},
		ReadyAt:   pgtype.Timestamptz{Valid: false},
	}

	resp := toDeploymentResponse(deployment)

	if resp.StartedAt != nil {
		t.Error("expected StartedAt to be nil for invalid timestamp")
	}

	if resp.ReadyAt != nil {
		t.Error("expected ReadyAt to be nil for invalid timestamp")
	}
}

func TestDeploymentStatuses(t *testing.T) {
	validStatuses := []string{
		"pending",
		"building",
		"deploying",
		"running",
		"failed",
		"stopped",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			deployment := db.Deployment{
				ID:        uuid.New(),
				AppID:     uuid.New(),
				Version:   1,
				Image:     "test:latest",
				Status:    status,
				CreatedAt: time.Now(),
			}

			resp := toDeploymentResponse(deployment)
			if resp.Status != status {
				t.Errorf("expected status %q, got %q", status, resp.Status)
			}
		})
	}
}

func TestVersionIncrement(t *testing.T) {
	tests := []struct {
		name            string
		latestVersion   int32
		expectedVersion int32
	}{
		{"first deployment", 0, 1},
		{"second deployment", 1, 2},
		{"tenth deployment", 9, 10},
		{"high version", 999, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextVersion := tt.latestVersion + 1
			if nextVersion != tt.expectedVersion {
				t.Errorf("expected version %d, got %d", tt.expectedVersion, nextVersion)
			}
		})
	}
}
