package store

import (
	"encoding/json"
	"testing"
)

func TestReset_ClearsEverything(t *testing.T) {
	s := newStore()

	// Populate store
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	s.RegisterAgent(RegisterInput{Mode: "gateway", Environment: "dev", Role: "gateway", InstanceID: "i2"})
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")
	result, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4100})
	s.ReportCommand(cmd.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "succeeded", Result: result})
	s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID: "agent-dev-gateway-001", DesiredStateVersion: 1,
		Environment: "dev", Type: "gateway_routes", Status: "applied",
	})
	_ = dep

	s.Reset()

	if len(s.ListAgents()) != 0 {
		t.Error("agents not cleared")
	}
	if len(s.ListDeployments()) != 0 {
		t.Error("deployments not cleared")
	}
	if len(s.ListCommandReports()) != 0 {
		t.Error("command reports not cleared")
	}
	if len(s.ListDesiredStateReports()) != 0 {
		t.Error("desired state reports not cleared")
	}
	if len(s.GetAllDesiredStates()) != 0 {
		t.Error("desired states not cleared")
	}
}

func TestReset_ResetsCounters(t *testing.T) {
	s := newStore()
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	s.InitDeploy(makeDeployInput("app", "dev", "a1"))

	s.Reset()

	// After reset, new objects must start from 001 again
	a := s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	if a.ID != "agent-dev-api-001" {
		t.Errorf("agent ID after reset: got %q, want agent-dev-api-001", a.ID)
	}

	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", a.ID))
	if cmd.ID != "cmd-001" {
		t.Errorf("cmd ID after reset: got %q, want cmd-001", cmd.ID)
	}
}

func TestReset_AgentIndexCleared(t *testing.T) {
	s := newStore()
	a1 := s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	_ = a1

	s.Reset()

	// Re-registering the same instance_id should create a fresh agent with 001
	a2 := s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	if a2.ID != "agent-dev-api-001" {
		t.Errorf("expected agent-dev-api-001 after reset, got %q", a2.ID)
	}
}
