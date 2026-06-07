package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/waelson/fake-platform-api/internal/store"
)

// ---- helpers ----

// doRequest sends a request through the router and returns the response recorder.
func doRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

// ---- POST /api/agents/register ----

func TestHandleRegisterAgent_Success(t *testing.T) {
	router := newTestRouter(defaultCfg())

	w := doRequest(router, http.MethodPost, "/api/agents/register", `{
		"mode":"runtime","environment":"dev","role":"api",
		"hostname":"host-1","instance_id":"inst-1",
		"private_ip":"10.0.0.1","version":"0.1.0"
	}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	body := decodeBody(t, w)
	if body["agent_id"] != "agent-dev-api-001" {
		t.Errorf("agent_id: got %v", body["agent_id"])
	}
	if body["status"] != "registered" {
		t.Errorf("status: got %v", body["status"])
	}
}

func TestHandleRegisterAgent_Upsert_PreservesID(t *testing.T) {
	router := newTestRouter(defaultCfg())

	payload := `{"mode":"runtime","environment":"dev","role":"api","instance_id":"inst-1","hostname":"h1"}`
	doRequest(router, http.MethodPost, "/api/agents/register", payload)
	w := doRequest(router, http.MethodPost, "/api/agents/register", payload)

	body := decodeBody(t, w)
	if body["agent_id"] != "agent-dev-api-001" {
		t.Errorf("re-register must preserve agent_id: got %v", body["agent_id"])
	}
}

func TestHandleRegisterAgent_MissingFields(t *testing.T) {
	router := newTestRouter(defaultCfg())

	w := doRequest(router, http.MethodPost, "/api/agents/register", `{"mode":"runtime"}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
	body := decodeBody(t, w)
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "INVALID_REQUEST" {
		t.Errorf("error code: %v", errObj["code"])
	}
}

func TestHandleRegisterAgent_BadJSON(t *testing.T) {
	router := newTestRouter(defaultCfg())
	w := doRequest(router, http.MethodPost, "/api/agents/register", `not-json`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ---- POST /api/agents/{agentID}/heartbeat ----

func registerTestAgent(t *testing.T, st *store.Store) string {
	t.Helper()
	a := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		InstanceID: "inst-1", Hostname: "host-1", Version: "0.1.0",
	})
	return a.ID
}

func TestHandleHeartbeat_Success(t *testing.T) {
	cfg := defaultCfg()
	st := store.New()
	router := NewRouter(cfg, st)
	agentID := registerTestAgent(t, st)

	w := doRequest(router, http.MethodPost, "/api/agents/"+agentID+"/heartbeat", `{
		"status":"online","version":"0.2.0","running_containers":1
	}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	body := decodeBody(t, w)
	if body["status"] != "accepted" {
		t.Errorf("status: got %v", body["status"])
	}
	if _, ok := body["server_time"]; !ok {
		t.Error("missing server_time")
	}
}

func TestHandleHeartbeat_NotFound(t *testing.T) {
	router := newTestRouter(defaultCfg())
	w := doRequest(router, http.MethodPost, "/api/agents/agent-dev-api-999/heartbeat", `{"status":"online"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandleHeartbeat_UpdatesAgentFields(t *testing.T) {
	cfg := defaultCfg()
	st := store.New()
	router := NewRouter(cfg, st)
	agentID := registerTestAgent(t, st)

	doRequest(router, http.MethodPost, "/api/agents/"+agentID+"/heartbeat", `{
		"status":"degraded","version":"0.3.0","private_ip":"10.0.0.9"
	}`)

	a, _ := st.GetAgent(agentID)
	if a.Status != "degraded" {
		t.Errorf("status not updated: got %q", a.Status)
	}
	if a.Version != "0.3.0" {
		t.Errorf("version not updated: got %q", a.Version)
	}
	if a.PrivateIP != "10.0.0.9" {
		t.Errorf("private_ip not updated: got %q", a.PrivateIP)
	}
}
