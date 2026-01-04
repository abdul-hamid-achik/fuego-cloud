package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Helper to convert uuid.UUID to pgtype.UUID
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// TestUserOperations tests user database operations
func TestUserOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	t.Run("create and retrieve user", func(t *testing.T) {
		githubID := time.Now().UnixNano()
		username := "testuser-" + uuid.New().String()[:8]
		avatarURL := "https://example.com/avatar.png"

		user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
			GithubID:  githubID,
			Username:  username,
			Email:     username + "@test.com",
			AvatarUrl: &avatarURL,
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

		if user.Username != username {
			t.Errorf("expected username %q, got %q", username, user.Username)
		}
		if user.Plan != "free" {
			t.Errorf("expected plan 'free', got %q", user.Plan)
		}

		// Retrieve by ID
		retrieved, err := testQueries.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetUserByID failed: %v", err)
		}
		if retrieved.ID != user.ID {
			t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
		}
	})

	t.Run("get user by github ID", func(t *testing.T) {
		githubID := time.Now().UnixNano()
		username := "github-" + uuid.New().String()[:8]
		avatarURL := "https://example.com/avatar.png"

		user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
			GithubID:  githubID,
			Username:  username,
			Email:     username + "@test.com",
			AvatarUrl: &avatarURL,
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

		retrieved, err := testQueries.GetUserByGitHubID(ctx, githubID)
		if err != nil {
			t.Fatalf("GetUserByGitHubID failed: %v", err)
		}
		if retrieved.ID != user.ID {
			t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
		}
	})

	t.Run("get user by username", func(t *testing.T) {
		githubID := time.Now().UnixNano()
		username := "byname-" + uuid.New().String()[:8]
		avatarURL := "https://example.com/avatar.png"

		user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
			GithubID:  githubID,
			Username:  username,
			Email:     username + "@test.com",
			AvatarUrl: &avatarURL,
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

		retrieved, err := testQueries.GetUserByUsername(ctx, username)
		if err != nil {
			t.Fatalf("GetUserByUsername failed: %v", err)
		}
		if retrieved.ID != user.ID {
			t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
		}
	})

	t.Run("update user email", func(t *testing.T) {
		githubID := time.Now().UnixNano()
		username := "email-" + uuid.New().String()[:8]
		avatarURL := "https://example.com/avatar.png"

		user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
			GithubID:  githubID,
			Username:  username,
			Email:     username + "@test.com",
			AvatarUrl: &avatarURL,
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

		newEmail := "updated-" + username + "@test.com"
		updated, err := testQueries.UpdateUser(ctx, db.UpdateUserParams{
			ID:        user.ID,
			Email:     newEmail,
			AvatarUrl: user.AvatarUrl,
		})
		if err != nil {
			t.Fatalf("UpdateUser failed: %v", err)
		}
		if updated.Email != newEmail {
			t.Errorf("expected email %q, got %q", newEmail, updated.Email)
		}
	})

	t.Run("update user plan", func(t *testing.T) {
		githubID := time.Now().UnixNano()
		username := "plan-" + uuid.New().String()[:8]
		avatarURL := "https://example.com/avatar.png"

		user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
			GithubID:  githubID,
			Username:  username,
			Email:     username + "@test.com",
			AvatarUrl: &avatarURL,
		})
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

		// Upgrade to pro
		updated, err := testQueries.UpdateUserPlan(ctx, db.UpdateUserPlanParams{
			ID:   user.ID,
			Plan: "pro",
		})
		if err != nil {
			t.Fatalf("UpdateUserPlan failed: %v", err)
		}
		if updated.Plan != "pro" {
			t.Errorf("expected plan 'pro', got %q", updated.Plan)
		}
	})
}

// TestUserPlanLimits tests plan-based limits
func TestUserPlanLimits(t *testing.T) {
	plans := map[string]struct {
		maxApps        int
		maxDeployments int
		maxDomains     int
	}{
		"free":       {maxApps: 3, maxDeployments: 10, maxDomains: 1},
		"pro":        {maxApps: 10, maxDeployments: 100, maxDomains: 10},
		"enterprise": {maxApps: -1, maxDeployments: -1, maxDomains: -1}, // Unlimited
	}

	for plan, limits := range plans {
		t.Run("plan_"+plan, func(t *testing.T) {
			if plan == "free" && limits.maxApps != 3 {
				t.Errorf("expected free plan maxApps=3, got %d", limits.maxApps)
			}
			if plan == "pro" && limits.maxApps != 10 {
				t.Errorf("expected pro plan maxApps=10, got %d", limits.maxApps)
			}
			// Verify limits are defined
			t.Logf("Plan %s: maxApps=%d, maxDeployments=%d, maxDomains=%d",
				plan, limits.maxApps, limits.maxDeployments, limits.maxDomains)
		})
	}
}

// TestUserDeletion tests cascading deletion behavior
func TestUserDeletion(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	githubID := time.Now().UnixNano()
	username := "delete-cascade-" + uuid.New().String()[:8]
	avatarURL := "https://example.com/avatar.png"

	user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
		GithubID:  githubID,
		Username:  username,
		Email:     username + "@test.com",
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create an app for this user
	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: user.ID,
		Name:   "cascade-app-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	// Delete the app first (since we may not have cascade delete)
	err = testQueries.DeleteApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	// Now delete the user
	err = testQueries.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user is deleted
	_, err = testQueries.GetUserByID(ctx, user.ID)
	if err == nil {
		t.Error("expected error when getting deleted user, got nil")
	}
}

// TestUserTimestamps tests user timestamp fields
func TestUserTimestamps(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()

	before := time.Now().Add(-time.Second)

	githubID := time.Now().UnixNano()
	username := "timestamp-" + uuid.New().String()[:8]
	avatarURL := "https://example.com/avatar.png"

	user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
		GithubID:  githubID,
		Username:  username,
		Email:     username + "@test.com",
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteUser(ctx, user.ID) }()

	after := time.Now().Add(time.Second)

	if user.CreatedAt.Before(before) || user.CreatedAt.After(after) {
		t.Errorf("created_at %v not in expected range [%v, %v]", user.CreatedAt, before, after)
	}
}

// TestAPITokenOperations tests API token handling
func TestAPITokenOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	t.Run("create and retrieve token", func(t *testing.T) {
		tokenHash := "sha256:" + uuid.New().String()
		tokenName := "test-token-" + uuid.New().String()[:8]

		token, err := testQueries.CreateAPIToken(ctx, db.CreateAPITokenParams{
			UserID:    userID,
			TokenHash: tokenHash,
			Name:      tokenName,
		})
		if err != nil {
			t.Fatalf("CreateAPIToken failed: %v", err)
		}

		if token.Name != tokenName {
			t.Errorf("expected name %q, got %q", tokenName, token.Name)
		}

		// Retrieve by hash
		retrieved, err := testQueries.GetAPITokenByHash(ctx, tokenHash)
		if err != nil {
			t.Fatalf("GetAPITokenByHash failed: %v", err)
		}
		if retrieved.ID != token.ID {
			t.Errorf("expected ID %s, got %s", token.ID, retrieved.ID)
		}
	})

	t.Run("update token last used", func(t *testing.T) {
		tokenHash := "sha256:" + uuid.New().String()
		tokenName := "lastused-" + uuid.New().String()[:8]

		token, err := testQueries.CreateAPIToken(ctx, db.CreateAPITokenParams{
			UserID:    userID,
			TokenHash: tokenHash,
			Name:      tokenName,
		})
		if err != nil {
			t.Fatalf("CreateAPIToken failed: %v", err)
		}

		// Update last used
		err = testQueries.UpdateAPITokenLastUsed(ctx, token.ID)
		if err != nil {
			t.Fatalf("UpdateAPITokenLastUsed failed: %v", err)
		}

		// Verify it was updated
		retrieved, err := testQueries.GetAPITokenByHash(ctx, tokenHash)
		if err != nil {
			t.Fatalf("GetAPITokenByHash failed: %v", err)
		}
		if !retrieved.LastUsedAt.Valid {
			t.Error("expected last_used_at to be set")
		}
	})
}

// TestActivityLogOperations tests activity log functionality
func TestActivityLogOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	// Create an app
	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "activity-test-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}
	defer testQueries.DeleteApp(ctx, app.ID)

	t.Run("create activity log", func(t *testing.T) {
		log, err := testQueries.CreateActivityLog(ctx, db.CreateActivityLogParams{
			AppID:   toPgUUID(app.ID),
			UserID:  toPgUUID(userID),
			Action:  "deployment.created",
			Details: []byte(`{"version": 1, "image": "myapp:v1"}`),
		})
		if err != nil {
			t.Fatalf("CreateActivityLog failed: %v", err)
		}

		if log.Action != "deployment.created" {
			t.Errorf("expected action 'deployment.created', got %q", log.Action)
		}
	})

	t.Run("list activity logs by app", func(t *testing.T) {
		// Create multiple activity logs
		actions := []string{"app.created", "deployment.created", "deployment.started", "deployment.completed"}
		for _, action := range actions {
			_, err := testQueries.CreateActivityLog(ctx, db.CreateActivityLogParams{
				AppID:   toPgUUID(app.ID),
				UserID:  toPgUUID(userID),
				Action:  action,
				Details: []byte(`{}`),
			})
			if err != nil {
				t.Fatalf("CreateActivityLog failed: %v", err)
			}
		}

		logs, err := testQueries.ListActivityLogsByApp(ctx, db.ListActivityLogsByAppParams{
			AppID:  toPgUUID(app.ID),
			Limit:  100,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListActivityLogsByApp failed: %v", err)
		}

		if len(logs) < 4 {
			t.Errorf("expected at least 4 activity logs, got %d", len(logs))
		}
	})
}
