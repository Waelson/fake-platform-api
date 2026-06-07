package store

import (
	"encoding/json"
	"testing"
)

func makeDeployInput(app, env, agentID string) InitDeployInput {
	return InitDeployInput{
		Application:           app,
		Environment:           env,
		Image:                 "fake-api:v1",
		Host:                  app + ".dev.local",
		TargetAgentID:         agentID,
		ContainerInternalPort: 3000,
		HealthCheckPath:       "/health",
		RequiresRoute:         true,
		CommandTimeoutSeconds: 600,
	}
}

// ---- InitDeploy ----

func TestInitDeploy_IDsAndStatus(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("billing-api", "dev", "agent-dev-api-001"))

	if dep.ID != "dep-001" {
		t.Errorf("dep ID: got %q, want dep-001", dep.ID)
	}
	if cmd.ID != "cmd-001" {
		t.Errorf("cmd ID: got %q, want cmd-001", cmd.ID)
	}
	if dep.Status != "command_created" {
		t.Errorf("dep status: got %q, want command_created", dep.Status)
	}
	if dep.CommandID != cmd.ID {
		t.Errorf("dep.CommandID not linked: got %q", dep.CommandID)
	}
	if cmd.Status != "pending" {
		t.Errorf("cmd status: got %q, want pending", cmd.Status)
	}
	if cmd.DeploymentID != dep.ID {
		t.Errorf("cmd.DeploymentID not linked: got %q", cmd.DeploymentID)
	}
	if cmd.Type != "DEPLOY_APPLICATION" {
		t.Errorf("cmd type: got %q", cmd.Type)
	}
	if cmd.TimeoutSeconds != 600 {
		t.Errorf("cmd timeout: got %d", cmd.TimeoutSeconds)
	}
}

func TestInitDeploy_IncrementCounters(t *testing.T) {
	s := newStore()
	s.InitDeploy(makeDeployInput("app1", "dev", "agent-1"))
	dep2, cmd2 := s.InitDeploy(makeDeployInput("app2", "dev", "agent-1"))

	if dep2.ID != "dep-002" {
		t.Errorf("dep2 ID: got %q, want dep-002", dep2.ID)
	}
	if cmd2.ID != "cmd-002" {
		t.Errorf("cmd2 ID: got %q, want cmd-002", cmd2.ID)
	}
}

func TestInitDeploy_Atomicity(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))

	if _, ok := s.GetDeployment(dep.ID); !ok {
		t.Error("deployment not in store")
	}
	if _, ok := s.GetCommand(cmd.ID); !ok {
		t.Error("command not in store")
	}
}

func TestInitDeploy_AutoContainerName(t *testing.T) {
	s := newStore()
	// No ContainerName provided
	dep, cmd := s.InitDeploy(makeDeployInput("billing-api", "dev", "agent-dev-api-001"))

	expected := "billing-api-dev-dep-001"
	if dep.ContainerName != expected {
		t.Errorf("ContainerName: got %q, want %q", dep.ContainerName, expected)
	}

	// Also verify the payload inside the command carries the same container_name
	var p DeployApplicationPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p.ContainerName != expected {
		t.Errorf("payload ContainerName: got %q, want %q", p.ContainerName, expected)
	}
}

func TestInitDeploy_ExplicitContainerName(t *testing.T) {
	s := newStore()
	in := makeDeployInput("app", "dev", "a1")
	in.ContainerName = "custom-name"
	dep, _ := s.InitDeploy(in)

	if dep.ContainerName != "custom-name" {
		t.Errorf("ContainerName: got %q, want custom-name", dep.ContainerName)
	}
}

func TestInitDeploy_MandatoryLabels(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("billing-api", "dev", "agent-dev-api-001"))

	var p DeployApplicationPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	want := map[string]string{
		"devex.managed":       "true",
		"devex.application":   "billing-api",
		"devex.environment":   "dev",
		"devex.deployment_id": dep.ID,
		"devex.command_id":    cmd.ID,
	}
	for k, v := range want {
		if p.Labels[k] != v {
			t.Errorf("label %q: got %q, want %q", k, p.Labels[k], v)
		}
	}
	if len(p.Labels) != 5 {
		t.Errorf("expected exactly 5 mandatory labels, got %d: %v", len(p.Labels), p.Labels)
	}
}

func TestInitDeploy_CustomLabels_CannotOverrideMandatory(t *testing.T) {
	s := newStore()
	in := makeDeployInput("app", "dev", "a1")
	in.CustomLabels = map[string]string{
		"custom.team":         "platform",
		"devex.managed":       "false", // attempt to override mandatory
		"devex.deployment_id": "injected",
	}
	_, cmd := s.InitDeploy(in)

	var p DeployApplicationPayload
	json.Unmarshal(cmd.Payload, &p)

	if p.Labels["devex.managed"] != "true" {
		t.Errorf("mandatory label must not be overridden: devex.managed = %q", p.Labels["devex.managed"])
	}
	if p.Labels["devex.deployment_id"] != "dep-001" {
		t.Errorf("mandatory label must not be overridden: devex.deployment_id = %q", p.Labels["devex.deployment_id"])
	}
	if p.Labels["custom.team"] != "platform" {
		t.Errorf("custom label missing: custom.team = %q", p.Labels["custom.team"])
	}
}

// ---- CreateCommand (standalone) ----

func TestCreateCommand_StandaloneIDs(t *testing.T) {
	s := newStore()
	s.InitDeploy(makeDeployInput("app", "dev", "a1"))

	cmd := s.CreateCommand(CreateCommandInput{
		Type:           "STOP_APPLICATION",
		DeploymentID:   "dep-001",
		TargetAgentID:  "agent-dev-api-001",
		TimeoutSeconds: 60,
		Payload:        json.RawMessage(`{}`),
	})

	if cmd.ID != "cmd-002" {
		t.Errorf("cmd ID: got %q, want cmd-002", cmd.ID)
	}
	if cmd.Type != "STOP_APPLICATION" {
		t.Errorf("cmd type: got %q", cmd.Type)
	}
}

// ---- GetDeployment / ListDeployments ----

func TestGetDeployment_MissingReturnsNotFound(t *testing.T) {
	s := newStore()
	_, ok := s.GetDeployment("dep-999")
	if ok {
		t.Error("expected not found")
	}
}

func TestListDeployments(t *testing.T) {
	s := newStore()
	s.InitDeploy(makeDeployInput("app1", "dev", "a1"))
	s.InitDeploy(makeDeployInput("app2", "dev", "a1"))

	list := s.ListDeployments()
	if len(list) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(list))
	}
}
