package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Stable tool error classes (RFC 0001).
const (
	ToolErrInvalidInput = "invalid_input"
	ToolErrTimeout      = "timeout"
	ToolErrDenied       = "denied"
	ToolErrToolError    = "tool_error"
)

// ToolError is a classified tool failure for eval and callers.
type ToolError struct {
	Class   string
	Message string
	Err     error
}

func (e *ToolError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Class, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Class, e.Err)
	}
	return e.Class
}

func (e *ToolError) Unwrap() error { return e.Err }

func toolErrorf(class, msg string, args ...any) error {
	return &ToolError{Class: class, Message: fmt.Sprintf(msg, args...)}
}

// Tool is a named capability with schema validation and bounded execution (RFC 0001).
type Tool interface {
	Name() string
	Description() string
	// InputSchema is a JSON-Schema-like object (required, properties).
	InputSchema() map[string]any
	// RequiresApproval pauses the run until a human approves (dangerous tools).
	RequiresApproval() bool
	// Timeout for a single invocation.
	Timeout() time.Duration
	// Execute runs the tool after validation (callers should prefer Registry.Invoke).
	Execute(ctx context.Context, input map[string]any) (string, error)
}

// Schema is an alias for InputSchema used by older call sites / tests.
// Prefer InputSchema on new code.
func ToolSchema(t Tool) map[string]any {
	if t == nil {
		return nil
	}
	return t.InputSchema()
}

// ToolRegistry maps tool name → implementation.
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry(tools ...Tool) *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]Tool)}
	for _, t := range tools {
		if t != nil {
			r.tools[t.Name()] = t
		}
	}
	return r
}

func DefaultToolRegistry() *ToolRegistry {
	return NewToolRegistry(
		EchoTool{},
		CalculatorTool{},
		HTTPGetTool{Client: http.DefaultClient},
		DangerEchoTool{}, // requires approval
	)
}

// Register adds or replaces a tool by name.
func (r *ToolRegistry) Register(t Tool) error {
	if r == nil {
		return errors.New("tool registry is nil")
	}
	if t == nil || strings.TrimSpace(t.Name()) == "" {
		return toolErrorf(ToolErrInvalidInput, "tool name is required")
	}
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	r.tools[t.Name()] = t
	return nil
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

// List returns registered tools in arbitrary order.
func (r *ToolRegistry) List() []Tool {
	if r == nil {
		return nil
	}
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// InvokeOptions controls Registry.Invoke.
type InvokeOptions struct {
	// Timeout overrides tool.Timeout when > 0.
	Timeout time.Duration
	// Approved must be true for tools with RequiresApproval.
	Approved bool
}

// Invoke validates, enforces approval, applies timeout, and executes via the registry.
func (r *ToolRegistry) Invoke(ctx context.Context, name string, input map[string]any, opts InvokeOptions) (string, error) {
	if r == nil {
		return "", toolErrorf(ToolErrToolError, "tool registry not configured")
	}
	tool, ok := r.Get(name)
	if !ok {
		return "", toolErrorf(ToolErrInvalidInput, "unknown tool %q", name)
	}
	if tool.RequiresApproval() && !opts.Approved {
		return "", toolErrorf(ToolErrDenied, "tool %q requires approval", name)
	}
	if err := ValidateToolInput(tool, input); err != nil {
		return "", &ToolError{Class: ToolErrInvalidInput, Message: err.Error(), Err: err}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = tool.Timeout()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan struct {
		out string
		err error
	}, 1)
	go func() {
		out, err := tool.Execute(cctx, input)
		done <- struct {
			out string
			err error
		}{out, err}
	}()
	select {
	case <-cctx.Done():
		return "", &ToolError{Class: ToolErrTimeout, Message: fmt.Sprintf("tool timeout after %s", timeout), Err: cctx.Err()}
	case res := <-done:
		if res.err != nil {
			var te *ToolError
			if errors.As(res.err, &te) {
				return "", res.err
			}
			return "", &ToolError{Class: ToolErrToolError, Message: res.err.Error(), Err: res.err}
		}
		return res.out, nil
	}
}

// ValidateToolInput checks required keys and simple types from InputSchema().
func ValidateToolInput(tool Tool, input map[string]any) error {
	if input == nil {
		input = map[string]any{}
	}
	schema := tool.InputSchema()
	required, _ := schema["required"].([]string)
	if required == nil {
		if raw, ok := schema["required"].([]any); ok {
			for _, v := range raw {
				if s, ok := v.(string); ok {
					required = append(required, s)
				}
			}
		}
	}
	props, _ := schema["properties"].(map[string]any)
	for _, key := range required {
		v, ok := input[key]
		if !ok || v == nil || strings.TrimSpace(fmt.Sprint(v)) == "" {
			return fmt.Errorf("schema: missing required field %q", key)
		}
	}
	if props != nil {
		for key, v := range input {
			spec, ok := props[key].(map[string]any)
			if !ok {
				continue
			}
			want, _ := spec["type"].(string)
			if want == "" {
				continue
			}
			if err := checkType(want, v); err != nil {
				return fmt.Errorf("schema: field %q: %w", key, err)
			}
		}
	}
	return nil
}

func checkType(want string, v any) error {
	switch want {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("want string")
		}
	case "number":
		switch v.(type) {
		case float64, float32, int, int64, int32, json.Number:
			return nil
		default:
			if s, ok := v.(string); ok {
				if _, err := strconv.ParseFloat(s, 64); err == nil {
					return nil
				}
			}
			return fmt.Errorf("want number")
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("want boolean")
		}
	}
	return nil
}

// RunTool validates, applies timeout, and executes (does not check approval).
// Prefer ToolRegistry.Invoke for RFC-aligned call path with approval + error classes.
func RunTool(ctx context.Context, tool Tool, input map[string]any) (string, error) {
	if tool == nil {
		return "", toolErrorf(ToolErrToolError, "tool is nil")
	}
	// Use a transient registry to share invoke timeout/validation logic without approval gate.
	r := NewToolRegistry(tool)
	return r.Invoke(ctx, tool.Name(), input, InvokeOptions{Approved: true})
}

// --- built-in tools ---

type EchoTool struct{}

func (EchoTool) Name() string        { return "echo" }
func (EchoTool) Description() string { return "Echoes the provided text unchanged" }
func (EchoTool) RequiresApproval() bool { return false }
func (EchoTool) Timeout() time.Duration { return 2 * time.Second }
func (EchoTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"text"},
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
}
func (EchoTool) Execute(_ context.Context, input map[string]any) (string, error) {
	return fmt.Sprint(input["text"]), nil
}

// DangerEchoTool is a side-effect-ish mock that requires approval.
type DangerEchoTool struct{}

func (DangerEchoTool) Name() string        { return "danger_echo" }
func (DangerEchoTool) Description() string { return "Echoes text after human approval (dangerous mock)" }
func (DangerEchoTool) RequiresApproval() bool { return true }
func (DangerEchoTool) Timeout() time.Duration { return 2 * time.Second }
func (DangerEchoTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"text"},
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
}
func (DangerEchoTool) Execute(_ context.Context, input map[string]any) (string, error) {
	return "DANGER_OK:" + fmt.Sprint(input["text"]), nil
}

type CalculatorTool struct{}

func (CalculatorTool) Name() string        { return "calculator" }
func (CalculatorTool) Description() string { return "Evaluates a simple a+b / a-b / a*b / a/b expression" }
func (CalculatorTool) RequiresApproval() bool { return false }
func (CalculatorTool) Timeout() time.Duration { return 2 * time.Second }
func (CalculatorTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"expression"},
		"properties": map[string]any{
			"expression": map[string]any{"type": "string"},
		},
	}
}
func (CalculatorTool) Execute(_ context.Context, input map[string]any) (string, error) {
	expr := strings.ReplaceAll(fmt.Sprint(input["expression"]), " ", "")
	for _, op := range []string{"+", "-", "*", "/"} {
		if i := strings.Index(expr, op); i > 0 {
			left, err1 := strconv.ParseFloat(expr[:i], 64)
			right, err2 := strconv.ParseFloat(expr[i+1:], 64)
			if err1 != nil || err2 != nil {
				return "", toolErrorf(ToolErrInvalidInput, "invalid expression %q", expr)
			}
			var v float64
			switch op {
			case "+":
				v = left + right
			case "-":
				v = left - right
			case "*":
				v = left * right
			case "/":
				if right == 0 {
					return "", toolErrorf(ToolErrInvalidInput, "division by zero")
				}
				v = left / right
			}
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		}
	}
	return "", toolErrorf(ToolErrInvalidInput, "unsupported expression %q (use a+b form)", expr)
}

// HTTPGetTool fetches a URL (http/https). Host can be constrained in production.
type HTTPGetTool struct {
	Client *http.Client
}

func (HTTPGetTool) Name() string        { return "http_get" }
func (HTTPGetTool) Description() string { return "HTTP GET a URL and return status + body (64KiB cap)" }
func (HTTPGetTool) RequiresApproval() bool { return false }
func (HTTPGetTool) Timeout() time.Duration { return 5 * time.Second }
func (HTTPGetTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"url"},
		"properties": map[string]any{
			"url": map[string]any{"type": "string"},
		},
	}
}
func (t HTTPGetTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	urlStr := fmt.Sprint(input["url"])
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return "", toolErrorf(ToolErrInvalidInput, "url must start with http:// or https://")
	}
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body)), nil
}
