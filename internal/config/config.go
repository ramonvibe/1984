package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ramon/trackline/internal/updatecheck"
)

type Config struct {
	Address             string
	BaseURL             string
	DatabaseURL         string
	SecureCookies       bool
	GitHubAppID         int64
	GitHubAppSlug       string
	GitHubPrivateKey    string
	GitHubWebhookSecret string
	GitHubAPIURL        string
	UpdateCheckURL      string
	AppVersion          string
	ShutdownTimeout     time.Duration
}

func Load() (Config, error) {
	config := Config{
		Address:             env("ADDR", ":8080"),
		BaseURL:             strings.TrimRight(env("BASE_URL", "http://localhost:8080"), "/"),
		DatabaseURL:         env("DATABASE_URL", "postgres://trackline:trackline@localhost:5432/trackline?sslmode=disable"),
		GitHubAppSlug:       os.Getenv("GITHUB_APP_SLUG"),
		GitHubPrivateKey:    strings.ReplaceAll(os.Getenv("GITHUB_PRIVATE_KEY"), `\n`, "\n"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		GitHubAPIURL:        strings.TrimRight(env("GITHUB_API_URL", "https://api.github.com"), "/"),
		UpdateCheckURL:      env("UPDATE_CHECK_URL", "https://api.github.com/repos/ramonvibe/1984/releases/latest"),
		AppVersion:          env("APP_VERSION", updatecheck.CurrentVersion),
		ShutdownTimeout:     10 * time.Second,
	}
	config.SecureCookies = strings.HasPrefix(config.BaseURL, "https://")
	if value := os.Getenv("SECURE_COOKIES"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("SECURE_COOKIES: %w", err)
		}
		config.SecureCookies = parsed
	}
	if value := os.Getenv("GITHUB_APP_ID"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("GITHUB_APP_ID: %w", err)
		}
		config.GitHubAppID = parsed
	}
	return config, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
