package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	appName := getEnv("APP_NAME", "fake-app")
	appVersion := getEnv("APP_VERSION", "v1")
	appMode := getEnv("APP_MODE", "healthy")
	port := getEnv("PORT", "3000")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"app":     appName,
			"version": appVersion,
			"mode":    appMode,
		})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		switch appMode {
		case "broken":
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
		case "slow":
			time.Sleep(10 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	})

	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version": appVersion,
		})
	})

	mux.HandleFunc("GET /env", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"APP_NAME":    appName,
			"APP_VERSION": appVersion,
			"APP_MODE":    appMode,
			"PORT":        port,
		})
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("fake-app listening on %s (name=%s version=%s mode=%s)", addr, appName, appVersion, appMode)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
