package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/waelson/fake-platform-api/internal/ids"
)

// InitDeployInput carries all fields needed to atomically create a deployment + command pair.
// ContainerName is auto-generated as {Application}-{Environment}-{DepID} when empty.
// The 5 mandatory devex.* labels are always injected into the command payload and cannot
// be overridden by CustomLabels.
type InitDeployInput struct {
	Application           string
	Environment           string
	Image                 string
	Host                  string
	TargetAgentID         string
	ContainerName         string // auto-generated if empty
	ContainerInternalPort int
	HealthCheckPath       string
	RequiresRoute         bool
	EnvironmentVariables  map[string]string
	CustomLabels          map[string]string // merged with mandatory labels
	CommandTimeoutSeconds int
}

// InitDeploy atomically creates a Deployment (requested→command_created) and its DEPLOY_APPLICATION command.
func (s *Store) InitDeploy(in InitDeployInput) (*Deployment, *Command) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reserve IDs before building anything that depends on them.
	s.Counters.Deployment++
	depID := ids.Deployment(s.Counters.Deployment)
	s.Counters.Command++
	cmdID := ids.Command(s.Counters.Command)

	// Auto-generate container_name if not provided (decision 12).
	containerName := in.ContainerName
	if containerName == "" {
		containerName = fmt.Sprintf("%s-%s-%s", in.Application, in.Environment, depID)
	}

	// Build mandatory labels (decision 16). CustomLabels cannot override them.
	labels := map[string]string{
		"devex.managed":       "true",
		"devex.application":   in.Application,
		"devex.environment":   in.Environment,
		"devex.deployment_id": depID,
		"devex.command_id":    cmdID,
	}
	for k, v := range in.CustomLabels {
		if _, mandatory := labels[k]; !mandatory {
			labels[k] = v
		}
	}

	// Build payload.
	payload, _ := json.Marshal(DeployApplicationPayload{
		Application:           in.Application,
		Environment:           in.Environment,
		Image:                 in.Image,
		ContainerName:         containerName,
		ContainerInternalPort: in.ContainerInternalPort,
		HealthCheckPath:       in.HealthCheckPath,
		RequiresRoute:         in.RequiresRoute,
		EnvironmentVariables:  in.EnvironmentVariables,
		Labels:                labels,
	})

	now := time.Now().UTC()

	dep := &Deployment{
		ID:                    depID,
		Application:           in.Application,
		Environment:           in.Environment,
		Image:                 in.Image,
		Host:                  in.Host,
		Status:                "requested",
		TargetAgentID:         in.TargetAgentID,
		ContainerName:         containerName,
		ContainerInternalPort: in.ContainerInternalPort,
		HealthCheckPath:       in.HealthCheckPath,
		RequiresRoute:         in.RequiresRoute,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.Deployments[depID] = dep

	cmd := &Command{
		ID:             cmdID,
		Type:           "DEPLOY_APPLICATION",
		DeploymentID:   depID,
		TargetAgentID:  in.TargetAgentID,
		Status:         "pending",
		TimeoutSeconds: in.CommandTimeoutSeconds,
		Payload:        payload,
		CreatedAt:      now,
	}
	s.Commands[cmdID] = cmd

	// Link deployment → command and advance to command_created.
	dep.CommandID = cmdID
	dep.Status = "command_created"
	dep.UpdatedAt = now

	return dep, cmd
}

// CreateCommandInput is used for standalone commands (STOP, REMOVE, CLEANUP).
type CreateCommandInput struct {
	Type           string
	DeploymentID   string
	TargetAgentID  string
	TimeoutSeconds int
	Payload        json.RawMessage
}

func (s *Store) CreateCommand(in CreateCommandInput) *Command {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Counters.Command++
	cmdID := ids.Command(s.Counters.Command)

	cmd := &Command{
		ID:             cmdID,
		Type:           in.Type,
		DeploymentID:   in.DeploymentID,
		TargetAgentID:  in.TargetAgentID,
		Status:         "pending",
		TimeoutSeconds: in.TimeoutSeconds,
		Payload:        in.Payload,
		CreatedAt:      time.Now().UTC(),
	}
	s.Commands[cmdID] = cmd
	return cmd
}

func (s *Store) GetDeployment(depID string) (*Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.Deployments[depID]
	return d, ok
}

func (s *Store) ListDeployments() []*Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Deployment, 0, len(s.Deployments))
	for _, d := range s.Deployments {
		out = append(out, d)
	}
	return out
}
