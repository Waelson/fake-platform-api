package store

import (
	"encoding/json"
	"time"
)

// ---- Agent ----

type Agent struct {
	ID            string          `json:"id"`
	Mode          string          `json:"mode"`
	Environment   string          `json:"environment"`
	Role          string          `json:"role"`
	Hostname      string          `json:"hostname"`
	InstanceID    string          `json:"instance_id"`
	PrivateIP     string          `json:"private_ip"`
	PublicIP      *string         `json:"public_ip"`
	Version       string          `json:"version"`
	Status        string          `json:"status"`
	Capabilities  map[string]any  `json:"capabilities"`
	LastHeartbeat json.RawMessage `json:"last_heartbeat"`
	RegisteredAt  time.Time       `json:"registered_at"`
	LastSeenAt    time.Time       `json:"last_seen_at"`
}

// ---- Command ----

type Command struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	DeploymentID  string          `json:"deployment_id,omitempty"`
	TargetAgentID string          `json:"target_agent_id"`
	Status        string          `json:"status"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
	ClaimedAt     *time.Time      `json:"claimed_at,omitempty"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	ClaimedBy     string          `json:"claimed_by,omitempty"`
}

// ---- Deployment ----

type Deployment struct {
	ID                    string    `json:"id"`
	Application           string    `json:"application"`
	Environment           string    `json:"environment"`
	Image                 string    `json:"image"`
	Host                  string    `json:"host,omitempty"`
	Status                string    `json:"status"`
	TargetAgentID         string    `json:"target_agent_id"`
	CommandID             string    `json:"command_id"`
	ContainerName         string    `json:"container_name"`
	ContainerInternalPort int       `json:"container_internal_port"`
	HealthCheckPath       string    `json:"health_check_path"`
	RequiresRoute         bool      `json:"requires_route"`
	RuntimePrivateIP      string    `json:"runtime_private_ip"`
	HostPort              int       `json:"host_port"`
	RouteID               string    `json:"route_id,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ---- Route ----

type Route struct {
	ID              string `json:"id"`
	Environment     string `json:"environment"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	Upstream        string `json:"upstream"`
	DeploymentID    string `json:"deployment_id"`
	HealthCheckPath string `json:"health_check_path"`
}

// ---- DesiredState ----

type DesiredState struct {
	Version     int     `json:"version"`
	Type        string  `json:"type"`
	Environment string  `json:"environment"`
	Routes      []Route `json:"routes"`
}

// ---- CommandReport ----

type ReportError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation,omitempty"`
	Retryable bool   `json:"retryable"`
}

type CommandReport struct {
	CommandID    string          `json:"command_id"`
	DeploymentID string          `json:"deployment_id"`
	AgentID      string          `json:"agent_id"`
	Status       string          `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        *ReportError    `json:"error,omitempty"`
	ReceivedAt   time.Time       `json:"received_at"`
}

// ---- DesiredStateReport ----

type DesiredStateReport struct {
	AgentID                    string     `json:"agent_id"`
	DesiredStateVersion        int        `json:"desired_state_version"`
	CurrentDesiredStateVersion int        `json:"current_desired_state_version"`
	Stale                      bool       `json:"stale"`
	Environment                string     `json:"environment"`
	Type                       string     `json:"type"`
	Status                     string     `json:"status"`
	RoutesTotal                int        `json:"routes_total"`
	ValidatedRoutes            int        `json:"validated_routes"`
	FailedRoutes               int        `json:"failed_routes"`
	Error                      *ReportError `json:"error,omitempty"`
	ReceivedAt                 time.Time  `json:"received_at"`
}

// ---- Counters ----

type Counters struct {
	AgentByEnvironmentRole          map[string]int
	Command                         int
	Deployment                      int
	Route                           int
	DesiredStateVersionByEnvironment map[string]int
}

// ---- Payload structs ----

type DeployApplicationPayload struct {
	Application          string            `json:"application"`
	Environment          string            `json:"environment"`
	Image                string            `json:"image"`
	ContainerName        string            `json:"container_name"`
	ContainerInternalPort int              `json:"container_internal_port"`
	HealthCheckPath      string            `json:"health_check_path"`
	RequiresRoute        bool              `json:"requires_route"`
	EnvironmentVariables map[string]string `json:"environment_variables,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
}

type StopApplicationPayload struct {
	DeploymentID       string `json:"deployment_id"`
	ContainerName      string `json:"container_name"`
	StopTimeoutSeconds int    `json:"stop_timeout_seconds"`
}

type RemoveDeploymentPayload struct {
	DeploymentID  string `json:"deployment_id"`
	ContainerName string `json:"container_name"`
	ReleasePort   bool   `json:"release_port"`
}

type CleanupDrainingPayload struct {
	Environment      string `json:"environment"`
	OlderThanSeconds int    `json:"older_than_seconds"`
}
