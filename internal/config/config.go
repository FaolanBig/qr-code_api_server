package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address          string
	APIKey           string
	FallbackLogoPath string
	LogoMaxBytes     int
	OpenAPIPath      string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	ShutdownTimeout  time.Duration
}

func Load() (Config, error) {
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		return Config{}, fmt.Errorf("API_KEY is required")
	}
	if len(apiKey) < 16 {
		return Config{}, fmt.Errorf("API_KEY must be at least 16 characters")
	}

	readTimeout, err := durationFromEnv("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := durationFromEnv("WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := durationFromEnv("IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := durationFromEnv("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	address := strings.TrimSpace(os.Getenv("LISTEN_ADDRESS"))
	if address == "" {
		address = ":8080"
	}

	fallbackLogoPath := strings.TrimSpace(os.Getenv("FALLBACK_LOGO_PATH"))
	if fallbackLogoPath == "" {
		fallbackLogoPath = "assets/fallback-logo.png"
	}

	logoMaxBytes, err := intFromEnv("LOGO_MAX_BYTES", 1024*1024)
	if err != nil {
		return Config{}, err
	}
	if logoMaxBytes < 1024 || logoMaxBytes > 10*1024*1024 {
		return Config{}, fmt.Errorf("LOGO_MAX_BYTES must be between 1024 and 10485760")
	}

	openAPIPath := strings.TrimSpace(os.Getenv("OPENAPI_PATH"))
	if openAPIPath == "" {
		openAPIPath = "docs/openapi.yaml"
	}

	return Config{
		Address: address, APIKey: apiKey,
		FallbackLogoPath: fallbackLogoPath, LogoMaxBytes: logoMaxBytes, OpenAPIPath: openAPIPath,
		ReadTimeout: readTimeout, WriteTimeout: writeTimeout,
		IdleTimeout: idleTimeout, ShutdownTimeout: shutdownTimeout,
	}, nil
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return parsed, nil
}

func intFromEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}
