package orchestrator

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPostgresStoreRestartReload(t *testing.T) {
	url := strings.TrimSpace(os.Getenv("ATLAS_ORCHESTRATOR_DATABASE_URL"))
	if url == "" {
		url = strings.TrimSpace(os.Getenv("ATLAS_DATABASE_URL"))
	}
	if url == "" {
		// default local compose credentials if postgres is up
		url = "postgres://atlas:atlas-dev-password@127.0.0.1:5432/atlas?sslmode=disable"
	}
	ctx := context.Background()
	store, err := OpenPostgresStore(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer store.Close()

	// unique id per test run
	id := "pg-test-" + newID()
	run := Run{
		RunID:     id,
		Goal:      "persist me",
		Status:    StatusSucceeded,
		Result:    "fake response to: persist me",
		Steps:     DefaultPlan(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	for i := range run.Steps {
		run.Steps[i].Status = StepSucceeded
		run.Steps[i].Output = "ok"
	}
	if err := store.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendEvent(ctx, Event{EventID: newID(), RunID: id, Type: "test", Message: "saved"}); err != nil {
		t.Fatal(err)
	}

	// New store connection = restart simulation
	store2, err := OpenPostgresStore(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	got, err := store2.GetRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Goal != "persist me" || got.Status != StatusSucceeded {
		t.Fatalf("%+v", got)
	}
	if got.Result != "fake response to: persist me" {
		t.Fatalf("result=%q", got.Result)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("steps=%d", len(got.Steps))
	}
	events, err := store2.ListEvents(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 1 {
		t.Fatal("expected events")
	}
}

func TestPostgresStoreThroughServiceExecute(t *testing.T) {
	url := "postgres://atlas:atlas-dev-password@127.0.0.1:5432/atlas?sslmode=disable"
	ctx := context.Background()
	store, err := OpenPostgresStore(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer store.Close()

	// httptest gateway
	// use net/http/httptest in this file — import via service_test patterns
	// inline minimal client
	gw := &staticGateway{content: "fake response to: pg-service"}
	id := "pg-svc-" + newID()[:8]
	svc := NewService(ServiceConfig{
		Store:   store,
		Gateway: gw,
		IDFunc:  func() string { return id },
	})
	created, err := svc.CreateRun(ctx, "pg-service")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.StartRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("%s %s", run.Status, run.Error)
	}

	store2, err := OpenPostgresStore(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	got, err := store2.GetRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusSucceeded || !strings.Contains(got.Result, "fake response to:") {
		t.Fatalf("%+v", got)
	}
}

type staticGateway struct{ content string }

func (s *staticGateway) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	var r GenerateResponse
	r.Message.Role = "assistant"
	r.Message.Content = s.content
	r.Provider = "fake"
	r.Model = "fake-model"
	return r, nil
}
