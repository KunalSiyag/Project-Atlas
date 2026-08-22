package inference

import (
	"errors"
	"strings"
	"time"
)

// Route binds one logical task class to a provider and its default model.
type Route struct {
	Provider     ModelProvider
	DefaultModel string
}

// StaticRouter is a deterministic in-process router. It is intentionally
// simple now; a later implementation can load routes from configuration or a
// model registry without changing the gateway contract.
type StaticRouter struct {
	Default Route
	Routes  map[string]Route
}


func (r StaticRouter) Resolve(taskClass, requestedProvider, apiKey, requestedModel string) (ModelProvider, string, error) {
	if taskClass == "" {
		taskClass = "act"
	}

	route, ok := r.Routes[taskClass]
	if !ok {
		route = r.Default
	}
	if route.Provider == nil {
		return nil, "", errors.New("no provider configured for task class " + taskClass)
	}

	model := requestedModel
	if model == "" {
		model = route.DefaultModel
	}
	return route.Provider, model, nil
}
// DynamicRouter dynamically resolves providers and models.
// It supports "auto" mode (choosing based on availability and task class),
// explicit provider requests ("nvidia", "openai", "ollama", "fake"), and static fallback routes.
type DynamicRouter struct {
	Config   Config
	Default  Route
	Routes   map[string]Route
	Fallback ModelProvider
}

func (r DynamicRouter) Resolve(taskClass, requestedProvider, apiKey, requestedModel string) (ModelProvider, string, error) {
	reqProvider := strings.ToLower(strings.TrimSpace(requestedProvider))
	reqModel := strings.TrimSpace(requestedModel)
	reqAPIKey := strings.TrimSpace(apiKey)

	// Handle "auto" mode or empty provider request
	if reqProvider == "" || reqProvider == "auto" {
		if reqAPIKey != "" || r.Config.OpenAIAPIKey != "" {
			reqProvider = "openai"
		} else {
			reqProvider = strings.ToLower(strings.TrimSpace(r.Config.ProviderName))
		}
	}

	// Direct provider construction or route lookup
	switch reqProvider {
	case "nvidia":
		// NVIDIA NIM OpenAI-compatible endpoint
		baseURL := r.Config.OpenAIBaseURL
		if baseURL == defaultOpenAIBaseURL || baseURL == "" {
			baseURL = "https://integrate.api.nvidia.com/v1"
		}
		model := reqModel
		if model == "" {
			model = "meta/llama-3.1-70b-instruct"
		}
		key := reqAPIKey
		if key == "" {
			key = r.Config.OpenAIAPIKey
		}
		provider := OpenAIProvider{
			BaseURL:      baseURL,
			APIKey:       key,
			ProviderName: "nvidia",
		}
		return RetryingProvider{Provider: provider, Policy: RetryPolicy{MaxAttempts: r.Config.MaxAttempts, Backoff: 100 * time.Millisecond}}, model, nil

	case "openai":
		model := reqModel
		if model == "" {
			model = "gpt-4o"
		}
		key := reqAPIKey
		if key == "" {
			key = r.Config.OpenAIAPIKey
		}
		provider := OpenAIProvider{
			BaseURL:      r.Config.OpenAIBaseURL,
			APIKey:       key,
			ProviderName: "openai",
		}
		return RetryingProvider{Provider: provider, Policy: RetryPolicy{MaxAttempts: r.Config.MaxAttempts, Backoff: 100 * time.Millisecond}}, model, nil

	case "ollama":
		model := reqModel
		if model == "" {
			model = "llama3"
		}
		provider := OllamaProvider{BaseURL: r.Config.OllamaURL}
		return RetryingProvider{Provider: provider, Policy: RetryPolicy{MaxAttempts: r.Config.MaxAttempts, Backoff: 100 * time.Millisecond}}, model, nil

	case "fake":
		model := reqModel
		if model == "" {
			model = "fake-model"
		}
		return FakeProvider{ProviderName: "fake"}, model, nil

	default:
		// Fallback to static route resolution
		return StaticRouter{Default: r.Default, Routes: r.Routes}.Resolve(taskClass, reqProvider, reqAPIKey, reqModel)
	}
}
