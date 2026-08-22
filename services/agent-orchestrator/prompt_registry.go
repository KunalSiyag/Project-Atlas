package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// PromptVar declares a template variable.
type PromptVar struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
}

// PromptEntry is one versioned prompt template.
type PromptEntry struct {
	// WorkspaceID scopes multi-tenant publish (empty treated as "default" on store).
	WorkspaceID  string      `json:"workspace_id,omitempty" yaml:"workspace_id,omitempty"`
	ID           string      `json:"id" yaml:"id"`
	Version      string      `json:"version" yaml:"version"`
	Status       string      `json:"status" yaml:"status"` // draft | active | retired
	Text         string      `json:"text" yaml:"text"`
	Variables    []PromptVar `json:"variables,omitempty" yaml:"variables,omitempty"`
	ModelClass   string      `json:"model_class,omitempty" yaml:"model_class,omitempty"`
	ApproxTokens int         `json:"approx_tokens,omitempty" yaml:"approx_tokens,omitempty"`
}

// PromptRegistry is an in-memory index of loaded prompt entries.
// Supports file hot-reload (Reload) and optional PromptStore publish/load.
// Multi-tenant isolation for admin HTTP uses PromptStore with workspace keys.
type PromptRegistry struct {
	mu sync.RWMutex
	// byID maps id -> all versions (any status)
	byID  map[string][]PromptEntry
	dir   string      // source directory for Reload; empty if not file-backed
	store PromptStore // optional durable publish path
	// publishWorkspace is the default workspace for Publish/LoadFromStore (default "default").
	publishWorkspace string
}

// NewPromptRegistry returns an empty registry (optionally with a store for publish).
func NewPromptRegistry(store PromptStore) *PromptRegistry {
	return &PromptRegistry{byID: make(map[string][]PromptEntry), store: store}
}

// WithStore attaches or replaces the durable prompt store used by Publish/LoadFromStore.
func (r *PromptRegistry) WithStore(store PromptStore) *PromptRegistry {
	if r == nil {
		return NewPromptRegistry(store)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store = store
	return r
}

// LoadPromptDir walks dir for .json and .yaml/.yml prompt files.
func LoadPromptDir(dir string) (*PromptRegistry, error) {
	reg := &PromptRegistry{byID: make(map[string][]PromptEntry), dir: strings.TrimSpace(dir)}
	if reg.dir == "" {
		return nil, fmt.Errorf("prompt dir is required")
	}
	if err := reg.loadDirLocked(reg.dir); err != nil {
		return nil, err
	}
	return reg, nil
}

// WithPublishWorkspace sets the workspace used by Publish/LoadFromStore/Reload store merge.
func (r *PromptRegistry) WithPublishWorkspace(workspaceID string) *PromptRegistry {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publishWorkspace = NormalizeWorkspace(workspaceID)
	return r
}

func (r *PromptRegistry) storeWorkspace() string {
	if r == nil || r.publishWorkspace == "" {
		return "default"
	}
	return r.publishWorkspace
}

// Reload re-reads the file-backed prompt directory (hot-reload without process restart)
// and re-merges any attached store for the publish workspace. Resolve sees the new contents immediately.
func (r *PromptRegistry) Reload() error {
	if r == nil {
		return fmt.Errorf("prompt registry is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dir == "" {
		return fmt.Errorf("prompt registry has no file directory to reload")
	}
	r.byID = make(map[string][]PromptEntry)
	if err := r.loadDirLocked(r.dir); err != nil {
		return err
	}
	if r.store != nil {
		ws := r.publishWorkspace
		if ws == "" {
			ws = "default"
		}
		entries, err := r.store.List(context.Background(), ws)
		if err != nil {
			return fmt.Errorf("reload store: %w", err)
		}
		for _, e := range entries {
			r.upsertLocked(e)
		}
	}
	return nil
}

// Publish stores entry via PromptStore (required) and merges it into the in-memory index
// so Resolve works without restart. Same id/version keys as file prompts within the workspace.
func (r *PromptRegistry) Publish(ctx context.Context, entry PromptEntry) error {
	if r == nil {
		return fmt.Errorf("prompt registry is nil")
	}
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("prompt id is required")
	}
	if entry.Version == "" {
		entry.Version = "0"
	}
	if entry.Status == "" {
		entry.Status = "active"
	}
	if strings.TrimSpace(entry.Text) == "" {
		return fmt.Errorf("prompt text is required")
	}
	r.mu.Lock()
	store := r.store
	ws := r.publishWorkspace
	r.mu.Unlock()
	if store == nil {
		return fmt.Errorf("prompt store not configured")
	}
	if entry.WorkspaceID != "" {
		ws = entry.WorkspaceID
	}
	if ws == "" {
		ws = "default"
	}
	if err := store.Publish(ctx, ws, entry); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upsertLocked(entry)
	return nil
}

// LoadFromStore loads published prompts for the publish workspace into the index.
func (r *PromptRegistry) LoadFromStore(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("prompt registry is nil")
	}
	r.mu.Lock()
	store := r.store
	ws := r.publishWorkspace
	r.mu.Unlock()
	if store == nil {
		return fmt.Errorf("prompt store not configured")
	}
	if ws == "" {
		ws = "default"
	}
	entries, err := store.List(ctx, ws)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		r.upsertLocked(e)
	}
	return nil
}

func (r *PromptRegistry) upsertLocked(entry PromptEntry) {
	if r.byID == nil {
		r.byID = make(map[string][]PromptEntry)
	}
	list := r.byID[entry.ID]
	for i, e := range list {
		if e.Version == entry.Version {
			list[i] = entry
			r.byID[entry.ID] = list
			return
		}
	}
	r.byID[entry.ID] = append(list, entry)
}

// loadDirLocked reads prompt files into byID. Caller must hold mu (or be sole owner during construction).
func (r *PromptRegistry) loadDirLocked(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("prompt path is not a directory: %s", dir)
	}
	if r.byID == nil {
		r.byID = make(map[string][]PromptEntry)
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
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
		r.upsertLocked(entry)
		return nil
	})
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
	r.mu.RLock()
	defer r.mu.RUnlock()
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
