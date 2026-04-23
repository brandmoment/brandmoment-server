package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// analysisRule is one entry from stage 1 output.
type analysisRule struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

// analysisResult is the full stage 1 JSON output.
type analysisResult struct {
	Count int            `json:"count"`
	Rules []analysisRule `json:"rules"`
}

// StageResult tracks one stage's metrics.
type StageResult struct {
	Stage        string  `json:"stage"`
	LatencyMS    float64 `json:"latency_ms"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
}

// MultiStageResult is the full outcome of the 3-stage pipeline.
type MultiStageResult struct {
	RulesJSON    string           `json:"rules_json"`
	Stages       []StageResult    `json:"stages"`
	Confidence   ConfidenceStatus `json:"confidence"`
	TotalCalls   int              `json:"total_calls"`
	TotalLatency float64          `json:"total_latency_ms"`
	TotalIn      int              `json:"total_input_tokens"`
	TotalOut     int              `json:"total_output_tokens"`
}

// MultiStageParser runs a 3-stage LLM pipeline to parse natural-language phrases
// into structured publisher rules.
type MultiStageParser struct {
	client ChatClient
}

// NewMultiStageParser creates a MultiStageParser backed by the given ChatClient.
func NewMultiStageParser(client ChatClient) *MultiStageParser {
	return &MultiStageParser{client: client}
}

const stage1SystemPrompt = `You are a rule-type analyzer for an ad-network rule engine.
Given a natural-language phrase, identify the number and types of rules it describes.

VALID RULE TYPES (use only these exact names):
- blocklist
- allowlist
- frequency_cap
- geo_filter
- platform_filter

OUTPUT FORMAT — return ONLY a JSON object, nothing else:
{"count": <number>, "rules": [{"type": "<rule_type>", "summary": "<one-sentence summary>"}, ...]}

If the phrase cannot be expressed with these rule types, return: {"count": 0, "rules": []}
Do NOT include markdown fences, commentary, or any text outside the JSON object.`

const stage2SystemPromptTemplate = `You are a precise JSON config extractor for an ad-network rule engine.
Given a rule type and a description, extract only the config object for that rule.

Rule type: %s

CONFIG SCHEMAS:
- blocklist:       {"domains": ["example.com"], "bundle_ids": []}
- allowlist:       {"domains": ["good.com"], "bundle_ids": []}
- frequency_cap:   {"max_impressions": 5, "window_seconds": 3600}
- geo_filter:      {"mode": "exclude", "country_codes": ["RU", "KZ"]}
- platform_filter: {"mode": "include", "platforms": ["ios", "android"]}

OUTPUT FORMAT — return ONLY the config JSON object, nothing else.
Do NOT include markdown fences, the rule type wrapper, or any text outside the JSON object.`

// Parse runs the 3-stage pipeline and returns a MultiStageResult.
// Stage 1: analyze phrase → identify rule count and types.
// Stage 2: extract config per rule (one LLM call per rule).
// Stage 3: assemble + validate (no LLM call).
func (p *MultiStageParser) Parse(ctx context.Context, phrase string) MultiStageResult {
	var result MultiStageResult

	// Stage 1: Analyze.
	analysis, stage1, err := p.analyzePhrase(ctx, phrase)
	result.Stages = append(result.Stages, stage1)
	result.TotalCalls++
	result.TotalIn += stage1.InputTokens
	result.TotalOut += stage1.OutputTokens
	result.TotalLatency += stage1.LatencyMS

	if err != nil {
		slog.InfoContext(ctx, "multistage stage1 failed", slog.String("error", err.Error()))
		result.RulesJSON = "[]"
		result.Confidence = ConfidenceStatusFail
		return result
	}

	slog.InfoContext(ctx, "multistage stage1 done",
		slog.Int("rule_count", analysis.Count),
		slog.Float64("latency_ms", stage1.LatencyMS),
	)

	if analysis.Count == 0 || len(analysis.Rules) == 0 {
		result.RulesJSON = "[]"
		result.Confidence = ConfidenceStatusFail
		return result
	}

	// Stage 2: Extract config per rule.
	configs := make([]json.RawMessage, 0, len(analysis.Rules))
	for i, ar := range analysis.Rules {
		cfg, stage2, extractErr := p.extractConfig(ctx, phrase, ar.Type, ar.Summary)
		result.Stages = append(result.Stages, stage2)
		result.TotalCalls++
		result.TotalIn += stage2.InputTokens
		result.TotalOut += stage2.OutputTokens
		result.TotalLatency += stage2.LatencyMS

		slog.InfoContext(ctx, "multistage stage2 done",
			slog.Int("rule_index", i),
			slog.String("rule_type", ar.Type),
			slog.Float64("latency_ms", stage2.LatencyMS),
		)

		if extractErr != nil {
			slog.InfoContext(ctx, "multistage stage2 extract failed",
				slog.Int("rule_index", i),
				slog.String("error", extractErr.Error()),
			)
			result.RulesJSON = "[]"
			result.Confidence = ConfidenceStatusFail
			return result
		}
		configs = append(configs, cfg)
	}

	// Stage 3: Assemble (no LLM).
	rulesJSON, constraintResult := assembleRules(analysis.Rules, configs)
	stage3 := StageResult{Stage: "assemble"}
	result.Stages = append(result.Stages, stage3)

	result.RulesJSON = rulesJSON
	result.Confidence = constraintResult.Status

	slog.InfoContext(ctx, "multistage stage3 done",
		slog.String("confidence", string(constraintResult.Status)),
		slog.Int("total_calls", result.TotalCalls),
		slog.Float64("total_latency_ms", result.TotalLatency),
	)

	return result
}

// analyzePhrase calls the LLM to count and name the rules in phrase.
func (p *MultiStageParser) analyzePhrase(ctx context.Context, phrase string) (analysisResult, StageResult, error) {
	start := time.Now()
	stage := StageResult{Stage: "analyze"}

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: stage1SystemPrompt},
			{Role: RoleUser, Content: phrase},
		},
		Temperature: 0,
	}

	resp, err := p.client.Complete(ctx, req)
	stage.LatencyMS = float64(time.Since(start).Milliseconds())
	stage.InputTokens = resp.InputTokens
	stage.OutputTokens = resp.OutputTokens

	if err != nil {
		return analysisResult{}, stage, fmt.Errorf("stage1 analyze: %w", err)
	}

	raw := StripMarkdownFences(resp.Content)
	var analysis analysisResult
	if err := json.Unmarshal([]byte(raw), &analysis); err != nil {
		return analysisResult{}, stage, fmt.Errorf("stage1 parse json: %w", err)
	}
	return analysis, stage, nil
}

// extractConfig calls the LLM to extract the config object for a single rule.
func (p *MultiStageParser) extractConfig(ctx context.Context, phrase, ruleType, summary string) (json.RawMessage, StageResult, error) {
	start := time.Now()
	stage := StageResult{Stage: "extract:" + ruleType}

	systemMsg := fmt.Sprintf(stage2SystemPromptTemplate, ruleType)
	userMsg := fmt.Sprintf("Phrase: %s\nRule summary: %s", phrase, summary)

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: systemMsg},
			{Role: RoleUser, Content: userMsg},
		},
		Temperature: 0,
	}

	resp, err := p.client.Complete(ctx, req)
	stage.LatencyMS = float64(time.Since(start).Milliseconds())
	stage.InputTokens = resp.InputTokens
	stage.OutputTokens = resp.OutputTokens

	if err != nil {
		return nil, stage, fmt.Errorf("stage2 extract %s: %w", ruleType, err)
	}

	raw := StripMarkdownFences(resp.Content)
	// Validate it's valid JSON.
	var check json.RawMessage
	if err := json.Unmarshal([]byte(raw), &check); err != nil {
		return nil, stage, fmt.Errorf("stage2 parse json for %s: %w", ruleType, err)
	}
	return json.RawMessage(raw), stage, nil
}

// assembleRules combines types and configs into the final JSON array and validates it.
func assembleRules(analyses []analysisRule, configs []json.RawMessage) (string, ConstraintResult) {
	type ruleObj struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}

	rules := make([]ruleObj, 0, len(analyses))
	for i, ar := range analyses {
		if i >= len(configs) {
			break
		}
		rules = append(rules, ruleObj{Type: ar.Type, Config: configs[i]})
	}

	b, err := json.Marshal(rules)
	if err != nil {
		return "[]", ConstraintResult{
			Status: ConfidenceStatusFail,
			Errors: []string{fmt.Sprintf("assemble marshal: %s", err)},
		}
	}

	rulesJSON := string(b)
	constraint := CheckConstraint(context.Background(), rulesJSON)
	return rulesJSON, constraint
}
