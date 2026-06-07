package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/store"
)

// testingCfg returns a config with environment "dev" (default for testing endpoints).
func testingCfg() *config.Config {
	return &config.Config{
		Port:        "8080",
		AuthEnabled: false,
		Token:       "dev-token",
		Environment: "dev",
	}
}

// setupTestingWithAgent registers a runtime agent in environment "dev" and returns router+store+agentID.
func setupTestingWithAgent(t *testing.T) (http.Handler, *store.Store, string) {
	t.Helper()
	cfg := testingCfg()
	st := store.New()
	router := NewRouter(cfg, st)
	a := st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		InstanceID: "inst-rt-1", Version: "0.1.0",
	})
	return router, st, a.ID
}

// ---- POST /testing/commands/deploy ----

func TestTestingDeploy_Success(t *testing.T) {
	router, _, agentID := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"billing-api",
		"image":"fake-api:v1",
		"host":"billing-api.dev.local",
		"container_internal_port":3000,
		"health_check_path":"/health",
		"requires_route":true
	}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["deployment_id"] != "dep-001" {
		t.Errorf("deployment_id: got %v", body["deployment_id"])
	}
	if body["command_id"] != "cmd-001" {
		t.Errorf("command_id: got %v", body["command_id"])
	}
	if body["target_agent_id"] != agentID {
		t.Errorf("target_agent_id: got %v", body["target_agent_id"])
	}
	if body["status"] != "pending" {
		t.Errorf("status: got %v", body["status"])
	}
}

func TestTestingDeploy_DefaultEnvironment(t *testing.T) {
	router, st, _ := setupTestingWithAgent(t)

	// No environment in request body — should use cfg.Environment="dev"
	w := doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app",
		"image":"img:v1",
		"host":"app.dev.local",
		"container_internal_port":3000,
		"requires_route":false
	}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	dep, ok := st.GetDeployment("dep-001")
	if !ok {
		t.Fatal("deployment not found")
	}
	if dep.Environment != "dev" {
		t.Errorf("environment: got %q, want dev", dep.Environment)
	}
}

func TestTestingDeploy_AutoContainerName(t *testing.T) {
	router, st, _ := setupTestingWithAgent(t)

	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"billing-api",
		"image":"img:v1",
		"host":"billing-api.dev.local",
		"container_internal_port":3000,
		"requires_route":false
	}`)

	dep, _ := st.GetDeployment("dep-001")
	expected := "billing-api-dev-dep-001"
	if dep.ContainerName != expected {
		t.Errorf("container_name: got %q, want %q", dep.ContainerName, expected)
	}
}

func TestTestingDeploy_MandatoryLabels(t *testing.T) {
	router, st, _ := setupTestingWithAgent(t)

	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"billing-api",
		"image":"img:v1",
		"host":"billing-api.dev.local",
		"container_internal_port":3000,
		"requires_route":false
	}`)

	_, ok := st.GetDeployment("dep-001")
	if !ok {
		t.Fatal("deployment not found")
	}
	cmd, _ := st.GetCommand("cmd-001")
	var p store.DeployApplicationPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for _, k := range []string{"devex.managed", "devex.application", "devex.environment", "devex.deployment_id", "devex.command_id"} {
		if p.Labels[k] == "" {
			t.Errorf("mandatory label %q is missing", k)
		}
	}
}

func TestTestingDeploy_AgentNotFound(t *testing.T) {
	cfg := testingCfg()
	st := store.New()
	router := NewRouter(cfg, st)
	// No agents registered

	w := doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestTestingDeploy_MissingApplication(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"image":"img:v1","host":"app.dev.local","container_internal_port":3000
	}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ---- POST /testing/commands/stop ----

func TestTestingStop_Success(t *testing.T) {
	router, st, agentID := setupTestingWithAgent(t)

	// Deploy first
	dep, cmd := st.InitDeploy(store.InitDeployInput{
		Application: "app", Environment: "dev", Image: "img:v1",
		Host: "app.dev.local", TargetAgentID: agentID,
		ContainerInternalPort: 3000, RequiresRoute: false, CommandTimeoutSeconds: 600,
	})
	// Advance to healthy
	st.ClaimCommand(cmd.ID, agentID)
	st.StartCommand(cmd.ID, agentID)
	result, _ := json.Marshal(map[string]any{"requires_route": false})
	st.ReportCommand(cmd.ID, store.ReportCommandInput{AgentID: agentID, Status: "succeeded", Result: result})

	w := doRequest(router, http.MethodPost, "/testing/commands/stop", fmt.Sprintf(`{
		"deployment_id":"%s"
	}`, dep.ID))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["command_id"] == nil {
		t.Error("command_id must be present")
	}
	if body["target_agent_id"] != agentID {
		t.Errorf("target_agent_id: got %v", body["target_agent_id"])
	}
}

func TestTestingStop_DeploymentNotFound(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)
	w := doRequest(router, http.MethodPost, "/testing/commands/stop", `{"deployment_id":"dep-999"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---- POST /testing/commands/remove ----

func TestTestingRemove_Success(t *testing.T) {
	router, st, agentID := setupTestingWithAgent(t)

	dep, cmd := st.InitDeploy(store.InitDeployInput{
		Application: "app", Environment: "dev", Image: "img:v1",
		Host: "app.dev.local", TargetAgentID: agentID,
		ContainerInternalPort: 3000, RequiresRoute: false, CommandTimeoutSeconds: 600,
	})
	st.ClaimCommand(cmd.ID, agentID)
	st.StartCommand(cmd.ID, agentID)
	result, _ := json.Marshal(map[string]any{"requires_route": false})
	st.ReportCommand(cmd.ID, store.ReportCommandInput{AgentID: agentID, Status: "succeeded", Result: result})

	w := doRequest(router, http.MethodPost, "/testing/commands/remove", fmt.Sprintf(`{
		"deployment_id":"%s"
	}`, dep.ID))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["command_id"] == nil {
		t.Error("command_id must be present")
	}
}

// ---- POST /testing/commands/cleanup-draining ----

func TestTestingCleanupDraining_Success(t *testing.T) {
	router, _, agentID := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodPost, "/testing/commands/cleanup-draining", `{"target_agent_role":"api"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	if body["target_agent_id"] != agentID {
		t.Errorf("target_agent_id: got %v", body["target_agent_id"])
	}
	if body["command_id"] == nil {
		t.Error("command_id must be present")
	}
}

// ---- GET /testing/agents ----

func TestTestingListAgents_Empty(t *testing.T) {
	cfg := testingCfg()
	st := store.New()
	router := NewRouter(cfg, st)

	w := doRequest(router, http.MethodGet, "/testing/agents", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var agents []any
	json.NewDecoder(w.Body).Decode(&agents)
	if len(agents) != 0 {
		t.Errorf("expected empty array, got %d", len(agents))
	}
}

func TestTestingListAgents_WithAgents(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodGet, "/testing/agents", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var agents []any
	json.NewDecoder(w.Body).Decode(&agents)
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

// ---- GET /testing/commands ----

func TestTestingListCommands(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	w := doRequest(router, http.MethodGet, "/testing/commands", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var cmds []any
	json.NewDecoder(w.Body).Decode(&cmds)
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}
}

// ---- GET /testing/deployments ----

func TestTestingListDeployments(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	w := doRequest(router, http.MethodGet, "/testing/deployments", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var deps []any
	json.NewDecoder(w.Body).Decode(&deps)
	if len(deps) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(deps))
	}
}

// ---- GET /testing/reports ----

func TestTestingListReports_EmptyArray(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodGet, "/testing/reports", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var reports []any
	json.NewDecoder(w.Body).Decode(&reports)
	if len(reports) != 0 {
		t.Errorf("expected empty array, got %d", len(reports))
	}
}

// ---- GET /testing/desired-state ----

func TestTestingDesiredState_WithEnvironment_Empty(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodGet, "/testing/desired-state?environment=dev", "")
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
}

func TestTestingDesiredState_AllEnvironments(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	// Deploy to create a desired state entry
	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	w := doRequest(router, http.MethodGet, "/testing/desired-state", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["desired_states"] == nil {
		t.Error("expected desired_states key")
	}
}

// ---- GET /testing/debug ----

func TestTestingDebug(t *testing.T) {
	router, _, _ := setupTestingWithAgent(t)

	w := doRequest(router, http.MethodGet, "/testing/debug", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["agents"] == nil {
		t.Error("debug must include agents")
	}
}

// ---- POST /testing/reset ----

func TestTestingReset_ClearsEverything(t *testing.T) {
	router, st, _ := setupTestingWithAgent(t)

	// Trigger a deploy so we have deployments, commands, etc.
	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	w := doRequest(router, http.MethodPost, "/testing/reset", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	body := decodeBody(t, w)
	if body["status"] != "reset" {
		t.Errorf("status: got %v", body["status"])
	}

	if len(st.ListAgents()) != 0 {
		t.Error("agents must be cleared after reset")
	}
	if len(st.ListDeployments()) != 0 {
		t.Error("deployments must be cleared after reset")
	}
}

func TestTestingReset_CountersRestart(t *testing.T) {
	router, st, _ := setupTestingWithAgent(t)

	// First deploy → dep-001, cmd-001
	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"application":"app","image":"img:v1","host":"app.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	// Reset
	doRequest(router, http.MethodPost, "/testing/reset", "")

	// Register a new agent after reset
	st.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "inst-rt-2",
	})

	// Deploy again — IDs must restart from 001
	doRequest(router, http.MethodPost, "/testing/commands/deploy", `{
		"target_agent_role":"api",
		"application":"app2","image":"img:v2","host":"app2.dev.local",
		"container_internal_port":3000,"requires_route":false
	}`)

	dep, ok := st.GetDeployment("dep-001")
	if !ok {
		t.Fatal("after reset, first new deployment must be dep-001")
	}
	if dep.Application != "app2" {
		t.Errorf("application: got %q", dep.Application)
	}
}
