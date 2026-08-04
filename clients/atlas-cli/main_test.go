package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/generate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Request-ID", "req-1")
		w.Header().Set("X-Run-ID", "run-1")
		_ = json.NewEncoder(w).Encode(ChatResponse{
			Message:  Message{Role: "assistant", Content: "fake response to: hi"},
			Provider: "fake",
			Model:    "fake-model",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Generate(t.Context(), GenerateRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Model:    "fake-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Message.Content, "hi") {
		t.Fatalf("unexpected content: %q", result.Message.Content)
	}
	if result.RunID != "run-1" {
		t.Fatalf("run id: %q", result.RunID)
	}
}

func TestClientHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	body, err := client.Health(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "ok") {
		t.Fatalf("body: %s", body)
	}
}

func TestShortID(t *testing.T) {
	if shortID("") != "—" {
		t.Fatal(shortID(""))
	}
	if shortID("abcdefghijklmnop") != "abcdefgh…" {
		t.Fatal(shortID("abcdefghijklmnop"))
	}
}
