package orchestrator

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireWorkspaceRejectsProtectedRoutesWithoutHeader(t *testing.T) {
	inner := &Handler{Service: NewService(ServiceConfig{
		Store:   NewMemoryStore(),
		Planner: NewRulePlanner(),
	})}
	h := RequireWorkspace(inner)
	server := httptest.NewServer(h)
	defer server.Close()

	// Protected: create run without header → 401
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/runs", strings.NewReader(`{"goal":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without workspace header, got %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "X-Workspace-ID") {
		t.Fatalf("body=%s", body)
	}

	// Protected with header → not 401 (may be 201 or other app status)
	req2, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/runs", strings.NewReader(`{"goal":"echo ok"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Workspace-ID", "ws-test")
	resp2, err := server.Client().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode == http.StatusUnauthorized {
		t.Fatal("with X-Workspace-ID must not be 401")
	}
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("want 201 create with workspace header, got %d", resp2.StatusCode)
	}

	// OpenAPI is protected when require workspace is on
	oa, err := server.Client().Get(server.URL + "/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_ = oa.Body.Close()
	if oa.StatusCode != http.StatusUnauthorized {
		t.Fatalf("openapi without header want 401, got %d", oa.StatusCode)
	}
}

func TestRequireWorkspaceHealthBypass(t *testing.T) {
	inner := &Handler{Service: NewService(ServiceConfig{Store: NewMemoryStore()})}
	h := RequireWorkspace(inner)
	server := httptest.NewServer(h)
	defer server.Close()

	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s want 200 without workspace header, got %d body=%s", path, resp.StatusCode, body)
		}
	}
}
