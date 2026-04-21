package llm

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const defaultGeminiModel = "gemini-2.0-flash"

// geminiClient wraps the Google generative-ai-go SDK.
type geminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient creates a ChatClient backed by Google Gemini using the given API key.
// If model is empty, "gemini-2.0-flash" is used.
func NewGeminiClient(ctx context.Context, apiKey, model string) (ChatClient, error) {
	if model == "" {
		model = defaultGeminiModel
	}
	c, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &geminiClient{client: c, model: model}, nil
}

func (g *geminiClient) Provider() Provider { return ProviderGemini }

func (g *geminiClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = g.model
	}

	m := g.client.GenerativeModel(modelName)
	m.SetTemperature(req.Temperature)

	// Separate system prompt from conversation turns.
	var systemText string
	var parts []genai.Part
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleSystem:
			systemText += msg.Content + "\n"
		default:
			parts = append(parts, genai.Text(msg.Content))
		}
	}
	if systemText != "" {
		m.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemText)},
		}
	}

	cs := m.StartChat()
	// Feed all but the last user message into history.
	userParts := parts
	if len(userParts) == 0 {
		return ChatResponse{}, fmt.Errorf("gemini complete: no user messages")
	}
	// Send the last user turn.
	lastPart := userParts[len(userParts)-1]
	resp, err := cs.SendMessage(ctx, lastPart)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini complete: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return ChatResponse{}, fmt.Errorf("gemini complete: empty response")
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return ChatResponse{}, fmt.Errorf("gemini complete: unexpected response part type")
	}

	var inputTok, outputTok int
	if resp.UsageMetadata != nil {
		inputTok = int(resp.UsageMetadata.PromptTokenCount)
		outputTok = int(resp.UsageMetadata.CandidatesTokenCount)
	}

	return ChatResponse{
		Content:      string(text),
		InputTokens:  inputTok,
		OutputTokens: outputTok,
	}, nil
}
