package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KunalSiyag/Project-Atlas/services/inference-gateway"
)

func main() {
	config, err := inference.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	shutdownTelemetry, err := inference.InstallTelemetry(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(c); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}()

	provider, err := inference.NewConfiguredProvider(config)
	if err != nil {
		log.Fatal(err)
	}
	embedder, err := inference.NewConfiguredEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	retryingProvider := inference.RetryingProvider{
		Provider: provider,
		Policy: inference.RetryPolicy{
			MaxAttempts: config.MaxAttempts,
			Backoff:     100 * time.Millisecond,
		},
	}
	var runStore inference.RunStore
	var conversationStore inference.ConversationStore
	if config.DatabaseURL != "" {
		store, err := inference.OpenPostgresRunStore(context.Background(), config.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer store.DB.Close()
		runStore = store
		conversationStore, err = inference.NewPostgresConversationStore(context.Background(), store.DB)
		if err != nil {
			log.Fatal(err)
		}
	}
	router := inference.DynamicRouter{
		Config:  config,
		Default: inference.Route{Provider: retryingProvider, DefaultModel: config.ModelName},
		Routes: map[string]inference.Route{
			"plan": {Provider: retryingProvider, DefaultModel: config.ModelName},
			"act":  {Provider: retryingProvider, DefaultModel: config.ModelName},
		},
	}
	handler := inference.NewGatewayWithRouterAndStores(router, runStore, conversationStore).WithEmbeddings(embedder)
	server := &http.Server{
		Addr:    config.Address,
		Handler: handler,
	}

	go func() {
		log.Printf("inference gateway listening on %s using provider %s embeddings=%s otel=%s",
			config.Address, config.ProviderName, config.EmbeddingProvider, config.OTELExporter)
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
}
