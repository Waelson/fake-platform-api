package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/waelson/fake-platform-api/internal/response"
	"github.com/waelson/fake-platform-api/internal/store"
)

// ---- GET /api/agents/{agentID}/commands/pending ----

func handleListPendingCommands(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")

		if _, ok := st.GetAgent(agentID); !ok {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "agent not found", false)
			return
		}

		cmds := st.ListPendingByAgent(agentID)
		response.JSON(w, http.StatusOK, cmds)
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/claim ----

type claimResponse struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	ClaimedBy string     `json:"claimed_by"`
	ClaimedAt *time.Time `json:"claimed_at,omitempty"`
}

func handleClaimCommand(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")
		commandID := chi.URLParam(r, "commandID")

		cmd, err := st.ClaimCommand(commandID, agentID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "command not found", false)
				return
			}
			response.Error(w, http.StatusConflict, "CONFLICT", err.Error(), false)
			return
		}

		response.JSON(w, http.StatusOK, claimResponse{
			ID:        cmd.ID,
			Status:    cmd.Status,
			ClaimedBy: cmd.ClaimedBy,
			ClaimedAt: cmd.ClaimedAt,
		})
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/start ----

type startResponse struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	StartedAt *time.Time `json:"started_at,omitempty"`
}

func handleStartCommand(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")
		commandID := chi.URLParam(r, "commandID")

		cmd, err := st.StartCommand(commandID, agentID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "command not found", false)
				return
			}
			response.Error(w, http.StatusConflict, "CONFLICT", err.Error(), false)
			return
		}

		response.JSON(w, http.StatusOK, startResponse{
			ID:        cmd.ID,
			Status:    cmd.Status,
			StartedAt: cmd.StartedAt,
		})
	}
}

// ---- POST /api/agents/{agentID}/commands/{commandID}/report ----

type commandReportRequest struct {
	Status       string             `json:"status"`
	DeploymentID string             `json:"deployment_id"`
	Result       json.RawMessage    `json:"result"`
	Error        *store.ReportError `json:"error"`
}

func handleReportCommand(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")
		commandID := chi.URLParam(r, "commandID")

		var req commandReportRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}

		_, err := st.ReportCommand(commandID, store.ReportCommandInput{
			AgentID:      agentID,
			DeploymentID: req.DeploymentID,
			Status:       req.Status,
			Result:       req.Result,
			Error:        req.Error,
		})
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "command not found", false)
				return
			}
			response.Error(w, http.StatusConflict, "CONFLICT", err.Error(), false)
			return
		}

		response.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
	}
}
