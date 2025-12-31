package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

func TestHealthGet_NoDatabase(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	c := fuego.NewContext(w, req)

	err := Get(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}

	expectedFields := []string{"status", "database", "version"}
	for _, field := range expectedFields {
		if !contains(body, field) {
			t.Errorf("expected response to contain %q, got %s", field, body)
		}
	}

	if !contains(body, "disconnected") {
		t.Errorf("expected database status to be 'disconnected' when no db, got %s", body)
	}
}

func TestHealthGet_ResponseFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	c := fuego.NewContext(w, req)

	err := Get(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
