package orchestrator

import (
	"strings"
	"testing"
)

func TestContextBuilderRespectsBudgetAndKeepsRecent(t *testing.T) {
	b := NewContextBuilder()
	// Build oversized history: many old turns + recent important ones.
	var history []Message
	for i := 0; i < 50; i++ {
		history = append(history, Message{Role: "user", Content: "old noise message number " + strings.Repeat("x", 40) + string(rune('A'+i%26))})
		history = append(history, Message{Role: "assistant", Content: "old reply " + strings.Repeat("y", 40)})
	}
	history = append(history, Message{Role: "user", Content: "RECENT_USER_GOAL please answer"})
	history = append(history, Message{Role: "assistant", Content: "RECENT_ASSISTANT_HINT"})

	out, err := b.Build(ContextRequest{
		System:    "sys",
		History:   history,
		MaxTokens: 80,
		MemoryNotes: []string{
			strings.Repeat("mem ", 100), // large — may drop
			"IMPORTANT_MEMORY_FACT",
		},
		DocSnippets: []string{strings.Repeat("doc ", 100), "IMPORTANT_DOC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.EstimatedTokens > 80 {
		t.Fatalf("tokens %d exceed budget 80", out.EstimatedTokens)
	}
	joined := ""
	for _, m := range out.Messages {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "RECENT_USER_GOAL") {
		t.Fatalf("expected recent user retained: %s", joined)
	}
	// Old noise should be mostly dropped
	if out.Dropped < 10 {
		t.Fatalf("expected many drops, dropped=%d", out.Dropped)
	}
}

func TestContextBuilderSystemOverflowFails(t *testing.T) {
	b := NewContextBuilder()
	_, err := b.Build(ContextRequest{
		System:    strings.Repeat("S", 400),
		MaxTokens: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
