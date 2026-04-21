package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const selfCheckVerifyPromptTemplate = `You are a strict auditor.

Given the original phrase and the parsed JSON rules array, answer:
Does the JSON array EXACTLY and COMPLETELY capture the intent of the phrase?

Respond with exactly one word on the first line: YES or NO
Then on the next line explain briefly what is correct or what is missing/wrong.

Original phrase: %s

Parsed rules:
%s`

// SelfCheckResult is the outcome of the self-check confidence approach.
type SelfCheckResult struct {
	Status       ConfidenceStatus `json:"status"`
	Explanation  string           `json:"explanation"`
	Latency      time.Duration    `json:"-"`
	InputTokens  int              `json:"input_tokens"`
	OutputTokens int              `json:"output_tokens"`
}

// CheckSelfCheck performs two LLM calls:
//  1. Parse phrase → rules JSON.
//  2. Ask the model to verify whether the JSON matches the phrase.
//
// Returns the rules JSON from call 1 and the verification result.
func CheckSelfCheck(ctx context.Context, client ChatClient, phrase string) (string, SelfCheckResult) {
	start := time.Now()
	totalIn, totalOut := 0, 0

	// Call 1: parse.
	parseResp, err := client.Complete(ctx, ChatRequest{
		Temperature: 0,
		Messages: []Message{
			{Role: RoleSystem, Content: SystemPrompt},
			{Role: RoleUser, Content: phrase},
		},
	})
	if err != nil {
		return "", SelfCheckResult{Status: ConfidenceStatusFail, Latency: time.Since(start)}
	}
	totalIn += parseResp.InputTokens
	totalOut += parseResp.OutputTokens
	rulesJSON := strings.TrimSpace(parseResp.Content)

	// Call 2: verify.
	verifyPrompt := fmt.Sprintf(selfCheckVerifyPromptTemplate, phrase, rulesJSON)
	verifyResp, err := client.Complete(ctx, ChatRequest{
		Temperature: 0,
		Messages: []Message{
			{Role: RoleUser, Content: verifyPrompt},
		},
	})
	if err != nil {
		return rulesJSON, SelfCheckResult{
			Status:       ConfidenceStatusUnsure,
			Explanation:  "verify call failed",
			Latency:      time.Since(start),
			InputTokens:  totalIn,
			OutputTokens: totalOut,
		}
	}
	totalIn += verifyResp.InputTokens
	totalOut += verifyResp.OutputTokens

	verdict, explanation := parseSelfCheckResponse(verifyResp.Content)
	status := ConfidenceStatusOK
	if !verdict {
		status = ConfidenceStatusUnsure
	}

	return rulesJSON, SelfCheckResult{
		Status:       status,
		Explanation:  explanation,
		Latency:      time.Since(start),
		InputTokens:  totalIn,
		OutputTokens: totalOut,
	}
}

// parseSelfCheckResponse extracts YES/NO from the first line and the explanation
// from the rest of the response.
func parseSelfCheckResponse(raw string) (yes bool, explanation string) {
	lines := strings.SplitN(strings.TrimSpace(raw), "\n", 2)
	verdict := strings.TrimSpace(strings.ToUpper(lines[0]))
	yes = strings.HasPrefix(verdict, "YES")
	if len(lines) > 1 {
		explanation = strings.TrimSpace(lines[1])
	}
	return yes, explanation
}
