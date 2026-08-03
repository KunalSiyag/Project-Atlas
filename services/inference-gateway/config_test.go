package inference

import "testing"

func TestLoadConfigUsesLocalDefaults(t *testing.T) {
	t.Setenv("ATLAS_INFERENCE_ADDR", "")
	t.Setenv("ATLAS_INFERENCE_PROVIDER", "")

	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != ":8080" {
		t.Fatalf("address = %q, want :8080", config.Address)
	}
	if config.ProviderName != "fake" {
		t.Fatalf("provider = %q, want fake", config.ProviderName)
	}
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	t.Setenv("ATLAS_INFERENCE_ADDR", "127.0.0.1:9090")
	t.Setenv("ATLAS_INFERENCE_PROVIDER", "local-test")

	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Address != "127.0.0.1:9090" || config.ProviderName != "local-test" {
		t.Fatalf("config = %+v, want configured values", config)
	}
}

func TestLoadConfigRejectsWhitespaceOnlyValues(t *testing.T) {
	t.Setenv("ATLAS_INFERENCE_ADDR", "   ")
	t.Setenv("ATLAS_INFERENCE_PROVIDER", "   ")

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v, want defaults for blank values", err)
	}
}
