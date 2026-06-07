package api

import (
	"net/http"

	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/response"
)

// corsMiddleware allows browser-based tools (e.g. local test consoles) to call
// this fake API from any origin — it's a dev/test tool, not production.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.AuthEnabled {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("Authorization") != "Bearer "+cfg.Token {
				response.Error(w, http.StatusUnauthorized,
					"AUTHENTICATION_FAILED", "Invalid or missing token", false)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
