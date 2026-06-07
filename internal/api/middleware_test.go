package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/waelson/fake-platform-api/internal/config"
)

func TestAuthMiddleware_Disabled(t *testing.T) {
	router := newTestRouter(&config.Config{AuthEnabled: false, Token: "secret"})

	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{
		"mode":"runtime","environment":"dev","role":"api","instance_id":"i1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// No auth header needed — must not be 401
	if w.Code == http.StatusUnauthorized {
		t.Error("auth disabled: must not return 401")
	}
}

func TestAuthMiddleware_Enabled_ValidToken(t *testing.T) {
	router := newTestRouter(&config.Config{AuthEnabled: true, Token: "my-token"})

	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{
		"mode":"runtime","environment":"dev","role":"api","instance_id":"i1"
	}`))
	req.Header.Set("Authorization", "Bearer my-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Error("valid token must not return 401")
	}
}

func TestAuthMiddleware_Enabled_MissingToken(t *testing.T) {
	router := newTestRouter(&config.Config{AuthEnabled: true, Token: "my-token"})

	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing token: got %d, want 401", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "AUTHENTICATION_FAILED" {
		t.Errorf("error code: got %v", errObj["code"])
	}
}

func TestAuthMiddleware_Enabled_WrongToken(t *testing.T) {
	router := newTestRouter(&config.Config{AuthEnabled: true, Token: "correct"})

	req := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", w.Code)
	}
}
