// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the Lore API.
type Config struct {
	Port        string
	Env         string
	DatabaseURL string
	AdminToken  string
	GithubToken string
}

// Load reads configuration from the environment. It best-effort loads a local
// .env file (ignored if absent) so that local development is zero-setup.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("APP_ENV", "development"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
		GithubToken: os.Getenv("GITHUB_TOKEN"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
