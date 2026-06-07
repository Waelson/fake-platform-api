package store

import (
	"encoding/json"
	"testing"
)

func newStore() *Store { return New("host.docker.internal") }

// ---- RegisterAgent ----

func TestRegisterAgent_New(t *testing.T) {
	s := newStore()
	in := RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		Hostname: "host-1", InstanceID: "inst-1",
		PrivateIP: "10.0.0.1", Version: "0.1.0",
		Capabilities: map[string]any{"max_containers": 10},
	}
	a := s.RegisterAgent(in)

	if a.ID != "agent-dev-api-001" {
		t.Errorf("ID: got %q, want agent-dev-api-001", a.ID)
	}
	if a.Status != "online" {
		t.Errorf("Status: got %q", a.Status)
	}
	if a.Environment != "dev" || a.Role != "api" {
		t.Errorf("env/role mismatch")
	}
}

func TestRegisterAgent_CounterPerEnvRole(t *testing.T) {
	s := newStore()
	base := RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"}
	s.RegisterAgent(base)

	base.InstanceID = "i2"
	a2 := s.RegisterAgent(base)
	if a2.ID != "agent-dev-api-002" {
		t.Errorf("second agent ID: got %q, want agent-dev-api-002", a2.ID)
	}

	// different role — counter restarts
	base.InstanceID = "i3"
	base.Role = "worker"
	a3 := s.RegisterAgent(base)
	if a3.ID != "agent-dev-worker-001" {
		t.Errorf("worker ID: got %q, want agent-dev-worker-001", a3.ID)
	}
}

func TestRegisterAgent_Upsert(t *testing.T) {
	s := newStore()
	in := RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		Hostname: "old-host", InstanceID: "inst-1",
		PrivateIP: "10.0.0.1", Version: "0.1.0",
	}
	first := s.RegisterAgent(in)

	in.Hostname = "new-host"
	in.Version = "0.2.0"
	second := s.RegisterAgent(in)

	if first.ID != second.ID {
		t.Error("re-register must preserve agent ID")
	}
	if second.Hostname != "new-host" {
		t.Errorf("Hostname not updated: %q", second.Hostname)
	}
	if second.Version != "0.2.0" {
		t.Errorf("Version not updated: %q", second.Version)
	}
	if s.Counters.AgentByEnvironmentRole["dev-api"] != 1 {
		t.Error("counter must not increment on re-register")
	}
}

func TestRegisterAgent_StoreLength(t *testing.T) {
	s := newStore()
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "a"})
	s.RegisterAgent(RegisterInput{Mode: "gateway", Environment: "dev", Role: "gateway", InstanceID: "b"})

	if len(s.ListAgents()) != 2 {
		t.Errorf("expected 2 agents, got %d", len(s.ListAgents()))
	}
}

// ---- Heartbeat ----

func TestHeartbeat_UpdatesFields(t *testing.T) {
	s := newStore()
	a := s.RegisterAgent(RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		InstanceID: "inst-1", PrivateIP: "10.0.0.1", Version: "0.1.0",
	})
	agentID := a.ID

	payload := json.RawMessage(`{"status":"degraded","version":"0.2.0","private_ip":"10.0.0.2","extra":"field"}`)
	got, err := s.Heartbeat(agentID, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Status != "degraded" {
		t.Errorf("Status: got %q, want degraded", got.Status)
	}
	if got.Version != "0.2.0" {
		t.Errorf("Version: got %q", got.Version)
	}
	if got.PrivateIP != "10.0.0.2" {
		t.Errorf("PrivateIP: got %q", got.PrivateIP)
	}
	if string(got.LastHeartbeat) != string(payload) {
		t.Error("LastHeartbeat not stored verbatim")
	}
}

func TestHeartbeat_PartialPayload(t *testing.T) {
	s := newStore()
	a := s.RegisterAgent(RegisterInput{
		Mode: "gateway", Environment: "dev", Role: "gateway",
		InstanceID: "gw-1", Version: "1.0.0", PrivateIP: "10.0.0.5",
	})

	// Gateway heartbeat doesn't have private_ip — existing value must be preserved
	payload := json.RawMessage(`{"status":"online","caddy_status":"healthy"}`)
	got, _ := s.Heartbeat(a.ID, payload)

	if got.PrivateIP != "10.0.0.5" {
		t.Errorf("PrivateIP should be preserved: got %q", got.PrivateIP)
	}
	if got.Status != "online" {
		t.Errorf("Status: got %q", got.Status)
	}
}

func TestHeartbeat_NotFound(t *testing.T) {
	s := newStore()
	_, err := s.Heartbeat("agent-dev-api-999", json.RawMessage(`{}`))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- GetAgent / FindAgentByEnvironmentRole ----

func TestGetAgent_Exists(t *testing.T) {
	s := newStore()
	a := s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	got, ok := s.GetAgent(a.ID)
	if !ok || got.ID != a.ID {
		t.Error("GetAgent failed")
	}
}

func TestGetAgent_Missing(t *testing.T) {
	s := newStore()
	_, ok := s.GetAgent("agent-dev-api-999")
	if ok {
		t.Error("expected not found")
	}
}

func TestFindAgentByEnvironmentRole(t *testing.T) {
	s := newStore()
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "stage", Role: "api", InstanceID: "i2"})

	a, ok := s.FindAgentByEnvironmentRole("dev", "api")
	if !ok {
		t.Fatal("expected to find agent")
	}
	if a.Environment != "dev" {
		t.Errorf("env: got %q", a.Environment)
	}

	_, ok = s.FindAgentByEnvironmentRole("prod", "api")
	if ok {
		t.Error("should not find agent for prod")
	}
}
