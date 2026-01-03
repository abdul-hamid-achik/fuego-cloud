package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestGetUserIDFromContext_Valid(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)

	got, ok := GetUserIDFromContext(ctx)

	if !ok {
		t.Error("expected ok to be true")
	}
	if got != userID {
		t.Errorf("expected %s, got %s", userID, got)
	}
}

func TestGetUserIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	_, ok := GetUserIDFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when user ID is missing")
	}
}

func TestGetUserIDFromContext_WrongType(t *testing.T) {
	// Set a string instead of UUID
	ctx := context.WithValue(context.Background(), UserIDKey, "not-a-uuid")

	_, ok := GetUserIDFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when value is wrong type")
	}
}

func TestGetUserIDFromContext_NilUUID(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, uuid.Nil)

	got, ok := GetUserIDFromContext(ctx)

	if !ok {
		t.Error("expected ok to be true for nil UUID")
	}
	if got != uuid.Nil {
		t.Errorf("expected nil UUID, got %s", got)
	}
}

func TestGetUsernameFromContext_Valid(t *testing.T) {
	username := "testuser"
	ctx := context.WithValue(context.Background(), UsernameKey, username)

	got, ok := GetUsernameFromContext(ctx)

	if !ok {
		t.Error("expected ok to be true")
	}
	if got != username {
		t.Errorf("expected %q, got %q", username, got)
	}
}

func TestGetUsernameFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	_, ok := GetUsernameFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when username is missing")
	}
}

func TestGetUsernameFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UsernameKey, 12345)

	_, ok := GetUsernameFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when value is wrong type")
	}
}

func TestGetUsernameFromContext_EmptyString(t *testing.T) {
	ctx := context.WithValue(context.Background(), UsernameKey, "")

	got, ok := GetUsernameFromContext(ctx)

	if !ok {
		t.Error("expected ok to be true for empty string")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetClaimsFromContext_Valid(t *testing.T) {
	userID := uuid.New()
	claims := &Claims{
		UserID:   userID,
		Username: "testuser",
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	got, ok := GetClaimsFromContext(ctx)

	if !ok {
		t.Error("expected ok to be true")
	}
	if got.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, got.UserID)
	}
	if got.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %q", got.Username)
	}
}

func TestGetClaimsFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	_, ok := GetClaimsFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when claims are missing")
	}
}

func TestGetClaimsFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, "not-claims")

	_, ok := GetClaimsFromContext(ctx)

	if ok {
		t.Error("expected ok to be false when value is wrong type")
	}
}

func TestGetClaimsFromContext_NilClaims(t *testing.T) {
	var claims *Claims = nil
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	got, ok := GetClaimsFromContext(ctx)

	// Type assertion succeeds but value is nil
	if !ok {
		t.Error("expected ok to be true for nil claims pointer")
	}
	if got != nil {
		t.Error("expected nil claims")
	}
}

func TestSetUserInContext(t *testing.T) {
	userID := uuid.New()
	claims := &Claims{
		UserID:   userID,
		Username: "testuser",
	}

	ctx := SetUserInContext(context.Background(), claims)

	// Verify UserID is set
	gotUserID, ok := GetUserIDFromContext(ctx)
	if !ok {
		t.Error("expected UserID to be set in context")
	}
	if gotUserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, gotUserID)
	}

	// Verify Username is set
	gotUsername, ok := GetUsernameFromContext(ctx)
	if !ok {
		t.Error("expected Username to be set in context")
	}
	if gotUsername != "testuser" {
		t.Errorf("expected Username 'testuser', got %q", gotUsername)
	}

	// Verify Claims is set
	gotClaims, ok := GetClaimsFromContext(ctx)
	if !ok {
		t.Error("expected Claims to be set in context")
	}
	if gotClaims != claims {
		t.Error("expected same claims pointer")
	}
}

func TestSetUserInContext_PreservesExistingValues(t *testing.T) {
	// Set some existing value in context
	ctx := context.WithValue(context.Background(), contextKey("other"), "existing-value")

	claims := &Claims{
		UserID:   uuid.New(),
		Username: "newuser",
	}

	ctx = SetUserInContext(ctx, claims)

	// Verify existing value is preserved
	if ctx.Value(contextKey("other")) != "existing-value" {
		t.Error("expected existing context value to be preserved")
	}

	// Verify new values are set
	_, ok := GetUserIDFromContext(ctx)
	if !ok {
		t.Error("expected UserID to be set")
	}
}

func TestIsPublicPath_HealthEndpoint(t *testing.T) {
	if !IsPublicPath("/api/health") {
		t.Error("expected /api/health to be public")
	}
}

func TestIsPublicPath_HealthSubPath(t *testing.T) {
	if !IsPublicPath("/api/health/check") {
		t.Error("expected /api/health/check to be public")
	}
}

func TestIsPublicPath_AuthLogin(t *testing.T) {
	if !IsPublicPath("/api/auth/login") {
		t.Error("expected /api/auth/login to be public")
	}
}

func TestIsPublicPath_AuthCallback(t *testing.T) {
	if !IsPublicPath("/api/auth/callback") {
		t.Error("expected /api/auth/callback to be public")
	}
}

func TestIsPublicPath_AuthCallbackSubPath(t *testing.T) {
	if !IsPublicPath("/api/auth/callback/github") {
		t.Error("expected /api/auth/callback/github to be public")
	}
}

func TestIsPublicPath_PrivateEndpoints(t *testing.T) {
	privateEndpoints := []string{
		"/api/apps",
		"/api/apps/myapp",
		"/api/users/me",
		"/api/auth/token",
		"/api/registry/token",
		"/api/metrics",
		"/dashboard",
		"/",
	}

	for _, path := range privateEndpoints {
		if IsPublicPath(path) {
			t.Errorf("expected %q to be private", path)
		}
	}
}

func TestIsPublicPath_EmptyPath(t *testing.T) {
	if IsPublicPath("") {
		t.Error("expected empty path to be private")
	}
}

func TestIsPublicPath_PartialMatch(t *testing.T) {
	// /api/health-check should NOT match /api/health
	if IsPublicPath("/api/health-check") {
		t.Error("expected /api/health-check to be private (not a subpath of /api/health)")
	}
}

func TestIsPublicPath_CaseSensitive(t *testing.T) {
	// Paths should be case-sensitive
	if IsPublicPath("/api/HEALTH") {
		t.Error("expected /api/HEALTH to be private (case mismatch)")
	}
	if IsPublicPath("/API/health") {
		t.Error("expected /API/health to be private (case mismatch)")
	}
}

func TestContextKeys_Unique(t *testing.T) {
	// Ensure context keys are unique
	if UserIDKey == UsernameKey {
		t.Error("UserIDKey and UsernameKey should be different")
	}
	if UserIDKey == ClaimsKey {
		t.Error("UserIDKey and ClaimsKey should be different")
	}
	if UsernameKey == ClaimsKey {
		t.Error("UsernameKey and ClaimsKey should be different")
	}
}

func TestContextKey_String(t *testing.T) {
	// Verify context keys have expected string values
	if string(UserIDKey) != "user_id" {
		t.Errorf("expected UserIDKey to be 'user_id', got %q", string(UserIDKey))
	}
	if string(UsernameKey) != "username" {
		t.Errorf("expected UsernameKey to be 'username', got %q", string(UsernameKey))
	}
	if string(ClaimsKey) != "claims" {
		t.Errorf("expected ClaimsKey to be 'claims', got %q", string(ClaimsKey))
	}
}
