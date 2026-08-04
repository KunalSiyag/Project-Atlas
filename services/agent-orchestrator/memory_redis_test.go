package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisShortMemoryPutListRoundTrip(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	long := NewInMemoryMemory()
	mem := NewRedisShortMemory(client, long)
	mem.TTL = time.Hour

	ctx := context.Background()
	runID := "run-redis-1"
	if err := mem.PutShort(ctx, runID, "SHORT_REDIS_PAYLOAD_ALPHA"); err != nil {
		t.Fatal(err)
	}
	if err := mem.PutShort(ctx, runID, "SHORT_REDIS_PAYLOAD_BETA"); err != nil {
		t.Fatal(err)
	}

	listed, err := mem.ListShort(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("len=%d listed=%+v", len(listed), listed)
	}
	joined := ""
	for _, f := range listed {
		joined += f.Content + " "
		if f.RunID != runID {
			t.Fatalf("run_id=%q", f.RunID)
		}
		if f.Scope != "short" {
			t.Fatalf("scope=%q", f.Scope)
		}
	}
	if !strings.Contains(joined, "SHORT_REDIS_PAYLOAD_ALPHA") || !strings.Contains(joined, "SHORT_REDIS_PAYLOAD_BETA") {
		t.Fatalf("listed content missing payloads: %q", joined)
	}

	other, err := mem.ListShort(ctx, "other-run")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("leak into other run: %+v", other)
	}

	if err := mem.PutLong(ctx, MemoryFact{WorkspaceID: "ws", Content: "LONG_VIA_REDIS_WRAPPER"}); err != nil {
		t.Fatal(err)
	}
	longs, err := mem.ListLong(ctx, "ws", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(longs) == 0 || longs[0].Content != "LONG_VIA_REDIS_WRAPPER" {
		t.Fatalf("long=%+v", longs)
	}
}

func TestRedisShortMemoryRequiresFields(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	mem := NewRedisShortMemory(client, nil)
	if err := mem.PutShort(context.Background(), "", "x"); err == nil {
		t.Fatal("expected error for empty run_id")
	}
}
