package db_test

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/fuego-cloud/generated/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool
var testQueries *db.Queries

// TestMain sets up and tears down the test database
func TestMain(m *testing.M) {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Skip database tests if no database is available
		println("DB TESTS: Skipping - DATABASE_URL not set")
		os.Exit(0)
	}

	println("DB TESTS: Connecting to", dbURL)
	ctx := context.Background()

	// Connect to database
	var err error
	testPool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		// Skip if can't connect
		println("DB TESTS: Failed to create pool:", err.Error())
		os.Exit(0)
	}

	// Verify connection
	if err := testPool.Ping(ctx); err != nil {
		println("DB TESTS: Ping failed:", err.Error())
		testPool.Close()
		os.Exit(0)
	}

	println("DB TESTS: Connected successfully, running tests...")
	testQueries = db.New(testPool)

	// Run tests
	code := m.Run()

	// Cleanup
	testPool.Close()
	os.Exit(code)
}

// Helper to create a test user
func createTestUser(t *testing.T, ctx context.Context) db.User {
	t.Helper()

	avatarURL := "https://example.com/avatar.png"
	user, err := testQueries.CreateUser(ctx, db.CreateUserParams{
		GithubID:  int64(time.Now().UnixNano() % 1000000000),
		Username:  "testuser-" + uuid.New().String()[:8],
		Email:     "test-" + uuid.New().String()[:8] + "@example.com",
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// Helper to cleanup test user
func deleteTestUser(t *testing.T, ctx context.Context, id uuid.UUID) {
	t.Helper()
	_ = testQueries.DeleteUser(ctx, id)
}

// Helper to create a test app
func createTestApp(t *testing.T, ctx context.Context, userID uuid.UUID) db.App {
	t.Helper()

	app, err := testQueries.CreateApp(ctx, db.CreateAppParams{
		UserID: userID,
		Name:   "testapp-" + uuid.New().String()[:8],
		Region: "gdl",
		Size:   "starter",
	})
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	return app
}

// Helper to cleanup test app
func deleteTestApp(t *testing.T, ctx context.Context, id uuid.UUID) {
	t.Helper()
	_ = testQueries.DeleteApp(ctx, id)
}

// ============================================================================
// User Tests
// ============================================================================

func TestCreateUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	if user.ID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
	if user.Username == "" {
		t.Error("expected non-empty username")
	}
	if user.Plan != "free" {
		t.Errorf("expected default plan 'free', got %q", user.Plan)
	}
}

func TestGetUserByID(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	got, err := testQueries.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, got.ID)
	}
	if got.Username != user.Username {
		t.Errorf("expected username %q, got %q", user.Username, got.Username)
	}
}

func TestGetUserByGitHubID(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	got, err := testQueries.GetUserByGitHubID(ctx, user.GithubID)
	if err != nil {
		t.Fatalf("GetUserByGitHubID failed: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, got.ID)
	}
}

func TestGetUserByUsername(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	got, err := testQueries.GetUserByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, got.ID)
	}
}

func TestUpdateUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	newUsername := "updated-" + uuid.New().String()[:8]
	newEmail := "updated-" + uuid.New().String()[:8] + "@example.com"

	updated, err := testQueries.UpdateUser(ctx, db.UpdateUserParams{
		ID:        user.ID,
		Username:  newUsername,
		Email:     newEmail,
		AvatarUrl: user.AvatarUrl,
	})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if updated.Username != newUsername {
		t.Errorf("expected username %q, got %q", newUsername, updated.Username)
	}
	if updated.Email != newEmail {
		t.Errorf("expected email %q, got %q", newEmail, updated.Email)
	}
}

func TestUpdateUserPlan(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	stripeID := "cus_test123"
	updated, err := testQueries.UpdateUserPlan(ctx, db.UpdateUserPlanParams{
		ID:               user.ID,
		Plan:             "pro",
		StripeCustomerID: &stripeID,
	})
	if err != nil {
		t.Fatalf("UpdateUserPlan failed: %v", err)
	}

	if updated.Plan != "pro" {
		t.Errorf("expected plan 'pro', got %q", updated.Plan)
	}
}

func TestDeleteUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)

	err := testQueries.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify deletion
	_, err = testQueries.GetUserByID(ctx, user.ID)
	if err == nil {
		t.Error("expected error when getting deleted user")
	}
}

// ============================================================================
// App Tests
// ============================================================================

func TestCreateApp(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	if app.ID == uuid.Nil {
		t.Error("expected non-nil app ID")
	}
	if app.UserID != user.ID {
		t.Errorf("expected UserID %s, got %s", user.ID, app.UserID)
	}
	if app.Status != "stopped" {
		t.Errorf("expected status 'stopped', got %q", app.Status)
	}
	if app.DeploymentCount != 0 {
		t.Errorf("expected deployment count 0, got %d", app.DeploymentCount)
	}
}

func TestGetAppByName(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	got, err := testQueries.GetAppByName(ctx, db.GetAppByNameParams{
		UserID: user.ID,
		Name:   app.Name,
	})
	if err != nil {
		t.Fatalf("GetAppByName failed: %v", err)
	}

	if got.ID != app.ID {
		t.Errorf("expected ID %s, got %s", app.ID, got.ID)
	}
}

func TestListAppsByUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	// Create multiple apps
	app1 := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app1.ID)
	app2 := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app2.ID)

	apps, err := testQueries.ListAppsByUser(ctx, db.ListAppsByUserParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListAppsByUser failed: %v", err)
	}

	if len(apps) < 2 {
		t.Errorf("expected at least 2 apps, got %d", len(apps))
	}
}

func TestUpdateAppStatus(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	deploymentID := uuid.New()
	updated, err := testQueries.UpdateAppStatus(ctx, db.UpdateAppStatusParams{
		ID:                  app.ID,
		Status:              "running",
		CurrentDeploymentID: pgtype.UUID{Bytes: deploymentID, Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateAppStatus failed: %v", err)
	}

	if updated.Status != "running" {
		t.Errorf("expected status 'running', got %q", updated.Status)
	}
}

func TestIncrementDeploymentCount(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	originalCount := app.DeploymentCount

	updated, err := testQueries.IncrementDeploymentCount(ctx, app.ID)
	if err != nil {
		t.Fatalf("IncrementDeploymentCount failed: %v", err)
	}

	if updated.DeploymentCount != originalCount+1 {
		t.Errorf("expected deployment count %d, got %d", originalCount+1, updated.DeploymentCount)
	}
}

func TestCountAppsByUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	// Get initial count
	initialCount, err := testQueries.CountAppsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountAppsByUser failed: %v", err)
	}

	// Create app
	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	// Count should increase
	newCount, err := testQueries.CountAppsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountAppsByUser failed: %v", err)
	}

	if newCount != initialCount+1 {
		t.Errorf("expected count %d, got %d", initialCount+1, newCount)
	}
}

// ============================================================================
// Deployment Tests
// ============================================================================

func TestCreateDeployment(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 1,
		Image:   "nginx:alpine",
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteDeployment(ctx, deployment.ID) }()

	if deployment.ID == uuid.Nil {
		t.Error("expected non-nil deployment ID")
	}
	if deployment.Version != 1 {
		t.Errorf("expected version 1, got %d", deployment.Version)
	}
	if deployment.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", deployment.Status)
	}
}

func TestUpdateDeploymentStatus(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	deployment, err := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID:   app.ID,
		Version: 1,
		Image:   "nginx:alpine",
		Status:  "pending",
	})
	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteDeployment(ctx, deployment.ID) }()

	message := "Deployment successful"
	updated, err := testQueries.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
		ID:      deployment.ID,
		Status:  "running",
		Message: &message,
		Error:   nil,
	})
	if err != nil {
		t.Fatalf("UpdateDeploymentStatus failed: %v", err)
	}

	if updated.Status != "running" {
		t.Errorf("expected status 'running', got %q", updated.Status)
	}
}

func TestGetLatestDeployment(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	// Create multiple deployments
	d1, _ := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID: app.ID, Version: 1, Image: "nginx:1", Status: "running",
	})
	defer func() { _ = testQueries.DeleteDeployment(ctx, d1.ID) }()

	d2, _ := testQueries.CreateDeployment(ctx, db.CreateDeploymentParams{
		AppID: app.ID, Version: 2, Image: "nginx:2", Status: "running",
	})
	defer func() { _ = testQueries.DeleteDeployment(ctx, d2.ID) }()

	latest, err := testQueries.GetLatestDeployment(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetLatestDeployment failed: %v", err)
	}

	if latest.Version != 2 {
		t.Errorf("expected latest version 2, got %d", latest.Version)
	}
}

// ============================================================================
// Domain Tests
// ============================================================================

func TestCreateDomain(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

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
		t.Error("expected verified to be false initially")
	}
}

func TestUpdateDomainVerified(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	domain, _ := testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID:  app.ID,
		Domain: "verify-" + uuid.New().String()[:8] + ".example.com",
	})
	defer func() { _ = testQueries.DeleteDomain(ctx, domain.ID) }()

	updated, err := testQueries.UpdateDomainVerified(ctx, domain.ID)
	if err != nil {
		t.Fatalf("UpdateDomainVerified failed: %v", err)
	}

	if !updated.Verified {
		t.Error("expected domain to be verified")
	}
	if !updated.VerifiedAt.Valid {
		t.Error("expected VerifiedAt to be set")
	}
}

func TestListDomainsByApp(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	// Create multiple domains
	d1, _ := testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID: app.ID, Domain: "d1-" + uuid.New().String()[:8] + ".example.com",
	})
	defer func() { _ = testQueries.DeleteDomain(ctx, d1.ID) }()

	d2, _ := testQueries.CreateDomain(ctx, db.CreateDomainParams{
		AppID: app.ID, Domain: "d2-" + uuid.New().String()[:8] + ".example.com",
	})
	defer func() { _ = testQueries.DeleteDomain(ctx, d2.ID) }()

	domains, err := testQueries.ListDomainsByApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("ListDomainsByApp failed: %v", err)
	}

	if len(domains) < 2 {
		t.Errorf("expected at least 2 domains, got %d", len(domains))
	}
}

// ============================================================================
// API Token Tests
// ============================================================================

func TestCreateAPIToken(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	token, err := testQueries.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:    user.ID,
		Name:      "test-token",
		TokenHash: "hashed-token-value-" + uuid.New().String(),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateAPIToken failed: %v", err)
	}
	defer func() { _ = testQueries.DeleteAPIToken(ctx, token.ID) }()

	if token.Name != "test-token" {
		t.Errorf("expected name 'test-token', got %q", token.Name)
	}
}

func TestGetAPITokenByHash(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	tokenHash := "unique-hash-" + uuid.New().String()
	token, _ := testQueries.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:    user.ID,
		Name:      "hash-test-token",
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Valid: false},
	})
	defer func() { _ = testQueries.DeleteAPIToken(ctx, token.ID) }()

	got, err := testQueries.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetAPITokenByHash failed: %v", err)
	}

	if got.ID != token.ID {
		t.Errorf("expected ID %s, got %s", token.ID, got.ID)
	}
}

func TestUpdateAPITokenLastUsed(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	token, _ := testQueries.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:    user.ID,
		Name:      "last-used-test",
		TokenHash: "hash-" + uuid.New().String(),
		ExpiresAt: pgtype.Timestamptz{Valid: false},
	})
	defer func() { _ = testQueries.DeleteAPIToken(ctx, token.ID) }()

	// Should not error
	err := testQueries.UpdateAPITokenLastUsed(ctx, token.ID)
	if err != nil {
		t.Fatalf("UpdateAPITokenLastUsed failed: %v", err)
	}

	// Verify it was updated
	got, _ := testQueries.GetAPITokenByID(ctx, token.ID)
	if !got.LastUsedAt.Valid {
		t.Error("expected LastUsedAt to be set")
	}
}

// ============================================================================
// Activity Log Tests
// ============================================================================

func TestCreateActivityLog(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	ip := netip.MustParseAddr("192.168.1.1")
	log, err := testQueries.CreateActivityLog(ctx, db.CreateActivityLogParams{
		UserID:    pgtype.UUID{Bytes: user.ID, Valid: true},
		AppID:     pgtype.UUID{Bytes: app.ID, Valid: true},
		Action:    "deployment.created",
		Details:   []byte(`{"version": 1}`),
		IpAddress: &ip,
	})
	if err != nil {
		t.Fatalf("CreateActivityLog failed: %v", err)
	}

	if log.Action != "deployment.created" {
		t.Errorf("expected action 'deployment.created', got %q", log.Action)
	}
}

func TestListActivityLogsByApp(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not available")
	}

	ctx := context.Background()
	user := createTestUser(t, ctx)
	defer deleteTestUser(t, ctx, user.ID)

	app := createTestApp(t, ctx, user.ID)
	defer deleteTestApp(t, ctx, app.ID)

	// Create activity logs
	for i := 0; i < 3; i++ {
		_, _ = testQueries.CreateActivityLog(ctx, db.CreateActivityLogParams{
			UserID:    pgtype.UUID{Bytes: user.ID, Valid: true},
			AppID:     pgtype.UUID{Bytes: app.ID, Valid: true},
			Action:    "test.action",
			Details:   nil,
			IpAddress: nil,
		})
	}

	logs, err := testQueries.ListActivityLogsByApp(ctx, db.ListActivityLogsByAppParams{
		AppID:  pgtype.UUID{Bytes: app.ID, Valid: true},
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListActivityLogsByApp failed: %v", err)
	}

	if len(logs) < 3 {
		t.Errorf("expected at least 3 logs, got %d", len(logs))
	}
}
