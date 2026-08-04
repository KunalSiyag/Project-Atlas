package inference

import (
	"fmt"
	"strings"
)

// NewConfiguredProvider creates the adapter selected by configuration.
func NewConfiguredProvider(config Config) (ModelProvider, error) {
	switch strings.ToLower(strings.TrimSpace(config.ProviderName)) {
	case "fake":
		return FakeProvider{ProviderName: "fake"}, nil
	case "ollama":
		return OllamaProvider{BaseURL: config.OllamaURL}, nil
	case "openai":
		return OpenAIProvider{
			BaseURL:      config.OpenAIBaseURL,
			APIKey:       config.OpenAIAPIKey,
			ProviderName: "openai",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported inference provider %q", config.ProviderName)
	}
}
