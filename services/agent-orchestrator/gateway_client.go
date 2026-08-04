package orchestrator

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

// Message is one chat turn sent to the inference gateway.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateRequest matches inference-gateway POST /v1/generate body (subset).
type GenerateRequest struct {
	Messages  []Message         `json:"messages"`
	TaskClass string            `json:"task_class,omitempty"`
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// GenerateResponse matches gateway chat response (subset).
type GenerateResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// InferenceClient calls the inference gateway.
type InferenceClient interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
}

// HTTPGatewayClient is a real HTTP client for POST /v1/generate.
type HTTPGatewayClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Model      string
}

func NewHTTPGatewayClient(baseURL, model string) *HTTPGatewayClient {
	if model == "" {
		model = "fake-model"
	}
	return &HTTPGatewayClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *HTTPGatewayClient) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if c == nil || c.BaseURL == "" {
		return GenerateResponse{}, fmt.Errorf("gateway base URL is not configured")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.TaskClass == "" {
		req.TaskClass = "act"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/generate", bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("gateway HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out GenerateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode gateway response: %w", err)
	}
	if strings.TrimSpace(out.Message.Content) == "" {
		return GenerateResponse{}, fmt.Errorf("gateway returned empty assistant content")
	}
	return out, nil
}
