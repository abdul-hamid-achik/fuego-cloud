package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateTokenPair(t *testing.T) {
	userID := uuid.New()
	username := "testuser"
	secret := "test-secret-key-for-jwt"

	tokens, err := GenerateTokenPair(userID, username, secret)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	if tokens.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}

	if tokens.TokenType != "Bearer" {
		t.Errorf("expected token type 'Bearer', got %q", tokens.TokenType)
	}

	if tokens.ExpiresAt.Before(time.Now()) {
		t.Error("expected expires_at to be in the future")
	}

	if tokens.ExpiresAt.After(time.Now().Add(16 * time.Minute)) {
		t.Error("expected expires_at to be within 16 minutes")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	userID := uuid.New()
	username := "testuser"
	secret := "test-secret-key-for-jwt"

	tokens, err := GenerateTokenPair(userID, username, secret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateToken(tokens.AccessToken, secret)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, claims.UserID)
	}

	if claims.Username != username {
		t.Errorf("expected username %q, got %q", username, claims.Username)
	}

	if claims.Issuer != "fuego-cloud" {
		t.Errorf("expected issuer 'fuego-cloud', got %q", claims.Issuer)
	}
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	userID := uuid.New()
	username := "testuser"

	tokens, err := GenerateTokenPair(userID, username, "secret1")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateToken(tokens.AccessToken, "secret2")
	if err == nil {
		t.Error("expected error for invalid secret")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	expiredClaims := Claims{
		UserID:   userID,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "fuego-cloud",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ValidateToken(tokenString, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	_, err := ValidateToken("not-a-valid-token", "secret")
	if err == nil {
		t.Error("expected error for malformed token")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := ValidateToken("", "secret")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestGenerateAPIToken(t *testing.T) {
	token, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.HasPrefix(token, "fgt_") {
		t.Errorf("expected token to start with 'fgt_', got %q", token)
	}

	if len(token) < 68 {
		t.Errorf("expected token length >= 68, got %d", len(token))
	}

	token2, _ := GenerateAPIToken()
	if token == token2 {
		t.Error("expected unique tokens")
	}
}

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(state) != 32 {
		t.Errorf("expected state length 32, got %d", len(state))
	}

	state2, _ := GenerateState()
	if state == state2 {
		t.Error("expected unique states")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer abc123",
			expected: "abc123",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "no bearer prefix",
			header:   "abc123",
			expected: "",
		},
		{
			name:     "lowercase bearer",
			header:   "bearer abc123",
			expected: "abc123",
		},
		{
			name:     "bearer only no token",
			header:   "Bearer",
			expected: "",
		},
		{
			name:     "token with spaces is full second part",
			header:   "Bearer token with spaces",
			expected: "token with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractBearerToken(tt.header)
			if got != tt.expected {
				t.Errorf("ExtractBearerToken(%q) = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "test-token"
	hash := HashToken(token)

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	if hash == token {
		t.Error("expected hash to be different from token")
	}

	hash2 := HashToken(token)
	if hash != hash2 {
		t.Error("expected same hash for same input")
	}
}
