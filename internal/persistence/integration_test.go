package persistence

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/waelson/fake-platform-api/internal/store"
)

// TestRestart_RestoresFullLifecycleState builds up a realistic store (agent,
// deployment, command, desired-state route) by driving the same lifecycle the
// real handlers use, persists it, and then "restarts" by loading the snapshot
// into a brand new Store — mirroring what main.go does on boot. It confirms
// agents, deployments, commands and desired-state routes all remain accessible
// after the simulated restart.
func TestRestart_RestoresFullLifecycleState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	before := store.New()

	agent := before.RegisterAgent(store.RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		Hostname: "host-1", InstanceID: "inst-1",
		PrivateIP: "10.0.0.5", Version: "0.1.0",
	})

	dep, cmd := before.InitDeploy(store.InitDeployInput{
		Application:           "billing-api",
		Environment:           "dev",
		Image:                 "fake-api:v1",
		Host:                  "billing-api.dev.local",
		TargetAgentID:         agent.ID,
		ContainerInternalPort: 3000,
		HealthCheckPath:       "/health",
		RequiresRoute:         true,
		CommandTimeoutSeconds: 60,
	})

	if _, err := before.ClaimCommand(cmd.ID, agent.ID); err != nil {
		t.Fatalf("ClaimCommand: %v", err)
	}
	if _, err := before.StartCommand(cmd.ID, agent.ID); err != nil {
		t.Fatalf("StartCommand: %v", err)
	}

	result, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.5", "host_port": 30001})
	if _, err := before.ReportCommand(cmd.ID, store.ReportCommandInput{
		AgentID:      agent.ID,
		DeploymentID: dep.ID,
		Status:       "succeeded",
		Result:       result,
	}); err != nil {
		t.Fatalf("ReportCommand: %v", err)
	}

	if err := Save(path, before.Snapshot()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// --- simulate restart: brand new Store, load snapshot from disk ---
	snap, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot to be loaded, got nil")
	}

	after := store.New()
	after.Restore(*snap)

	// Agent
	gotAgent, ok := after.GetAgent(agent.ID)
	if !ok {
		t.Fatalf("agent %s not found after restart", agent.ID)
	}
	if gotAgent.Hostname != "host-1" || gotAgent.PrivateIP != "10.0.0.5" {
		t.Errorf("agent fields not restored: %+v", gotAgent)
	}

	// Deployment
	gotDep, ok := after.GetDeployment(dep.ID)
	if !ok {
		t.Fatalf("deployment %s not found after restart", dep.ID)
	}
	if gotDep.Status != "route_pending" {
		t.Errorf("deployment status: got %q, want route_pending", gotDep.Status)
	}
	if gotDep.RouteID == "" {
		t.Errorf("expected deployment to have a route id after restart")
	}

	// Command
	gotCmd, ok := after.GetCommand(cmd.ID)
	if !ok {
		t.Fatalf("command %s not found after restart", cmd.ID)
	}
	if gotCmd.Status != "succeeded" {
		t.Errorf("command status: got %q, want succeeded", gotCmd.Status)
	}

	// Desired state / route
	ds := after.GetDesiredState("dev")
	if len(ds.Routes) != 1 {
		t.Fatalf("expected 1 route in desired state, got %d", len(ds.Routes))
	}
	if ds.Routes[0].Host != "billing-api.dev.local" {
		t.Errorf("route host: got %q", ds.Routes[0].Host)
	}
	if ds.Routes[0].DeploymentID != dep.ID {
		t.Errorf("route deployment id: got %q, want %q", ds.Routes[0].DeploymentID, dep.ID)
	}

	// Counters must be preserved so new IDs don't collide with restored ones.
	if after.Counters.Deployment != before.Counters.Deployment {
		t.Errorf("deployment counter: got %d, want %d", after.Counters.Deployment, before.Counters.Deployment)
	}
	if after.Counters.Command != before.Counters.Command {
		t.Errorf("command counter: got %d, want %d", after.Counters.Command, before.Counters.Command)
	}
}
