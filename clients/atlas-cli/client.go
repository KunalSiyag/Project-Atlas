package main

import (
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
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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

func (c *Client) PostMessage(ctx context.Context, conversationID, content, model, taskClass string) (GenerateResult, error) {
	body := map[string]any{
		"content": content,
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
