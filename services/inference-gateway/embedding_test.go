package inference

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFakeEmbeddingDeterministicShape(t *testing.T) {
	p := FakeEmbeddingProvider{ProviderName: "fake", Dimensions: 8}
	a, err := p.Embed(t.Context(), EmbeddingRequest{Inputs: []string{"hello", "world"}, Model: "fake-embed"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Dimensions != 8 || len(a.Vectors) != 2 {
		t.Fatalf("resp = %+v", a)
	}
	if len(a.Vectors[0]) != 8 || len(a.Vectors[1]) != 8 {
		t.Fatalf("vector lens %d %d", len(a.Vectors[0]), len(a.Vectors[1]))
	}
	b, err := p.Embed(t.Context(), EmbeddingRequest{Inputs: []string{"hello"}})
	if err != nil {
		t.Fatal(err)
	}
	for i := range a.Vectors[0] {
		if a.Vectors[0][i] != b.Vectors[0][i] {
			t.Fatalf("not deterministic at %d", i)
		}
	}
}

func TestGatewayEmbeddingsHandler(t *testing.T) {
	gateway := NewGateway(FakeProvider{ProviderName: "fake"}).WithEmbeddings(FakeEmbeddingProvider{Dimensions: 8})
	server := httptest.NewServer(gateway)
	defer server.Close()

	body := `{"inputs":["atlas","rocks"],"model":"fake-embed"}`
	resp, err := server.Client().Post(server.URL+"/v1/embeddings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var out EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Vectors) != 2 || out.Dimensions != 8 {
		t.Fatalf("out = %+v", out)
	}
	if len(out.Vectors[0]) == 0 {
		t.Fatal("empty vector")
	}
	if out.Provider != "fake" {
		t.Fatalf("provider = %q", out.Provider)
	}
}

func TestGatewayEmbeddingsRejectsEmptyInputs(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()
	resp, err := server.Client().Post(server.URL+"/v1/embeddings", "application/json", strings.NewReader(`{"inputs":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
