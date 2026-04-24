package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultEmbedModel = "text-embedding-3-small"

// EmbedResponse is the result of a single text embedding call.
type EmbedResponse struct {
	// Vector is the embedding as float32 slice (converted from float64 API response).
	Vector []float32
	// InputTokens is the number of prompt tokens consumed.
	InputTokens int
}

// EmbedClient is the common interface for text-embedding providers.
type EmbedClient interface {
	// Embed returns the embedding vector for the given text.
	// Returns an error if text is empty or the API call fails.
	Embed(ctx context.Context, text string) (EmbedResponse, error)
}

// openAIEmbedClient wraps the official openai-go SDK for embeddings.
type openAIEmbedClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIEmbedClient creates an EmbedClient backed by OpenAI.
// If model is empty, "text-embedding-3-small" is used.
func NewOpenAIEmbedClient(apiKey, model string) EmbedClient {
	if model == "" {
		model = defaultEmbedModel
	}
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &openAIEmbedClient{client: &c, model: model}
}

// Embed sends a single text to the OpenAI Embeddings API.
// Returns an error if text is empty.
func (e *openAIEmbedClient) Embed(ctx context.Context, text string) (EmbedResponse, error) {
	if text == "" {
		return EmbedResponse{}, fmt.Errorf("embed input: text must not be empty")
	}

	params := openai.EmbeddingNewParams{
		Model: e.model,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	}

	resp, err := e.client.Embeddings.New(ctx, params)
	if err != nil {
		return EmbedResponse{}, fmt.Errorf("embed openai: %w", err)
	}
	if len(resp.Data) == 0 {
		return EmbedResponse{}, fmt.Errorf("embed openai: no embedding returned")
	}

	raw := resp.Data[0].Embedding
	vec := make([]float32, len(raw))
	for i, v := range raw {
		vec[i] = float32(v)
	}

	return EmbedResponse{
		Vector:      vec,
		InputTokens: int(resp.Usage.PromptTokens),
	}, nil
}
