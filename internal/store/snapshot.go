package store

// SchemaVersion identifies the shape of Snapshot. Bump it whenever the
// exported Store fields change in a way that would break restoring an
// older snapshot, so Load can fail-safe (warn + start empty) on mismatch.
const SchemaVersion = 1

// Snapshot mirrors the exported state of the Store for persistence.
type Snapshot struct {
	SchemaVersion int `json:"schema_version"`

	Agents     map[string]*Agent `json:"agents"`
	AgentIndex map[string]string `json:"agent_index"`

	Commands    map[string]*Command    `json:"commands"`
	Deployments map[string]*Deployment `json:"deployments"`

	CommandReports      []CommandReport          `json:"command_reports"`
	DesiredStates       map[string]*DesiredState `json:"desired_states"`
	DesiredStateReports []DesiredStateReport     `json:"desired_state_reports"`

	Counters Counters `json:"counters"`
}

// Snapshot copies the current state of the store under a read lock.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make(map[string]*Agent, len(s.Agents))
	for id, a := range s.Agents {
		cp := *a
		agents[id] = &cp
	}

	agentIndex := make(map[string]string, len(s.AgentIndex))
	for k, v := range s.AgentIndex {
		agentIndex[k] = v
	}

	commands := make(map[string]*Command, len(s.Commands))
	for id, c := range s.Commands {
		cp := *c
		commands[id] = &cp
	}

	deployments := make(map[string]*Deployment, len(s.Deployments))
	for id, d := range s.Deployments {
		cp := *d
		deployments[id] = &cp
	}

	commandReports := make([]CommandReport, len(s.CommandReports))
	copy(commandReports, s.CommandReports)

	desiredStates := make(map[string]*DesiredState, len(s.DesiredStates))
	for k, ds := range s.DesiredStates {
		cp := *ds
		cp.Routes = make([]Route, len(ds.Routes))
		copy(cp.Routes, ds.Routes)
		desiredStates[k] = &cp
	}

	desiredStateReports := make([]DesiredStateReport, len(s.DesiredStateReports))
	copy(desiredStateReports, s.DesiredStateReports)

	agentByEnvRole := make(map[string]int, len(s.Counters.AgentByEnvironmentRole))
	for k, v := range s.Counters.AgentByEnvironmentRole {
		agentByEnvRole[k] = v
	}
	desiredStateVersionByEnv := make(map[string]int, len(s.Counters.DesiredStateVersionByEnvironment))
	for k, v := range s.Counters.DesiredStateVersionByEnvironment {
		desiredStateVersionByEnv[k] = v
	}

	return Snapshot{
		SchemaVersion: SchemaVersion,

		Agents:     agents,
		AgentIndex: agentIndex,

		Commands:    commands,
		Deployments: deployments,

		CommandReports:      commandReports,
		DesiredStates:       desiredStates,
		DesiredStateReports: desiredStateReports,

		Counters: Counters{
			AgentByEnvironmentRole:           agentByEnvRole,
			Command:                          s.Counters.Command,
			Deployment:                       s.Counters.Deployment,
			Route:                            s.Counters.Route,
			DesiredStateVersionByEnvironment: desiredStateVersionByEnv,
		},
	}
}

// Restore replaces the store's state with the contents of snap. It must only
// be called during initialization, before the store is exposed to concurrent
// requests.
func (s *Store) Restore(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Agents = snap.Agents
	if s.Agents == nil {
		s.Agents = make(map[string]*Agent)
	}

	s.AgentIndex = snap.AgentIndex
	if s.AgentIndex == nil {
		s.AgentIndex = make(map[string]string)
	}

	s.Commands = snap.Commands
	if s.Commands == nil {
		s.Commands = make(map[string]*Command)
	}

	s.Deployments = snap.Deployments
	if s.Deployments == nil {
		s.Deployments = make(map[string]*Deployment)
	}

	s.CommandReports = snap.CommandReports

	s.DesiredStates = snap.DesiredStates
	if s.DesiredStates == nil {
		s.DesiredStates = make(map[string]*DesiredState)
	}

	s.DesiredStateReports = snap.DesiredStateReports

	s.Counters = snap.Counters
	if s.Counters.AgentByEnvironmentRole == nil {
		s.Counters.AgentByEnvironmentRole = make(map[string]int)
	}
	if s.Counters.DesiredStateVersionByEnvironment == nil {
		s.Counters.DesiredStateVersionByEnvironment = make(map[string]int)
	}
}
