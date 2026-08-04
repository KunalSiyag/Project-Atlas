package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Tool is a named capability with schema validation and bounded execution.
type Tool interface {
	Name() string
	// Schema is a minimal JSON-schema-like map: required fields and types ("string","number").
	Schema() map[string]any
	// Dangerous tools require human approval before invoke.
	Dangerous() bool
	// Timeout for a single invocation.
	Timeout() time.Duration
	Invoke(ctx context.Context, input map[string]any) (string, error)
}

// ToolRegistry maps tool name → implementation.
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry(tools ...Tool) *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]Tool)}
	for _, t := range tools {
		r.tools[t.Name()] = t
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

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

// ValidateToolInput checks required keys and simple types from Schema().
func ValidateToolInput(tool Tool, input map[string]any) error {
	if input == nil {
		input = map[string]any{}
	}
	schema := tool.Schema()
	required, _ := schema["required"].([]string)
	if required == nil {
		// also accept []any from JSON-ish maps
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
			// allow numeric strings
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

// RunTool validates, applies timeout, and invokes.
func RunTool(ctx context.Context, tool Tool, input map[string]any) (string, error) {
	if err := ValidateToolInput(tool, input); err != nil {
		return "", err
	}
	timeout := tool.Timeout()
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
		out, err := tool.Invoke(cctx, input)
		done <- struct {
			out string
			err error
		}{out, err}
	}()
	select {
	case <-cctx.Done():
		return "", fmt.Errorf("tool timeout after %s", timeout)
	case res := <-done:
		return res.out, res.err
	}
}

// --- mock tools ---

type EchoTool struct{}

func (EchoTool) Name() string { return "echo" }
func (EchoTool) Dangerous() bool { return false }
func (EchoTool) Timeout() time.Duration { return 2 * time.Second }
func (EchoTool) Schema() map[string]any {
	return map[string]any{
		"required": []string{"text"},
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
}
func (EchoTool) Invoke(_ context.Context, input map[string]any) (string, error) {
	return fmt.Sprint(input["text"]), nil
}

// DangerEchoTool is a side-effect-ish mock that requires approval.
type DangerEchoTool struct{}

func (DangerEchoTool) Name() string { return "danger_echo" }
func (DangerEchoTool) Dangerous() bool { return true }
func (DangerEchoTool) Timeout() time.Duration { return 2 * time.Second }
func (DangerEchoTool) Schema() map[string]any {
	return map[string]any{
		"required": []string{"text"},
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
	}
}
func (DangerEchoTool) Invoke(_ context.Context, input map[string]any) (string, error) {
	return "DANGER_OK:" + fmt.Sprint(input["text"]), nil
}

type CalculatorTool struct{}

func (CalculatorTool) Name() string { return "calculator" }
func (CalculatorTool) Dangerous() bool { return false }
func (CalculatorTool) Timeout() time.Duration { return 2 * time.Second }
func (CalculatorTool) Schema() map[string]any {
	return map[string]any{
		"required": []string{"expression"},
		"properties": map[string]any{
			"expression": map[string]any{"type": "string"},
		},
	}
}
func (CalculatorTool) Invoke(_ context.Context, input map[string]any) (string, error) {
	expr := strings.ReplaceAll(fmt.Sprint(input["expression"]), " ", "")
	// support a+b, a-b, a*b, a/b with two numbers
	for _, op := range []string{"+", "-", "*", "/"} {
		if i := strings.Index(expr, op); i > 0 {
			left, err1 := strconv.ParseFloat(expr[:i], 64)
			right, err2 := strconv.ParseFloat(expr[i+1:], 64)
			if err1 != nil || err2 != nil {
				return "", fmt.Errorf("invalid expression %q", expr)
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
					return "", fmt.Errorf("division by zero")
				}
				v = left / right
			}
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		}
	}
	return "", fmt.Errorf("unsupported expression %q (use a+b form)", expr)
}

// HTTPGetTool fetches a URL (http/https). Host can be constrained in production.
type HTTPGetTool struct {
	Client *http.Client
}

func (HTTPGetTool) Name() string { return "http_get" }
func (HTTPGetTool) Dangerous() bool { return false }
func (HTTPGetTool) Timeout() time.Duration { return 5 * time.Second }
func (HTTPGetTool) Schema() map[string]any {
	return map[string]any{
		"required": []string{"url"},
		"properties": map[string]any{
			"url": map[string]any{"type": "string"},
		},
	}
}
func (t HTTPGetTool) Invoke(ctx context.Context, input map[string]any) (string, error) {
	urlStr := fmt.Sprint(input["url"])
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return "", fmt.Errorf("url must start with http:// or https://")
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
