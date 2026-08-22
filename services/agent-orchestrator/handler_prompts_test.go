package orchestrator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPromptAdminHTTPMultiTenant(t *testing.T) {
	store := NewMemoryPromptStore()
	svc := NewService(ServiceConfig{
		Store:     NewMemoryStore(),
		Prompts:   store,
		Retriever: NewInProcessRetriever(FakeEmbedder{Dims: 8}),
	})
	h := &Handler{Service: svc}

	// Publish in workspace A
	body := `{"id":"sys.greet","version":"1","status":"active","text":"hello from A"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/prompts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", "ws-a")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("publish A status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Publish same id in workspace B with different text
	bodyB := `{"id":"sys.greet","version":"1","status":"active","text":"hello from B"}`
	reqB := httptest.NewRequest(http.MethodPost, "/v1/prompts", bytes.NewBufferString(bodyB))
	reqB.Header.Set("Content-Type", "application/json")
	reqB.Header.Set("X-Workspace-ID", "ws-b")
	rrB := httptest.NewRecorder()
	h.ServeHTTP(rrB, reqB)
	if rrB.Code != http.StatusCreated {
		t.Fatalf("publish B status=%d body=%s", rrB.Code, rrB.Body.String())
	}

	// List A — only A's prompt
	reqList := httptest.NewRequest(http.MethodGet, "/v1/prompts", nil)
	reqList.Header.Set("X-Workspace-ID", "ws-a")
	rrList := httptest.NewRecorder()
	h.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("list A status=%d", rrList.Code)
	}
	var listA struct {
		WorkspaceID string        `json:"workspace_id"`
		Prompts     []PromptEntry `json:"prompts"`
	}
	if err := json.Unmarshal(rrList.Body.Bytes(), &listA); err != nil {
		t.Fatal(err)
	}
	if listA.WorkspaceID != "ws-a" || len(listA.Prompts) != 1 || listA.Prompts[0].Text != "hello from A" {
		t.Fatalf("list A: %+v", listA)
	}

	// Get A by id+version
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/prompts/sys.greet?version=1", nil)
	reqGet.Header.Set("X-Workspace-ID", "ws-a")
	rrGet := httptest.NewRecorder()
	h.ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("get A status=%d body=%s", rrGet.Code, rrGet.Body.String())
	}
	var got PromptEntry
	if err := json.Unmarshal(rrGet.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Text != "hello from A" || got.WorkspaceID != "ws-a" {
		t.Fatalf("get A: %+v", got)
	}

	// Cross-workspace: B cannot get A's content via list (isolated)
	reqListB := httptest.NewRequest(http.MethodGet, "/v1/prompts", nil)
	reqListB.Header.Set("X-Workspace-ID", "ws-b")
	rrListB := httptest.NewRecorder()
	h.ServeHTTP(rrListB, reqListB)
	var listB struct {
		Prompts []PromptEntry `json:"prompts"`
	}
	_ = json.Unmarshal(rrListB.Body.Bytes(), &listB)
	if len(listB.Prompts) != 1 || listB.Prompts[0].Text != "hello from B" {
		t.Fatalf("list B should only see B: %+v", listB)
	}
	// B get of A's text must not return A
	if listB.Prompts[0].Text == "hello from A" {
		t.Fatal("cross-workspace leak")
	}

	// Missing workspace header denied
	reqNo := httptest.NewRequest(http.MethodGet, "/v1/prompts", nil)
	rrNo := httptest.NewRecorder()
	h.ServeHTTP(rrNo, reqNo)
	if rrNo.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without workspace, got %d", rrNo.Code)
	}
}

func TestPluginCatalogHTTP(t *testing.T) {
	svc := NewService(ServiceConfig{Store: NewMemoryStore()})
	h := &Handler{Service: svc}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plugin-ok"))
	}))
	defer srv.Close()

	body := `{"name":"catalog_remote","kind":"http","url":"` + srv.URL + `","input_schema":{"required":["payload"],"properties":{"payload":{"type":"string"}}}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register plugin %d %s", rr.Code, rr.Body.String())
	}
	reqL := httptest.NewRequest(http.MethodGet, "/v1/plugins", nil)
	rrL := httptest.NewRecorder()
	h.ServeHTTP(rrL, reqL)
	if rrL.Code != http.StatusOK {
		t.Fatal(rrL.Body.String())
	}
	var out struct {
		Plugins []ToolDescriptor `json:"plugins"`
	}
	if err := json.Unmarshal(rrL.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Plugins) != 1 || out.Plugins[0].Name != "catalog_remote" {
		t.Fatalf("%+v", out)
	}
	// Invoke via registry path
	res, err := svc.Tools.Invoke(req.Context(), "catalog_remote", map[string]any{"payload": "x"}, InvokeOptions{})
	if err != nil || res != "plugin-ok" {
		t.Fatalf("%q %v", res, err)
	}
}
