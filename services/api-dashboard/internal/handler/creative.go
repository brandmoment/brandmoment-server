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

type CreativeHandler struct {
	service *service.CreativeService
}

// NewCreativeHandler returns a new CreativeHandler instance.
  func NewCreativeHandler(svc *service.CreativeService) *CreativeHandler {
	return &CreativeHandler{service: svc}
}

func (h *CreativeHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	campaignIDStr := chi.URLParam(r, "id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "campaign id must be a valid UUID")
		return
	}

	var req service.CreateCreativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	creative, err := h.service.Create(r.Context(), orgID, campaignID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, creative)
}

func (h *CreativeHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	campaignIDStr := chi.URLParam(r, "id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "campaign id must be a valid UUID")
		return
	}

	result, err := h.service.ListByCampaign(r.Context(), orgID, campaignID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, result)
}
