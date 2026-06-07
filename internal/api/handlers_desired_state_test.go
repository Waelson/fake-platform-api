package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/waelson/fake-platform-api/internal/store"
)

// setupWithRoute creates a store where a deploy completed successfully (route exists, dep is route_pending).
func setupWithRoute(t *testing.T) (http.Handler, *store.Store, string, string, string) {
	t.Helper()
	cfg := defaultCfg()
	st := store.New("host.docker.internal")
	router := NewRouter(cfg, st)

	runtime := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "rt-1",
	})
	gateway := st.RegisterAgent(store.RegisterInput{
		Mode: "gateway", Environment: "dev", Role: "gateway", InstanceID: "gw-1",
	})

	dep, cmd := st.InitDeploy(store.InitDeployInput{
		Application:           "app",
		Environment:           "dev",
		Image:                 "img:v1",
		Host:                  "app.dev.local",
		TargetAgentID:         runtime.ID,
		ContainerInternalPort: 3000,
		HealthCheckPath:       "/health",
		RequiresRoute:         true,
		CommandTimeoutSeconds: 600,
	})

	st.ClaimCommand(cmd.ID, runtime.ID)
	st.StartCommand(cmd.ID, runtime.ID)
	result, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4100})
	st.ReportCommand(cmd.ID, store.ReportCommandInput{
		AgentID: runtime.ID, Status: "succeeded", Result: result,
	})

	return router, st, runtime.ID, gateway.ID, dep.ID
}

// ---- GET /api/agents/{agentID}/desired-state ----

func TestHandleGetDesiredState_Empty(t *testing.T) {
	cfg := defaultCfg()
	st := store.New("host.docker.internal")
	router := NewRouter(cfg, st)
	a := st.RegisterAgent(store.RegisterInput{
		Mode: "gateway", Environment: "dev", Role: "gateway", InstanceID: "gw-1",
	})

	w := doRequest(router, http.MethodGet, "/api/agents/"+a.ID+"/desired-state", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var ds map[string]any
	json.NewDecoder(w.Body).Decode(&ds)
	if ds["version"].(float64) != 0 {
		t.Errorf("version: got %v, want 0", ds["version"])
	}
	if ds["environment"] != "dev" {
		t.Errorf("environment: got %v", ds["environment"])
	}
	if ds["type"] != "gateway_routes" {
		t.Errorf("type: got %v", ds["type"])
	}
	routes, _ := ds["routes"].([]any)
	if len(routes) != 0 {
		t.Errorf("routes: got %d, want 0", len(routes))
	}
}

func TestHandleGetDesiredState_WithRoutes(t *testing.T) {
	router, _, _, gatewayID, _ := setupWithRoute(t)

	w := doRequest(router, http.MethodGet, "/api/agents/"+gatewayID+"/desired-state", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var ds map[string]any
	json.NewDecoder(w.Body).Decode(&ds)
	if ds["version"].(float64) != 1 {
		t.Errorf("version: got %v, want 1", ds["version"])
	}
	routes, _ := ds["routes"].([]any)
	if len(routes) != 1 {
		t.Fatalf("routes: got %d, want 1", len(routes))
	}
	route := routes[0].(map[string]any)
	if route["host"] != "app.dev.local" {
		t.Errorf("route host: got %v", route["host"])
	}
	if route["upstream"] != "host.docker.internal:4100" {
		t.Errorf("upstream: got %v", route["upstream"])
	}
}

func TestHandleGetDesiredState_AgentNotFound(t *testing.T) {
	router := newTestRouter(defaultCfg())
	w := doRequest(router, http.MethodGet, "/api/agents/agent-dev-gw-999/desired-state", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---- POST /api/agents/{agentID}/desired-state/report ----

func TestHandleDesiredStateReport_Applied(t *testing.T) {
	router, st, _, gatewayID, depID := setupWithRoute(t)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/desired-state/report", gatewayID),
		`{"status":"applied","desired_state_version":1,"type":"gateway_routes","environment":"dev","routes_total":1,"validated_routes":1,"failed_routes":0}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["status"] != "accepted" {
		t.Errorf("status: got %v", body["status"])
	}

	dep, _ := st.GetDeployment(depID)
	if dep.Status != "route_active" {
		t.Errorf("dep status: got %q, want route_active", dep.Status)
	}
}

func TestHandleDesiredStateReport_Failed(t *testing.T) {
	router, st, _, gatewayID, depID := setupWithRoute(t)

	doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/desired-state/report", gatewayID),
		`{"status":"failed","desired_state_version":1,"type":"gateway_routes","environment":"dev"}`)

	dep, _ := st.GetDeployment(depID)
	if dep.Status != "route_failed" {
		t.Errorf("dep status: got %q, want route_failed", dep.Status)
	}
}

func TestHandleDesiredStateReport_Stale(t *testing.T) {
	router, _, _, gatewayID, _ := setupWithRoute(t)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/desired-state/report", gatewayID),
		`{"status":"applied","desired_state_version":0,"type":"gateway_routes","environment":"dev"}`)
	if w.Code != http.StatusOK {
		t.Errorf("stale must return 200: got %d", w.Code)
	}
}

func TestHandleDesiredStateReport_FutureVersion(t *testing.T) {
	router, _, _, gatewayID, _ := setupWithRoute(t)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/desired-state/report", gatewayID),
		`{"status":"applied","desired_state_version":99,"type":"gateway_routes","environment":"dev"}`)

	if w.Code != http.StatusConflict {
		t.Errorf("future version must return 409: got %d", w.Code)
	}
	body := decodeBody(t, w)
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "INVALID_VERSION" {
		t.Errorf("error code: got %v", errObj["code"])
	}
}

func TestHandleDesiredStateReport_BadJSON(t *testing.T) {
	router, _, _, gatewayID, _ := setupWithRoute(t)
	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/desired-state/report", gatewayID), `not-json`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad JSON: got %d, want 400", w.Code)
	}
}
