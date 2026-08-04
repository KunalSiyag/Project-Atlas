package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// PromptVar declares a template variable.
type PromptVar struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
}

// PromptEntry is one versioned prompt template.
type PromptEntry struct {
	ID           string      `json:"id" yaml:"id"`
	Version      string      `json:"version" yaml:"version"`
	Status       string      `json:"status" yaml:"status"` // draft | active | retired
	Text         string      `json:"text" yaml:"text"`
	Variables    []PromptVar `json:"variables,omitempty" yaml:"variables,omitempty"`
	ModelClass   string      `json:"model_class,omitempty" yaml:"model_class,omitempty"`
	ApproxTokens int         `json:"approx_tokens,omitempty" yaml:"approx_tokens,omitempty"`
}

// PromptRegistry is an in-memory index of loaded prompt entries.
type PromptRegistry struct {
	// byID maps id -> all versions (any status)
	byID map[string][]PromptEntry
}

// LoadPromptDir walks dir for .json and .yaml/.yml prompt files.
func LoadPromptDir(dir string) (*PromptRegistry, error) {
	reg := &PromptRegistry{byID: make(map[string][]PromptEntry)}
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("prompt dir is required")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("prompt path is not a directory: %s", dir)
	}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json", ".yaml", ".yml":
		default:
			return nil
		}
		entry, err := loadPromptFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if entry.ID == "" {
			return fmt.Errorf("%s: missing id", path)
		}
		if entry.Version == "" {
			entry.Version = "0"
		}
		if entry.Status == "" {
			entry.Status = "active"
		}
		reg.byID[entry.ID] = append(reg.byID[entry.ID], entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func loadPromptFile(path string) (PromptEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PromptEntry{}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var entry PromptEntry
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &entry); err != nil {
			return PromptEntry{}, err
		}
	case ".yaml", ".yml":
		// Minimal YAML subset via JSON after converting common cases, or use simple line parser.
		// Prefer gopkg-free: if content looks like JSON object, parse as JSON; else map key: value.
		trimmed := strings.TrimSpace(string(raw))
		if strings.HasPrefix(trimmed, "{") {
			if err := json.Unmarshal(raw, &entry); err != nil {
				return PromptEntry{}, err
			}
			break
		}
		parsed, err := parseSimpleYAMLPrompt(trimmed)
		if err != nil {
			return PromptEntry{}, err
		}
		entry = parsed
	}
	return entry, nil
}

// parseSimpleYAMLPrompt supports a small subset used in fixtures (scalars + multiline | text).
func parseSimpleYAMLPrompt(s string) (PromptEntry, error) {
	// Convert to a map via extremely small YAML: key: value lines; text: | blocks.
	entry := PromptEntry{}
	lines := strings.Split(s, "\n")
	var textLines []string
	inText := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if inText {
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") || strings.TrimSpace(line) == "" {
				textLines = append(textLines, strings.TrimPrefix(strings.TrimPrefix(line, "  "), "\t"))
				continue
			}
			inText = false
			entry.Text = strings.Join(textLines, "\n")
			textLines = nil
		}
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		key, val, ok := strings.Cut(trim, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "id":
			entry.ID = strings.Trim(val, `"'`)
		case "version":
			entry.Version = strings.Trim(val, `"'`)
		case "status":
			entry.Status = strings.Trim(val, `"'`)
		case "model_class":
			entry.ModelClass = strings.Trim(val, `"'`)
		case "approx_tokens":
			n, _ := strconv.Atoi(val)
			entry.ApproxTokens = n
		case "text":
			if val == "|" || val == ">" {
				inText = true
				textLines = nil
			} else {
				entry.Text = strings.Trim(val, `"'`)
			}
		case "variables":
			// ignore complex YAML lists in simple parser; JSON files carry variables
		}
	}
	if inText {
		entry.Text = strings.Join(textLines, "\n")
	}
	if entry.ID == "" || entry.Text == "" {
		return entry, fmt.Errorf("yaml prompt requires id and text")
	}
	return entry, nil
}

// Resolve returns the prompt for id. If version is empty, highest active version wins.
func (r *PromptRegistry) Resolve(id, version string) (PromptEntry, error) {
	if r == nil {
		return PromptEntry{}, fmt.Errorf("prompt registry is nil")
	}
	list := r.byID[id]
	if len(list) == 0 {
		return PromptEntry{}, fmt.Errorf("prompt %q not found", id)
	}
	if version != "" {
		for _, e := range list {
			if e.Version == version {
				return e, nil
			}
		}
		return PromptEntry{}, fmt.Errorf("prompt %q version %q not found", id, version)
	}
	// highest active by version string (numeric-aware when possible)
	var active []PromptEntry
	for _, e := range list {
		if strings.EqualFold(e.Status, "active") {
			active = append(active, e)
		}
	}
	if len(active) == 0 {
		return PromptEntry{}, fmt.Errorf("prompt %q has no active version", id)
	}
	sort.Slice(active, func(i, j int) bool {
		return versionLess(active[i].Version, active[j].Version)
	})
	return active[len(active)-1], nil
}

func versionLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return ai < bi
	}
	return a < b
}

var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

// Render substitutes {{var}} placeholders. Missing required vars error (no silent empty).
// Undeclared placeholders in text are treated as required.
func (e PromptEntry) Render(vars map[string]string) (string, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	required := map[string]bool{}
	for _, v := range e.Variables {
		if v.Required {
			required[v.Name] = true
		}
	}
	// All placeholders in text are required unless declared optional (required:false).
	matches := varPattern.FindAllStringSubmatch(e.Text, -1)
	for _, m := range matches {
		name := m[1]
		// if declared with Required:false, not required
		declaredOptional := false
		for _, v := range e.Variables {
			if v.Name == name && !v.Required {
				declaredOptional = true
				break
			}
		}
		if !declaredOptional {
			required[name] = true
		}
	}
	for name := range required {
		if _, ok := vars[name]; !ok {
			return "", fmt.Errorf("missing required prompt variable %q", name)
		}
	}
	out := varPattern.ReplaceAllStringFunc(e.Text, func(m string) string {
		sub := varPattern.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return vars[sub[1]]
	})
	return out, nil
}

// ResolveAndRender is a convenience for callers.
func (r *PromptRegistry) ResolveAndRender(id, version string, vars map[string]string) (string, PromptEntry, error) {
	e, err := r.Resolve(id, version)
	if err != nil {
		return "", PromptEntry{}, err
	}
	text, err := e.Render(vars)
	return text, e, err
}

// DefaultPromptIDs used by the orchestrator planner.
const (
	PromptPlanSystem = "orchestrator.plan.system"
	PromptPlanUser   = "orchestrator.plan.user"
)
