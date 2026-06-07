package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/waelson/fake-platform-api/internal/store"
)

// setupWithDeploy creates a router+store with a registered runtime agent and a pending deploy command.
// Returns router, store, agentID, depID, cmdID.
func setupWithDeploy(t *testing.T) (http.Handler, *store.Store, string, string, string) {
	t.Helper()
	cfg := defaultCfg()
	st := store.New("host.docker.internal")
	router := NewRouter(cfg, st)

	a := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		InstanceID: "inst-1", Version: "0.1.0",
	})

	dep, cmd := st.InitDeploy(store.InitDeployInput{
		Application:           "app",
		Environment:           "dev",
		Image:                 "img:v1",
		Host:                  "app.dev.local",
		TargetAgentID:         a.ID,
		ContainerInternalPort: 3000,
		HealthCheckPath:       "/health",
		RequiresRoute:         true,
		CommandTimeoutSeconds: 600,
	})
	return router, st, a.ID, dep.ID, cmd.ID
}

// ---- GET /api/agents/{agentID}/commands/pending ----

func TestHandleListPendingCommands_EmptyArray(t *testing.T) {
	cfg := defaultCfg()
	st := store.New("host.docker.internal")
	router := NewRouter(cfg, st)
	a := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1",
	})

	w := doRequest(router, http.MethodGet, "/api/agents/"+a.ID+"/commands/pending", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var cmds []any
	json.NewDecoder(w.Body).Decode(&cmds)
	if len(cmds) != 0 {
		t.Errorf("expected empty array, got %d items", len(cmds))
	}
}

func TestHandleListPendingCommands_WithCommands(t *testing.T) {
	router, _, agentID, _, cmdID := setupWithDeploy(t)

	w := doRequest(router, http.MethodGet, "/api/agents/"+agentID+"/commands/pending", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var cmds []map[string]any
	json.NewDecoder(w.Body).Decode(&cmds)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0]["id"] != cmdID {
		t.Errorf("command id: got %v", cmds[0]["id"])
	}
	if cmds[0]["status"] != "pending" {
		t.Errorf("status: got %v", cmds[0]["status"])
	}
	if cmds[0]["payload"] == nil {
		t.Error("payload must be present in pending command")
	}
}

func TestHandleListPendingCommands_AgentNotFound(t *testing.T) {
	router := newTestRouter(defaultCfg())
	w := doRequest(router, http.MethodGet, "/api/agents/agent-dev-api-999/commands/pending", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/claim ----

func TestHandleClaimCommand_Success(t *testing.T) {
	router, _, agentID, _, cmdID := setupWithDeploy(t)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/claim", agentID, cmdID),
		`{"status":"claimed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["status"] != "claimed" {
		t.Errorf("status: got %v", body["status"])
	}
	if body["claimed_by"] != agentID {
		t.Errorf("claimed_by: got %v", body["claimed_by"])
	}
	if body["claimed_at"] == nil {
		t.Error("claimed_at must be set")
	}
}

func TestHandleClaimCommand_Idempotent(t *testing.T) {
	router, _, agentID, _, cmdID := setupWithDeploy(t)
	path := fmt.Sprintf("/api/agents/%s/commands/%s/claim", agentID, cmdID)
	doRequest(router, http.MethodPost, path, `{"status":"claimed"}`)

	w := doRequest(router, http.MethodPost, path, `{"status":"claimed"}`)
	if w.Code != http.StatusOK {
		t.Errorf("second claim: got %d, want 200", w.Code)
	}
}

func TestHandleClaimCommand_Conflict_DifferentAgent(t *testing.T) {
	router, st, _, _, cmdID := setupWithDeploy(t)

	a2 := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i2",
	})
	doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/agent-dev-api-001/commands/%s/claim", cmdID),
		`{"status":"claimed"}`)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/claim", a2.ID, cmdID),
		`{"status":"claimed"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("different agent claim: got %d, want 409", w.Code)
	}
}

func TestHandleClaimCommand_NotFound(t *testing.T) {
	router, _, agentID, _, _ := setupWithDeploy(t)
	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/cmd-999/claim", agentID),
		`{"status":"claimed"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/start ----

func TestHandleStartCommand_Success(t *testing.T) {
	router, st, agentID, depID, cmdID := setupWithDeploy(t)

	doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/claim", agentID, cmdID),
		`{"status":"claimed"}`)

	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/start", agentID, cmdID),
		`{"status":"running"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["status"] != "running" {
		t.Errorf("status: got %v", body["status"])
	}
	if body["started_at"] == nil {
		t.Error("started_at must be set")
	}

	dep, _ := st.GetDeployment(depID)
	if dep.Status != "deploying" {
		t.Errorf("dep status: got %q, want deploying", dep.Status)
	}
}

func TestHandleStartCommand_Conflict_NotClaimed(t *testing.T) {
	router, _, agentID, _, cmdID := setupWithDeploy(t)
	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/start", agentID, cmdID),
		`{"status":"running"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("status: got %d, want 409", w.Code)
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/report ----

func TestHandleReportCommand_Succeeded(t *testing.T) {
	router, st, agentID, depID, cmdID := setupWithDeploy(t)

	claimPath := fmt.Sprintf("/api/agents/%s/commands/%s/claim", agentID, cmdID)
	startPath := fmt.Sprintf("/api/agents/%s/commands/%s/start", agentID, cmdID)
	reportPath := fmt.Sprintf("/api/agents/%s/commands/%s/report", agentID, cmdID)
	doRequest(router, http.MethodPost, claimPath, `{"status":"claimed"}`)
	doRequest(router, http.MethodPost, startPath, `{"status":"running"}`)

	w := doRequest(router, http.MethodPost, reportPath, `{
		"status":"succeeded",
		"deployment_id":"dep-001",
		"result":{"runtime_private_ip":"10.0.0.1","host_port":4100,"requires_route":true}
	}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["status"] != "accepted" {
		t.Errorf("status: got %v", body["status"])
	}

	dep, _ := st.GetDeployment(depID)
	if dep.Status != "route_pending" {
		t.Errorf("dep status: got %q, want route_pending", dep.Status)
	}
}

func TestHandleReportCommand_Failed(t *testing.T) {
	router, st, agentID, depID, cmdID := setupWithDeploy(t)

	doRequest(router, http.MethodPost, fmt.Sprintf("/api/agents/%s/commands/%s/claim", agentID, cmdID), `{}`)
	doRequest(router, http.MethodPost, fmt.Sprintf("/api/agents/%s/commands/%s/start", agentID, cmdID), `{}`)
	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/%s/report", agentID, cmdID),
		`{"status":"failed","deployment_id":"dep-001","error":{"code":"HEALTH_CHECK_FAILED","message":"unhealthy","retryable":false}}`)

	if w.Code != http.StatusOK {
		t.Fatalf("failed report should still return 200: got %d", w.Code)
	}

	dep, _ := st.GetDeployment(depID)
	if dep.Status != "failed" {
		t.Errorf("dep status: got %q, want failed", dep.Status)
	}
}

func TestHandleReportCommand_NotFound(t *testing.T) {
	router, _, agentID, _, _ := setupWithDeploy(t)
	w := doRequest(router, http.MethodPost,
		fmt.Sprintf("/api/agents/%s/commands/cmd-999/report", agentID),
		`{"status":"succeeded"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}
