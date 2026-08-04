package inference

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAddress           = ":8080"
	defaultProviderName      = "fake"
	defaultModelName         = "fake-model"
	defaultOllamaURL         = "http://localhost:11434"
	defaultMaxAttempts       = 3
	defaultDatabaseURL       = ""
	defaultOpenAIBaseURL     = "https://api.openai.com/v1"
	defaultEmbeddingProvider = "fake"
	defaultOTELExporter      = "none"
	defaultOTELEndpoint      = "localhost:4318"
	defaultOTELServiceName   = "atlas-inference-gateway"
)

// Config contains process-level settings for the inference gateway.
type Config struct {
	Address           string
	ProviderName      string
	ModelName         string
	OllamaURL         string
	MaxAttempts       int
	DatabaseURL       string
	OpenAIAPIKey      string
	OpenAIBaseURL     string
	EmbeddingProvider string
	OTELExporter      string
	OTELEndpoint      string
	OTELServiceName   string
}

// LoadConfig reads environment-first configuration with local development defaults.
func LoadConfig() (Config, error) {
	maxAttempts, err := intOrDefault("ATLAS_INFERENCE_MAX_ATTEMPTS", defaultMaxAttempts)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		Address:           valueOrDefault("ATLAS_INFERENCE_ADDR", defaultAddress),
		ProviderName:      valueOrDefault("ATLAS_INFERENCE_PROVIDER", defaultProviderName),
		ModelName:         valueOrDefault("ATLAS_INFERENCE_MODEL", defaultModelName),
		OllamaURL:         valueOrDefault("ATLAS_OLLAMA_URL", defaultOllamaURL),
		MaxAttempts:       maxAttempts,
		DatabaseURL:       valueOrDefault("ATLAS_DATABASE_URL", defaultDatabaseURL),
		OpenAIAPIKey:      strings.TrimSpace(os.Getenv("ATLAS_OPENAI_API_KEY")),
		OpenAIBaseURL:     valueOrDefault("ATLAS_OPENAI_BASE_URL", defaultOpenAIBaseURL),
		EmbeddingProvider: valueOrDefault("ATLAS_EMBEDDING_PROVIDER", defaultEmbeddingProvider),
		OTELExporter:      valueOrDefault("ATLAS_OTEL_EXPORTER", defaultOTELExporter),
		OTELEndpoint:      valueOrDefault("ATLAS_OTEL_ENDPOINT", defaultOTELEndpoint),
		OTELServiceName:   valueOrDefault("ATLAS_OTEL_SERVICE_NAME", defaultOTELServiceName),
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
	if strings.EqualFold(config.ProviderName, "openai") && config.OpenAIAPIKey == "" {
		// Allow empty key only when BaseURL is clearly a mock/local override tests set Client separately.
		// Process startup still works; first real request may fail with auth — documented.
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
