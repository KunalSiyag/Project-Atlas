package orchestrator

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RunStore persists orchestration runs.
type RunStore interface {
	SaveRun(ctx context.Context, run Run) error
	GetRun(ctx context.Context, runID string) (Run, error)
	// AppendEvent is optional; memory store keeps events in process for tests.
	AppendEvent(ctx context.Context, event Event) error
	ListEvents(ctx context.Context, runID string) ([]Event, error)
}

// InMemoryRunStore is the in-process fallback when no database URL is configured.
type InMemoryRunStore struct {
	mu     sync.RWMutex
	runs   map[string]Run
	events map[string][]Event
}

func NewMemoryStore() *InMemoryRunStore {
	return &InMemoryRunStore{
		runs:   make(map[string]Run),
		events: make(map[string][]Event),
	}
}

func (m *InMemoryRunStore) SaveRun(_ context.Context, run Run) error {
	if m == nil {
		return errors.New("memory store nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// deep-ish copy steps slice
	cp := run
	cp.Steps = append([]Step(nil), run.Steps...)
	for i := range cp.Steps {
		cp.Steps[i].DependsOn = append([]string(nil), run.Steps[i].DependsOn...)
		if run.Steps[i].ToolInput != nil {
			cp.Steps[i].ToolInput = map[string]any{}
			for k, v := range run.Steps[i].ToolInput {
				cp.Steps[i].ToolInput[k] = v
			}
		}
	}
	m.runs[run.RunID] = cp
	return nil
}

func (m *InMemoryRunStore) GetRun(_ context.Context, runID string) (Run, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	run, ok := m.runs[runID]
	if !ok {
		return Run{}, errors.New("run not found")
	}
	cp := run
	cp.Steps = append([]Step(nil), run.Steps...)
	return cp, nil
}

func (m *InMemoryRunStore) AppendEvent(_ context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	m.events[event.RunID] = append(m.events[event.RunID], event)
	return nil
}

func (m *InMemoryRunStore) ListEvents(_ context.Context, runID string) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ev := m.events[runID]
	out := make([]Event, len(ev))
	copy(out, ev)
	return out, nil
}
