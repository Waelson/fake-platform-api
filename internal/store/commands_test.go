package store

import (
	"encoding/json"
	"errors"
	"testing"
)

// claimStart is a helper that runs the full pending→claimed→running lifecycle.
func claimStart(t *testing.T, s *Store, cmdID, agentID string) {
	t.Helper()
	if _, err := s.ClaimCommand(cmdID, agentID); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := s.StartCommand(cmdID, agentID); err != nil {
		t.Fatalf("start: %v", err)
	}
}

// ---- ClaimCommand ----

func TestClaimCommand_Pending_TargetAgent(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	_ = dep

	got, err := s.ClaimCommand(cmd.ID, "agent-dev-api-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != "claimed" {
		t.Errorf("status: got %q, want claimed", got.Status)
	}
	if got.ClaimedBy != "agent-dev-api-001" {
		t.Errorf("claimed_by: got %q", got.ClaimedBy)
	}
	if got.ClaimedAt == nil {
		t.Error("claimed_at must be set")
	}
}

func TestClaimCommand_Pending_NonTargetAgent_Conflict(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))

	_, err := s.ClaimCommand(cmd.ID, "agent-dev-api-002")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestClaimCommand_Claimed_SameAgent_Idempotent(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	s.ClaimCommand(cmd.ID, "agent-dev-api-001")

	// second claim by same agent → 200 idempotent
	got, err := s.ClaimCommand(cmd.ID, "agent-dev-api-001")
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if got.Status != "claimed" {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestClaimCommand_Claimed_DifferentAgent_Conflict(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	s.ClaimCommand(cmd.ID, "agent-dev-api-001")

	_, err := s.ClaimCommand(cmd.ID, "agent-dev-api-002")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestClaimCommand_Running_Conflict(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	_, err := s.ClaimCommand(cmd.ID, "agent-dev-api-001")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestClaimCommand_NotFound(t *testing.T) {
	s := newStore()
	_, err := s.ClaimCommand("cmd-999", "agent-dev-api-001")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- StartCommand ----

func TestStartCommand_FromClaimed_DeployApplication(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	s.ClaimCommand(cmd.ID, "agent-dev-api-001")

	got, err := s.StartCommand(cmd.ID, "agent-dev-api-001")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("cmd status: got %q", got.Status)
	}
	if got.StartedAt == nil {
		t.Error("started_at must be set")
	}

	// deployment must advance to deploying
	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "deploying" {
		t.Errorf("dep status: got %q, want deploying", d.Status)
	}
}

func TestStartCommand_Idempotent_SameAgent(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	got, err := s.StartCommand(cmd.ID, "agent-dev-api-001")
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestStartCommand_DifferentAgent_Conflict(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	_, err := s.StartCommand(cmd.ID, "agent-dev-api-002")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestStartCommand_Pending_Conflict(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))

	_, err := s.StartCommand(cmd.ID, "agent-dev-api-001")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict for pending, got %v", err)
	}
}

// ---- ReportCommand ----

func TestReportCommand_DeploySucceeded_RequiresRoute(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("billing-api", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	result, _ := json.Marshal(map[string]any{
		"runtime_private_ip": "10.0.0.1",
		"host_port":          4100,
		"requires_route":     true,
	})
	report, err := s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "succeeded",
		Result:  result,
	})
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report.Status != "succeeded" {
		t.Errorf("report status: %q", report.Status)
	}

	c, _ := s.GetCommand(cmd.ID)
	if c.Status != "succeeded" {
		t.Errorf("cmd status: got %q, want succeeded", c.Status)
	}
	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "route_pending" {
		t.Errorf("dep status: got %q, want route_pending", d.Status)
	}
	if d.RouteID == "" {
		t.Error("RouteID must be set after route creation")
	}
	if d.HostPort != 4100 {
		t.Errorf("HostPort: got %d", d.HostPort)
	}

	ds := s.GetDesiredState("dev")
	if ds.Version != 1 {
		t.Errorf("desired state version: got %d, want 1", ds.Version)
	}
	if len(ds.Routes) != 1 {
		t.Errorf("routes: got %d, want 1", len(ds.Routes))
	}
	if ds.Routes[0].Upstream != "10.0.0.1:4100" {
		t.Errorf("upstream: got %q", ds.Routes[0].Upstream)
	}
}

func TestReportCommand_DeploySucceeded_NoRoute(t *testing.T) {
	s := newStore()
	in := makeDeployInput("worker", "dev", "agent-dev-api-001")
	in.RequiresRoute = false
	in.Host = ""
	dep, cmd := s.InitDeploy(in)
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "succeeded",
		Result:  json.RawMessage(`{"runtime_private_ip":"10.0.0.1","host_port":0}`),
	})

	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "healthy" {
		t.Errorf("dep status: got %q, want healthy", d.Status)
	}
	ds := s.GetDesiredState("dev")
	if ds.Version != 0 {
		t.Errorf("desired state version must stay 0 for no-route deploy: got %d", ds.Version)
	}
}

func TestReportCommand_DeployFailed(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "failed",
		Error:   &ReportError{Code: "HEALTH_CHECK_FAILED", Message: "unhealthy"},
	})

	c, _ := s.GetCommand(cmd.ID)
	d, _ := s.GetDeployment(dep.ID)
	if c.Status != "failed" {
		t.Errorf("cmd status: %q", c.Status)
	}
	if d.Status != "failed" {
		t.Errorf("dep status: %q", d.Status)
	}
}

func TestReportCommand_StopSucceeded(t *testing.T) {
	s := newStore()
	// Deploy first to get a healthy deployment
	dep, deployCMD := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, deployCMD.ID, "agent-dev-api-001")
	result, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4100})
	s.ReportCommand(deployCMD.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "succeeded", Result: result})

	// Create STOP command
	stopCMD := s.CreateCommand(CreateCommandInput{
		Type: "STOP_APPLICATION", DeploymentID: dep.ID,
		TargetAgentID: "agent-dev-api-001", TimeoutSeconds: 60,
		Payload: json.RawMessage(`{}`),
	})
	claimStart(t, s, stopCMD.ID, "agent-dev-api-001")
	s.ReportCommand(stopCMD.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "succeeded",
		Result:  json.RawMessage(`{"stopped":true}`),
	})

	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "removed" {
		t.Errorf("dep status: got %q, want removed", d.Status)
	}
}

func TestReportCommand_RemoveSucceeded_WithRoute(t *testing.T) {
	s := newStore()
	dep, deployCMD := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, deployCMD.ID, "agent-dev-api-001")
	result, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4100})
	s.ReportCommand(deployCMD.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "succeeded", Result: result})

	versionBefore := s.GetDesiredState("dev").Version // should be 1

	removeCMD := s.CreateCommand(CreateCommandInput{
		Type: "REMOVE_DEPLOYMENT", DeploymentID: dep.ID,
		TargetAgentID: "agent-dev-api-001", TimeoutSeconds: 120,
		Payload: json.RawMessage(`{}`),
	})
	claimStart(t, s, removeCMD.ID, "agent-dev-api-001")
	s.ReportCommand(removeCMD.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "succeeded",
		Result:  json.RawMessage(`{"removed":true}`),
	})

	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "removed" {
		t.Errorf("dep status: got %q", d.Status)
	}
	if d.RouteID != "" {
		t.Error("RouteID must be cleared after remove")
	}

	ds := s.GetDesiredState("dev")
	if ds.Version != versionBefore+1 {
		t.Errorf("desired state version: got %d, want %d", ds.Version, versionBefore+1)
	}
	if len(ds.Routes) != 0 {
		t.Errorf("routes: got %d, want 0 after remove", len(ds.Routes))
	}
}

func TestReportCommand_Cleanup_OnlyStoresReport(t *testing.T) {
	s := newStore()
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "dev", Role: "api", InstanceID: "i1"})

	cmd := s.CreateCommand(CreateCommandInput{
		Type: "CLEANUP_DRAINING", TargetAgentID: "agent-dev-api-001",
		TimeoutSeconds: 300, Payload: json.RawMessage(`{}`),
	})
	claimStart(t, s, cmd.ID, "agent-dev-api-001")
	s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID: "agent-dev-api-001",
		Status:  "succeeded",
		Result:  json.RawMessage(`{"cleaned":3}`),
	})

	reports := s.ListCommandReports()
	if len(reports) != 1 {
		t.Errorf("reports: got %d, want 1", len(reports))
	}
	if reports[0].Status != "succeeded" {
		t.Errorf("report status: %q", reports[0].Status)
	}
}

func TestReportCommand_DeploymentIDFallback(t *testing.T) {
	s := newStore()
	dep, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	// Agent does not send deployment_id in report body; store should fall back to cmd.DeploymentID
	report, err := s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID:      "agent-dev-api-001",
		Status:       "failed",
		DeploymentID: "", // intentionally empty
		Error:        &ReportError{Code: "ERR", Message: "fail"},
	})
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report.DeploymentID != dep.ID {
		t.Errorf("effective dep ID: got %q, want %q", report.DeploymentID, dep.ID)
	}
}

func TestReportCommand_UpdateUpsertRouteOnV2(t *testing.T) {
	s := newStore()
	// Deploy v1
	in := makeDeployInput("billing-api", "dev", "agent-dev-api-001")
	in.Host = "billing-api.dev.local"
	dep1, cmd1 := s.InitDeploy(in)
	claimStart(t, s, cmd1.ID, "agent-dev-api-001")
	result1, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4100})
	s.ReportCommand(cmd1.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "succeeded", Result: result1})

	if s.GetDesiredState("dev").Version != 1 {
		t.Fatal("expected version 1 after v1 deploy")
	}

	// Deploy v2 same host
	in.ContainerName = "billing-api-dev-dep-002"
	dep2, cmd2 := s.InitDeploy(in)
	claimStart(t, s, cmd2.ID, "agent-dev-api-001")
	result2, _ := json.Marshal(map[string]any{"runtime_private_ip": "10.0.0.1", "host_port": 4101})
	s.ReportCommand(cmd2.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "succeeded", Result: result2})

	ds := s.GetDesiredState("dev")
	if ds.Version != 2 {
		t.Errorf("version: got %d, want 2 after v2 deploy", ds.Version)
	}
	if len(ds.Routes) != 1 {
		t.Errorf("routes: got %d, want 1 (no duplicate)", len(ds.Routes))
	}
	if ds.Routes[0].Upstream != "10.0.0.1:4101" {
		t.Errorf("upstream not updated: %q", ds.Routes[0].Upstream)
	}
	if ds.Routes[0].DeploymentID != dep2.ID {
		t.Errorf("deploymentID not updated to v2: %q", ds.Routes[0].DeploymentID)
	}
	_ = dep1
}

func TestReportCommand_InvalidStatus(t *testing.T) {
	s := newStore()
	_, cmd := s.InitDeploy(makeDeployInput("app", "dev", "agent-dev-api-001"))
	claimStart(t, s, cmd.ID, "agent-dev-api-001")

	_, err := s.ReportCommand(cmd.ID, ReportCommandInput{AgentID: "agent-dev-api-001", Status: "bogus"})
	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestListPendingByAgent(t *testing.T) {
	s := newStore()
	_, cmd1 := s.InitDeploy(makeDeployInput("app1", "dev", "agent-dev-api-001"))
	_, cmd2 := s.InitDeploy(makeDeployInput("app2", "dev", "agent-dev-api-001"))
	_, cmd3 := s.InitDeploy(makeDeployInput("app3", "dev", "agent-dev-api-002"))

	// claim cmd1 — it leaves pending list
	s.ClaimCommand(cmd1.ID, "agent-dev-api-001")

	pending := s.ListPendingByAgent("agent-dev-api-001")
	if len(pending) != 1 || pending[0].ID != cmd2.ID {
		t.Errorf("pending: got %d items (expected cmd2 only)", len(pending))
	}
	_ = cmd3
}
