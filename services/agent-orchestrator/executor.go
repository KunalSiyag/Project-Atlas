package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrWaitingApproval is returned when a dangerous tool pauses the run.
var ErrWaitingApproval = errors.New("waiting for approval")

// Executor runs a plan graph: dependency order, retries, cancellation, tools, gateway.
type Executor struct {
	Store      RunStore
	Gateway    InferenceClient
	Builder    *ContextBuilder
	Tools      *ToolRegistry
	Memory     MemoryStore
	Retriever  Retriever
	Model      string
	TokenBudget int
	WorkspaceID string
	UserID      string
}

// ExecuteRun loads the run, marks running, executes steps, persists terminal or waiting state.
func (e *Executor) ExecuteRun(ctx context.Context, runID string) (Run, error) {
	if e == nil || e.Store == nil {
		return Run{}, fmt.Errorf("executor not configured")
	}
	run, err := e.Store.GetRun(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	// Allow resume from waiting_approval after approve.
	if run.Status != StatusPending && run.Status != StatusRunning && run.Status != StatusWaitingApproval {
		return Run{}, fmt.Errorf("run is not startable (status=%s)", run.Status)
	}
	run.Status = StatusRunning
	run.Error = ""
	run.UpdatedAt = time.Now().UTC()
	if err := e.Store.SaveRun(ctx, run); err != nil {
		return Run{}, err
	}
	_ = e.Store.AppendEvent(ctx, Event{EventID: newID(), RunID: runID, Type: "run_started", Message: "execution started"})

	for {
		if err := ctx.Err(); err != nil {
			return e.failRun(ctx, runID, "context cancelled: "+err.Error())
		}
		run, err = e.Store.GetRun(ctx, runID)
		if err != nil {
			return Run{}, err
		}
		if run.CancelRequested {
			return e.cancelRun(ctx, run)
		}
		if e.hasWaitingApproval(run) {
			run.Status = StatusWaitingApproval
			run.UpdatedAt = time.Now().UTC()
			_ = e.Store.SaveRun(ctx, run)
			return run, nil
		}

		next, ok := e.nextReadyStep(run)
		if !ok {
			if e.allSucceeded(run) {
				return e.succeedRun(ctx, run)
			}
			if e.hasFailedExhausted(run) {
				return e.failRun(ctx, runID, e.firstStepError(run))
			}
			return e.failRun(ctx, runID, "plan stuck: no runnable steps")
		}

		if err := e.runStep(ctx, runID, next.StepID); err != nil {
			run, _ = e.Store.GetRun(ctx, runID)
			if errors.Is(err, ErrWaitingApproval) {
				return run, nil
			}
			if run.Status == StatusFailed || run.Status == StatusCancelled || run.Status == StatusWaitingApproval {
				return run, nil
			}
			st := stepByID(run, next.StepID)
			if st != nil && st.Status == StepFailed && st.Attempts >= st.MaxAttempts {
				return e.failRun(ctx, runID, st.Error)
			}
			continue
		}
	}
}

func (e *Executor) runStep(ctx context.Context, runID, stepID string) error {
	run, err := e.Store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.CancelRequested {
		_, err := e.cancelRun(ctx, run)
		return err
	}
	idx := -1
	for i := range run.Steps {
		if run.Steps[i].StepID == stepID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("step %q not found", stepID)
	}
	step := &run.Steps[idx]

	// Resume path: waiting_approval + Approved → execute tool once.
	if step.Status == StepWaitingApproval {
		if !step.Approved {
			return ErrWaitingApproval
		}
		// fall through to execute without re-incrementing beyond needed
	} else {
		step.Status = StepRunning
		step.Attempts++
	}
	if err := e.Store.SaveRun(ctx, run); err != nil {
		return err
	}
	_ = e.Store.AppendEvent(ctx, Event{
		EventID: newID(), RunID: runID, StepID: stepID, Type: "step_started",
		Message: fmt.Sprintf("attempt %d/%d", step.Attempts, step.MaxAttempts),
	})

	var output string
	var stepErr error
	switch step.Kind {
	case StepKindContext:
		output, stepErr = e.execContext(ctx, run)
	case StepKindModel:
		output, stepErr = e.execModel(ctx, run, *step)
	case StepKindTool:
		output, stepErr = e.execTool(ctx, run, step)
		if errors.Is(stepErr, ErrWaitingApproval) {
			run, _ = e.Store.GetRun(ctx, runID)
			st := stepByID(run, stepID)
			if st != nil {
				st.Status = StepWaitingApproval
				st.Error = "requires approval"
			}
			run.Status = StatusWaitingApproval
			run.UpdatedAt = time.Now().UTC()
			_ = e.Store.SaveRun(ctx, run)
			_ = e.Store.AppendEvent(ctx, Event{
				EventID: newID(), RunID: runID, StepID: stepID, Type: "waiting_approval", Message: "approval required",
			})
			return ErrWaitingApproval
		}
	case StepKindFinalize:
		output, stepErr = e.execFinalize(run)
	default:
		stepErr = fmt.Errorf("unknown step kind %q", step.Kind)
	}

	run, err = e.Store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	step = stepByID(run, stepID)
	if step == nil {
		return fmt.Errorf("step %q missing after exec", stepID)
	}
	if stepErr != nil {
		step.Error = stepErr.Error()
		if step.Attempts < step.MaxAttempts {
			step.Status = StepPending
			_ = e.Store.SaveRun(ctx, run)
			_ = e.Store.AppendEvent(ctx, Event{
				EventID: newID(), RunID: runID, StepID: stepID, Type: "step_retry", Message: stepErr.Error(),
			})
			return stepErr
		}
		step.Status = StepFailed
		_ = e.Store.SaveRun(ctx, run)
		return stepErr
	}
	step.Status = StepSucceeded
	step.Output = output
	step.Error = ""
	_ = e.Store.SaveRun(ctx, run)
	_ = e.Store.AppendEvent(ctx, Event{
		EventID: newID(), RunID: runID, StepID: stepID, Type: "step_succeeded", Message: "ok",
	})
	// Persist short-term memory of tool/model outputs for later context.
	if e.Memory != nil && output != "" {
		_ = e.Memory.PutShort(ctx, runID, step.Name+": "+output)
	}
	return nil
}

func (e *Executor) execTool(ctx context.Context, run Run, step *Step) (string, error) {
	if e.Tools == nil {
		return "", fmt.Errorf("tool registry not configured")
	}
	tool, ok := e.Tools.Get(step.ToolName)
	if !ok {
		return "", fmt.Errorf("unknown tool %q", step.ToolName)
	}
	needsApproval := step.RequiresApproval || tool.RequiresApproval()
	if needsApproval && !step.Approved {
		return "", ErrWaitingApproval
	}
	return e.Tools.Invoke(ctx, step.ToolName, step.ToolInput, InvokeOptions{Approved: true})
}

func (e *Executor) execContext(ctx context.Context, run Run) (string, error) {
	builder := e.Builder
	if builder == nil {
		builder = NewContextBuilder()
	}
	history := []Message{{Role: "user", Content: run.Goal}}
	for _, st := range run.Steps {
		if st.Status == StepSucceeded && st.Output != "" && st.Kind != StepKindContext {
			history = append(history, Message{Role: "assistant", Content: st.Output})
		}
	}
	notes := CollectMemoryNotes(ctx, e.Memory, run.RunID, e.workspace(), e.UserID)
	var docs []string
	if e.Retriever != nil {
		if hits, err := e.Retriever.Search(ctx, run.Goal, 3); err == nil {
			docs = DocSnippets(hits)
		}
	}
	assembled, err := builder.Build(ContextRequest{
		System:      "You are Atlas orchestrator. Answer the user goal helpfully.",
		History:     history,
		MemoryNotes: notes,
		DocSnippets: docs,
		MaxTokens:   e.budget(),
	})
	if err != nil {
		return "", err
	}
	var parts []string
	for _, m := range assembled.Messages {
		parts = append(parts, m.Role+": "+m.Content)
	}
	return strings.Join(parts, "\n"), nil
}

func (e *Executor) execModel(ctx context.Context, run Run, step Step) (string, error) {
	if e.Gateway == nil {
		return "", fmt.Errorf("inference gateway client is not configured")
	}
	builder := e.Builder
	if builder == nil {
		builder = NewContextBuilder()
	}
	history := []Message{{Role: "user", Content: run.Goal}}
	// Prior tool/model outputs
	for _, st := range run.Steps {
		if st.Status == StepSucceeded && st.Output != "" && st.StepID != step.StepID {
			history = append(history, Message{Role: "assistant", Content: st.Name + ": " + st.Output})
		}
	}
	if step.Prompt != "" {
		history = append(history, Message{Role: "user", Content: step.Prompt})
	}
	notes := CollectMemoryNotes(ctx, e.Memory, run.RunID, e.workspace(), e.UserID)
	var docs []string
	if e.Retriever != nil {
		if hits, err := e.Retriever.Search(ctx, run.Goal, 3); err == nil {
			docs = DocSnippets(hits)
		}
	}
	assembled, err := builder.Build(ContextRequest{
		System:      "You are Atlas. Complete the user goal using provided context, memory, and tools results.",
		History:     history,
		MemoryNotes: notes,
		DocSnippets: docs,
		MaxTokens:   e.budget(),
	})
	if err != nil {
		return "", err
	}
	resp, err := e.Gateway.Generate(ctx, GenerateRequest{
		Messages:  assembled.Messages,
		Model:     e.Model,
		TaskClass: "act",
		Metadata:  map[string]string{"run_id": run.RunID, "step_id": step.StepID},
	})
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}

func (e *Executor) execFinalize(run Run) (string, error) {
	for i := len(run.Steps) - 1; i >= 0; i-- {
		if run.Steps[i].Kind == StepKindModel && run.Steps[i].Status == StepSucceeded && run.Steps[i].Output != "" {
			return run.Steps[i].Output, nil
		}
	}
	// fallback: last tool output
	for i := len(run.Steps) - 1; i >= 0; i-- {
		if run.Steps[i].Status == StepSucceeded && run.Steps[i].Output != "" {
			return run.Steps[i].Output, nil
		}
	}
	return "", fmt.Errorf("no output to finalize")
}

func (e *Executor) budget() int {
	if e.TokenBudget > 0 {
		return e.TokenBudget
	}
	return 256
}

func (e *Executor) workspace() string {
	if e.WorkspaceID != "" {
		return e.WorkspaceID
	}
	return "default"
}

func (e *Executor) succeedRun(ctx context.Context, run Run) (Run, error) {
	result := ""
	if fin := stepByID(run, "finalize"); fin != nil && fin.Output != "" {
		result = fin.Output
	} else {
		for i := len(run.Steps) - 1; i >= 0; i-- {
			if run.Steps[i].Output != "" && run.Steps[i].Status == StepSucceeded {
				result = run.Steps[i].Output
				break
			}
		}
	}
	run.Status = StatusSucceeded
	run.Result = result
	run.Error = ""
	run.UpdatedAt = time.Now().UTC()
	if err := e.Store.SaveRun(ctx, run); err != nil {
		return Run{}, err
	}
	_ = e.Store.AppendEvent(ctx, Event{EventID: newID(), RunID: run.RunID, Type: "run_succeeded", Message: "ok"})
	return run, nil
}

func (e *Executor) failRun(ctx context.Context, runID, msg string) (Run, error) {
	run, err := e.Store.GetRun(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	run.Status = StatusFailed
	run.Error = msg
	run.UpdatedAt = time.Now().UTC()
	for i := range run.Steps {
		if run.Steps[i].Status == StepPending || run.Steps[i].Status == StepRunning {
			run.Steps[i].Status = StepSkipped
		}
	}
	_ = e.Store.SaveRun(ctx, run)
	_ = e.Store.AppendEvent(ctx, Event{EventID: newID(), RunID: runID, Type: "run_failed", Message: msg})
	return run, nil
}

func (e *Executor) cancelRun(ctx context.Context, run Run) (Run, error) {
	run.Status = StatusCancelled
	run.Error = "cancelled"
	run.UpdatedAt = time.Now().UTC()
	for i := range run.Steps {
		if run.Steps[i].Status == StepPending || run.Steps[i].Status == StepRunning || run.Steps[i].Status == StepWaitingApproval {
			run.Steps[i].Status = StepCancelled
		}
	}
	_ = e.Store.SaveRun(ctx, run)
	_ = e.Store.AppendEvent(ctx, Event{EventID: newID(), RunID: run.RunID, Type: "run_cancelled", Message: "cancelled"})
	return run, nil
}

func (e *Executor) nextReadyStep(run Run) (Step, bool) {
	status := map[string]string{}
	for _, s := range run.Steps {
		status[s.StepID] = s.Status
	}
	for _, s := range run.Steps {
		// pending OR approved waiting_approval resume
		if s.Status != StepPending && !(s.Status == StepWaitingApproval && s.Approved) {
			continue
		}
		ready := true
		for _, dep := range s.DependsOn {
			if status[dep] != StepSucceeded {
				ready = false
				break
			}
		}
		if ready {
			return s, true
		}
	}
	return Step{}, false
}

func (e *Executor) hasWaitingApproval(run Run) bool {
	for _, s := range run.Steps {
		if s.Status == StepWaitingApproval && !s.Approved {
			return true
		}
	}
	return false
}

func (e *Executor) allSucceeded(run Run) bool {
	for _, s := range run.Steps {
		if s.Status != StepSucceeded {
			return false
		}
	}
	return len(run.Steps) > 0
}

func (e *Executor) hasFailedExhausted(run Run) bool {
	for _, s := range run.Steps {
		if s.Status == StepFailed && s.Attempts >= s.MaxAttempts {
			return true
		}
	}
	return false
}

func (e *Executor) firstStepError(run Run) string {
	for _, s := range run.Steps {
		if s.Status == StepFailed && s.Error != "" {
			return s.Error
		}
	}
	return "step failed"
}

func stepByID(run Run, id string) *Step {
	for i := range run.Steps {
		if run.Steps[i].StepID == id {
			return &run.Steps[i]
		}
	}
	return nil
}
