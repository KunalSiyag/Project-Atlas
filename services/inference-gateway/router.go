package inference

import (
	"errors"
	"fmt"
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

func (r StaticRouter) Resolve(taskClass, requestedModel string) (ModelProvider, string, error) {
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
	if model == "" {
		return nil, "", fmt.Errorf("no model configured for task class %s", taskClass)
	}
	return route.Provider, model, nil
}
