package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/auth"
	"github.com/abdul-hamid-achik/fuego-cloud/internal/config"
	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
	"github.com/google/uuid"
)

type TestApp struct {
	App    *fuego.App
	Config *config.Config
}

func NewTestApp() *TestApp {
	cfg := &config.Config{
		Port:             3000,
		Host:             "localhost",
		Environment:      "test",
		JWTSecret:        "test-jwt-secret-key-for-testing-purposes-only",
		EncryptionKey:    "test-encryption-key-32-bytes!!!",
		AppsDomainSuffix: "test.fuego.build",
		PlatformDomain:   "cloud.test.fuego.build",
	}

	app := fuego.New()
	app.Use(func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			c.Set("config", cfg)
			return next(c)
		}
	})

	return &TestApp{
		App:    app,
		Config: cfg,
	}
}

func (ta *TestApp) WithMockDB(mockDB *MockDB) *TestApp {
	ta.App.Use(func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			c.Set("db", mockDB)
			return next(c)
		}
	})
	return ta
}

func (ta *TestApp) WithAuth(userID uuid.UUID, username string) *TestApp {
	ta.App.Use(func(next fuego.HandlerFunc) fuego.HandlerFunc {
		return func(c *fuego.Context) error {
			c.Set("user_id", userID)
			c.Set("username", username)
			return next(c)
		}
	})
	return ta
}

func GenerateTestToken(t *testing.T, cfg *config.Config, userID uuid.UUID, username string) string {
	t.Helper()
	tokens, err := auth.GenerateTokenPair(userID, username, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return tokens.AccessToken
}

func MakeRequest(t *testing.T, method, path string, body any, headers map[string]string) *http.Request {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req
}

func ParseResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()

	var result T
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, w.Body.String())
	}
	return result
}

func AssertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected status %d, got %d (body: %s)", expected, w.Code, w.Body.String())
	}
}

func AssertJSONContains(t *testing.T, w *httptest.ResponseRecorder, key, expected string) {
	t.Helper()

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if val, ok := result[key].(string); !ok || val != expected {
		t.Errorf("expected %s=%q, got %q", key, expected, val)
	}
}

type MockDB struct {
	Users       map[uuid.UUID]db.User
	Apps        map[uuid.UUID]db.App
	Deployments map[uuid.UUID]db.Deployment
	Domains     map[uuid.UUID]db.Domain
	APITokens   map[uuid.UUID]db.ApiToken
	OAuthStates map[string]db.OauthState
}

func NewMockDB() *MockDB {
	return &MockDB{
		Users:       make(map[uuid.UUID]db.User),
		Apps:        make(map[uuid.UUID]db.App),
		Deployments: make(map[uuid.UUID]db.Deployment),
		Domains:     make(map[uuid.UUID]db.Domain),
		APITokens:   make(map[uuid.UUID]db.ApiToken),
		OAuthStates: make(map[string]db.OauthState),
	}
}

func (m *MockDB) SeedUser(id uuid.UUID, username, email string) db.User {
	user := db.User{
		ID:        id,
		GithubID:  12345,
		Username:  username,
		Email:     email,
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.Users[id] = user
	return user
}

func (m *MockDB) SeedApp(id, userID uuid.UUID, name string) db.App {
	app := db.App{
		ID:              id,
		UserID:          userID,
		Name:            name,
		Region:          "gdl",
		Size:            "starter",
		Status:          "running",
		DeploymentCount: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	m.Apps[id] = app
	return app
}

func (m *MockDB) SeedDeployment(id, appID uuid.UUID, version int32) db.Deployment {
	deployment := db.Deployment{
		ID:        id,
		AppID:     appID,
		Version:   version,
		Image:     "ghcr.io/test/image:v" + string(rune('0'+version)),
		Status:    "running",
		CreatedAt: time.Now(),
	}
	m.Deployments[id] = deployment
	return deployment
}

func (m *MockDB) SeedDomain(id, appID uuid.UUID, domain string) db.Domain {
	d := db.Domain{
		ID:        id,
		AppID:     appID,
		Domain:    domain,
		Verified:  false,
		SslStatus: "pending",
		CreatedAt: time.Now(),
	}
	m.Domains[id] = d
	return d
}

type MockQueries struct {
	db *MockDB
}

func NewMockQueries(mockDB *MockDB) *MockQueries {
	return &MockQueries{db: mockDB}
}

func (q *MockQueries) GetUserByGithubID(ctx context.Context, githubID int64) (db.User, error) {
	for _, u := range q.db.Users {
		if u.GithubID == githubID {
			return u, nil
		}
	}
	return db.User{}, context.DeadlineExceeded
}

func (q *MockQueries) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if user, ok := q.db.Users[id]; ok {
		return user, nil
	}
	return db.User{}, context.DeadlineExceeded
}

func (q *MockQueries) ListAppsByUser(ctx context.Context, userID uuid.UUID) ([]db.App, error) {
	var apps []db.App
	for _, app := range q.db.Apps {
		if app.UserID == userID {
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func (q *MockQueries) GetAppByName(ctx context.Context, params db.GetAppByNameParams) (db.App, error) {
	for _, app := range q.db.Apps {
		if app.UserID == params.UserID && app.Name == params.Name {
			return app, nil
		}
	}
	return db.App{}, context.DeadlineExceeded
}

func (q *MockQueries) CreateApp(ctx context.Context, params db.CreateAppParams) (db.App, error) {
	app := db.App{
		ID:              uuid.New(),
		UserID:          params.UserID,
		Name:            params.Name,
		Region:          params.Region,
		Size:            params.Size,
		Status:          "created",
		DeploymentCount: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	q.db.Apps[app.ID] = app
	return app, nil
}

func (q *MockQueries) UpdateApp(ctx context.Context, params db.UpdateAppParams) (db.App, error) {
	if app, ok := q.db.Apps[params.ID]; ok {
		app.Name = params.Name
		app.Region = params.Region
		app.Size = params.Size
		app.UpdatedAt = time.Now()
		q.db.Apps[params.ID] = app
		return app, nil
	}
	return db.App{}, context.DeadlineExceeded
}

func (q *MockQueries) DeleteApp(ctx context.Context, id uuid.UUID) error {
	if _, ok := q.db.Apps[id]; ok {
		delete(q.db.Apps, id)
		return nil
	}
	return context.DeadlineExceeded
}

func (q *MockQueries) ListDeploymentsByApp(ctx context.Context, params db.ListDeploymentsByAppParams) ([]db.Deployment, error) {
	var deps []db.Deployment
	for _, d := range q.db.Deployments {
		if d.AppID == params.AppID {
			deps = append(deps, d)
		}
	}
	return deps, nil
}

func (q *MockQueries) ListDomainsByApp(ctx context.Context, appID uuid.UUID) ([]db.Domain, error) {
	var domains []db.Domain
	for _, d := range q.db.Domains {
		if d.AppID == appID {
			domains = append(domains, d)
		}
	}
	return domains, nil
}
