package store

import (
	"fmt"
	"time"

	"github.com/waelson/fake-platform-api/internal/ids"
)

// GetDesiredState returns the desired state for an environment.
// Returns an empty state (version 0, routes []) if the environment has no state yet.
func (s *Store) GetDesiredState(environment string) DesiredState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ds, ok := s.DesiredStates[environment]; ok {
		return s.copyDesiredState(ds)
	}
	return DesiredState{
		Version:     0,
		Type:        "gateway_routes",
		Environment: environment,
		Routes:      []Route{},
	}
}

// GetAllDesiredStates returns a snapshot of all desired states keyed by environment.
func (s *Store) GetAllDesiredStates() map[string]DesiredState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]DesiredState, len(s.DesiredStates))
	for k, v := range s.DesiredStates {
		out[k] = s.copyDesiredState(v)
	}
	return out
}

func (s *Store) copyDesiredState(ds *DesiredState) DesiredState {
	routes := make([]Route, len(ds.Routes))
	copy(routes, ds.Routes)
	return DesiredState{
		Version:     ds.Version,
		Type:        ds.Type,
		Environment: ds.Environment,
		Routes:      routes,
	}
}

// ---- Internal helpers (called under s.mu held for write) ----

func (s *Store) getOrCreateDesiredState(environment string) *DesiredState {
	if ds, ok := s.DesiredStates[environment]; ok {
		return ds
	}
	ds := &DesiredState{
		Version:     0,
		Type:        "gateway_routes",
		Environment: environment,
		Routes:      []Route{},
	}
	s.DesiredStates[environment] = ds
	return ds
}

// upsertRoute creates or updates the route for a given environment+host.
// Route uniqueness is by environment + host (decision 6).
func (s *Store) upsertRoute(environment, host, path, upstream, deploymentID, healthCheckPath string) *Route {
	if ds, ok := s.DesiredStates[environment]; ok {
		for i := range ds.Routes {
			if ds.Routes[i].Host == host {
				ds.Routes[i].Upstream = upstream
				ds.Routes[i].DeploymentID = deploymentID
				return &ds.Routes[i]
			}
		}
	}

	s.Counters.Route++
	route := Route{
		ID:              ids.Route(s.Counters.Route),
		Environment:     environment,
		Host:            host,
		Path:            path,
		Upstream:        upstream,
		DeploymentID:    deploymentID,
		HealthCheckPath: healthCheckPath,
	}
	ds := s.getOrCreateDesiredState(environment)
	ds.Routes = append(ds.Routes, route)
	return &ds.Routes[len(ds.Routes)-1]
}

func (s *Store) removeRoute(environment, routeID string) {
	ds, ok := s.DesiredStates[environment]
	if !ok {
		return
	}
	filtered := ds.Routes[:0]
	for _, r := range ds.Routes {
		if r.ID != routeID {
			filtered = append(filtered, r)
		}
	}
	ds.Routes = filtered
}

func (s *Store) incrementDesiredState(environment string) {
	ds := s.getOrCreateDesiredState(environment)
	ds.Version++
	s.Counters.DesiredStateVersionByEnvironment[environment] = ds.Version
}

// ---- Desired-state report ----

type DesiredStateReportInput struct {
	AgentID             string
	DesiredStateVersion int
	Type                string
	Environment         string
	Status              string // "applied" | "failed"
	RoutesTotal         int
	ValidatedRoutes     int
	FailedRoutes        int
	Error               *ReportError
}

// ApplyDesiredStateReport processes a gateway agent report.
//
//   - version == current  → process transitions, return report
//   - version < current   → store as stale, return report (HTTP 200)
//   - version > current   → return ErrFutureVersion (HTTP 409)
func (s *Store) ApplyDesiredStateReport(in DesiredStateReportInput) (DesiredStateReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentVersion := 0
	if ds, ok := s.DesiredStates[in.Environment]; ok {
		currentVersion = ds.Version
	}

	report := DesiredStateReport{
		AgentID:                    in.AgentID,
		DesiredStateVersion:        in.DesiredStateVersion,
		CurrentDesiredStateVersion: currentVersion,
		Environment:                in.Environment,
		Type:                       in.Type,
		Status:                     in.Status,
		RoutesTotal:                in.RoutesTotal,
		ValidatedRoutes:            in.ValidatedRoutes,
		FailedRoutes:               in.FailedRoutes,
		Error:                      in.Error,
		ReceivedAt:                 time.Now().UTC(),
	}

	switch {
	case in.DesiredStateVersion > currentVersion:
		return DesiredStateReport{}, fmt.Errorf("%w: reported %d but current is %d",
			ErrFutureVersion, in.DesiredStateVersion, currentVersion)

	case in.DesiredStateVersion < currentVersion:
		report.Stale = true
		s.DesiredStateReports = append(s.DesiredStateReports, report)
		return report, nil

	default: // version == current
		s.DesiredStateReports = append(s.DesiredStateReports, report)
		now := time.Now().UTC()
		for _, dep := range s.Deployments {
			if dep.Environment != in.Environment || dep.Status != "route_pending" {
				continue
			}
			switch in.Status {
			case "applied":
				dep.Status = "route_active"
				dep.UpdatedAt = now
			case "failed":
				dep.Status = "route_failed"
				dep.UpdatedAt = now
			}
		}
		return report, nil
	}
}

func (s *Store) ListDesiredStateReports() []DesiredStateReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DesiredStateReport, len(s.DesiredStateReports))
	copy(out, s.DesiredStateReports)
	return out
}
