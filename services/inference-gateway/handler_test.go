package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGatewayGeneratesThroughProvider(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/generate", strings.NewReader(`{"model":"fake-model","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var result ChatResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "fake response to: hello" {
		t.Fatalf("content = %q, want fake response", result.Message.Content)
	}
}

func TestGatewayRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()

	response, err := server.Client().Post(server.URL+"/v1/generate", "application/json", strings.NewReader("{"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}
}

func TestGatewayHealthAndReadiness(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()

	for _, path := range []string{"/healthz", "/readyz"} {
		response, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, response.StatusCode)
		}
	}
}

func TestGatewayPassesRequestContextToProvider(t *testing.T) {
	provider := &contextCheckingProvider{}
	server := httptest.NewServer(NewGateway(provider))
	defer server.Close()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/v1/generate", strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !provider.called {
		t.Fatal("provider was not called")
	}
}

func TestGatewayPropagatesRequestIDToResponseAndProvider(t *testing.T) {
	provider := &metadataCheckingProvider{}
	server := httptest.NewServer(NewGateway(provider))
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/generate", strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Request-ID", "request-123")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.Header.Get("X-Request-ID") != "request-123" {
		t.Fatalf("response request id = %q, want request-123", response.Header.Get("X-Request-ID"))
	}
	if provider.metadata["request_id"] != "request-123" {
		t.Fatalf("provider metadata = %+v, want request id", provider.metadata)
	}
}

func TestGatewayPersistsRunLifecycle(t *testing.T) {
	store := &recordingRunStore{}
	server := httptest.NewServer(NewGatewayWithRouterAndStore(
		StaticRouter{Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "fake-model"}},
		store,
	))
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/generate", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if store.started.RunID == "" || store.completedRunID != store.started.RunID {
		t.Fatalf("store lifecycle = started %+v, completed %q", store.started, store.completedRunID)
	}
	if store.result.Message.Content == "" {
		t.Fatal("stored result is empty")
	}
}

func TestGatewayGetsPersistedRun(t *testing.T) {
	store := &recordingRunStore{view: InferenceRunView{RunID: "run-1", Status: "succeeded"}}
	server := httptest.NewServer(NewGatewayWithRouterAndStore(
		StaticRouter{Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "fake-model"}},
		store,
	))
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/v1/runs/run-1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var view InferenceRunView
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.RunID != "run-1" || view.Status != "succeeded" {
		t.Fatalf("view = %+v, want run-1/succeeded", view)
	}
}

func TestGatewayCreatesAndGetsConversation(t *testing.T) {
	store := &recordingConversationStore{conversation: Conversation{ConversationID: "conversation-1", Title: "Planning", Status: "open", Messages: []ConversationMessage{}}}
	server := httptest.NewServer(NewGatewayWithRouterAndStores(
		StaticRouter{Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "fake-model"}},
		nil,
		store,
	))
	defer server.Close()

	response, err := server.Client().Post(server.URL+"/v1/conversations", "application/json", strings.NewReader(`{"title":"Planning"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", response.StatusCode)
	}
	if store.createdTitle != "Planning" {
		t.Fatalf("created title = %q, want Planning", store.createdTitle)
	}

	response, err = server.Client().Get(server.URL + "/v1/conversations/conversation-1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", response.StatusCode)
	}
	var conversation Conversation
	if err := json.NewDecoder(response.Body).Decode(&conversation); err != nil {
		t.Fatal(err)
	}
	if conversation.ConversationID != "conversation-1" {
		t.Fatalf("conversation = %+v, want conversation-1", conversation)
	}
}

func TestGatewayPostsConversationMessage(t *testing.T) {
	store := &recordingConversationStore{conversation: Conversation{
		ConversationID: "conversation-1",
		Status:         "open",
		Messages:       []ConversationMessage{{Role: "user", Content: "Earlier context"}},
	}}
	server := httptest.NewServer(NewGatewayWithRouterAndStores(
		StaticRouter{Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "fake-model"}},
		nil,
		store,
	))
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/conversations/conversation-1/messages", strings.NewReader(`{"content":"Continue the conversation"}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var result ChatResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "fake response to: Continue the conversation" {
		t.Fatalf("result = %+v", result)
	}
	if len(store.appended) != 2 || store.appended[0].Role != "user" || store.appended[1].Role != "assistant" {
		t.Fatalf("appended messages = %+v, want user and assistant", store.appended)
	}
}

func TestGatewayReusesIdempotentConversationMessage(t *testing.T) {
	runStore := &recordingRunStore{}
	conversationStore := &recordingConversationStore{conversation: Conversation{ConversationID: "conversation-1", Status: "open"}}
	server := httptest.NewServer(NewGatewayWithRouterAndStores(
		StaticRouter{Default: Route{Provider: FakeProvider{ProviderName: "fake"}, DefaultModel: "fake-model"}},
		runStore,
		conversationStore,
	))
	defer server.Close()

	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/conversations/conversation-1/messages", strings.NewReader(`{"content":"Retry me"}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Idempotency-Key", "message-key-1")
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", attempt+1, response.StatusCode)
		}
	}
	if len(conversationStore.appended) != 2 {
		t.Fatalf("appended messages = %d, want one user and one assistant", len(conversationStore.appended))
	}
}

type contextCheckingProvider struct{ called bool }

func (p *contextCheckingProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	p.called = true
	return ChatResponse{Message: Message{Role: "assistant", Content: "ok"}}, nil
}

type metadataCheckingProvider struct{ metadata map[string]string }

func (p *metadataCheckingProvider) Generate(_ context.Context, request ChatRequest) (ChatResponse, error) {
	p.metadata = request.Metadata
	return ChatResponse{Message: Message{Role: "assistant", Content: "ok"}}, nil
}

type recordingRunStore struct {
	started        InferenceRun
	completedRunID string
	result         ChatResponse
	view           InferenceRunView
}

func (s *recordingRunStore) StartRun(_ context.Context, run InferenceRun) error {
	s.started = run
	return nil
}

func (s *recordingRunStore) CompleteRun(_ context.Context, runID string, result ChatResponse, _ time.Time) error {
	s.completedRunID = runID
	s.result = result
	s.view = InferenceRunView{RunID: runID, Status: "succeeded", Result: &result}
	return nil
}

func (s *recordingRunStore) FailRun(context.Context, string, string, string, time.Time) error {
	return nil
}

func (s *recordingRunStore) GetRun(context.Context, string) (InferenceRunView, error) {
	return s.view, nil
}

type recordingConversationStore struct {
	createdTitle string
	conversation Conversation
	appended     []ConversationMessage
}

func (s *recordingConversationStore) CreateConversation(_ context.Context, title string) (Conversation, error) {
	s.createdTitle = title
	return s.conversation, nil
}

func (s *recordingConversationStore) GetConversation(context.Context, string) (Conversation, error) {
	return s.conversation, nil
}

func (s *recordingConversationStore) AppendMessage(_ context.Context, _ string, role, content, runID, idempotencyKey string) (ConversationMessage, error) {
	for _, existing := range s.appended {
		if idempotencyKey != "" && existing.idempotencyKey == idempotencyKey {
			return existing, nil
		}
	}
	message := ConversationMessage{MessageID: "message-" + string(rune(len(s.appended)+1)), Role: role, Content: content, RunID: runID}
	message.idempotencyKey = idempotencyKey
	s.appended = append(s.appended, message)
	return message, nil
}
