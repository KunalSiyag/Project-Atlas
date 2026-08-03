package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/KunalSiyag/Project-Atlas/services/inference-gateway"
)

func main() {
	config, err := inference.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	provider, err := inference.NewConfiguredProvider(config)
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
	router := inference.StaticRouter{
		Default: inference.Route{Provider: retryingProvider, DefaultModel: config.ModelName},
		Routes: map[string]inference.Route{
			"plan": {Provider: retryingProvider, DefaultModel: config.ModelName},
			"act":  {Provider: retryingProvider, DefaultModel: config.ModelName},
		},
	}
	handler := inference.NewGatewayWithRouterAndStores(router, runStore, conversationStore)
	server := &http.Server{
		Addr:    config.Address,
		Handler: handler,
	}

	log.Printf("inference gateway listening on %s using provider %s", config.Address, config.ProviderName)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
