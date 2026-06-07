package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	AuthEnabled bool
	Token       string
	Environment string

	StateFile                string
	StateSaveIntervalSeconds int
}

func Load() *Config {
	return &Config{
		Port:        getEnv("DEVEX_FAKE_PORT", "8080"),
		AuthEnabled: getEnv("DEVEX_FAKE_AUTH_ENABLED", "false") == "true",
		Token:       getEnv("DEVEX_FAKE_TOKEN", "dev-token"),
		Environment: getEnv("DEVEX_FAKE_ENVIRONMENT", "dev"),

		StateFile:                getEnv("DEVEX_FAKE_STATE_FILE", ""),
		StateSaveIntervalSeconds: getEnvInt("DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS", 2),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
