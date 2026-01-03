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

func TestHashToken_NotReversible(t *testing.T) {
	// This test ensures the hash is NOT just hex encoding (the old vulnerable implementation)
	token := "fgt_abc123def456"
	hash := HashToken(token)

	// The old implementation would produce: hex.EncodeToString([]byte(token))
	// which is reversible. The new implementation uses SHA-256.

	// SHA-256 produces a 64-character hex string (256 bits = 32 bytes = 64 hex chars)
	if len(hash) != 64 {
		t.Errorf("expected SHA-256 hash length of 64, got %d", len(hash))
	}

	// Verify it's not just hex encoding of the input (old vulnerable behavior)
	// Old behavior would have produced a much longer string for this input
	oldVulnerableHash := make([]byte, len(token)*2)
	for i, b := range []byte(token) {
		oldVulnerableHash[i*2] = "0123456789abcdef"[b>>4]
		oldVulnerableHash[i*2+1] = "0123456789abcdef"[b&0xf]
	}

	if hash == string(oldVulnerableHash) {
		t.Error("SECURITY VULNERABILITY: HashToken is using reversible hex encoding instead of proper hashing")
	}
}

func TestHashToken_UniqueOutputs(t *testing.T) {
	// Different inputs should produce different hashes
	tokens := []string{
		"token1",
		"token2",
		"Token1",  // case sensitivity
		"token1 ", // trailing space
		" token1", // leading space
		"fgt_abc123",
		"fgt_abc124", // single char difference
	}

	hashes := make(map[string]string)
	for _, token := range tokens {
		hash := HashToken(token)
		if existingToken, exists := hashes[hash]; exists {
			t.Errorf("hash collision: %q and %q produce the same hash", token, existingToken)
		}
		hashes[hash] = token
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "fgt_deterministic_test_token_12345"

	// Hash the same token multiple times
	hashes := make([]string, 100)
	for i := 0; i < 100; i++ {
		hashes[i] = HashToken(token)
	}

	// All hashes should be identical
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("hash at index %d differs from hash at index 0", i)
		}
	}
}

func TestHashToken_EmptyInput(t *testing.T) {
	hash := HashToken("")

	// SHA-256 of empty string is a known value
	// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if hash != expectedEmptyHash {
		t.Errorf("expected SHA-256 of empty string, got %s", hash)
	}
}

func TestHashToken_KnownValue(t *testing.T) {
	// Test against a known SHA-256 hash to verify implementation
	// SHA-256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	hash := HashToken("hello")
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}
