package crypto

import (
	"testing"
)

const testKey = "12345678901234567890123456789012"

func TestEncryptDecrypt(t *testing.T) {
	data := map[string]string{
		"DATABASE_URL": "postgres://localhost/myapp",
		"API_KEY":      "secret-api-key-123",
		"DEBUG":        "true",
	}

	encrypted, err := Encrypt(data, testKey)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("expected non-empty ciphertext")
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if len(decrypted) != len(data) {
		t.Errorf("expected %d entries, got %d", len(data), len(decrypted))
	}

	for k, v := range data {
		if decrypted[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, decrypted[k])
		}
	}
}

func TestEncryptEmptyMap(t *testing.T) {
	data := map[string]string{}

	encrypted, err := Encrypt(data, testKey)
	if err != nil {
		t.Fatalf("failed to encrypt empty map: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty map, got %d entries", len(decrypted))
	}
}

func TestDecryptEmptySlice(t *testing.T) {
	decrypted, err := Decrypt([]byte{}, testKey)
	if err != nil {
		t.Fatalf("failed to decrypt empty slice: %v", err)
	}

	if decrypted == nil {
		t.Error("expected non-nil map")
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty map, got %d entries", len(decrypted))
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"too short", "short"},
		{"too long", "12345678901234567890123456789012345"},
		{"empty", ""},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Encrypt(map[string]string{"key": "value"}, tt.key)
			if err == nil {
				t.Error("expected error for invalid key length")
			}
		})
	}
}

func TestDecryptInvalidKeyLength(t *testing.T) {
	data := map[string]string{"key": "value"}
	encrypted, _ := Encrypt(data, testKey)

	_, err := Decrypt(encrypted, "short-key")
	if err == nil {
		t.Error("expected error for invalid key length")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	data := map[string]string{"secret": "value"}
	encrypted, _ := Encrypt(data, testKey)

	wrongKey := "abcdefghijklmnopqrstuvwxyz123456"
	_, err := Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	data := map[string]string{"key": "value"}
	encrypted, _ := Encrypt(data, testKey)

	encrypted[10] ^= 0xFF

	_, err := Decrypt(encrypted, testKey)
	if err == nil {
		t.Error("expected error for corrupted data")
	}
}

func TestDecryptTooShortCiphertext(t *testing.T) {
	_, err := Decrypt([]byte{1, 2, 3}, testKey)
	if err == nil {
		t.Error("expected error for ciphertext too short")
	}
}

func TestEncryptLargeData(t *testing.T) {
	data := make(map[string]string)
	for i := 0; i < 100; i++ {
		data[string(rune('A'+i%26))+string(rune('0'+i))] = "value-" + string(rune('0'+i))
	}

	encrypted, err := Encrypt(data, testKey)
	if err != nil {
		t.Fatalf("failed to encrypt large data: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("failed to decrypt large data: %v", err)
	}

	if len(decrypted) != len(data) {
		t.Errorf("expected %d entries, got %d", len(data), len(decrypted))
	}
}

func TestEncryptSpecialCharacters(t *testing.T) {
	data := map[string]string{
		"UNICODE":   "こんにちは世界 🌍",
		"MULTILINE": "line1\nline2\nline3",
		"QUOTES":    `"quoted" and 'single'`,
		"EQUALS":    "key=value=extra",
		"EMPTY":     "",
		"SPACES":    "  leading and trailing  ",
	}

	encrypted, err := Encrypt(data, testKey)
	if err != nil {
		t.Fatalf("failed to encrypt special characters: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("failed to decrypt special characters: %v", err)
	}

	for k, v := range data {
		if decrypted[k] != v {
			t.Errorf("key %q: expected %q, got %q", k, v, decrypted[k])
		}
	}
}

func TestEncryptDeterministic(t *testing.T) {
	data := map[string]string{"key": "value"}

	encrypted1, _ := Encrypt(data, testKey)
	encrypted2, _ := Encrypt(data, testKey)

	if string(encrypted1) == string(encrypted2) {
		t.Error("encryption should be non-deterministic (random nonce)")
	}

	decrypted1, _ := Decrypt(encrypted1, testKey)
	decrypted2, _ := Decrypt(encrypted2, testKey)

	if decrypted1["key"] != decrypted2["key"] {
		t.Error("both decryptions should produce same result")
	}
}
