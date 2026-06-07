package store

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidState  = errors.New("invalid state")
	ErrFutureVersion = errors.New("future version")
)

type Store struct {
	mu sync.RWMutex

	Agents     map[string]*Agent
	AgentIndex map[string]string // key: instance_id:mode:env:role → agent_id

	Commands    map[string]*Command
	Deployments map[string]*Deployment

	CommandReports      []CommandReport
	DesiredStates       map[string]*DesiredState
	DesiredStateReports []DesiredStateReport

	Counters Counters
}

func New() *Store {
	return &Store{
		Agents:        make(map[string]*Agent),
		AgentIndex:    make(map[string]string),
		Commands:      make(map[string]*Command),
		Deployments:   make(map[string]*Deployment),
		DesiredStates: make(map[string]*DesiredState),
		Counters: Counters{
			AgentByEnvironmentRole:           make(map[string]int),
			DesiredStateVersionByEnvironment: make(map[string]int),
		},
	}
}

// buildUpstream always points to the runtime instance's private IP — the
// Gateway Agent may run on a different host than the Runtime Agent, so any
// locally-resolvable hostname (e.g. host.docker.internal) would be invalid.
func (s *Store) buildUpstream(privateIP string, hostPort int) string {
	return fmt.Sprintf("%s:%d", privateIP, hostPort)
}

// Debug returns a snapshot of the full store for diagnostics.
func (s *Store) Debug() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"agents":                s.Agents,
		"agent_index":           s.AgentIndex,
		"commands":              s.Commands,
		"deployments":           s.Deployments,
		"command_reports":       s.CommandReports,
		"desired_states":        s.DesiredStates,
		"desired_state_reports": s.DesiredStateReports,
		"counters":              s.Counters,
	}
}
