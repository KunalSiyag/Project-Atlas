package orchestrator

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Service is the application API for the agent orchestrator.
type Service struct {
	Store     RunStore
	Executor  *Executor
	Planner   Planner
	Memory    MemoryStore
	Retriever Retriever
	// Prompts is the multi-tenant durable prompt store (admin HTTP).
	Prompts PromptStore
	// PromptReg optional in-memory registry for resolve/hot-reload.
	PromptReg *PromptRegistry
	// Plugins is the remote/plugin tool catalog (marketplace discovery).
	Plugins *PluginCatalog
	// Tools registry (shared with executor).
	Tools *ToolRegistry
	idFunc func() string
	now    func() time.Time
}

// ServiceConfig wires dependencies.
type ServiceConfig struct {
	Store       RunStore
	Gateway     InferenceClient
	Builder     *ContextBuilder
	Tools       *ToolRegistry
	Memory      MemoryStore
	Retriever   Retriever
	Prompts     PromptStore
	PromptReg   *PromptRegistry
	Plugins     *PluginCatalog
	Planner     Planner
	Model       string
	TokenBudget int
	WorkspaceID string
	UserID      string
	IDFunc      func() string
	Now         func() time.Time
}

// NewService constructs an orchestrator service.
func NewService(cfg ServiceConfig) *Service {
	store := cfg.Store
	if store == nil {
		store = NewMemoryStore()
	}
	builder := cfg.Builder
	if builder == nil {
		builder = NewContextBuilder()
	}
	tools := cfg.Tools
	if tools == nil {
		tools = DefaultToolRegistry()
	}
	mem := cfg.Memory
	if mem == nil {
		mem = NewInMemoryMemory()
	}
	planner := cfg.Planner
	if planner == nil {
		// Default: LLM plan when gateway is set, with rule fallback; pure rules otherwise.
		if cfg.Gateway != nil {
			planner = NewLLMPlanner(cfg.Gateway, cfg.Model, NewRulePlanner())
		} else {
			planner = NewRulePlanner()
		}
	}
	idFunc := cfg.IDFunc
	if idFunc == nil {
		idFunc = newID
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	plugins := cfg.Plugins
	if plugins == nil {
		plugins = NewPluginCatalog(tools)
	}
	// Prefer metered retriever for search SLA metrics when a plain retriever is passed.
	retriever := cfg.Retriever
	if retriever != nil {
		if _, ok := retriever.(*MeteredRetriever); !ok {
			retriever = NewMeteredRetriever(retriever)
		}
	}
	exec := &Executor{
		Store:       store,
		Gateway:     cfg.Gateway,
		Builder:     builder,
		Tools:       tools,
		Memory:      mem,
		Retriever:   retriever,
		Model:       cfg.Model,
		TokenBudget: cfg.TokenBudget,
		WorkspaceID: cfg.WorkspaceID,
		UserID:      cfg.UserID,
	}
	return &Service{
		Store: store, Executor: exec, Planner: planner, Memory: mem, Retriever: retriever,
		Prompts: cfg.Prompts, PromptReg: cfg.PromptReg, Plugins: plugins, Tools: tools,
		idFunc: idFunc, now: now,
	}
}

// CreateRun plans dynamically from the goal and stores a pending run.
func (s *Service) CreateRun(ctx context.Context, goal string) (Run, error) {
	if s == nil || s.Store == nil {
		return Run{}, errors.New("orchestrator not configured")
	}
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return Run{}, errors.New("goal is required")
	}
	steps, err := s.Planner.Plan(ctx, goal)
	if err != nil {
		return Run{}, err
	}
	now := s.now()
	run := Run{
		RunID:     s.idFunc(),
		Goal:      goal,
		Status:    StatusPending,
		Steps:     steps,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Store.SaveRun(ctx, run); err != nil {
		return Run{}, err
	}
	_ = s.Store.AppendEvent(ctx, Event{EventID: s.idFunc(), RunID: run.RunID, Type: "run_created", Message: goal, CreatedAt: now})
	return run, nil
}

// GetRun returns the current run snapshot.
func (s *Service) GetRun(ctx context.Context, runID string) (Run, error) {
	return s.Store.GetRun(ctx, runID)
}

// ListEvents returns lifecycle events for a run via the store ListEvents path.
// Returns an error if the run does not exist (callers map to 404).
func (s *Service) ListEvents(ctx context.Context, runID string) ([]Event, error) {
	if s == nil || s.Store == nil {
		return nil, errors.New("orchestrator not configured")
	}
	if _, err := s.Store.GetRun(ctx, runID); err != nil {
		return nil, err
	}
	return s.Store.ListEvents(ctx, runID)
}

// StartRun executes (or resumes) the plan.
func (s *Service) StartRun(ctx context.Context, runID string) (Run, error) {
	if s.Executor == nil {
		return Run{}, errors.New("executor not configured")
	}
	return s.Executor.ExecuteRun(ctx, runID)
}

// ApproveRun approves a waiting tool step and continues execution.
func (s *Service) ApproveRun(ctx context.Context, runID string) (Run, error) {
	run, err := s.Store.GetRun(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	found := false
	for i := range run.Steps {
		if run.Steps[i].Status == StepWaitingApproval {
			run.Steps[i].Approved = true
			found = true
		}
	}
	if !found {
		return Run{}, errors.New("no step waiting for approval")
	}
	run.Status = StatusRunning
	run.UpdatedAt = s.now()
	if err := s.Store.SaveRun(ctx, run); err != nil {
		return Run{}, err
	}
	_ = s.Store.AppendEvent(ctx, Event{EventID: s.idFunc(), RunID: runID, Type: "approved", Message: "human approved"})
	return s.Executor.ExecuteRun(ctx, runID)
}

// RejectRun fails a run that was waiting for approval.
func (s *Service) RejectRun(ctx context.Context, runID, reason string) (Run, error) {
	run, err := s.Store.GetRun(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	if reason == "" {
		reason = "rejected by operator"
	}
	for i := range run.Steps {
		if run.Steps[i].Status == StepWaitingApproval {
			run.Steps[i].Status = StepFailed
			run.Steps[i].Error = reason
		} else if run.Steps[i].Status == StepPending {
			run.Steps[i].Status = StepSkipped
		}
	}
	run.Status = StatusFailed
	run.Error = reason
	run.UpdatedAt = s.now()
	_ = s.Store.SaveRun(ctx, run)
	_ = s.Store.AppendEvent(ctx, Event{EventID: s.idFunc(), RunID: runID, Type: "rejected", Message: reason})
	return run, nil
}

// CancelRun requests cancellation.
func (s *Service) CancelRun(ctx context.Context, runID string) (Run, error) {
	run, err := s.Store.GetRun(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	if run.Status == StatusSucceeded || run.Status == StatusFailed || run.Status == StatusCancelled {
		return run, nil
	}
	run.CancelRequested = true
	run.UpdatedAt = s.now()
	if err := s.Store.SaveRun(ctx, run); err != nil {
		return Run{}, err
	}
	if run.Status == StatusPending || run.Status == StatusWaitingApproval {
		run.Status = StatusCancelled
		run.Error = "cancelled"
		for i := range run.Steps {
			if run.Steps[i].Status == StepPending || run.Steps[i].Status == StepWaitingApproval {
				run.Steps[i].Status = StepCancelled
			}
		}
		_ = s.Store.SaveRun(ctx, run)
	}
	return run, nil
}

// IngestDocument adds knowledge for RAG.
func (s *Service) IngestDocument(ctx context.Context, doc Document) (Document, error) {
	if s.Retriever == nil {
		return Document{}, errors.New("retriever not configured")
	}
	return s.Retriever.Ingest(ctx, doc)
}

// PutMemory stores a long-term fact.
func (s *Service) PutMemory(ctx context.Context, fact MemoryFact) error {
	if s.Memory == nil {
		return errors.New("memory not configured")
	}
	return s.Memory.PutLong(ctx, fact)
}
