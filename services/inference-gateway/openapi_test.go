package inference

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIDocumentDescribesImplementedRoutes(t *testing.T) {
	doc := string(OpenAPIDocument())
	if !strings.HasPrefix(strings.TrimSpace(doc), "openapi:") {
		t.Fatalf("expected OpenAPI YAML, got prefix %q", doc[:min(40, len(doc))])
	}
	required := []string{
		"/healthz",
		"/readyz",
		"/v1/generate",
		"/v1/runs/{run_id}",
		"/v1/conversations",
		"/v1/conversations/{conversation_id}/messages",
		"ChatRequest",
		"ChatResponse",
	}
	for _, needle := range required {
		if !strings.Contains(doc, needle) {
			t.Fatalf("openapi.yaml missing %q", needle)
		}
	}
	if !strings.Contains(doc, "/v1/embeddings") {
		t.Fatal("openapi should list implemented /v1/embeddings path")
	}
	if !strings.Contains(doc, "stream:") && !strings.Contains(doc, "stream\n") && !strings.Contains(doc, "StreamDelta") {
		t.Fatal("openapi should document stream generate / StreamDelta")
	}
	if !strings.Contains(doc, "text/event-stream") {
		t.Fatal("openapi should mention text/event-stream for streaming generate")
	}
}

func TestGatewayServesOpenAPI(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Atlas Inference Gateway") {
		t.Fatalf("body missing title: %s", string(body)[:min(200, len(body))])
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "yaml") {
		t.Fatalf("content-type = %q", resp.Header.Get("Content-Type"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
