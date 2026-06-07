package store

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSnapshotRestore_RoundTrip(t *testing.T) {
	s := New()

	agent := s.RegisterAgent(RegisterInput{
		Mode: "runtime", Environment: "dev", Role: "api",
		Hostname: "host-1", InstanceID: "inst-1",
		PrivateIP: "10.0.0.1", Version: "0.1.0",
		Capabilities: map[string]any{"max_containers": 10},
	})

	cmd := s.CreateCommand(CreateCommandInput{
		Type:           "deploy_application",
		TargetAgentID:  agent.ID,
		TimeoutSeconds: 30,
		Payload:        json.RawMessage(`{"application":"foo"}`),
	})
	_ = cmd

	snap := s.Snapshot()
	if snap.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion: got %d, want %d", snap.SchemaVersion, SchemaVersion)
	}

	restored := New()
	restored.Restore(snap)

	got := restored.Snapshot()
	want := s.Snapshot()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("restored snapshot does not match original\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestSnapshotRestore_EmptyStore(t *testing.T) {
	s := New()
	snap := s.Snapshot()

	restored := New()
	restored.Restore(snap)

	if len(restored.Agents) != 0 || len(restored.Commands) != 0 || len(restored.Deployments) != 0 {
		t.Errorf("expected restored store to be empty, got %+v", restored.Debug())
	}
	if restored.Counters.AgentByEnvironmentRole == nil || restored.Counters.DesiredStateVersionByEnvironment == nil {
		t.Errorf("expected counter maps to be initialized, got %+v", restored.Counters)
	}
}
