package store

import (
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) GetCommand(commandID string) (*Command, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.Commands[commandID]
	return c, ok
}

func (s *Store) ListPendingByAgent(agentID string) []*Command {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Command, 0)
	for _, c := range s.Commands {
		if c.TargetAgentID == agentID && c.Status == "pending" {
			out = append(out, c)
		}
	}
	return out
}

// ListCommands returns all commands, optionally filtered by status, type, and agent_id.
func (s *Store) ListCommands(status, cmdType, agentID string) []*Command {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Command, 0)
	for _, c := range s.Commands {
		if status != "" && c.Status != status {
			continue
		}
		if cmdType != "" && c.Type != cmdType {
			continue
		}
		if agentID != "" && c.TargetAgentID != agentID {
			continue
		}
		out = append(out, c)
	}
	return out
}

// ClaimCommand transitions a pending command to claimed (idempotent for same agent).
//
// Rules:
//   - pending + target agent  → claimed, nil
//   - claimed + same agent    → idempotent, nil
//   - claimed + other agent   → ErrConflict
//   - any other status        → ErrConflict
func (s *Store) ClaimCommand(commandID, agentID string) (*Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, ok := s.Commands[commandID]
	if !ok {
		return nil, ErrNotFound
	}

	switch cmd.Status {
	case "pending":
		if cmd.TargetAgentID != agentID {
			return nil, fmt.Errorf("%w: command belongs to agent %s", ErrConflict, cmd.TargetAgentID)
		}
		now := time.Now().UTC()
		cmd.Status = "claimed"
		cmd.ClaimedBy = agentID
		cmd.ClaimedAt = &now
		return cmd, nil

	case "claimed":
		if cmd.ClaimedBy != agentID {
			return nil, fmt.Errorf("%w: already claimed by %s", ErrConflict, cmd.ClaimedBy)
		}
		return cmd, nil // idempotent

	default:
		return nil, fmt.Errorf("%w: command status is %s", ErrConflict, cmd.Status)
	}
}

// StartCommand transitions a claimed command to running.
//
// For DEPLOY_APPLICATION it also advances the deployment from command_created → deploying.
// Idempotent if already running by the same agent.
func (s *Store) StartCommand(commandID, agentID string) (*Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, ok := s.Commands[commandID]
	if !ok {
		return nil, ErrNotFound
	}

	switch cmd.Status {
	case "claimed":
		if cmd.ClaimedBy != agentID {
			return nil, fmt.Errorf("%w: claimed by %s", ErrConflict, cmd.ClaimedBy)
		}
		now := time.Now().UTC()
		cmd.Status = "running"
		cmd.StartedAt = &now

		if cmd.Type == "DEPLOY_APPLICATION" && cmd.DeploymentID != "" {
			if dep, ok := s.Deployments[cmd.DeploymentID]; ok && dep.Status == "command_created" {
				dep.Status = "deploying"
				dep.UpdatedAt = now
			}
		}
		return cmd, nil

	case "running":
		if cmd.ClaimedBy != agentID {
			return nil, fmt.Errorf("%w: running by %s", ErrConflict, cmd.ClaimedBy)
		}
		return cmd, nil // idempotent

	default:
		return nil, fmt.Errorf("%w: command status is %s", ErrConflict, cmd.Status)
	}
}

type ReportCommandInput struct {
	AgentID      string
	DeploymentID string // may be empty; falls back to Command.DeploymentID
	Status       string // "succeeded" | "failed"
	Result       json.RawMessage
	Error        *ReportError
}

// ReportCommand finalises a command and applies all downstream state transitions.
func (s *Store) ReportCommand(commandID string, in ReportCommandInput) (CommandReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, ok := s.Commands[commandID]
	if !ok {
		return CommandReport{}, ErrNotFound
	}

	if cmd.Status != "running" && cmd.Status != "claimed" {
		return CommandReport{}, fmt.Errorf("%w: command status is %s", ErrConflict, cmd.Status)
	}
	if in.Status != "succeeded" && in.Status != "failed" {
		return CommandReport{}, fmt.Errorf("%w: invalid report status %q", ErrInvalidState, in.Status)
	}

	effectiveDepID := in.DeploymentID
	if effectiveDepID == "" {
		effectiveDepID = cmd.DeploymentID
	}

	now := time.Now().UTC()
	cmd.FinishedAt = &now

	report := CommandReport{
		CommandID:    commandID,
		DeploymentID: effectiveDepID,
		AgentID:      in.AgentID,
		Status:       in.Status,
		Result:       in.Result,
		Error:        in.Error,
		ReceivedAt:   now,
	}
	s.CommandReports = append(s.CommandReports, report)

	switch in.Status {
	case "succeeded":
		cmd.Status = "succeeded"
		s.applySucceeded(cmd, effectiveDepID, in.Result)
	case "failed":
		cmd.Status = "failed"
		if dep, ok2 := s.Deployments[effectiveDepID]; ok2 && dep.Status == "deploying" {
			dep.Status = "failed"
			dep.UpdatedAt = now
		}
	}

	return report, nil
}

// deployResult extracts upstream-relevant fields from a DEPLOY_APPLICATION result.
type deployResult struct {
	RuntimePrivateIP string `json:"runtime_private_ip"`
	HostPort         int    `json:"host_port"`
}

func (s *Store) applySucceeded(cmd *Command, depID string, result json.RawMessage) {
	now := time.Now().UTC()

	switch cmd.Type {
	case "DEPLOY_APPLICATION":
		dep, ok := s.Deployments[depID]
		if !ok {
			return
		}
		var res deployResult
		_ = json.Unmarshal(result, &res)

		dep.RuntimePrivateIP = res.RuntimePrivateIP
		dep.HostPort = res.HostPort
		dep.Status = "healthy"
		dep.UpdatedAt = now

		if dep.RequiresRoute && dep.Host != "" {
			upstream := s.buildUpstream(res.RuntimePrivateIP, res.HostPort)
			route := s.upsertRoute(dep.Environment, dep.Host, "/", upstream, depID, dep.HealthCheckPath)
			dep.RouteID = route.ID
			dep.Status = "route_pending"
			dep.UpdatedAt = now
			s.incrementDesiredState(dep.Environment)
		}

	case "STOP_APPLICATION":
		if dep, ok := s.Deployments[depID]; ok {
			dep.Status = "removed"
			dep.UpdatedAt = now
		}

	case "REMOVE_DEPLOYMENT":
		dep, ok := s.Deployments[depID]
		if !ok {
			return
		}
		routeID := dep.RouteID
		env := dep.Environment
		dep.Status = "removed"
		dep.UpdatedAt = now
		if routeID != "" {
			s.removeRoute(env, routeID)
			s.incrementDesiredState(env)
			dep.RouteID = ""
		}

	case "CLEANUP_DRAINING":
		// MVP: only store the report; no deployment transitions.
	}
}

func (s *Store) ListCommandReports() []CommandReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CommandReport, len(s.CommandReports))
	copy(out, s.CommandReports)
	return out
}
