package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/waelson/fake-platform-api/internal/ids"
)

type RegisterInput struct {
	Mode         string
	Environment  string
	Role         string
	Hostname     string
	InstanceID   string
	PrivateIP    string
	PublicIP     *string
	Version      string
	Capabilities map[string]any
}

func agentKey(instanceID, mode, environment, role string) string {
	return fmt.Sprintf("%s:%s:%s:%s", instanceID, mode, environment, role)
}

// RegisterAgent upserts an agent. Re-registration preserves the agent ID.
func (s *Store) RegisterAgent(in RegisterInput) *Agent {
	key := agentKey(in.InstanceID, in.Mode, in.Environment, in.Role)

	s.mu.Lock()
	defer s.mu.Unlock()

	if agentID, ok := s.AgentIndex[key]; ok {
		a := s.Agents[agentID]
		a.Hostname = in.Hostname
		a.Version = in.Version
		a.PrivateIP = in.PrivateIP
		a.PublicIP = in.PublicIP
		a.Capabilities = in.Capabilities
		a.Status = "online"
		a.LastSeenAt = time.Now().UTC()
		return a
	}

	envRoleKey := fmt.Sprintf("%s-%s", in.Environment, in.Role)
	s.Counters.AgentByEnvironmentRole[envRoleKey]++
	n := s.Counters.AgentByEnvironmentRole[envRoleKey]
	agentID := ids.Agent(in.Environment, in.Role, n)

	now := time.Now().UTC()
	a := &Agent{
		ID:           agentID,
		Mode:         in.Mode,
		Environment:  in.Environment,
		Role:         in.Role,
		Hostname:     in.Hostname,
		InstanceID:   in.InstanceID,
		PrivateIP:    in.PrivateIP,
		PublicIP:     in.PublicIP,
		Version:      in.Version,
		Status:       "online",
		Capabilities: in.Capabilities,
		RegisteredAt: now,
		LastSeenAt:   now,
	}

	s.Agents[agentID] = a
	s.AgentIndex[key] = agentID
	return a
}

// heartbeatCore extracts the common subset of fields present in every heartbeat payload.
type heartbeatCore struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	PrivateIP string `json:"private_ip"`
}

// Heartbeat stores the full raw payload and updates the agent's live fields.
func (s *Store) Heartbeat(agentID string, raw json.RawMessage) (*Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.Agents[agentID]
	if !ok {
		return nil, ErrNotFound
	}

	var core heartbeatCore
	_ = json.Unmarshal(raw, &core)

	a.LastHeartbeat = raw
	a.LastSeenAt = time.Now().UTC()
	if core.Status != "" {
		a.Status = core.Status
	}
	if core.Version != "" {
		a.Version = core.Version
	}
	if core.PrivateIP != "" {
		a.PrivateIP = core.PrivateIP
	}

	return a, nil
}

func (s *Store) GetAgent(agentID string) (*Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.Agents[agentID]
	return a, ok
}

func (s *Store) ListAgents() []*Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Agent, 0, len(s.Agents))
	for _, a := range s.Agents {
		out = append(out, a)
	}
	return out
}

// FindAgentByEnvironmentRole returns the first online agent matching env+role.
func (s *Store) FindAgentByEnvironmentRole(environment, role string) (*Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.Agents {
		if a.Environment == environment && a.Role == role {
			return a, true
		}
	}
	return nil, false
}
