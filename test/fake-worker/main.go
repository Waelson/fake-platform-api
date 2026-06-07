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
	workerName := getEnv("WORKER_NAME", "fake-worker")
	workerVersion := getEnv("WORKER_VERSION", "v1")
	port := getEnv("PORT", "")

	log.Printf("fake-worker started (name=%s version=%s)", workerName, workerVersion)

	// If PORT is set, expose a minimal /health endpoint.
	if port != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})
		go func() {
			addr := fmt.Sprintf(":%s", port)
			log.Printf("fake-worker health endpoint on %s", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Fatalf("health server error: %v", err)
			}
		}()
	}

	// Main worker loop — periodically writes logs.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		log.Printf("fake-worker %s@%s processing...", workerName, workerVersion)
	}
}
