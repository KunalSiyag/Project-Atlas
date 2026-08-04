package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"strings"
)

// EmbeddingRequest is the normalized embedding batch request.
type EmbeddingRequest struct {
	Inputs   []string          `json:"inputs"`
	Model    string            `json:"model,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmbeddingResponse is the normalized embedding batch result.
type EmbeddingResponse struct {
	Vectors    [][]float64 `json:"vectors"`
	Dimensions int         `json:"dimensions"`
	Provider   string      `json:"provider"`
	Model      string      `json:"model"`
	Usage      Usage       `json:"usage,omitempty"`
}

// EmbeddingProvider is the port for text → vector encoding.
type EmbeddingProvider interface {
	Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error)
}

// FakeEmbeddingProvider returns deterministic low-dimensional vectors without network I/O.
type FakeEmbeddingProvider struct {
	ProviderName string
	Dimensions   int
}

func (p FakeEmbeddingProvider) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	if err := ctx.Err(); err != nil {
		return EmbeddingResponse{}, err
	}
	if len(request.Inputs) == 0 {
		return EmbeddingResponse{}, fmt.Errorf("at least one input is required")
	}
	dims := p.Dimensions
	if dims <= 0 {
		dims = 8
	}
	name := p.ProviderName
	if name == "" {
		name = "fake"
	}
	model := request.Model
	if model == "" {
		model = "fake-embed"
	}
	vectors := make([][]float64, 0, len(request.Inputs))
	for _, text := range request.Inputs {
		vectors = append(vectors, fakeVector(text, dims))
	}
	return EmbeddingResponse{
		Vectors:    vectors,
		Dimensions: dims,
		Provider:   name,
		Model:      model,
		Usage:      Usage{InputTokens: len(request.Inputs), OutputTokens: 0},
	}, nil
}

func fakeVector(text string, dims int) []float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	out := make([]float64, dims)
	var sumSquares float64
	for i := 0; i < dims; i++ {
		seed = seed*6364136223846793005 + 1
		v := float64(int64(seed>>33)%1000)/500.0 - 1.0
		out[i] = v
		sumSquares += v * v
	}
	norm := math.Sqrt(sumSquares)
	if norm == 0 {
		return out
	}
	for i := range out {
		out[i] /= norm
	}
	return out
}

// NewConfiguredEmbedder selects an embedding adapter from config.
func NewConfiguredEmbedder(config Config) (EmbeddingProvider, error) {
	switch strings.ToLower(strings.TrimSpace(config.EmbeddingProvider)) {
	case "", "fake":
		return FakeEmbeddingProvider{ProviderName: "fake", Dimensions: 8}, nil
	case "openai":
		return OpenAIEmbeddingProvider{
			BaseURL: config.OpenAIBaseURL,
			APIKey:  config.OpenAIAPIKey,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", config.EmbeddingProvider)
	}
}

// OpenAIEmbeddingProvider calls OpenAI-compatible /embeddings.
type OpenAIEmbeddingProvider struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type openAIEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p OpenAIEmbeddingProvider) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	if len(request.Inputs) == 0 {
		return EmbeddingResponse{}, fmt.Errorf("at least one input is required")
	}
	model := request.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if p.Client == nil {
		p.Client = http.DefaultClient
	}
	body, err := json.Marshal(openAIEmbedRequest{Model: model, Input: request.Inputs})
	if err != nil {
		return EmbeddingResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return EmbeddingResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(p.APIKey); key != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+key)
	}
	httpResponse, err := p.Client.Do(httpRequest)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("call openai embeddings: %w", err)
	}
	defer httpResponse.Body.Close()
	raw, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return EmbeddingResponse{}, err
	}
	var parsed openAIEmbedResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return EmbeddingResponse{}, fmt.Errorf("decode embeddings: %w", err)
	}
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		class, retryable := classifyHTTPStatus(httpResponse.StatusCode)
		msg := string(raw)
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return EmbeddingResponse{}, &ProviderError{
			Class:     class,
			Retryable: retryable,
			Err:       fmt.Errorf("openai embeddings HTTP %d: %s", httpResponse.StatusCode, msg),
		}
	}
	if len(parsed.Data) == 0 {
		return EmbeddingResponse{}, fmt.Errorf("openai embeddings response empty")
	}
	vectors := make([][]float64, len(parsed.Data))
	dims := 0
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(vectors) {
			return EmbeddingResponse{}, fmt.Errorf("invalid embedding index %d", item.Index)
		}
		vectors[item.Index] = item.Embedding
		if dims == 0 {
			dims = len(item.Embedding)
		}
	}
	for i, v := range vectors {
		if len(v) == 0 {
			return EmbeddingResponse{}, fmt.Errorf("missing embedding at index %d", i)
		}
	}
	outModel := parsed.Model
	if outModel == "" {
		outModel = model
	}
	return EmbeddingResponse{
		Vectors:    vectors,
		Dimensions: dims,
		Provider:   "openai",
		Model:      outModel,
		Usage:      Usage{InputTokens: parsed.Usage.PromptTokens},
	}, nil
}
