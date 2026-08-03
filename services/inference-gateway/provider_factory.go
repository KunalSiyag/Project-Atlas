package inference

import "fmt"

// NewConfiguredProvider creates the adapter selected by configuration.
func NewConfiguredProvider(config Config) (ModelProvider, error) {
	switch config.ProviderName {
	case "fake":
		return FakeProvider{ProviderName: "fake"}, nil
	case "ollama":
		return OllamaProvider{BaseURL: config.OllamaURL}, nil
	default:
		return nil, fmt.Errorf("unsupported inference provider %q", config.ProviderName)
	}
}
