// Package llm provides a provider-agnostic chat completion interface and
// implementations for OpenAI and Gemini.
package llm

import "context"

// Provider identifies an LLM backend.
type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderGemini Provider = "gemini"
)

// Role identifies the author of a chat message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single turn in a chat conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the input to ChatClient.Complete.
type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature"`
	// Model is the provider-specific model identifier (e.g. "gpt-4o-mini").
	// If empty the client uses its configured default.
	Model string `json:"model,omitempty"`
}

// ChatResponse is the output from ChatClient.Complete.
type ChatResponse struct {
	Content      string `json:"content"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// ChatClient is the common interface for LLM chat completion providers.
type ChatClient interface {
	// Complete sends the request and returns the assistant's reply.
	Complete(ctx context.Context, req ChatRequest) (ChatResponse, error)
	// Provider returns the provider identifier.
	Provider() Provider
}
