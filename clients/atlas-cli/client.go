package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the Atlas inference gateway over HTTP/JSON.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ChatResponse struct {
	Message      Message `json:"message"`
	Usage        Usage   `json:"usage"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	FinishReason string  `json:"finish_reason"`
}

// Matches the gateway's writeError shape: {"error":{"code":"...","message":"..."}}.
type apiErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type Conversation struct {
	ConversationID string                `json:"conversation_id"`
	Title          string                `json:"title"`
	Status         string                `json:"status"`
	Messages       []ConversationMessage `json:"messages"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type ConversationMessage struct {
	MessageID string    `json:"message_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	RunID     string    `json:"run_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type GenerateRequest struct {
	Messages  []Message         `json:"messages"`
	TaskClass string            `json:"task_class,omitempty"`
	Provider  string            `json:"provider,omitempty"`
	APIKey    string            `json:"api_key,omitempty"`
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
}

// StreamDelta matches gateway SSE delta events.
type StreamDelta struct {
	Delta    string `json:"delta"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type GenerateResult struct {
	ChatResponse
	RequestID string
	RunID     string
}

// InferenceRunView matches GET /v1/runs/{id} from the gateway.
type InferenceRunView struct {
	RunID          string        `json:"run_id"`
	RequestID      string        `json:"request_id"`
	TaskClass      string        `json:"task_class"`
	RequestedModel string        `json:"requested_model,omitempty"`
	ResolvedModel  string        `json:"resolved_model"`
	Provider       string        `json:"provider,omitempty"`
	Status         string        `json:"status"`
	Result         *ChatResponse `json:"result,omitempty"`
	ErrorClass     string        `json:"error_class,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
}

type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider,omitempty"`
}

func (c *Client) ListModels(ctx context.Context, provider, apiKey string) ([]ModelInfo, error) {
	url := fmt.Sprintf("%s/v1/models?provider=%s&api_key=%s", c.BaseURL, provider, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models failed (%d)", resp.StatusCode)
	}
	var out struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) Health(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/healthz", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach gateway at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("health check failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Client) Ready(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/readyz", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("not ready (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Client) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	req.Stream = false
	var out GenerateResult
	status, headers, err := c.doJSON(ctx, http.MethodPost, "/v1/generate", req, &out.ChatResponse)
	if err != nil {
		return out, err
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("generate failed (HTTP %d)", status)
	}
	out.RequestID = headers.Get("X-Request-ID")
	out.RunID = headers.Get("X-Run-ID")
	return out, nil
}

// GenerateStream calls POST /v1/generate with stream=true and invokes onDelta for each SSE delta.
// The final assembled ChatResponse is returned from the done event.
func (c *Client) GenerateStream(ctx context.Context, req GenerateRequest, onDelta func(StreamDelta) error) (GenerateResult, error) {
	req.Stream = true
	raw, err := json.Marshal(req)
	if err != nil {
		return GenerateResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/generate", bytes.NewReader(raw))
	if err != nil {
		return GenerateResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	out := GenerateResult{
		RequestID: resp.Header.Get("X-Request-ID"),
		RunID:     resp.Header.Get("X-Run-ID"),
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var ae apiErrorEnvelope
		if json.Unmarshal(body, &ae) == nil && (ae.Error.Code != "" || ae.Error.Message != "") {
			return out, fmt.Errorf("%s: %s", ae.Error.Code, ae.Error.Message)
		}
		return out, fmt.Errorf("stream generate failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var event string
	var sawDone bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		switch event {
		case "delta":
			var d StreamDelta
			if err := json.Unmarshal([]byte(payload), &d); err != nil {
				return out, fmt.Errorf("decode stream delta: %w", err)
			}
			if onDelta != nil {
				if err := onDelta(d); err != nil {
					return out, err
				}
			}
		case "done":
			if err := json.Unmarshal([]byte(payload), &out.ChatResponse); err != nil {
				return out, fmt.Errorf("decode stream done: %w", err)
			}
			sawDone = true
		case "error":
			var ae apiErrorEnvelope
			if json.Unmarshal([]byte(payload), &ae) == nil && ae.Error.Message != "" {
				return out, fmt.Errorf("%s: %s", nonEmptyCLI(ae.Error.Code, "error"), ae.Error.Message)
			}
			return out, fmt.Errorf("stream error: %s", payload)
		}
		event = ""
	}
	if err := scanner.Err(); err != nil {
		return out, fmt.Errorf("read stream: %w", err)
	}
	if !sawDone {
		return out, fmt.Errorf("stream ended without done event")
	}
	return out, nil
}

func nonEmptyCLI(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (c *Client) CreateConversation(ctx context.Context, title string) (Conversation, error) {
	var out Conversation
	status, _, err := c.doJSON(ctx, http.MethodPost, "/v1/conversations", map[string]string{"title": title}, &out)
	if err != nil {
		return out, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return out, fmt.Errorf("create conversation failed (HTTP %d)", status)
	}
	return out, nil
}

func (c *Client) GetConversation(ctx context.Context, id string) (Conversation, error) {
	var out Conversation
	status, _, err := c.doJSON(ctx, http.MethodGet, "/v1/conversations/"+id, nil, &out)
	if err != nil {
		return out, err
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("get conversation failed (HTTP %d)", status)
	}
	return out, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (InferenceRunView, error) {
	var out InferenceRunView
	status, _, err := c.doJSON(ctx, http.MethodGet, "/v1/runs/"+runID, nil, &out)
	if err != nil {
		return out, err
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("get run failed (HTTP %d)", status)
	}
	return out, nil
}

func (c *Client) OpenAPI(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/openapi.yaml", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openapi failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func (c *Client) PostMessage(ctx context.Context, conversationID, content, provider, apiKey, model, taskClass string) (GenerateResult, error) {
	body := map[string]any{
		"content": content,
	}
	if provider != "" {
		body["provider"] = provider
	}
	if apiKey != "" {
		body["api_key"] = apiKey
	}
	if model != "" {
		body["model"] = model
	}
	if taskClass != "" {
		body["task_class"] = taskClass
	}
	var out GenerateResult
	status, headers, err := c.doJSON(ctx, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", body, &out.ChatResponse)
	if err != nil {
		return out, err
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("post message failed (HTTP %d)", status)
	}
	out.RequestID = headers.Get("X-Request-ID")
	out.RunID = headers.Get("X-Run-ID")
	return out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, dest any) (int, http.Header, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Header, err
	}
	if resp.StatusCode >= 400 {
		var ae apiErrorEnvelope
		if json.Unmarshal(raw, &ae) == nil && (ae.Error.Code != "" || ae.Error.Message != "") {
			msg := ae.Error.Message
			if msg == "" {
				msg = string(raw)
			}
			code := ae.Error.Code
			if code == "" {
				code = "error"
			}
			return resp.StatusCode, resp.Header, fmt.Errorf("%s: %s", code, msg)
		}
		return resp.StatusCode, resp.Header, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, dest); err != nil {
			return resp.StatusCode, resp.Header, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, resp.Header, nil
}
