package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/waelson/fake-platform-api/internal/api"
	"github.com/waelson/fake-platform-api/internal/config"
	"github.com/waelson/fake-platform-api/internal/persistence"
	"github.com/waelson/fake-platform-api/internal/store"
)

func main() {
	cfg := config.Load()
	st := store.New()

	if cfg.StateFile != "" {
		snap, err := persistence.Load(cfg.StateFile)
		if err != nil {
			log.Printf("state file %s: failed to load, starting empty: %v", cfg.StateFile, err)
		} else if snap != nil {
			if snap.SchemaVersion != store.SchemaVersion {
				log.Printf("state file %s: schema version mismatch (got %d, want %d), starting empty", cfg.StateFile, snap.SchemaVersion, store.SchemaVersion)
			} else {
				st.Restore(*snap)
				log.Printf("state file %s: restored snapshot (schema_version=%d)", cfg.StateFile, snap.SchemaVersion)
			}
		}
	}

	router := api.NewRouter(cfg, st)

	addr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{Addr: addr, Handler: router}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.StateFile != "" {
		go runSnapshotLoop(ctx, st, cfg.StateFile, time.Duration(cfg.StateSaveIntervalSeconds)*time.Second)
	}

	go func() {
		log.Printf("fake-platform-api listening on %s (env=%s auth=%v)", addr, cfg.Environment, cfg.AuthEnabled)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	if cfg.StateFile != "" {
		if err := persistence.Save(cfg.StateFile, st.Snapshot()); err != nil {
			log.Printf("state file %s: final save failed: %v", cfg.StateFile, err)
		} else {
			log.Printf("state file %s: final save complete", cfg.StateFile)
		}
	}
}

// runSnapshotLoop periodically saves the store's state to path until ctx is
// cancelled (e.g. on SIGTERM/SIGINT, where main performs one last save).
func runSnapshotLoop(ctx context.Context, st *store.Store, path string, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := persistence.Save(path, st.Snapshot()); err != nil {
				log.Printf("state file %s: periodic save failed: %v", path, err)
			}
		}
	}
}
