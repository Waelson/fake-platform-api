package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusUnauthorized, "AUTHENTICATION_FAILED", "Invalid or missing token", false)

	res := w.Result()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	var body ErrorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Code != "AUTHENTICATION_FAILED" {
		t.Errorf("code: got %q, want %q", body.Error.Code, "AUTHENTICATION_FAILED")
	}
	if body.Error.Message != "Invalid or missing token" {
		t.Errorf("message: got %q", body.Error.Message)
	}
	if body.Error.Retryable {
		t.Error("retryable: got true, want false")
	}
}

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("body[status]: got %q, want %q", body["status"], "ok")
	}
}
