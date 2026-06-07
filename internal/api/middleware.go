package api

import (
	"net/http"

	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/response"
)

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
