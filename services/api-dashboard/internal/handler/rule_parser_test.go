package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockLLMClientForHandler implements llm.ChatClient for handler-level tests.
type mockLLMClientForHandler struct {
	completeFn func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error)
}

func (m *mockLLMClientForHandler) Complete(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	return m.completeFn(ctx, req)
}

func (m *mockLLMClientForHandler) Provider() llm.Provider {
	return llm.ProviderOpenAI
}

var _ llm.ChatClient = (*mockLLMClientForHandler)(nil)

// buildRuleParserHandler creates a handler backed by a real RuleParserService using the given mock client.
func buildRuleParserHandler(client llm.ChatClient) *RuleParserHandler {
	svc := service.NewRuleParserService(client, noop.NewTracerProvider())
	return NewRuleParserHandler(svc)
}

// decodeErrorResponse decodes the standard error envelope from the response body.
func decodeErrorResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to decode response body: %v — body: %s", err, body)
	}
	return env
}

func TestRuleParserHandler_Parse_Disabled501(t *testing.T) {
	h := NewRuleParserHandlerDisabled()

	body := `{"phrase":"block evil.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501", w.Code)
	}
	env := decodeErrorResponse(t, w.Body.Bytes())
	errField, _ := env["error"].(map[string]any)
	if errField == nil {
		t.Fatal("expected error field in response")
	}
	if errField["code"] != "NOT_CONFIGURED" {
		t.Errorf("error code = %v, want NOT_CONFIGURED", errField["code"])
	}
}

func TestRuleParserHandler_Parse_MissingPhrase400(t *testing.T) {
	// Use a client that returns valid JSON so the service itself doesn't fail
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: `[{"type":"blocklist","config":{"domains":["x.com"]}}]`}, nil
		},
	}
	h := buildRuleParserHandler(client)

	body := `{"approaches":["constraint"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeErrorResponse(t, w.Body.Bytes())
	errField, _ := env["error"].(map[string]any)
	if errField == nil {
		t.Fatal("expected error field in response")
	}
	if errField["code"] != "INVALID_INPUT" {
		t.Errorf("error code = %v, want INVALID_INPUT", errField["code"])
	}
}

func TestRuleParserHandler_Parse_EmptyPhrase400(t *testing.T) {
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: `[]`}, nil
		},
	}
	h := buildRuleParserHandler(client)

	body := `{"phrase":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRuleParserHandler_Parse_InvalidBodyJSON400(t *testing.T) {
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: `[]`}, nil
		},
	}
	h := buildRuleParserHandler(client)

	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", strings.NewReader(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeErrorResponse(t, w.Body.Bytes())
	errField, _ := env["error"].(map[string]any)
	if errField == nil {
		t.Fatal("expected error field in response")
	}
	if errField["code"] != "INVALID_BODY" {
		t.Errorf("error code = %v, want INVALID_BODY", errField["code"])
	}
}

func TestRuleParserHandler_Parse_ValidRequest200(t *testing.T) {
	validJSON := `[{"type":"blocklist","config":{"domains":["evil.com"]}}]`
	callCount := 0
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return llm.ChatResponse{Content: validJSON, InputTokens: 10, OutputTokens: 5}, nil
			}
			// self_check verify
			return llm.ChatResponse{Content: "YES\nCorrect.", InputTokens: 15, OutputTokens: 3}, nil
		},
	}
	h := buildRuleParserHandler(client)

	reqBody, _ := json.Marshal(map[string]any{
		"phrase":     "block evil.com",
		"approaches": []string{"constraint"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}
	if data["rules"] == nil {
		t.Error("expected rules in response data")
	}
	if data["confidence"] == nil {
		t.Error("expected confidence in response data")
	}
}

func TestRuleParserHandler_Parse_ValidRequest_WithApproachesField200(t *testing.T) {
	validJSON := `[{"type":"frequency_cap","config":{"max_impressions":3,"window_seconds":86400}}]`
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validJSON, InputTokens: 8, OutputTokens: 4}, nil
		},
	}
	h := buildRuleParserHandler(client)

	reqBody, _ := json.Marshal(map[string]any{
		"phrase":     "max 3 impressions per day",
		"approaches": []string{"constraint"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestRuleParserHandler_Parse_ContentTypeJSON(t *testing.T) {
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: `[{"type":"allowlist","config":{"domains":["ok.com"]}}]`}, nil
		},
	}
	h := buildRuleParserHandler(client)

	reqBody, _ := json.Marshal(map[string]any{"phrase": "allow ok.com", "approaches": []string{"constraint"}})
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestRuleParserHandler_Parse_NoApproaches_UsesDefaults(t *testing.T) {
	validJSON := `[{"type":"platform_filter","config":{"mode":"include","platforms":["ios"]}}]`
	callCount := 0
	client := &mockLLMClientForHandler{
		completeFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			// call 1: self_check parse, call 2: self_check verify; constraint reuses JSON
			if callCount == 2 {
				return llm.ChatResponse{Content: "YES\nOK."}, nil
			}
			return llm.ChatResponse{Content: validJSON, InputTokens: 5, OutputTokens: 3}, nil
		},
	}
	h := buildRuleParserHandler(client)

	// No approaches field — should default to both
	reqBody, _ := json.Marshal(map[string]any{"phrase": "ios only"})
	req := httptest.NewRequest(http.MethodPost, "/v1/publisher-rules/parse", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Parse(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data := env["data"].(map[string]any)
	confidence := data["confidence"].(map[string]any)
	approaches, ok := confidence["approaches"].(map[string]any)
	if !ok {
		t.Fatal("expected approaches map in confidence")
	}
	if _, hasSC := approaches["self_check"]; !hasSC {
		t.Error("expected self_check approach in report")
	}
	if _, hasC := approaches["constraint"]; !hasC {
		t.Error("expected constraint approach in report")
	}
}
