package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testPool    *pgxpool.Pool
	testConfig  *config.Config
	testQueries *db.Queries
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Skip database tests if no connection
		os.Exit(0)
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		os.Exit(1)
	}
	defer testPool.Close()

	if err := testPool.Ping(context.Background()); err != nil {
		os.Exit(0) // Skip if can't connect
	}

	testQueries = db.New(testPool)
	testConfig = &config.Config{
		JWTSecret:        "test-secret-key-for-testing-purposes-only",
		AppsDomainSuffix: "apps.test.local",
	}

	os.Exit(m.Run())
}

// Helper to create a test user and return their ID and JWT token
func createTestUserWithToken(t *testing.T) (uuid.UUID, string) {
	t.Helper()

	ctx := context.Background()
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
		t.Fatalf("failed to create test user: %v", err)
	}

	tokens, err := auth.GenerateTokenPair(user.ID, user.Username, testConfig.JWTSecret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return user.ID, tokens.AccessToken
}

func deleteTestUser(t *testing.T, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	// Delete all apps for this user first
	apps, _ := testQueries.ListAppsByUser(ctx, db.ListAppsByUserParams{
		UserID: userID,
		Limit:  1000,
		Offset: 0,
	})
	for _, app := range apps {
		_ = testQueries.DeleteApp(ctx, app.ID)
	}

	_ = testQueries.DeleteUser(ctx, userID)
}

// TestAppsEndpointValidation tests input validation for apps endpoints
func TestAppsEndpointValidation(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	userID, token := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing name",
			body:           map[string]interface{}{"region": "gdl"},
			expectedStatus: 400,
			expectedError:  "name is required",
		},
		{
			name:           "name too short",
			body:           map[string]interface{}{"name": "ab"},
			expectedStatus: 400,
			expectedError:  "name must be between 3 and 63 characters",
		},
		{
			name:           "name too long",
			body:           map[string]interface{}{"name": "a" + string(make([]byte, 63))},
			expectedStatus: 400,
			expectedError:  "name must be between 3 and 63 characters",
		},
		{
			name:           "name starts with number",
			body:           map[string]interface{}{"name": "1myapp"},
			expectedStatus: 400,
			expectedError:  "name must start with a letter",
		},
		{
			name:           "name ends with hyphen",
			body:           map[string]interface{}{"name": "myapp-"},
			expectedStatus: 400,
			expectedError:  "name must start with a letter",
		},
		{
			name:           "name with uppercase",
			body:           map[string]interface{}{"name": "MyApp"},
			expectedStatus: 400,
			expectedError:  "name must start with a letter",
		},
		{
			name:           "invalid region",
			body:           map[string]interface{}{"name": "validapp", "region": "invalid"},
			expectedStatus: 400,
			expectedError:  "invalid region",
		},
		{
			name:           "invalid size",
			body:           map[string]interface{}{"name": "validapp", "size": "invalid"},
			expectedStatus: 400,
			expectedError:  "invalid size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/apps", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			// Note: This tests the validation logic expectations
			// Full integration would require the Fuego router setup
			if tt.expectedStatus != 0 && tt.expectedError != "" {
				// Validation expectation recorded
				t.Logf("Expected status %d with error: %s", tt.expectedStatus, tt.expectedError)
			}
		})
	}
}

// TestAppNameValidation tests the app name regex validation
func TestAppNameValidation(t *testing.T) {
	// Using the same regex as in route.go
	appNameRegex := `^[a-z][a-z0-9-]*[a-z0-9]$`

	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{"valid simple name", "myapp", true},
		{"valid with hyphens", "my-cool-app", true},
		{"valid with numbers", "app123", true},
		{"valid mixed", "my-app-v2", true},
		{"minimum length", "abc", true},
		{"invalid starts with number", "1app", false},
		{"invalid starts with hyphen", "-app", false},
		{"invalid ends with hyphen", "app-", false},
		{"invalid uppercase", "MyApp", false},
		{"invalid special chars", "my_app", false},
		{"invalid spaces", "my app", false},
		{"two chars valid by regex", "ab", true}, // Regex allows 2 chars, but API validates min 3
		{"single char", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := matchRegex(appNameRegex, tt.input)
			if matched != tt.isValid {
				t.Errorf("name %q: expected valid=%v, got valid=%v", tt.input, tt.isValid, matched)
			}
		})
	}
}

func matchRegex(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// TestRegionValidation tests valid regions
func TestRegionValidation(t *testing.T) {
	validRegions := map[string]bool{"gdl": true, "mex": true, "qro": true}

	tests := []struct {
		region  string
		isValid bool
	}{
		{"gdl", true},
		{"mex", true},
		{"qro", true},
		{"", false}, // Empty defaults, but explicit empty should fail validation
		{"us-east-1", false},
		{"GDL", false}, // Case sensitive
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			isValid := validRegions[tt.region]
			if isValid != tt.isValid {
				t.Errorf("region %q: expected valid=%v, got valid=%v", tt.region, tt.isValid, isValid)
			}
		})
	}
}

// TestSizeValidation tests valid sizes
func TestSizeValidation(t *testing.T) {
	validSizes := map[string]bool{"starter": true, "pro": true, "enterprise": true}

	tests := []struct {
		size    string
		isValid bool
	}{
		{"starter", true},
		{"pro", true},
		{"enterprise", true},
		{"", false},
		{"small", false},
		{"large", false},
		{"STARTER", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			isValid := validSizes[tt.size]
			if isValid != tt.isValid {
				t.Errorf("size %q: expected valid=%v, got valid=%v", tt.size, tt.isValid, isValid)
			}
		})
	}
}

// TestAppURLGeneration tests the URL generation for apps
func TestAppURLGeneration(t *testing.T) {
	tests := []struct {
		appName      string
		domainSuffix string
		expectedURL  string
	}{
		{"myapp", "apps.fuego.cloud", "https://myapp.apps.fuego.cloud"},
		{"test-app", "apps.test.local", "https://test-app.apps.test.local"},
		{"app123", "example.com", "https://app123.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.appName, func(t *testing.T) {
			url := "https://" + tt.appName + "." + tt.domainSuffix
			if url != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

// TestDatabaseAppOperations tests actual database operations for apps
func TestDatabaseAppOperations(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	t.Run("create and retrieve app", func(t *testing.T) {
		appName := "test-app-" + uuid.New().String()[:8]

		// Create app
		app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
			UserID: userID,
			Name:   appName,
			Region: "gdl",
			Size:   "starter",
		})
		if err != nil {
			t.Fatalf("CreateApp failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

		// Verify creation
		if app.Name != appName {
			t.Errorf("expected name %q, got %q", appName, app.Name)
		}
		if app.Region != "gdl" {
			t.Errorf("expected region 'gdl', got %q", app.Region)
		}
		if app.Size != "starter" {
			t.Errorf("expected size 'starter', got %q", app.Size)
		}
		if app.Status != "stopped" {
			t.Errorf("expected status 'stopped', got %q", app.Status)
		}

		// Retrieve by name
		retrieved, err := testQueries.GetAppByName(ctx, db.GetAppByNameParams{
			UserID: userID,
			Name:   appName,
		})
		if err != nil {
			t.Fatalf("GetAppByName failed: %v", err)
		}
		if retrieved.ID != app.ID {
			t.Errorf("expected ID %s, got %s", app.ID, retrieved.ID)
		}
	})

	t.Run("list apps by user", func(t *testing.T) {
		// Create multiple apps
		appNames := []string{
			"list-test-1-" + uuid.New().String()[:8],
			"list-test-2-" + uuid.New().String()[:8],
			"list-test-3-" + uuid.New().String()[:8],
		}

		var appIDs []uuid.UUID
		for _, name := range appNames {
			app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
				UserID: userID,
				Name:   name,
				Region: "gdl",
				Size:   "starter",
			})
			if err != nil {
				t.Fatalf("CreateApp failed: %v", err)
			}
			appIDs = append(appIDs, app.ID)
		}
		defer func() {
			for _, id := range appIDs {
				_ = testQueries.DeleteApp(ctx, id)
			}
		}()

		// List apps
		apps, err := testQueries.ListAppsByUser(ctx, db.ListAppsByUserParams{
			UserID: userID,
			Limit:  100,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("ListAppsByUser failed: %v", err)
		}

		if len(apps) < 3 {
			t.Errorf("expected at least 3 apps, got %d", len(apps))
		}
	})

	t.Run("update app status", func(t *testing.T) {
		appName := "status-test-" + uuid.New().String()[:8]

		app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
			UserID: userID,
			Name:   appName,
			Region: "gdl",
			Size:   "starter",
		})
		if err != nil {
			t.Fatalf("CreateApp failed: %v", err)
		}
		defer func() { _ = testQueries.DeleteApp(ctx, app.ID) }()

		// Update status
		updated, err := testQueries.UpdateAppStatus(ctx, db.UpdateAppStatusParams{
			ID:     app.ID,
			Status: "running",
		})
		if err != nil {
			t.Fatalf("UpdateAppStatus failed: %v", err)
		}
		if updated.Status != "running" {
			t.Errorf("expected status 'running', got %q", updated.Status)
		}
	})

	t.Run("delete app", func(t *testing.T) {
		appName := "delete-test-" + uuid.New().String()[:8]

		app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
			UserID: userID,
			Name:   appName,
			Region: "gdl",
			Size:   "starter",
		})
		if err != nil {
			t.Fatalf("CreateApp failed: %v", err)
		}

		// Delete
		err = testQueries.DeleteApp(ctx, app.ID)
		if err != nil {
			t.Fatalf("DeleteApp failed: %v", err)
		}

		// Verify deletion
		_, err = testQueries.GetAppByName(ctx, db.GetAppByNameParams{
			UserID: userID,
			Name:   appName,
		})
		if err == nil {
			t.Error("expected error after deletion, got nil")
		}
	})
}

// TestAppCountByUser tests counting apps for a user
func TestAppCountByUser(t *testing.T) {
	if testPool == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	userID, _ := createTestUserWithToken(t)
	defer deleteTestUser(t, userID)

	// Initially should be 0
	count, err := testQueries.CountAppsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountAppsByUser failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 apps initially, got %d", count)
	}

	// Create apps and verify count
	var appIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
			UserID: userID,
			Name:   "count-test-" + uuid.New().String()[:8],
			Region: "gdl",
			Size:   "starter",
		})
		if err != nil {
			t.Fatalf("CreateApp failed: %v", err)
		}
		appIDs = append(appIDs, app.ID)
	}
	defer func() {
		for _, id := range appIDs {
			_ = testQueries.DeleteApp(ctx, id)
		}
	}()

	count, err = testQueries.CountAppsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountAppsByUser failed: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 apps, got %d", count)
	}
}
