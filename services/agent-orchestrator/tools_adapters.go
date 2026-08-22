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

// GitHubIssueTool creates an issue via a GitHub-compatible HTTP API.
// BaseURL is injectable (e.g. httptest server); defaults to https://api.github.com when empty.
// Live github.com credentials are not required for unit tests.
type GitHubIssueTool struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (GitHubIssueTool) Name() string        { return "github_create_issue" }
func (GitHubIssueTool) Description() string { return "Create a GitHub issue via REST API (configurable base URL)" }
func (GitHubIssueTool) RequiresApproval() bool { return false }
func (GitHubIssueTool) Timeout() time.Duration { return 8 * time.Second }
func (GitHubIssueTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"owner", "repo", "title"},
		"properties": map[string]any{
			"owner": map[string]any{"type": "string"},
			"repo":  map[string]any{"type": "string"},
			"title": map[string]any{"type": "string"},
			"body":  map[string]any{"type": "string"},
		},
	}
}

func (t GitHubIssueTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	owner := fmt.Sprint(input["owner"])
	repo := fmt.Sprint(input["repo"])
	title := fmt.Sprint(input["title"])
	body, _ := input["body"].(string)
	base := strings.TrimRight(t.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body})
	url := fmt.Sprintf("%s/repos/%s/%s/issues", base, owner, repo)
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", toolErrorf(ToolErrToolError, "github request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", toolErrorf(ToolErrToolError, "github HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Number == 0 && out.HTMLURL == "" {
		return string(raw), nil
	}
	return fmt.Sprintf("issue #%d %s %s", out.Number, out.Title, out.HTMLURL), nil
}

// SlackPostMessageTool posts a message via a Slack-compatible chat.postMessage API.
// BaseURL injectable for tests (httptest); defaults to https://slack.com/api.
type SlackPostMessageTool struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (SlackPostMessageTool) Name() string        { return "slack_post_message" }
func (SlackPostMessageTool) Description() string { return "Post a Slack message via chat.postMessage (configurable base URL)" }
func (SlackPostMessageTool) RequiresApproval() bool { return false }
func (SlackPostMessageTool) Timeout() time.Duration { return 8 * time.Second }
func (SlackPostMessageTool) InputSchema() map[string]any {
	return map[string]any{
		"required": []string{"channel", "text"},
		"properties": map[string]any{
			"channel": map[string]any{"type": "string"},
			"text":    map[string]any{"type": "string"},
		},
	}
}

func (t SlackPostMessageTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	channel := fmt.Sprint(input["channel"])
	text := fmt.Sprint(input["text"])
	base := strings.TrimRight(t.BaseURL, "/")
	if base == "" {
		base = "https://slack.com/api"
	}
	payload, _ := json.Marshal(map[string]string{"channel": channel, "text": text})
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat.postMessage", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", toolErrorf(ToolErrToolError, "slack request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", toolErrorf(ToolErrToolError, "slack HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		TS    string `json:"ts"`
	}
	_ = json.Unmarshal(raw, &out)
	if !out.OK && out.Error != "" {
		return "", toolErrorf(ToolErrToolError, "slack error: %s", out.Error)
	}
	if out.TS != "" {
		return fmt.Sprintf("slack ok ts=%s channel=%s", out.TS, channel), nil
	}
	return string(raw), nil
}

// RegisterIntegrationTools adds GitHub and Slack tools with optional base URLs.
func RegisterIntegrationTools(reg *ToolRegistry, githubBase, slackBase string) error {
	if reg == nil {
		return fmt.Errorf("registry is nil")
	}
	if err := reg.Register(GitHubIssueTool{BaseURL: githubBase, Client: http.DefaultClient}); err != nil {
		return err
	}
	return reg.Register(SlackPostMessageTool{BaseURL: slackBase, Client: http.DefaultClient})
}
