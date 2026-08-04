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

// OrchestratorClient talks to agent-orchestrator HTTP API.
type OrchestratorClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOrchestratorClient(baseURL string) *OrchestratorClient {
	return &OrchestratorClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

type OrchStep struct {
	StepID   string `json:"step_id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
}

type OrchRun struct {
	RunID  string     `json:"run_id"`
	Goal   string     `json:"goal"`
	Status string     `json:"status"`
	Steps  []OrchStep `json:"steps"`
	Result string     `json:"result,omitempty"`
	Error  string     `json:"error,omitempty"`
}

func (c *OrchestratorClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("orchestrator unreachable at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("orchestrator health HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *OrchestratorClient) CreateRun(ctx context.Context, goal string) (OrchRun, error) {
	var out OrchRun
	err := c.doJSON(ctx, http.MethodPost, "/v1/runs", map[string]string{"goal": goal}, &out)
	return out, err
}

func (c *OrchestratorClient) StartRun(ctx context.Context, id string) (OrchRun, error) {
	var out OrchRun
	err := c.doJSON(ctx, http.MethodPost, "/v1/runs/"+id+"/start", nil, &out)
	return out, err
}

func (c *OrchestratorClient) GetRun(ctx context.Context, id string) (OrchRun, error) {
	var out OrchRun
	err := c.doJSON(ctx, http.MethodGet, "/v1/runs/"+id, nil, &out)
	return out, err
}

func (c *OrchestratorClient) ApproveRun(ctx context.Context, id string) (OrchRun, error) {
	var out OrchRun
	err := c.doJSON(ctx, http.MethodPost, "/v1/runs/"+id+"/approve", map[string]string{}, &out)
	return out, err
}

func (c *OrchestratorClient) doJSON(ctx context.Context, method, path string, payload any, dest any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if dest != nil && len(raw) > 0 {
		return json.Unmarshal(raw, dest)
	}
	return nil
}
