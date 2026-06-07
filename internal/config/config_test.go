package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	clearEnv()

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "8080")
	}
	if cfg.AuthEnabled != false {
		t.Error("AuthEnabled: got true, want false")
	}
	if cfg.Token != "dev-token" {
		t.Errorf("Token: got %q, want %q", cfg.Token, "dev-token")
	}
	if cfg.Environment != "dev" {
		t.Errorf("Environment: got %q, want %q", cfg.Environment, "dev")
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("DEVEX_FAKE_PORT", "9090")
	os.Setenv("DEVEX_FAKE_AUTH_ENABLED", "true")
	os.Setenv("DEVEX_FAKE_TOKEN", "secret-token")
	os.Setenv("DEVEX_FAKE_ENVIRONMENT", "stage")
	t.Cleanup(clearEnv)

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "9090")
	}
	if !cfg.AuthEnabled {
		t.Error("AuthEnabled: got false, want true")
	}
	if cfg.Token != "secret-token" {
		t.Errorf("Token: got %q, want %q", cfg.Token, "secret-token")
	}
	if cfg.Environment != "stage" {
		t.Errorf("Environment: got %q, want %q", cfg.Environment, "stage")
	}
}

func clearEnv() {
	os.Unsetenv("DEVEX_FAKE_PORT")
	os.Unsetenv("DEVEX_FAKE_AUTH_ENABLED")
	os.Unsetenv("DEVEX_FAKE_TOKEN")
	os.Unsetenv("DEVEX_FAKE_ENVIRONMENT")
}
