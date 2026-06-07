package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/waelson/fake-platform-api/internal/store"
)

func TestLoad_MissingFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	snap, err := Load(filepath.Join(dir, "does-not-exist.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap != nil {
		t.Fatalf("expected nil snapshot, got %+v", snap)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	want := store.Snapshot{
		SchemaVersion: store.SchemaVersion,
		Agents: map[string]*store.Agent{
			"agent-dev-api-001": {ID: "agent-dev-api-001", Mode: "runtime", Environment: "dev", Role: "api"},
		},
		AgentIndex:          map[string]string{"inst-1:runtime:dev:api": "agent-dev-api-001"},
		Commands:            map[string]*store.Command{},
		Deployments:         map[string]*store.Deployment{},
		CommandReports:      []store.CommandReport{},
		DesiredStates:       map[string]*store.DesiredState{},
		DesiredStateReports: []store.DesiredStateReport{},
		Counters: store.Counters{
			AgentByEnvironmentRole:           map[string]int{"dev:api": 1},
			DesiredStateVersionByEnvironment: map[string]int{},
		},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("expected snapshot, got nil")
	}

	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("round-trip mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestSave_AtomicWriteLeavesNoTmpFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	snap := store.Snapshot{SchemaVersion: store.SchemaVersion}
	if err := Save(path, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("expected no .tmp file to remain, stat err = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected final file to exist: %v", err)
	}
}
