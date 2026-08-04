package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Judge scores a candidate answer against a rubric (optional LLM-as-judge path).
type Judge interface {
	Judge(ctx context.Context, rubric, candidate string) (pass bool, score float64, rationale string, err error)
}

// JudgeResult is a parsed structured verdict.
type JudgeResult struct {
	Pass      bool    `json:"pass"`
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale,omitempty"`
}

// ParseJudgeResponse extracts pass/score/rationale from model text.
// Accepts JSON object or simple PASS/FAIL lines with optional SCORE=.
func ParseJudgeResponse(content string) (JudgeResult, error) {
	raw := strings.TrimSpace(content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	// Try JSON object first
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			var jr JudgeResult
			if err := json.Unmarshal([]byte(raw[i:j+1]), &jr); err == nil {
				if jr.Score == 0 && jr.Pass {
					jr.Score = 1
				}
				return jr, nil
			}
		}
	}
	upper := strings.ToUpper(raw)
	var jr JudgeResult
	if strings.Contains(upper, "PASS") && !strings.Contains(upper, "FAIL") {
		jr.Pass = true
		jr.Score = 1
	} else if strings.Contains(upper, "FAIL") {
		jr.Pass = false
		jr.Score = 0
	} else {
		return JudgeResult{}, fmt.Errorf("could not parse judge verdict from model reply")
	}
	if m := regexp.MustCompile(`(?i)score\s*[:=]\s*([0-9.]+)`).FindStringSubmatch(raw); len(m) == 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			jr.Score = v
		}
	}
	jr.Rationale = raw
	return jr, nil
}

// GatewayJudge uses InferenceClient with a fixed judge system prompt.
type GatewayJudge struct {
	Client InferenceClient
	Model  string
}

func (g *GatewayJudge) Judge(ctx context.Context, rubric, candidate string) (bool, float64, string, error) {
	if g == nil || g.Client == nil {
		return false, 0, "", fmt.Errorf("judge gateway not configured")
	}
	system := `You are an Atlas evaluation judge. Reply with JSON only: {"pass":bool,"score":number,"rationale":string}. score is 0.0-1.0.`
	user := fmt.Sprintf("Rubric:\n%s\n\nCandidate:\n%s\n", rubric, candidate)
	resp, err := g.Client.Generate(ctx, GenerateRequest{
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Model:     g.Model,
		TaskClass: "eval",
	})
	if err != nil {
		return false, 0, "", err
	}
	jr, err := ParseJudgeResponse(resp.Message.Content)
	if err != nil {
		return false, 0, resp.Message.Content, err
	}
	return jr.Pass, jr.Score, jr.Rationale, nil
}

// StaticJudge returns a fixed verdict (tests / offline stubs).
type StaticJudge struct {
	Pass      bool
	Score     float64
	Rationale string
	Err       error
}

func (s StaticJudge) Judge(context.Context, string, string) (bool, float64, string, error) {
	if s.Err != nil {
		return false, 0, "", s.Err
	}
	score := s.Score
	if score == 0 && s.Pass {
		score = 1
	}
	return s.Pass, score, s.Rationale, nil
}
