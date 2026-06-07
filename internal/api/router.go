package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/response"
	"github.com/waelson/fake-platform-api/internal/store"
)

func NewRouter(cfg *config.Config, st *store.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", handleHealth)

	// Official API — requires auth when enabled.
	r.Route("/api", func(r chi.Router) {
		r.Use(authMiddleware(cfg))
		r.Post("/agents/register", handleRegisterAgent(st))
		r.Post("/agents/{agentID}/heartbeat", handleHeartbeat(st))
		r.Get("/agents/{agentID}/commands/pending", handleListPendingCommands(st))
		r.Post("/agents/{agentID}/commands/{commandID}/claim", handleClaimCommand(st))
		r.Post("/agents/{agentID}/commands/{commandID}/start", handleStartCommand(st))
		r.Post("/agents/{agentID}/commands/{commandID}/report", handleReportCommand(st))
		r.Get("/agents/{agentID}/desired-state", handleGetDesiredState(st))
		r.Post("/agents/{agentID}/desired-state/report", handleDesiredStateReport(st))
	})

	// Testing endpoints — always public (no auth).
	r.Route("/testing", func(r chi.Router) {
		r.Post("/commands/deploy", handleTestingDeploy(cfg, st))
		r.Post("/commands/stop", handleTestingStop(st))
		r.Post("/commands/remove", handleTestingRemove(st))
		r.Post("/commands/cleanup-draining", handleTestingCleanupDraining(cfg, st))
		r.Get("/agents", handleTestingListAgents(st))
		r.Get("/commands", handleTestingListCommands(st))
		r.Get("/deployments", handleTestingListDeployments(st))
		r.Get("/reports", handleTestingListReports(st))
		r.Get("/desired-state", handleTestingDesiredState(st))
		r.Get("/desired-state/reports", handleTestingDesiredStateReports(st))
		r.Get("/debug", handleTestingDebug(st))
		r.Post("/reset", handleTestingReset(st))
	})

	return r
}

type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}
