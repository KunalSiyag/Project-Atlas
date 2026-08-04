package inference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Gateway exposes provider-independent inference through HTTP.
type Gateway struct {
	Router        ModelRouter
	Logger        *slog.Logger
	Telemetry     *Telemetry
	Store         RunStore
	Conversations ConversationStore
	Embeddings    EmbeddingProvider
}

// NewGateway constructs an HTTP gateway around a model provider.
func NewGateway(provider ModelProvider) *Gateway {
	return NewGatewayWithRouter(StaticRouter{Default: Route{
		Provider:     provider,
		DefaultModel: "fake-model",
	}})
}

// NewGatewayWithRouter constructs an HTTP gateway with explicit routing.
func NewGatewayWithRouter(router ModelRouter) *Gateway {
	return newGateway(router, slog.Default(), nil, nil)
}

// NewGatewayWithRouterAndStore enables durable inference-run persistence.
func NewGatewayWithRouterAndStore(router ModelRouter, store RunStore) *Gateway {
	return NewGatewayWithRouterAndStores(router, store, nil)
}

// NewGatewayWithRouterAndStores enables durable run and conversation persistence.
func NewGatewayWithRouterAndStores(router ModelRouter, store RunStore, conversations ConversationStore) *Gateway {
	return newGateway(router, slog.Default(), store, conversations)
}

// NewGatewayWithRouterAndLogger is useful for applications and tests that
// need an explicit structured-log destination.
func NewGatewayWithRouterAndLogger(router ModelRouter, logger *slog.Logger) *Gateway {
	return newGateway(router, logger, nil, nil)
}

func newGateway(router ModelRouter, logger *slog.Logger, store RunStore, conversations ConversationStore) *Gateway {
	if logger == nil {
		logger = slog.Default()
	}
	return &Gateway{
		Router:        router,
		Logger:        logger,
		Telemetry:     NewTelemetry(),
		Store:         store,
		Conversations: conversations,
		Embeddings:    FakeEmbeddingProvider{ProviderName: "fake", Dimensions: 8},
	}
}

// WithEmbeddings replaces the default embedder (tests and main wiring).
func (g *Gateway) WithEmbeddings(provider EmbeddingProvider) *Gateway {
	if g != nil {
		g.Embeddings = provider
	}
	return g
}

// ServeHTTP implements the initial internal HTTP API.
func (g *Gateway) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := request.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = newRequestID()
	}
	writer.Header().Set("X-Request-ID", requestID)

	if request.Method == http.MethodGet && request.URL.Path == "/healthz" {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if request.Method == http.MethodGet && request.URL.Path == "/readyz" {
		if g.Router == nil {
			writeError(writer, http.StatusServiceUnavailable, "not_ready", "provider is not configured")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
		return
	}
	if isOpenAPIPath(request.URL.Path) {
		g.handleOpenAPI(writer, request)
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/generate" {
		g.handleGenerate(writer, request, requestID)
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/embeddings" {
		g.handleEmbeddings(writer, request, requestID)
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/conversations" {
		g.handleCreateConversation(writer, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/v1/conversations/") {
		if request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/messages") {
			g.handlePostConversationMessage(writer, request, requestID)
			return
		}
		if request.Method == http.MethodGet {
			g.handleGetConversation(writer, request, requestID)
			return
		}
	}
	if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v1/runs/") {
		g.handleGetRun(writer, request, requestID)
		return
	}

	writeError(writer, http.StatusNotFound, "not_found", "route not found")
}

func (g *Gateway) handlePostConversationMessage(writer http.ResponseWriter, request *http.Request, requestID string) {
	if g.Conversations == nil || g.Router == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "conversation persistence and provider are required")
		return
	}
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" && g.Store == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "idempotent conversation messages require run persistence")
		return
	}
	conversationID := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v1/conversations/"), "/messages")
	if conversationID == "" || strings.Contains(conversationID, "/") {
		writeError(writer, http.StatusBadRequest, "invalid_request", "conversation id is required")
		return
	}
	var input struct {
		Content   string            `json:"content"`
		TaskClass string            `json:"task_class,omitempty"`
		Model     string            `json:"model,omitempty"`
		Metadata  map[string]string `json:"metadata,omitempty"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		writeError(writer, http.StatusBadRequest, "invalid_request", "content is required and must be valid JSON")
		return
	}
	conversation, err := g.Conversations.GetConversation(request.Context(), conversationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(writer, http.StatusNotFound, "not_found", "conversation not found")
			return
		}
		writeError(writer, http.StatusInternalServerError, "persistence_error", "conversation could not be loaded")
		return
	}
	provider, model, err := g.Router.Resolve(input.TaskClass, input.Model)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	runID := input.Metadata["run_id"]
	if runID == "" {
		runID = newRequestID()
	}
	writer.Header().Set("X-Run-ID", runID)
	metadata := withRequestAndRunID(input.Metadata, requestID, runID)
	userMessage, err := g.Conversations.AppendMessage(request.Context(), conversationID, "user", input.Content, runID, idempotencyKey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(writer, http.StatusNotFound, "not_found", "conversation not found")
			return
		}
		if strings.Contains(err.Error(), "not open") {
			writeError(writer, http.StatusConflict, "conversation_closed", "conversation is not open")
			return
		}
		if strings.Contains(err.Error(), "different content") {
			writeError(writer, http.StatusConflict, "idempotency_conflict", "idempotency key was already used with different content")
			return
		}
		g.logger().ErrorContext(request.Context(), "failed to append user message", "request_id", requestID, "conversation_id", conversationID, "error", err)
		writeError(writer, http.StatusServiceUnavailable, "persistence_error", "user message could not be persisted")
		return
	}
	runID = userMessage.RunID
	writer.Header().Set("X-Run-ID", runID)
	if userMessage.idempotencyKey != "" {
		view, err := g.Store.GetRun(request.Context(), runID)
		if err == nil && view.Status == "succeeded" && view.Result != nil {
			writeJSON(writer, http.StatusOK, *view.Result)
			return
		}
		if err == nil && view.Status == "running" {
			writeError(writer, http.StatusConflict, "request_in_progress", "an identical request is already in progress")
			return
		}
		if err == nil && view.Status == "failed" {
			writeError(writer, http.StatusInternalServerError, view.ErrorClass, "the identical request previously failed")
			return
		}
	}
	messages := make([]Message, 0, len(conversation.Messages)+1)
	for _, message := range conversation.Messages {
		messages = append(messages, Message{Role: message.Role, Content: message.Content})
	}
	messages = append(messages, Message{Role: userMessage.Role, Content: userMessage.Content})
	started := time.Now()
	if g.Store != nil {
		if err := g.Store.StartRun(request.Context(), InferenceRun{
			RunID:          runID,
			RequestID:      requestID,
			TaskClass:      input.TaskClass,
			RequestedModel: input.Model,
			ResolvedModel:  model,
			Status:         "running",
			Messages:       messages,
			StartedAt:      started,
		}); err != nil {
			writeError(writer, http.StatusServiceUnavailable, "persistence_error", "inference run could not be persisted")
			return
		}
	}
	result, err := provider.Generate(request.Context(), ChatRequest{Messages: messages, TaskClass: input.TaskClass, Model: model, Metadata: metadata})
	if err != nil {
		if g.Store != nil {
			_ = g.Store.FailRun(request.Context(), runID, errorClass(err), err.Error(), time.Now())
		}
		writeError(writer, http.StatusInternalServerError, "provider_error", "inference request failed")
		return
	}
	if g.Store != nil {
		if err := g.Store.CompleteRun(request.Context(), runID, result, time.Now()); err != nil {
			writeError(writer, http.StatusInternalServerError, "persistence_error", "inference result could not be persisted")
			return
		}
	}
	if _, err := g.Conversations.AppendMessage(request.Context(), conversationID, result.Message.Role, result.Message.Content, runID, ""); err != nil {
		g.logger().ErrorContext(request.Context(), "failed to append assistant message", "request_id", requestID, "conversation_id", conversationID, "run_id", runID, "error", err)
		writeError(writer, http.StatusInternalServerError, "persistence_error", "assistant message could not be persisted")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (g *Gateway) handleCreateConversation(writer http.ResponseWriter, request *http.Request) {
	if g.Conversations == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "conversation persistence is not configured")
		return
	}
	var input struct {
		Title string `json:"title"`
	}
	// Empty body is allowed (title defaults to empty) so simple POSTs work.
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	conversation, err := g.Conversations.CreateConversation(request.Context(), input.Title)
	if err != nil {
		g.logger().ErrorContext(request.Context(), "failed to create conversation", "error", err)
		writeError(writer, http.StatusInternalServerError, "persistence_error", "conversation could not be created")
		return
	}
	writeJSON(writer, http.StatusCreated, conversation)
}

func (g *Gateway) handleGetConversation(writer http.ResponseWriter, request *http.Request, requestID string) {
	if g.Conversations == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "conversation persistence is not configured")
		return
	}
	conversationID := strings.TrimPrefix(request.URL.Path, "/v1/conversations/")
	if conversationID == "" || strings.Contains(conversationID, "/") {
		writeError(writer, http.StatusBadRequest, "invalid_request", "conversation id is required")
		return
	}
	conversation, err := g.Conversations.GetConversation(request.Context(), conversationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(writer, http.StatusNotFound, "not_found", "conversation not found")
			return
		}
		g.logger().ErrorContext(request.Context(), "failed to get conversation", "request_id", requestID, "conversation_id", conversationID, "error", err)
		writeError(writer, http.StatusInternalServerError, "persistence_error", "conversation could not be loaded")
		return
	}
	writeJSON(writer, http.StatusOK, conversation)
}

func (g *Gateway) handleGetRun(writer http.ResponseWriter, request *http.Request, requestID string) {
	if g.Store == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "run persistence is not configured")
		return
	}
	runID := strings.TrimPrefix(request.URL.Path, "/v1/runs/")
	if runID == "" || strings.Contains(runID, "/") {
		writeError(writer, http.StatusBadRequest, "invalid_request", "run id is required")
		return
	}
	view, err := g.Store.GetRun(request.Context(), runID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(writer, http.StatusNotFound, "not_found", "run not found")
			return
		}
		g.logger().ErrorContext(request.Context(), "failed to get inference run", "request_id", requestID, "run_id", runID, "error", err)
		writeError(writer, http.StatusInternalServerError, "persistence_error", "run could not be loaded")
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (g *Gateway) handleEmbeddings(writer http.ResponseWriter, request *http.Request, requestID string) {
	if g.Embeddings == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "embedding provider is not configured")
		return
	}
	var input EmbeddingRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	if len(input.Inputs) == 0 {
		writeError(writer, http.StatusBadRequest, "invalid_request", "inputs are required")
		return
	}
	ctx, span := g.telemetry().StartRequest(request.Context(), requestID)
	defer span.End()
	// reuse request counter with a synthetic task class for embeddings
	g.telemetry().RecordStarted(ctx, "embed")
	started := time.Now()
	result, err := g.Embeddings.Embed(ctx, input)
	if err != nil {
		span.RecordError(err)
		g.telemetry().RecordFailure(ctx, "embed")
		status := http.StatusInternalServerError
		code := "provider_error"
		var pe *ProviderError
		if errors.As(err, &pe) {
			code = pe.Class
			if pe.Class == "invalid_request" {
				status = http.StatusBadRequest
			}
			if pe.Class == "auth" {
				status = http.StatusUnauthorized
			}
		}
		writeError(writer, status, code, "embedding request failed")
		return
	}
	g.telemetry().RecordCompleted(ctx, "embed", result.Provider, started)
	writeJSON(writer, http.StatusOK, result)
}

func (g *Gateway) handleGenerate(writer http.ResponseWriter, request *http.Request, requestID string) {
	if g.Router == nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "provider is not configured")
		return
	}

	var input ChatRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	if len(input.Messages) == 0 {
		writeError(writer, http.StatusBadRequest, "invalid_request", "messages are required")
		return
	}

	provider, model, err := g.Router.Resolve(input.TaskClass, input.Model)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	requestedModel := input.Model
	input.Model = model
	runID := input.Metadata["run_id"]
	if runID == "" {
		runID = newRequestID()
	}
	writer.Header().Set("X-Run-ID", runID)
	input.Metadata = withRequestAndRunID(input.Metadata, requestID, runID)
	ctx, span := g.telemetry().StartRequest(request.Context(), requestID)
	defer span.End()
	started := time.Now()
	g.telemetry().RecordStarted(ctx, input.TaskClass)
	g.logger().InfoContext(ctx, "inference request started",
		"request_id", requestID,
		"task_class", input.TaskClass,
		"model", model,
	)
	if g.Store != nil {
		if err := g.Store.StartRun(ctx, InferenceRun{
			RunID:          runID,
			RequestID:      requestID,
			TaskClass:      input.TaskClass,
			RequestedModel: requestedModel,
			ResolvedModel:  model,
			Status:         "running",
			Messages:       input.Messages,
			StartedAt:      started,
		}); err != nil {
			g.logger().ErrorContext(ctx, "inference run persistence failed", "request_id", requestID, "run_id", runID, "error", err)
			writeError(writer, http.StatusServiceUnavailable, "persistence_error", "inference run could not be persisted")
			return
		}
	}
	result, err := provider.Generate(ctx, input)
	if err != nil {
		span.RecordError(err)
		g.telemetry().RecordFailure(ctx, input.TaskClass)
		g.logger().ErrorContext(ctx, "inference request failed",
			"request_id", requestID,
			"error", err,
		)
		if g.Store != nil {
			if storeErr := g.Store.FailRun(ctx, runID, errorClass(err), err.Error(), time.Now()); storeErr != nil {
				g.logger().ErrorContext(ctx, "failed to persist inference failure", "request_id", requestID, "run_id", runID, "error", storeErr)
			}
		}
		status := http.StatusInternalServerError
		code := "provider_error"
		if err == context.Canceled || err == context.DeadlineExceeded {
			status = http.StatusRequestTimeout
			code = "timeout"
		}
		writeError(writer, status, code, "inference request failed")
		return
	}

	g.telemetry().RecordCompleted(ctx, input.TaskClass, result.Provider, started)
	if g.Store != nil {
		if err := g.Store.CompleteRun(ctx, runID, result, time.Now()); err != nil {
			g.logger().ErrorContext(ctx, "failed to persist inference result", "request_id", requestID, "run_id", runID, "error", err)
			writeError(writer, http.StatusInternalServerError, "persistence_error", "inference result could not be persisted")
			return
		}
	}
	g.logger().InfoContext(ctx, "inference request completed",
		"request_id", requestID,
		"provider", result.Provider,
		"model", result.Model,
		"finish_reason", result.FinishReason,
	)
	writeJSON(writer, http.StatusOK, result)
}

func (g *Gateway) logger() *slog.Logger {
	if g.Logger == nil {
		return slog.Default()
	}
	return g.Logger
}

func (g *Gateway) telemetry() *Telemetry {
	if g.Telemetry == nil {
		return NewTelemetry()
	}
	return g.Telemetry
}

func withRequestAndRunID(metadata map[string]string, requestID, runID string) map[string]string {
	result := make(map[string]string, len(metadata)+2)
	for key, value := range metadata {
		result[key] = value
	}
	result["request_id"] = requestID
	result["run_id"] = runID
	return result
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
