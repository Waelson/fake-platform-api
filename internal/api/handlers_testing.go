package api

import (
	"encoding/json"
	"net/http"

	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/response"
	"github.com/waelson/fake-platform-api/internal/store"
)

// ---- timeout defaults ----

const (
	timeoutDeploy  = 600
	timeoutStop    = 60
	timeoutRemove  = 120
	timeoutCleanup = 300
)

// ---- POST /testing/commands/deploy ----

type testingDeployRequest struct {
	TargetAgentRole       string            `json:"target_agent_role"`
	Application           string            `json:"application"`
	Environment           string            `json:"environment"`
	Image                 string            `json:"image"`
	ContainerName         string            `json:"container_name"`
	ContainerInternalPort int               `json:"container_internal_port"`
	HealthCheckPath       string            `json:"health_check_path"`
	RequiresRoute         bool              `json:"requires_route"`
	Host                  string            `json:"host"`
	EnvironmentVariables  map[string]string `json:"environment_variables"`
}

type testingDeployResponse struct {
	DeploymentID  string `json:"deployment_id"`
	CommandID     string `json:"command_id"`
	TargetAgentID string `json:"target_agent_id"`
	Status        string `json:"status"`
}

func handleTestingDeploy(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testingDeployRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}

		if req.Application == "" || req.Image == "" || req.TargetAgentRole == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST",
				"application, image and target_agent_role are required", false)
			return
		}
		if req.Environment == "" {
			req.Environment = cfg.Environment
		}

		agent, ok := st.FindAgentByEnvironmentRole(req.Environment, req.TargetAgentRole)
		if !ok {
			response.Error(w, http.StatusNotFound, "AGENT_NOT_FOUND",
				"no agent registered for the given environment and role", false)
			return
		}

		dep, cmd := st.InitDeploy(store.InitDeployInput{
			Application:           req.Application,
			Environment:           req.Environment,
			Image:                 req.Image,
			Host:                  req.Host,
			TargetAgentID:         agent.ID,
			ContainerName:         req.ContainerName,
			ContainerInternalPort: req.ContainerInternalPort,
			HealthCheckPath:       req.HealthCheckPath,
			RequiresRoute:         req.RequiresRoute,
			EnvironmentVariables:  req.EnvironmentVariables,
			CommandTimeoutSeconds: timeoutDeploy,
		})

		response.JSON(w, http.StatusOK, testingDeployResponse{
			DeploymentID:  dep.ID,
			CommandID:     cmd.ID,
			TargetAgentID: agent.ID,
			Status:        "pending",
		})
	}
}

// ---- POST /testing/commands/stop ----

type testingStopRequest struct {
	TargetAgentID      string `json:"target_agent_id"`
	DeploymentID       string `json:"deployment_id"`
	ContainerName      string `json:"container_name"`
	StopTimeoutSeconds int    `json:"stop_timeout_seconds"`
}

type testingCommandResponse struct {
	CommandID     string `json:"command_id"`
	DeploymentID  string `json:"deployment_id,omitempty"`
	TargetAgentID string `json:"target_agent_id"`
	Status        string `json:"status"`
}

func handleTestingStop(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testingStopRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}
		if req.DeploymentID == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST",
				"deployment_id is required", false)
			return
		}

		// Resolve container_name and target_agent_id from deployment if not provided.
		containerName := req.ContainerName
		if dep, ok := st.GetDeployment(req.DeploymentID); ok {
			if containerName == "" {
				containerName = dep.ContainerName
			}
			if req.TargetAgentID == "" {
				req.TargetAgentID = dep.TargetAgentID
			}
		} else {
			response.Error(w, http.StatusNotFound, "DEPLOYMENT_NOT_FOUND",
				"deployment not found", false)
			return
		}

		stopTimeout := req.StopTimeoutSeconds
		if stopTimeout <= 0 {
			stopTimeout = timeoutStop
		}

		payload, _ := json.Marshal(store.StopApplicationPayload{
			DeploymentID:       req.DeploymentID,
			ContainerName:      containerName,
			StopTimeoutSeconds: stopTimeout,
		})

		cmd := st.CreateCommand(store.CreateCommandInput{
			Type:           "STOP_APPLICATION",
			DeploymentID:   req.DeploymentID,
			TargetAgentID:  req.TargetAgentID,
			TimeoutSeconds: timeoutStop,
			Payload:        payload,
		})

		response.JSON(w, http.StatusOK, testingCommandResponse{
			CommandID:     cmd.ID,
			DeploymentID:  req.DeploymentID,
			TargetAgentID: req.TargetAgentID,
			Status:        "pending",
		})
	}
}

// ---- POST /testing/commands/remove ----

type testingRemoveRequest struct {
	TargetAgentID string `json:"target_agent_id"`
	DeploymentID  string `json:"deployment_id"`
	ContainerName string `json:"container_name"`
	ReleasePort   bool   `json:"release_port"`
}

func handleTestingRemove(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testingRemoveRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}
		if req.DeploymentID == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST",
				"deployment_id is required", false)
			return
		}

		// Resolve container_name and target_agent_id from deployment if not provided.
		containerName := req.ContainerName
		if dep, ok := st.GetDeployment(req.DeploymentID); ok {
			if containerName == "" {
				containerName = dep.ContainerName
			}
			if req.TargetAgentID == "" {
				req.TargetAgentID = dep.TargetAgentID
			}
		} else {
			response.Error(w, http.StatusNotFound, "DEPLOYMENT_NOT_FOUND",
				"deployment not found", false)
			return
		}

		payload, _ := json.Marshal(store.RemoveDeploymentPayload{
			DeploymentID:  req.DeploymentID,
			ContainerName: containerName,
			ReleasePort:   req.ReleasePort,
		})

		cmd := st.CreateCommand(store.CreateCommandInput{
			Type:           "REMOVE_DEPLOYMENT",
			DeploymentID:   req.DeploymentID,
			TargetAgentID:  req.TargetAgentID,
			TimeoutSeconds: timeoutRemove,
			Payload:        payload,
		})

		response.JSON(w, http.StatusOK, testingCommandResponse{
			CommandID:     cmd.ID,
			DeploymentID:  req.DeploymentID,
			TargetAgentID: req.TargetAgentID,
			Status:        "pending",
		})
	}
}

// ---- POST /testing/commands/cleanup-draining ----

type testingCleanupDrainingRequest struct {
	TargetAgentRole  string `json:"target_agent_role"`
	Environment      string `json:"environment"`
	OlderThanSeconds int    `json:"older_than_seconds"`
}

func handleTestingCleanupDraining(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testingCleanupDrainingRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}
		if req.TargetAgentRole == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST",
				"target_agent_role is required", false)
			return
		}
		if req.Environment == "" {
			req.Environment = cfg.Environment
		}
		if req.OlderThanSeconds <= 0 {
			req.OlderThanSeconds = timeoutCleanup
		}

		agent, ok := st.FindAgentByEnvironmentRole(req.Environment, req.TargetAgentRole)
		if !ok {
			response.Error(w, http.StatusNotFound, "AGENT_NOT_FOUND",
				"no agent registered for the given environment and role", false)
			return
		}

		payload, _ := json.Marshal(store.CleanupDrainingPayload{
			Environment:      req.Environment,
			OlderThanSeconds: req.OlderThanSeconds,
		})

		cmd := st.CreateCommand(store.CreateCommandInput{
			Type:           "CLEANUP_DRAINING",
			TargetAgentID:  agent.ID,
			TimeoutSeconds: timeoutCleanup,
			Payload:        payload,
		})

		response.JSON(w, http.StatusOK, testingCommandResponse{
			CommandID:     cmd.ID,
			TargetAgentID: agent.ID,
			Status:        "pending",
		})
	}
}

// ---- GET /testing/agents ----

func handleTestingListAgents(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, st.ListAgents())
	}
}

// ---- GET /testing/commands ----

func handleTestingListCommands(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cmds := st.ListCommands(q.Get("status"), q.Get("type"), q.Get("agent_id"))
		response.JSON(w, http.StatusOK, cmds)
	}
}

// ---- GET /testing/deployments ----

func handleTestingListDeployments(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, st.ListDeployments())
	}
}

// ---- GET /testing/reports ----

func handleTestingListReports(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, st.ListCommandReports())
	}
}

// ---- GET /testing/desired-state ----

// Two formats (decision 23):
//   - ?environment=X → single DesiredState
//   - no param       → {"desired_states": {env: DesiredState, ...}}

func handleTestingDesiredState(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		env := r.URL.Query().Get("environment")
		if env != "" {
			response.JSON(w, http.StatusOK, st.GetDesiredState(env))
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{
			"desired_states": st.GetAllDesiredStates(),
		})
	}
}

// ---- GET /testing/desired-state/reports ----

func handleTestingDesiredStateReports(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, st.ListDesiredStateReports())
	}
}

// ---- GET /testing/debug ----

func handleTestingDebug(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, st.Debug())
	}
}

// ---- POST /testing/reset ----

func handleTestingReset(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st.Reset()
		response.JSON(w, http.StatusOK, map[string]string{"status": "reset"})
	}
}
