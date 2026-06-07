// Package persistence handles loading and saving store.Snapshot to a local
// JSON file, so the Fake API can survive process/container restarts without
// requiring a database.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/waelson/fake-platform-api/internal/store"
)

// Load reads and unmarshals the snapshot at path. If the file does not exist,
// it returns (nil, nil) so callers can start with an empty store.
func Load(path string) (*store.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("persistence: read %s: %w", path, err)
	}

	var snap store.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("persistence: unmarshal %s: %w", path, err)
	}

	return &snap, nil
}

// Save serializes snap and writes it to path atomically: it writes to a
// temporary file in the same directory and renames it over path, so a crash
// mid-write never leaves a corrupted snapshot in place.
func Save(path string, snap store.Snapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("persistence: marshal snapshot: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("persistence: create dir %s: %w", dir, err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("persistence: write %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("persistence: rename %s to %s: %w", tmpPath, path, err)
	}

	return nil
}
