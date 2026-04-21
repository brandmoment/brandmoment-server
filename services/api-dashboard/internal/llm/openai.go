package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultOpenAIModel = "gpt-4o-mini"

// openAIClient wraps the official openai-go SDK.
type openAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a ChatClient backed by OpenAI using the given API key.
// If model is empty, "gpt-4o-mini" is used.
func NewOpenAIClient(apiKey, model string) ChatClient {
	if model == "" {
		model = defaultOpenAIModel
	}
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &openAIClient{client: &c, model: model}
}

func (o *openAIClient) Provider() Provider { return ProviderOpenAI }

func (o *openAIClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	msgs := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			msgs[i] = openai.SystemMessage(m.Content)
		case RoleUser:
			msgs[i] = openai.UserMessage(m.Content)
		case RoleAssistant:
			msgs[i] = openai.AssistantMessage(m.Content)
		default:
			msgs[i] = openai.UserMessage(m.Content)
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(model),
		Messages:    msgs,
		Temperature: openai.Float(float64(req.Temperature)),
	}

	resp, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai complete: %w", err)
	}
	if len(resp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai complete: no choices returned")
	}

	return ChatResponse{
		Content:      resp.Choices[0].Message.Content,
		InputTokens:  int(resp.Usage.PromptTokens),
		OutputTokens: int(resp.Usage.CompletionTokens),
	}, nil
}
