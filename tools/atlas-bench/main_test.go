package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBenchDrivesRealHTTPAndEmitsMetrics(t *testing.T) {
	// Real HTTP server path — not a hard-coded metrics stub.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "metrics.json")
	code := run([]string{
		"-url", srv.URL,
		"-path", "/healthz",
		"-n", "20",
		"-c", "4",
		"-out", out,
	})
	if code != 0 {
		t.Fatalf("bench exit %d", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var m Metrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Requests != 20 || m.Successes != 20 {
		t.Fatalf("metrics=%+v", m)
	}
	if m.ThroughputRPS <= 0 || m.LatencyP50MS < 0 || m.LatencyP95MS < 0 {
		t.Fatalf("latency/throughput missing: %+v", m)
	}
	if m.Path != "/healthz" || m.ErrorRatio != 0 {
		t.Fatalf("%+v", m)
	}
}
