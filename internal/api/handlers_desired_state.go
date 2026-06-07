package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waelson/fake-platform-api/internal/response"
	"github.com/waelson/fake-platform-api/internal/store"
)

// ---- GET /api/agents/{agentID}/desired-state ----

func handleGetDesiredState(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")

		agent, ok := st.GetAgent(agentID)
		if !ok {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "agent not found", false)
			return
		}

		ds := st.GetDesiredState(agent.Environment)
		response.JSON(w, http.StatusOK, ds)
	}
}

// ---- POST /api/agents/{agentID}/desired-state/report ----

type desiredStateReportRequest struct {
	Status              string             `json:"status"`
	DesiredStateVersion int                `json:"desired_state_version"`
	Type                string             `json:"type"`
	Environment         string             `json:"environment"`
	RoutesTotal         int                `json:"routes_total"`
	ValidatedRoutes     int                `json:"validated_routes"`
	FailedRoutes        int                `json:"failed_routes"`
	Error               *store.ReportError `json:"error"`
}

func handleDesiredStateReport(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := chi.URLParam(r, "agentID")

		var req desiredStateReportRequest
		if err := decode(r, &req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", false)
			return
		}

		_, err := st.ApplyDesiredStateReport(store.DesiredStateReportInput{
			AgentID:             agentID,
			DesiredStateVersion: req.DesiredStateVersion,
			Type:                req.Type,
			Environment:         req.Environment,
			Status:              req.Status,
			RoutesTotal:         req.RoutesTotal,
			ValidatedRoutes:     req.ValidatedRoutes,
			FailedRoutes:        req.FailedRoutes,
			Error:               req.Error,
		})
		if err != nil {
			if errors.Is(err, store.ErrFutureVersion) {
				response.Error(w, http.StatusConflict, "INVALID_VERSION",
					"reported version is ahead of current", false)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), false)
			return
		}

		response.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
	}
}
