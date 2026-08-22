package orchestrator

import (
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed events.schema.json
var eventsSchemaDoc string

func TestOpenAPIDocumentsImplementedRoutes(t *testing.T) {
	doc := string(OpenAPIDocument())
	if !strings.HasPrefix(strings.TrimSpace(doc), "openapi:") {
		t.Fatal("openapi.yaml must start with openapi:")
	}
	needles := []string{
		"/healthz",
		"/readyz",
		"/openapi.yaml",
		"/v1/runs",
		"/v1/runs/{run_id}",
		"/v1/runs/{run_id}/events",
		"/v1/runs/{run_id}/start",
		"/v1/runs/{run_id}/approve",
		"/v1/runs/{run_id}/reject",
		"/v1/runs/{run_id}/cancel",
		"/v1/knowledge/ingest",
		"/v1/memory",
		"waiting_approval",
		"createRun",
		"startRun",
		"listRunEvents",
	}
	for _, n := range needles {
		if !strings.Contains(doc, n) {
			t.Fatalf("openapi.yaml missing %q", n)
		}
	}
	if strings.Contains(doc, "execution_id") && !strings.Contains(doc, "run_id") {
		t.Fatal("openapi should document run_id for implemented runs")
	}
}

func TestEventsSchemaListsLifecycleTypes(t *testing.T) {
	doc := eventsSchemaDoc
	if len(strings.TrimSpace(doc)) < 50 {
		t.Fatal("events.schema.json is empty or stub")
	}
	for _, typ := range []string{
		"run_created", "run_started", "step_started", "step_succeeded", "step_retry",
		"waiting_approval", "approved", "rejected",
		"run_succeeded", "run_failed", "run_cancelled",
	} {
		if !strings.Contains(doc, typ) {
			t.Fatalf("events.schema.json missing event type %q", typ)
		}
	}
}

func TestBuiltinsImplementToolInterface(t *testing.T) {
	reg := DefaultToolRegistry()
	for _, name := range []string{"echo", "calculator", "http_get", "danger_echo"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("missing built-in %s", name)
		}
		var _ Tool = tool
		if tool.Name() != name {
			t.Fatalf("Name()=%q want %q", tool.Name(), name)
		}
		if tool.InputSchema() == nil {
			t.Fatalf("%s missing InputSchema", name)
		}
	}
}

func TestHandlerServesOpenAPI(t *testing.T) {
	h := &Handler{Service: NewService(ServiceConfig{Store: NewMemoryStore()})}
	server := httptest.NewServer(h)
	defer server.Close()
	for _, path := range []string{"/openapi.yaml", "/openapi.yml", "/v1/openapi.yaml"} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status %d", path, resp.StatusCode)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "yaml") {
			t.Fatalf("%s Content-Type=%q want yaml", path, ct)
		}
		if !strings.HasPrefix(strings.TrimSpace(string(body)), "openapi:") {
			t.Fatalf("%s body prefix not openapi:", path)
		}
		if !strings.Contains(string(body), "/v1/runs/{run_id}/events") {
			t.Fatalf("%s missing list-events path", path)
		}
	}
}
