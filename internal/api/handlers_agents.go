package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/waelson/fake-platform-api/internal/response"
	"github.com/waelson/fake-platform-api/internal/store"
)

// decode is a shared helper used by all handlers.
func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// ---- POST /api/agents/register ----

type registerRequest struct {
	Mode         string         `json:"mode"`
	Environment  string         `json:"environment"`
	Role         string         `json:"role"`
	Hostname     string         `json:"hostname"`
	InstanceID   string         `json:"instance_id"`
	PrivateIP    string         `json:"private_ip"`
	PublicIP     *string        `json:"public_ip"`
	Version      string         `json:"version"`
	Capabilities map[string]any `json:"capabilities"`
}

type registerResponse struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

func handleRegisterAgent(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}
		if req.InstanceID == "" || req.Environment == "" || req.Role == "" || req.Mode == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST",
				"instance_id, environment, role and mode are required", false)
			return
		}

		agent := st.RegisterAgent(store.RegisterInput{
			Mode:         req.Mode,
			Environment:  req.Environment,
			Role:         req.Role,
			Hostname:     req.Hostname,
			InstanceID:   req.InstanceID,
			PrivateIP:    req.PrivateIP,
			PublicIP:     req.PublicIP,
			Version:      req.Version,
			Capabilities: req.Capabilities,
		})

		response.JSON(w, http.StatusOK, registerResponse{
			AgentID: agent.ID,
			Status:  "registered",
		})
	}
}

// ---- POST /api/agents/{agentID}/heartbeat ----

type heartbeatResponse struct {
	Status     string    `json:"status"`
	ServerTime time.Time `json:"server_time"`
}

func handleHeartbeat(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")

		var raw json.RawMessage
		if err := decode(r, &raw); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}

		if _, err := st.Heartbeat(agentID, raw); err != nil {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "agent not found", false)
			return
		}

		response.JSON(w, http.StatusOK, heartbeatResponse{
			Status:     "accepted",
			ServerTime: time.Now().UTC(),
		})
	}
}
