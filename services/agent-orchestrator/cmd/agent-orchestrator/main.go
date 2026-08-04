package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KunalSiyag/Project-Atlas/services/agent-orchestrator"
)

func main() {
	addr := envOr("ATLAS_ORCHESTRATOR_ADDR", ":8090")
	gatewayURL := envOr("ATLAS_INFERENCE_GATEWAY_URL", "http://127.0.0.1:8080")
	model := envOr("ATLAS_INFERENCE_MODEL", "fake-model")
	dbURL := firstEnv("ATLAS_ORCHESTRATOR_DATABASE_URL", "ATLAS_DATABASE_URL")
	qdrantURL := envOr("ATLAS_QDRANT_URL", "http://127.0.0.1:6333")
	workspace := envOr("ATLAS_WORKSPACE_ID", "default")

	ctx := context.Background()
	var store orchestrator.RunStore = orchestrator.NewMemoryStore()
	var pg *orchestrator.PostgresStore
	if dbURL != "" {
		s, err := orchestrator.OpenPostgresStore(ctx, dbURL)
		if err != nil {
			log.Fatal(err)
		}
		pg = s
		store = s
		log.Printf("orchestrator persistence: postgres")
	} else {
		log.Printf("orchestrator persistence: memory")
	}

	var mem orchestrator.MemoryStore = orchestrator.NewInMemoryMemory()
	var pgMem *orchestrator.PostgresMemory
	var redisMem *orchestrator.RedisShortMemory
	if dbURL != "" {
		m, err := orchestrator.OpenPostgresMemory(ctx, dbURL)
		if err != nil {
			log.Printf("memory postgres unavailable, using in-memory: %v", err)
		} else {
			pgMem = m
			mem = m
			log.Printf("memory: postgres long-term + in-process short-term")
		}
	}
	// Optional Redis short-term (ATLAS_REDIS_URL=redis://host:6379/0).
	if redisURL := strings.TrimSpace(os.Getenv("ATLAS_REDIS_URL")); redisURL != "" {
		rm, err := orchestrator.OpenRedisShortMemory(ctx, redisURL, mem)
		if err != nil {
			log.Printf("redis short-term unavailable, keeping previous memory store: %v", err)
		} else {
			redisMem = rm
			mem = rm
			log.Printf("memory: redis short-term + long-term backend")
		}
	}

	embedder := &orchestrator.GatewayEmbedder{BaseURL: gatewayURL, Model: "fake-embed"}
	// Prefer Qdrant with in-process fallback always attached.
	retriever := orchestrator.NewQdrantRetriever(qdrantURL, embedder)

	gateway := orchestrator.NewHTTPGatewayClient(gatewayURL, model)
	// Hybrid LLM planner with deterministic rule fallback + file-backed prompts.
	planner := orchestrator.NewLLMPlanner(gateway, model, orchestrator.NewRulePlanner())
	promptDir := envOr("ATLAS_PROMPTS_DIR", "")
	if promptDir == "" {
		// default: repo prompts/ relative to cwd, or alongside binary search
		for _, c := range []string{"prompts", filepath.Join("..", "..", "prompts"), filepath.Join("..", "prompts")} {
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				promptDir = c
				break
			}
		}
	}
	if promptDir != "" {
		if reg, err := orchestrator.LoadPromptDir(promptDir); err != nil {
			log.Printf("prompt registry unavailable (%s): %v", promptDir, err)
		} else {
			planner = planner.WithPrompts(reg)
			log.Printf("prompt registry: %s", promptDir)
		}
	}
	svc := orchestrator.NewService(orchestrator.ServiceConfig{
		Store:       store,
		Gateway:     gateway,
		Builder:     orchestrator.NewContextBuilder(),
		Tools:       orchestrator.DefaultToolRegistry(),
		Memory:      mem,
		Retriever:   retriever,
		Planner:     planner,
		Model:       model,
		TokenBudget: 512,
		WorkspaceID: workspace,
	})
	handler := &orchestrator.Handler{Service: svc}
	// Light workspace isolation when required.
	var h http.Handler = handler
	if envOr("ATLAS_REQUIRE_WORKSPACE", "") == "1" {
		h = requireWorkspace(handler)
	}
	server := &http.Server{Addr: addr, Handler: h}

	go func() {
		log.Printf("agent-orchestrator listening on %s gateway=%s", addr, gatewayURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	if pg != nil {
		_ = pg.Close()
	}
	if pgMem != nil {
		_ = pgMem.Close()
	}
	if redisMem != nil {
		_ = redisMem.Close()
	}
}

func requireWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.TrimSpace(r.Header.Get("X-Workspace-ID")) == "" {
			http.Error(w, `{"error":"X-Workspace-ID required"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
