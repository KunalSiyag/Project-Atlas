package orchestrator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// PluginCatalog is a discovery/registry surface for remote HTTP tool plugins
// (marketplace catalog). Does not run a WASM guest VM (that remains icebox 0006-g).
type PluginCatalog struct {
	mu    sync.RWMutex
	items []ToolDescriptor
	reg   *ToolRegistry
}

// NewPluginCatalog returns a catalog that registers tools into reg when listed.
func NewPluginCatalog(reg *ToolRegistry) *PluginCatalog {
	if reg == nil {
		reg = NewToolRegistry()
	}
	return &PluginCatalog{items: nil, reg: reg}
}

// Register adds a descriptor for discovery and registers the tool on the registry.
func (c *PluginCatalog) Register(d ToolDescriptor) error {
	if c == nil {
		return fmt.Errorf("plugin catalog is nil")
	}
	if err := c.reg.RegisterDescriptor(d); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// replace by name
	for i, existing := range c.items {
		if existing.Name == d.Name {
			c.items[i] = d
			return nil
		}
	}
	c.items = append(c.items, d)
	return nil
}

// List returns catalog descriptors (discovery).
func (c *PluginCatalog) List() []ToolDescriptor {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ToolDescriptor, len(c.items))
	copy(out, c.items)
	return out
}

func (h *Handler) listPlugins(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Plugins == nil {
		writeJSON(w, http.StatusOK, map[string]any{"plugins": []ToolDescriptor{}})
		return
	}
	list := h.Service.Plugins.List()
	if list == nil {
		list = []ToolDescriptor{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": list, "note": "HTTP remote plugins only; WASM guest runtime is icebox"})
}

func (h *Handler) registerPlugin(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Plugins == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "plugin catalog not configured"})
		return
	}
	var d ToolDescriptor
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := h.Service.Plugins.Register(d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, d)
}
