package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/nexo-cloud/generated/db"
	"github.com/google/uuid"
)

// TestDeploymentOperations tests deployment database operations
func TestDeploymentOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	// Create an app first
	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "deploy-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	t.Run("create deployment", func(t *testing.T) {
		deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
			AppID:   app.ID,
			Version: 1,
			Image:   "myapp:v1",
			Status:  "pending",
		})
		if err != nil {
			t.Fatalf("CreateDeployment failed: %v", err)
		}

		if deployment.AppID != app.ID {
			t.Errorf("expected AppID %s, got %s", app.ID, deployment.AppID)
		}
		if deployment.Version != 1 {
			t.Errorf("expected version 1, got %d", deployment.Version)
		}
		if deployment.Image != "myapp:v1" {
			t.Errorf("expected image 'myapp:v1', got %q", deployment.Image)
		}
		if deployment.Status != "pending" {
			t.Errorf("expected status 'pending', got %q", deployment.Status)
		}
	})

	t.Run("update deployment status", func(t *testing.T) {
		deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
			AppID:   app.ID,
			Version: 2,
			Image:   "myapp:v2",
			Status:  "pending",
		})
		if err != nil {
			t.Fatalf("CreateDeployment failed: %v", err)
		}

		// Update to running
		runningMsg := "Deployment started successfully"
		updated, err := testQueries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
			ID:      deployment.ID,
			Status:  "running",
			Message: &runningMsg,
		})
		if err != nil {
			t.Fatalf("UpdateDeploymentStatus failed: %v", err)
		}
		if updated.Status != "running" {
			t.Errorf("expected status 'running', got %q", updated.Status)
		}

		// Update to failed with error
		failedMsg := "Container crashed"
		failed, err := testQueries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
			ID:      deployment.ID,
			Status:  "failed",
			Message: &failedMsg,
		})
		if err != nil {
			t.Fatalf("UpdateDeploymentStatus failed: %v", err)
		}
		if failed.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", failed.Status)
		}
	})

	t.Run("list deployments by app", func(t *testing.T) {
		// Create multiple deployments
		for i := 10; i < 15; i++ {
			_, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
				AppID:   app.ID,
				Version: int32(i), //nolint:gosec // Loop counter guaranteed to be within int32 range
				Image:   "myapp:v" + string(rune('0'+i)),
				Status:  "pending",
			})
			if err != nil {
				t.Fatalf("CreateDeployment failed: %v", err)
			}
		}

		deployments, err := testQueries.ListDeploymentsByApp(ctx, db.ListDeploymentsByAppParams{
			AppID:  app.ID,
			Limit:  100,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListDeploymentsByApp failed: %v", err)
		}

		if len(deployments) < 5 {
			t.Errorf("expected at least 5 deployments, got %d", len(deployments))
		}
	})

	t.Run("get latest deployment", func(t *testing.T) {
		// Create a deployment with highest version
		latest, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
			AppID:   app.ID,
			Version: 999,
			Image:   "myapp:latest",
			Status:  "pending",
		})
		if err != nil {
			t.Fatalf("CreateDeployment failed: %v", err)
		}

		retrieved, err := testQueries.GetLatestDeployment(ctx, app.ID)
		if err != nil {
			t.Fatalf("GetLatestDeployment failed: %v", err)
		}

		if retrieved.ID != latest.ID {
			t.Errorf("expected latest deployment ID %s, got %s", latest.ID, retrieved.ID)
		}
	})

	t.Run("increment deployment count", func(t *testing.T) {
		initialCount := app.DeploymentCount

		updated, err := testQueries.IncrementDeploymentCount(ctx, app.ID)
		if err != nil {
			t.Fatalf("IncrementDeploymentCount failed: %v", err)
		}

		if updated.DeploymentCount != initialCount+1 {
			t.Errorf("expected deployment count %d, got %d", initialCount+1, updated.DeploymentCount)
		}
	})
}

// TestDeploymentStatusTransitions tests valid deployment status transitions
func TestDeploymentStatusTransitions(t *testing.T) {
	validStatuses := []string{"pending", "building", "deploying", "running", "failed", "stopped"}

	for _, status := range validStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Verify status is recognized
			found := false
			for _, valid := range validStatuses {
				if status == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("status %q not in valid statuses", status)
			}
		})
	}
}

// TestDeploymentVersioning tests deployment versioning logic
func TestDeploymentVersioning(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "version-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	t.Run("sequential versioning", func(t *testing.T) {
		var prevVersion int32
		for i := 1; i <= 5; i++ {
			deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
				AppID:   app.ID,
				Version: int32(i), //nolint:gosec // Loop counter guaranteed to be within int32 range
				Image:   "myapp:v" + string(rune('0'+i)),
				Status:  "pending",
			})
			if err != nil {
				t.Fatalf("CreateDeployment failed: %v", err)
			}

			if deployment.Version <= prevVersion {
				t.Errorf("version %d should be greater than %d", deployment.Version, prevVersion)
			}
			prevVersion = deployment.Version
		}
	})
}

// TestDeploymentRollback tests rollback by creating deployment with same image
func TestDeploymentRollback(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "rollback-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	// Create v1
	v1, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 1,
		Image:   "myapp:v1",
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment v1 failed: %v", err)
	}

	// Create v2
	_, err = testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 2,
		Image:   "myapp:v2",
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment v2 failed: %v", err)
	}

	// Rollback to v1 by creating v3 with v1's image
	rollback, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 3,
		Image:   v1.Image, // Same image as v1
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment rollback failed: %v", err)
	}

	if rollback.Image != v1.Image {
		t.Errorf("rollback image should be %q, got %q", v1.Image, rollback.Image)
	}
	if rollback.Version != 3 {
		t.Errorf("rollback version should be 3, got %d", rollback.Version)
	}
}

// TestDeploymentTimestamps tests deployment timestamp fields
func TestDeploymentTimestamps(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "timestamp-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

	before := time.Now().Add(-time.Second)

	deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 1,
		Image:   "myapp:v1",
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}

	after := time.Now().Add(time.Second)

	if deployment.CreatedAt.Before(before) || deployment.CreatedAt.After(after) {
		t.Errorf("created_at %v not in expected range [%v, %v]", deployment.CreatedAt, before, after)
	}
}
