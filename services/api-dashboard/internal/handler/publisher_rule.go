package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type PublisherRuleHandler struct {
	service *service.PublisherRuleService
}

func NewPublisherRuleHandler(svc *service.PublisherRuleService) *PublisherRuleHandler {
	return &PublisherRuleHandler{service: svc}
}

func (h *PublisherRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appID, ok := parseAppIDParam(w, r)
	if !ok {
		return
	}

	var req service.CreatePublisherRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	rule, err := h.service.Create(r.Context(), orgID, appID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, rule)
}

func (h *PublisherRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appID, ok := parseAppIDParam(w, r)
	if !ok {
		return
	}

	limit, offset := parsePagination(r)

	result, err := h.service.List(r.Context(), orgID, appID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, result)
}

func (h *PublisherRuleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appID, ok := parseAppIDParam(w, r)
	if !ok {
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "rule id must be a valid UUID")
		return
	}

	rule, err := h.service.GetByID(r.Context(), orgID, appID, ruleID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rule)
}

func (h *PublisherRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appID, ok := parseAppIDParam(w, r)
	if !ok {
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "rule id must be a valid UUID")
		return
	}

	var req service.UpdatePublisherRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	rule, err := h.service.Update(r.Context(), orgID, appID, ruleID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, rule)
}

func (h *PublisherRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appID, ok := parseAppIDParam(w, r)
	if !ok {
		return
	}

	ruleIDStr := chi.URLParam(r, "ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "rule id must be a valid UUID")
		return
	}

	if err := h.service.Delete(r.Context(), orgID, appID, ruleID); err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, map[string]string{"id": ruleID.String()})
}

// parseAppIDParam extracts and parses the {id} URL param used as the app ID.
// Returns false and writes an error response if parsing fails.
func parseAppIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "app id must be a valid UUID")
		return uuid.UUID{}, false
	}
	return id, true
}
