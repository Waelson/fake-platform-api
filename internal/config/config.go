package config

import "os"

type Config struct {
	Port        string
	AuthEnabled bool
	Token       string
	Environment string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("DEVEX_FAKE_PORT", "8080"),
		AuthEnabled: getEnv("DEVEX_FAKE_AUTH_ENABLED", "false") == "true",
		Token:       getEnv("DEVEX_FAKE_TOKEN", "dev-token"),
		Environment: getEnv("DEVEX_FAKE_ENVIRONMENT", "dev"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
