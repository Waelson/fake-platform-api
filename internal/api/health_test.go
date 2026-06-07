package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/store"
)

func newTestRouter(cfg *config.Config) http.Handler {
	return NewRouter(cfg, store.New())
}

func defaultCfg() *config.Config {
	return &config.Config{Port: "8080", Environment: "dev", AuthEnabled: false, Token: "dev-token"}
}

func TestHealthEndpoint(t *testing.T) {
	router := newTestRouter(defaultCfg())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field: got %v", body["status"])
	}
	if _, ok := body["timestamp"]; !ok {
		t.Error("missing timestamp field")
	}
}

func TestHealthNotFound(t *testing.T) {
	router := newTestRouter(defaultCfg())

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Result().StatusCode)
	}
}

func TestHealthBypassesAuth(t *testing.T) {
	router := newTestRouter(&config.Config{
		Port: "8080", Environment: "dev", AuthEnabled: true, Token: "secret",
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// No Authorization header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("health must bypass auth: got %d", w.Result().StatusCode)
	}
}
