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

type CampaignHandler struct {
	service *service.CampaignService
}

// NewCampaignHandler returns a new CampaignHandler instance.
  func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{service: svc}
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	var req service.CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	campaign, err := h.service.Create(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, campaign)
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())
	limit, offset := parsePagination(r)

	var statusFilter *string
	if s := r.URL.Query().Get("status"); s != "" {
		statusFilter = &s
	}

	result, err := h.service.List(r.Context(), orgID, statusFilter, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, result)
}

func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	campaign, err := h.service.GetByID(r.Context(), orgID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, campaign)
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	var req service.UpdateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	campaign, err := h.service.Update(r.Context(), orgID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, campaign)
}

func (h *CampaignHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	var req service.UpdateCampaignStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	campaign, err := h.service.UpdateStatus(r.Context(), orgID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, campaign)
}
