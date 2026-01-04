package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/google/uuid"
)

func TestNewTestApp(t *testing.T) {
	ta := NewTestApp()

	if ta == nil {
		t.Fatal("expected non-nil TestApp")
	}
	if ta.App == nil {
		t.Error("expected non-nil App")
	}
	if ta.Config == nil {
		t.Error("expected non-nil Config")
	}
	if ta.Config.Environment != "test" {
		t.Errorf("expected environment 'test', got %q", ta.Config.Environment)
	}
	if ta.Config.JWTSecret == "" {
		t.Error("expected non-empty JWTSecret")
	}
	if ta.Config.EncryptionKey == "" {
		t.Error("expected non-empty EncryptionKey")
	}
}

func TestTestApp_WithMockDB(t *testing.T) {
	ta := NewTestApp()
	mockDB := NewMockDB()

	result := ta.WithMockDB(mockDB)

	if result != ta {
		t.Error("expected WithMockDB to return same TestApp instance")
	}
}

func TestTestApp_WithAuth(t *testing.T) {
	ta := NewTestApp()
	userID := uuid.New()
	username := "testuser"

	result := ta.WithAuth(userID, username)

	if result != ta {
		t.Error("expected WithAuth to return same TestApp instance")
	}
}

func TestGenerateTestToken(t *testing.T) {
	ta := NewTestApp()
	userID := uuid.New()
	username := "testuser"

	token := GenerateTestToken(t, ta.Config, userID, username)

	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestMakeRequest(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		body := map[string]string{"key": "value"}
		headers := map[string]string{"Authorization": "Bearer token123"}

		req := MakeRequest(t, http.MethodPost, "/api/test", body, headers)

		if req.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/test" {
			t.Errorf("expected path /api/test, got %s", req.URL.Path)
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("expected Authorization header, got %s", req.Header.Get("Authorization"))
		}
	})

	t.Run("without body", func(t *testing.T) {
		req := MakeRequest(t, http.MethodGet, "/api/test", nil, nil)

		if req.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", req.Method)
		}
	})

	t.Run("with multiple headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Another":       "another-value",
		}

		req := MakeRequest(t, http.MethodGet, "/api/test", nil, headers)

		if req.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected X-Custom-Header, got %s", req.Header.Get("X-Custom-Header"))
		}
		if req.Header.Get("X-Another") != "another-value" {
			t.Errorf("expected X-Another, got %s", req.Header.Get("X-Another"))
		}
	})
}

func TestParseResponse(t *testing.T) {
	type TestResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	w := httptest.NewRecorder()
	_, _ = w.WriteString(`{"id":"123","name":"test"}`)

	result := ParseResponse[TestResponse](t, w)

	if result.ID != "123" {
		t.Errorf("expected ID '123', got %q", result.ID)
	}
	if result.Name != "test" {
		t.Errorf("expected Name 'test', got %q", result.Name)
	}
}

func TestAssertStatusCode(t *testing.T) {
	t.Run("matching status", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)

		// This should not fail
		AssertStatusCode(t, w, http.StatusOK)
	})
}

func TestAssertJSONContains(t *testing.T) {
	t.Run("key exists with correct value", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, _ = w.WriteString(`{"status":"success","message":"done"}`)

		// This should not fail
		AssertJSONContains(t, w, "status", "success")
		AssertJSONContains(t, w, "message", "done")
	})
}

func TestNewMockDB(t *testing.T) {
	mockDB := NewMockDB()

	if mockDB == nil {
		t.Fatal("expected non-nil MockDB")
	}
	if mockDB.Users == nil {
		t.Error("expected non-nil Users map")
	}
	if mockDB.Apps == nil {
		t.Error("expected non-nil Apps map")
	}
	if mockDB.Deployments == nil {
		t.Error("expected non-nil Deployments map")
	}
	if mockDB.Domains == nil {
		t.Error("expected non-nil Domains map")
	}
	if mockDB.APITokens == nil {
		t.Error("expected non-nil APITokens map")
	}
	if mockDB.OAuthStates == nil {
		t.Error("expected non-nil OAuthStates map")
	}
}

func TestMockDB_SeedUser(t *testing.T) {
	mockDB := NewMockDB()
	userID := uuid.New()

	user := mockDB.SeedUser(userID, "testuser", "test@example.com")

	if user.ID != userID {
		t.Errorf("expected ID %s, got %s", userID, user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", user.Email)
	}
	if user.Plan != "free" {
		t.Errorf("expected plan 'free', got %q", user.Plan)
	}

	// Verify user was added to map
	if _, ok := mockDB.Users[userID]; !ok {
		t.Error("expected user to be in Users map")
	}
}

func TestMockDB_SeedApp(t *testing.T) {
	mockDB := NewMockDB()
	appID := uuid.New()
	userID := uuid.New()

	app := mockDB.SeedApp(appID, userID, "myapp")

	if app.ID != appID {
		t.Errorf("expected ID %s, got %s", appID, app.ID)
	}
	if app.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, app.UserID)
	}
	if app.Name != "myapp" {
		t.Errorf("expected name 'myapp', got %q", app.Name)
	}
	if app.Region != "gdl" {
		t.Errorf("expected region 'gdl', got %q", app.Region)
	}
	if app.Size != "starter" {
		t.Errorf("expected size 'starter', got %q", app.Size)
	}
	if app.Status != "running" {
		t.Errorf("expected status 'running', got %q", app.Status)
	}

	// Verify app was added to map
	if _, ok := mockDB.Apps[appID]; !ok {
		t.Error("expected app to be in Apps map")
	}
}

func TestMockDB_SeedDeployment(t *testing.T) {
	mockDB := NewMockDB()
	deploymentID := uuid.New()
	appID := uuid.New()

	deployment := mockDB.SeedDeployment(deploymentID, appID, 1)

	if deployment.ID != deploymentID {
		t.Errorf("expected ID %s, got %s", deploymentID, deployment.ID)
	}
	if deployment.AppID != appID {
		t.Errorf("expected AppID %s, got %s", appID, deployment.AppID)
	}
	if deployment.Version != 1 {
		t.Errorf("expected version 1, got %d", deployment.Version)
	}
	if deployment.Status != "running" {
		t.Errorf("expected status 'running', got %q", deployment.Status)
	}

	// Verify deployment was added to map
	if _, ok := mockDB.Deployments[deploymentID]; !ok {
		t.Error("expected deployment to be in Deployments map")
	}
}

func TestMockDB_SeedDomain(t *testing.T) {
	mockDB := NewMockDB()
	domainID := uuid.New()
	appID := uuid.New()

	domain := mockDB.SeedDomain(domainID, appID, "example.com")

	if domain.ID != domainID {
		t.Errorf("expected ID %s, got %s", domainID, domain.ID)
	}
	if domain.AppID != appID {
		t.Errorf("expected AppID %s, got %s", appID, domain.AppID)
	}
	if domain.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got %q", domain.Domain)
	}
	if domain.Verified {
		t.Error("expected Verified to be false")
	}
	if domain.SslStatus != "pending" {
		t.Errorf("expected SslStatus 'pending', got %q", domain.SslStatus)
	}

	// Verify domain was added to map
	if _, ok := mockDB.Domains[domainID]; !ok {
		t.Error("expected domain to be in Domains map")
	}
}

func TestNewMockQueries(t *testing.T) {
	mockDB := NewMockDB()
	queries := NewMockQueries(mockDB)

	if queries == nil {
		t.Fatal("expected non-nil MockQueries")
	}
	if queries.db != mockDB {
		t.Error("expected MockQueries.db to reference MockDB")
	}
}

func TestMockQueries_GetUserByGithubID(t *testing.T) {
	mockDB := NewMockDB()
	userID := uuid.New()
	mockDB.SeedUser(userID, "testuser", "test@example.com")

	queries := NewMockQueries(mockDB)

	t.Run("user found", func(t *testing.T) {
		user, err := queries.GetUserByGithubID(context.TODO(), 12345)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.ID != userID {
			t.Errorf("expected user ID %s, got %s", userID, user.ID)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := queries.GetUserByGithubID(context.TODO(), 99999)
		if err == nil {
			t.Error("expected error for non-existent user")
		}
	})
}

func TestMockQueries_GetUserByID(t *testing.T) {
	mockDB := NewMockDB()
	userID := uuid.New()
	mockDB.SeedUser(userID, "testuser", "test@example.com")

	queries := NewMockQueries(mockDB)

	t.Run("user found", func(t *testing.T) {
		user, err := queries.GetUserByID(context.TODO(), userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %q", user.Username)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := queries.GetUserByID(context.TODO(), uuid.New())
		if err == nil {
			t.Error("expected error for non-existent user")
		}
	})
}

func TestMockQueries_ListAppsByUser(t *testing.T) {
	mockDB := NewMockDB()
	userID := uuid.New()
	otherUserID := uuid.New()

	mockDB.SeedApp(uuid.New(), userID, "app1")
	mockDB.SeedApp(uuid.New(), userID, "app2")
	mockDB.SeedApp(uuid.New(), otherUserID, "other-app")

	queries := NewMockQueries(mockDB)

	apps, err := queries.ListAppsByUser(context.TODO(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(apps))
	}
}

func TestMockQueries_GetAppByName(t *testing.T) {
	mockDB := NewMockDB()
	userID := uuid.New()
	mockDB.SeedApp(uuid.New(), userID, "myapp")

	queries := NewMockQueries(mockDB)

	t.Run("app found", func(t *testing.T) {
		app, err := queries.GetAppByName(context.TODO(), db.GetAppByNameParams{
			UserID: userID,
			Name:   "myapp",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if app.Name != "myapp" {
			t.Errorf("expected name 'myapp', got %q", app.Name)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := queries.GetAppByName(context.TODO(), db.GetAppByNameParams{
			UserID: userID,
			Name:   "nonexistent",
		})
		if err == nil {
			t.Error("expected error for non-existent app")
		}
	})
}

func TestMockQueries_CreateApp(t *testing.T) {
	mockDB := NewMockDB()
	queries := NewMockQueries(mockDB)
	userID := uuid.New()

	app, err := queries.CreateApp(context.TODO(), db.CreateAppParams{
		UserID: userID,
		Name:   "newapp",
		Region: "mex",
		Size:   "pro",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if app.Name != "newapp" {
		t.Errorf("expected name 'newapp', got %q", app.Name)
	}
	if app.Region != "mex" {
		t.Errorf("expected region 'mex', got %q", app.Region)
	}
	if app.Size != "pro" {
		t.Errorf("expected size 'pro', got %q", app.Size)
	}
	if app.Status != "created" {
		t.Errorf("expected status 'created', got %q", app.Status)
	}

	// Verify app was added to map
	if _, ok := mockDB.Apps[app.ID]; !ok {
		t.Error("expected app to be in Apps map")
	}
}

func TestMockQueries_UpdateApp(t *testing.T) {
	mockDB := NewMockDB()
	appID := uuid.New()
	userID := uuid.New()
	mockDB.SeedApp(appID, userID, "myapp")

	queries := NewMockQueries(mockDB)

	t.Run("app exists", func(t *testing.T) {
		updated, err := queries.UpdateApp(context.TODO(), db.UpdateAppParams{
			ID:     appID,
			Name:   "updated-app",
			Region: "qro",
			Size:   "enterprise",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if updated.Name != "updated-app" {
			t.Errorf("expected name 'updated-app', got %q", updated.Name)
		}
		if updated.Region != "qro" {
			t.Errorf("expected region 'qro', got %q", updated.Region)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := queries.UpdateApp(context.TODO(), db.UpdateAppParams{
			ID:     uuid.New(),
			Name:   "updated-app",
			Region: "qro",
			Size:   "enterprise",
		})
		if err == nil {
			t.Error("expected error for non-existent app")
		}
	})
}

func TestMockQueries_DeleteApp(t *testing.T) {
	mockDB := NewMockDB()
	appID := uuid.New()
	userID := uuid.New()
	mockDB.SeedApp(appID, userID, "myapp")

	queries := NewMockQueries(mockDB)

	t.Run("app exists", func(t *testing.T) {
		err := queries.DeleteApp(context.TODO(), appID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if _, ok := mockDB.Apps[appID]; ok {
			t.Error("expected app to be deleted from map")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		err := queries.DeleteApp(context.TODO(), uuid.New())
		if err == nil {
			t.Error("expected error for non-existent app")
		}
	})
}

func TestMockQueries_ListDeploymentsByApp(t *testing.T) {
	mockDB := NewMockDB()
	appID := uuid.New()
	otherAppID := uuid.New()

	mockDB.SeedDeployment(uuid.New(), appID, 1)
	mockDB.SeedDeployment(uuid.New(), appID, 2)
	mockDB.SeedDeployment(uuid.New(), otherAppID, 1)

	queries := NewMockQueries(mockDB)

	deployments, err := queries.ListDeploymentsByApp(context.TODO(), db.ListDeploymentsByAppParams{
		AppID: appID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(deployments))
	}
}

func TestMockQueries_ListDomainsByApp(t *testing.T) {
	mockDB := NewMockDB()
	appID := uuid.New()
	otherAppID := uuid.New()

	mockDB.SeedDomain(uuid.New(), appID, "example.com")
	mockDB.SeedDomain(uuid.New(), appID, "api.example.com")
	mockDB.SeedDomain(uuid.New(), otherAppID, "other.com")

	queries := NewMockQueries(mockDB)

	domains, err := queries.ListDomainsByApp(context.TODO(), appID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}
}
