package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/waelson/fake-platform-api/internal/api"
	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/store"
)

func main() {
	cfg := config.Load()
	st := store.New(cfg.UpstreamHost)
	router := api.NewRouter(cfg, st)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("fake-platform-api listening on %s (env=%s auth=%v)", addr, cfg.Environment, cfg.AuthEnabled)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
