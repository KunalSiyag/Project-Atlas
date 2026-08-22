package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListEventsHTTPViaRealStorePath(t *testing.T) {
	// Drive create (which AppendEvents run_created) then GET /events through handler.
	// No hand-built event list that skips ListEvents.
	var idSeq int
	svc := NewService(ServiceConfig{
		Store: NewMemoryStore(),
		IDFunc: func() string {
			idSeq++
			return "id-" + itoa(idSeq)
		},
		// No gateway: rule planner only for simple goals still needs model for default plans —
		// use RulePlanner with a tool goal so create succeeds without gateway.
		Planner: NewRulePlanner(),
	})
	h := &Handler{Service: svc}
	server := httptest.NewServer(h)
	defer server.Close()

	// Create run → Store.AppendEvent(run_created)
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/runs", strings.NewReader(`{"goal":"echo hello from events test"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var created Run
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", resp.StatusCode)
	}
	if created.RunID == "" {
		t.Fatal("empty run_id")
	}

	// GET events via HTTP → Service.ListEvents → Store.ListEvents
	er, err := server.Client().Get(server.URL + "/v1/runs/" + created.RunID + "/events")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(er.Body)
	_ = er.Body.Close()
	if er.StatusCode != http.StatusOK {
		t.Fatalf("events status %d body %s", er.StatusCode, raw)
	}
	var events []Event
	if err := json.Unmarshal(raw, &events); err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least run_created from real AppendEvent path")
	}
	foundCreated := false
	for _, e := range events {
		if e.EventID == "" || e.RunID == "" || e.Type == "" {
			t.Fatalf("event missing required fields: %+v", e)
		}
		if e.RunID != created.RunID {
			t.Fatalf("run_id mismatch %q want %q", e.RunID, created.RunID)
		}
		if e.Type == "run_created" {
			foundCreated = true
			if !strings.Contains(e.Message, "echo hello") {
				t.Fatalf("run_created message=%q", e.Message)
			}
		}
	}
	if !foundCreated {
		t.Fatalf("missing run_created; events=%+v", events)
	}

	// Cross-check service API returns the same store path
	svcEvents, err := svc.ListEvents(context.Background(), created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(svcEvents) != len(events) {
		t.Fatalf("service ListEvents len=%d http=%d", len(svcEvents), len(events))
	}

	// Unknown run → 404
	nf, err := server.Client().Get(server.URL + "/v1/runs/does-not-exist/events")
	if err != nil {
		t.Fatal(err)
	}
	_ = nf.Body.Close()
	if nf.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 for unknown run, got %d", nf.StatusCode)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
