package orchestrator

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ContextRequest is input for assembling a bounded model context.
type ContextRequest struct {
	// System is always included when non-empty (highest priority after hard budget checks).
	System string
	// History is oldest-first conversation or stub messages.
	History []Message
	// MemoryNotes are optional facts (treated like lower priority than recent history).
	MemoryNotes []string
	// DocSnippets are optional retrieved docs (lowest priority).
	DocSnippets []string
	// MaxTokens is a simple budget: we estimate ~4 chars per token.
	MaxTokens int
}

// AssembledContext is the budgeted output for /v1/generate.
type AssembledContext struct {
	Messages       []Message `json:"messages"`
	EstimatedTokens int       `json:"estimated_tokens"`
	Dropped        int       `json:"dropped"`
	// Inclusion is a short human-readable summary for debugging.
	Inclusion string `json:"inclusion,omitempty"`
}

// ContextBuilder ranks and trims material under a token budget.
//
// Rules (documented + tested):
//  1. Estimate tokens as ceil(utf8_runes/4), minimum 1 for non-empty strings.
//  2. Always try to keep System if it fits alone; if system alone exceeds budget, return error.
//  3. Prefer newest History turns (walk from end); keep pairs when possible.
//  4. Then memory notes (newest/last first), then docs (last first).
//  5. Never exceed MaxTokens estimated total.
type ContextBuilder struct {
	// DefaultMaxTokens used when request.MaxTokens <= 0.
	DefaultMaxTokens int
}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{DefaultMaxTokens: 256}
}

// Build returns messages suitable for the gateway under budget.
func (b *ContextBuilder) Build(req ContextRequest) (AssembledContext, error) {
	budget := req.MaxTokens
	if budget <= 0 {
		if b != nil && b.DefaultMaxTokens > 0 {
			budget = b.DefaultMaxTokens
		} else {
			budget = 256
		}
	}

	var out []Message
	used := 0
	dropped := 0

	if s := strings.TrimSpace(req.System); s != "" {
		t := estimateTokens(s)
		if t > budget {
			return AssembledContext{}, fmt.Errorf("system prompt alone exceeds token budget (%d > %d)", t, budget)
		}
		out = append(out, Message{Role: "system", Content: s})
		used += t
	}

	// History newest-first selection, then reverse to chronological for the model.
	type scored struct {
		msg Message
		tok int
	}
	var keptHist []scored
	for i := len(req.History) - 1; i >= 0; i-- {
		m := req.History[i]
		if strings.TrimSpace(m.Content) == "" {
			dropped++
			continue
		}
		t := estimateTokens(m.Content) + 1 // role overhead
		if used+t > budget {
			dropped++
			continue
		}
		keptHist = append(keptHist, scored{msg: m, tok: t})
		used += t
	}
	// reverse keptHist (was newest-first) to chronological
	for i := len(keptHist) - 1; i >= 0; i-- {
		out = append(out, keptHist[i].msg)
	}

	appendNotes := func(role string, notes []string) {
		for i := len(notes) - 1; i >= 0; i-- {
			n := strings.TrimSpace(notes[i])
			if n == "" {
				dropped++
				continue
			}
			content := n
			if role == "system" {
				content = "memory: " + n
			}
			if role == "tool" {
				content = "doc: " + n
			}
			t := estimateTokens(content) + 1
			if used+t > budget {
				dropped++
				continue
			}
			// memory as system-ish user notes for simplicity
			r := "user"
			if strings.HasPrefix(content, "doc:") {
				r = "user"
			}
			out = append(out, Message{Role: r, Content: content})
			used += t
		}
	}
	appendNotes("memory", req.MemoryNotes)
	appendNotes("doc", req.DocSnippets)

	if len(out) == 0 {
		return AssembledContext{}, fmt.Errorf("no messages fit within token budget %d", budget)
	}

	return AssembledContext{
		Messages:        out,
		EstimatedTokens: used,
		Dropped:         dropped,
		Inclusion:       fmt.Sprintf("messages=%d tokens≈%d budget=%d dropped=%d", len(out), used, budget, dropped),
	}, nil
}

func estimateTokens(s string) int {
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return 0
	}
	// ~4 chars per token heuristic
	t := (n + 3) / 4
	if t < 1 {
		return 1
	}
	return t
}
