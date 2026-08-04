package orchestrator

import "time"

// Run terminal and intermediate statuses.
const (
	StatusPending          = "pending"
	StatusRunning          = "running"
	StatusSucceeded        = "succeeded"
	StatusFailed           = "failed"
	StatusCancelled        = "cancelled"
	StatusWaitingApproval  = "waiting_approval"
)

// Step kinds executed by the plan runner.
const (
	StepKindContext  = "context"  // build/trim context from goal + history + memory + RAG
	StepKindModel    = "model"    // call inference gateway /v1/generate
	StepKindTool     = "tool"     // invoke a registered tool
	StepKindFinalize = "finalize"
)

// Step statuses.
const (
	StepPending          = "pending"
	StepRunning          = "running"
	StepSucceeded        = "succeeded"
	StepFailed           = "failed"
	StepSkipped          = "skipped"
	StepCancelled        = "cancelled"
	StepWaitingApproval  = "waiting_approval"
)

// Step is one node in a run plan DAG.
type Step struct {
	StepID      string   `json:"step_id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Status      string   `json:"status"`
	Attempts    int      `json:"attempts"`
	MaxAttempts int      `json:"max_attempts"`
	Output      string   `json:"output,omitempty"`
	Error       string   `json:"error,omitempty"`
	// Prompt is optional model-step user content override; empty uses goal + prior outputs.
	Prompt string `json:"prompt,omitempty"`
	// ToolName + ToolInput apply when Kind == tool.
	ToolName         string         `json:"tool_name,omitempty"`
	ToolInput        map[string]any `json:"tool_input,omitempty"`
	RequiresApproval bool           `json:"requires_approval,omitempty"`
	// Approved is set by the approve API for dangerous tools.
	Approved bool `json:"approved,omitempty"`
}

// Run is a durable orchestration execution record.
type Run struct {
	RunID     string    `json:"run_id"`
	Goal      string    `json:"goal"`
	Status    string    `json:"status"`
	Steps     []Step    `json:"steps"`
	Result    string    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	// CancelRequested is set by CancelRun; the executor checks between steps.
	CancelRequested bool      `json:"cancel_requested,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DefaultPlan returns a real multi-step plan graph (not a three-string stub).
// prepare → infer (depends prepare) → finalize (depends infer).
func DefaultPlan() []Step {
	return []Step{
		{
			StepID:      "prepare",
			Name:        "prepare_context",
			Kind:        StepKindContext,
			DependsOn:   nil,
			Status:      StepPending,
			MaxAttempts: 1,
		},
		{
			StepID:      "infer",
			Name:        "call_inference",
			Kind:        StepKindModel,
			DependsOn:   []string{"prepare"},
			Status:      StepPending,
			MaxAttempts: 2, // one retry by default
		},
		{
			StepID:      "finalize",
			Name:        "return_answer",
			Kind:        StepKindFinalize,
			DependsOn:   []string{"infer"},
			Status:      StepPending,
			MaxAttempts: 1,
		},
	}
}

// Event is an optional audit record for step transitions.
type Event struct {
	EventID   string    `json:"event_id"`
	RunID     string    `json:"run_id"`
	StepID    string    `json:"step_id,omitempty"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
