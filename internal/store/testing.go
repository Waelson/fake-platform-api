package store

// Reset clears all state and resets all counters (IDs restart from 001).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Agents = make(map[string]*Agent)
	s.AgentIndex = make(map[string]string)
	s.Commands = make(map[string]*Command)
	s.Deployments = make(map[string]*Deployment)
	s.CommandReports = nil
	s.DesiredStates = make(map[string]*DesiredState)
	s.DesiredStateReports = nil
	s.Counters = Counters{
		AgentByEnvironmentRole:           make(map[string]int),
		DesiredStateVersionByEnvironment: make(map[string]int),
	}
}
