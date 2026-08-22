package inference

import "testing"

func TestStaticRouterUsesTaskSpecificRoute(t *testing.T) {
	planning := FakeProvider{ProviderName: "planning"}
	acting := FakeProvider{ProviderName: "acting"}
	router := StaticRouter{
		Default: Route{Provider: acting, DefaultModel: "act-model"},
		Routes: map[string]Route{
			"plan": {Provider: planning, DefaultModel: "plan-model"},
		},
	}

	provider, model, err := router.Resolve("plan", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if provider != planning || model != "plan-model" {
		t.Fatalf("route = (%T, %q), want planning/plan-model", provider, model)
	}
}

func TestStaticRouterAllowsRequestedModelOverride(t *testing.T) {
	provider, model, err := (StaticRouter{
		Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "default-model"},
	}).Resolve("act", "", "", "requested-model")
	if err != nil {
		t.Fatal(err)
	}
	if model != "requested-model" {
		t.Fatalf("model = %q, want requested-model", model)
	}
	if provider == nil {
		t.Fatal("provider = nil, want provider")
	}
}

func TestStaticRouterRejectsMissingProvider(t *testing.T) {
	if _, _, err := (StaticRouter{}).Resolve("plan", "", "", ""); err == nil {
		t.Fatal("Resolve() error = nil, want missing-provider error")
	}
}

func TestDynamicRouterNvidiaOverride(t *testing.T) {
	router := DynamicRouter{
		Config: Config{
			OpenAIBaseURL: "https://integrate.api.nvidia.com/v1",
			OpenAIAPIKey:  "test-key",
			MaxAttempts:   1,
		},
	}
	provider, model, err := router.Resolve("act", "nvidia", "custom-api-key", "meta/llama-3.1-70b-instruct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "meta/llama-3.1-70b-instruct" {
		t.Fatalf("model = %q, want meta/llama-3.1-70b-instruct", model)
	}
	if provider == nil {
		t.Fatal("expected provider instance")
	}
}
