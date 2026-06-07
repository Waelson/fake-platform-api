package store

import (
	"encoding/json"
	"errors"
	"testing"
)

// helper: deploy + succeed a command to put a route in desired state
func deployAndSucceed(t *testing.T, s *Store, app, env, host, agentID string, port int) (*Deployment, *Command) {
	t.Helper()
	in := makeDeployInput(app, env, agentID)
	in.Host = host
	dep, cmd := s.InitDeploy(in)
	claimStart(t, s, cmd.ID, agentID)
	result, _ := json.Marshal(map[string]any{
		"runtime_private_ip": "10.0.0.1",
		"host_port":          port,
	})
	s.ReportCommand(cmd.ID, ReportCommandInput{
		AgentID: agentID, Status: "succeeded", Result: result,
	})
	return dep, cmd
}

// ---- GetDesiredState ----

func TestGetDesiredState_NonExistent(t *testing.T) {
	s := newStore()
	ds := s.GetDesiredState("prod")

	if ds.Version != 0 {
		t.Errorf("version: got %d, want 0", ds.Version)
	}
	if ds.Environment != "prod" {
		t.Errorf("env: got %q", ds.Environment)
	}
	if ds.Type != "gateway_routes" {
		t.Errorf("type: got %q", ds.Type)
	}
	if ds.Routes == nil || len(ds.Routes) != 0 {
		t.Errorf("routes: expected empty slice, got %v", ds.Routes)
	}
}

func TestGetDesiredState_AfterDeploy(t *testing.T) {
	s := newStore()
	deployAndSucceed(t, s, "billing-api", "dev", "billing.dev.local", "agent-dev-api-001", 4100)

	ds := s.GetDesiredState("dev")
	if ds.Version != 1 {
		t.Errorf("version: got %d, want 1", ds.Version)
	}
	if len(ds.Routes) != 1 {
		t.Fatalf("routes: got %d", len(ds.Routes))
	}
	if ds.Routes[0].Host != "billing.dev.local" {
		t.Errorf("host: got %q", ds.Routes[0].Host)
	}
	if ds.Routes[0].ID != "route-001" {
		t.Errorf("route ID: got %q", ds.Routes[0].ID)
	}
}

func TestGetAllDesiredStates_MultipleEnvironments(t *testing.T) {
	s := newStore()
	deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)
	// Register stage agent and deploy there
	s.RegisterAgent(RegisterInput{Mode: "runtime", Environment: "stage", Role: "api", InstanceID: "stage-1"})
	deployAndSucceed(t, s, "app", "stage", "app.stage.local", "agent-stage-api-001", 4200)

	all := s.GetAllDesiredStates()
	if len(all) != 2 {
		t.Errorf("expected 2 environments, got %d", len(all))
	}
	if _, ok := all["dev"]; !ok {
		t.Error("missing dev")
	}
	if _, ok := all["stage"]; !ok {
		t.Error("missing stage")
	}
}

func TestGetDesiredState_ReturnsCopy(t *testing.T) {
	s := newStore()
	deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)

	ds1 := s.GetDesiredState("dev")
	ds1.Routes[0].Host = "mutated"

	ds2 := s.GetDesiredState("dev")
	if ds2.Routes[0].Host == "mutated" {
		t.Error("GetDesiredState should return a copy, not a mutable reference")
	}
}

// ---- ApplyDesiredStateReport ----

func TestApplyDesiredStateReport_CurrentVersion_Applied(t *testing.T) {
	s := newStore()
	dep, _ := deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)
	// dep is now route_pending, version == 1

	report, err := s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID:             "agent-dev-gateway-001",
		DesiredStateVersion: 1,
		Environment:         "dev",
		Type:                "gateway_routes",
		Status:              "applied",
		RoutesTotal:         1,
		ValidatedRoutes:     1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Stale {
		t.Error("should not be stale")
	}
	if report.CurrentDesiredStateVersion != 1 {
		t.Errorf("current version: got %d", report.CurrentDesiredStateVersion)
	}

	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "route_active" {
		t.Errorf("dep status: got %q, want route_active", d.Status)
	}
}

func TestApplyDesiredStateReport_CurrentVersion_Failed(t *testing.T) {
	s := newStore()
	dep, _ := deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)

	s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID:             "agent-dev-gateway-001",
		DesiredStateVersion: 1,
		Environment:         "dev",
		Type:                "gateway_routes",
		Status:              "failed",
	})

	d, _ := s.GetDeployment(dep.ID)
	if d.Status != "route_failed" {
		t.Errorf("dep status: got %q, want route_failed", d.Status)
	}
}

func TestApplyDesiredStateReport_StaleVersion(t *testing.T) {
	s := newStore()
	dep, _ := deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)
	// version is now 1; deploy again to bump to 2
	deployAndSucceed(t, s, "app2", "dev", "app2.dev.local", "agent-dev-api-001", 4101)
	// version is now 2

	report, err := s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID:             "agent-dev-gateway-001",
		DesiredStateVersion: 1, // stale
		Environment:         "dev",
		Status:              "applied",
	})
	if err != nil {
		t.Fatalf("stale report should not return error: %v", err)
	}
	if !report.Stale {
		t.Error("report should be marked stale")
	}
	if report.CurrentDesiredStateVersion != 2 {
		t.Errorf("current version: got %d, want 2", report.CurrentDesiredStateVersion)
	}

	// deployment must NOT have changed
	d, _ := s.GetDeployment(dep.ID)
	if d.Status == "route_active" {
		t.Error("stale report must not transition deployments")
	}
}

func TestApplyDesiredStateReport_FutureVersion(t *testing.T) {
	s := newStore()
	deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)
	// current version = 1

	_, err := s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID:             "agent-dev-gateway-001",
		DesiredStateVersion: 99, // future
		Environment:         "dev",
		Status:              "applied",
	})
	if !errors.Is(err, ErrFutureVersion) {
		t.Errorf("expected ErrFutureVersion, got %v", err)
	}
}

func TestApplyDesiredStateReport_StoredInList(t *testing.T) {
	s := newStore()
	deployAndSucceed(t, s, "app", "dev", "app.dev.local", "agent-dev-api-001", 4100)

	s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID: "gw-1", DesiredStateVersion: 1, Environment: "dev",
		Type: "gateway_routes", Status: "applied",
	})
	// stale
	s.ApplyDesiredStateReport(DesiredStateReportInput{
		AgentID: "gw-1", DesiredStateVersion: 0, Environment: "dev",
		Type: "gateway_routes", Status: "applied",
	})

	reports := s.ListDesiredStateReports()
	if len(reports) != 2 {
		t.Errorf("expected 2 reports, got %d", len(reports))
	}
}
