package handler

import (
	"encoding/json"
	"net/http"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// RuleParserHandler exposes the NL → PublisherRule parsing endpoint.
type RuleParserHandler struct {
	service *service.RuleParserService
}

// NewRuleParserHandler returns a RuleParserHandler wired to the given RuleParserService.
func NewRuleParserHandler(svc *service.RuleParserService) *RuleParserHandler {
	return &RuleParserHandler{service: svc}
}

// NewRuleParserHandlerDisabled returns a RuleParserHandler that always responds 501.
// Use when no LLM API key is configured.
func NewRuleParserHandlerDisabled() *RuleParserHandler {
	return &RuleParserHandler{service: nil}
}

type parseRuleRequest struct {
	Phrase     string   `json:"phrase"`
	Approaches []string `json:"approaches"`
}

type parseRuleResponse struct {
	Rules      any `json:"rules"`
	Confidence any `json:"confidence"`
}

// Parse handles POST /v1/publisher-rules/parse.
func (h *RuleParserHandler) Parse(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		httputil.RespondError(w, http.StatusNotImplemented, "NOT_CONFIGURED", "LLM API key not configured")
		return
	}

	defer r.Body.Close()
	var req parseRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request body")
		return
	}

	if req.Phrase == "" {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_INPUT", "phrase is required")
		return
	}

	rules, confidence, err := h.service.Parse(r.Context(), req.Phrase, req.Approaches)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, parseRuleResponse{
		Rules:      rules,
		Confidence: confidence,
	})
}
