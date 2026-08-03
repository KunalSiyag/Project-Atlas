package inference

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAddress      = ":8080"
	defaultProviderName = "fake"
	defaultModelName    = "fake-model"
	defaultOllamaURL    = "http://localhost:11434"
	defaultMaxAttempts  = 3
	defaultDatabaseURL  = ""
)

// Config contains process-level settings for the inference gateway.
type Config struct {
	Address      string
	ProviderName string
	ModelName    string
	OllamaURL    string
	MaxAttempts  int
	DatabaseURL  string
}

// LoadConfig reads environment-first configuration with local development
// defaults. Secrets are intentionally not part of this first configuration
// slice because the fake provider does not need credentials.
func LoadConfig() (Config, error) {
	maxAttempts, err := intOrDefault("ATLAS_INFERENCE_MAX_ATTEMPTS", defaultMaxAttempts)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		Address:      valueOrDefault("ATLAS_INFERENCE_ADDR", defaultAddress),
		ProviderName: valueOrDefault("ATLAS_INFERENCE_PROVIDER", defaultProviderName),
		ModelName:    valueOrDefault("ATLAS_INFERENCE_MODEL", defaultModelName),
		OllamaURL:    valueOrDefault("ATLAS_OLLAMA_URL", defaultOllamaURL),
		MaxAttempts:  maxAttempts,
		DatabaseURL:  valueOrDefault("ATLAS_DATABASE_URL", defaultDatabaseURL),
	}
	if strings.TrimSpace(config.Address) == "" {
		return Config{}, errors.New("ATLAS_INFERENCE_ADDR must not be empty")
	}
	if strings.TrimSpace(config.ProviderName) == "" {
		return Config{}, errors.New("ATLAS_INFERENCE_PROVIDER must not be empty")
	}
	if strings.TrimSpace(config.ModelName) == "" {
		return Config{}, errors.New("ATLAS_INFERENCE_MODEL must not be empty")
	}
	if strings.TrimSpace(config.OllamaURL) == "" {
		return Config{}, errors.New("ATLAS_OLLAMA_URL must not be empty")
	}
	if config.MaxAttempts < 1 {
		return Config{}, errors.New("ATLAS_INFERENCE_MAX_ATTEMPTS must be at least 1")
	}
	return config, nil
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func intOrDefault(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}
	return parsed, nil
}
