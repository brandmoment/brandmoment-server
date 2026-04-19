package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type PublisherAppHandler struct {
	service *service.PublisherAppService
}

// NewPublisherAppHandler returns a new PublisherAppHandler instance.
  func NewPublisherAppHandler(svc *service.PublisherAppService) *PublisherAppHandler {
	return &PublisherAppHandler{service: svc}
}

func (h *PublisherAppHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	var req service.CreatePublisherAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	app, err := h.service.Create(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, app)
}

func (h *PublisherAppHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())
	limit, offset := parsePagination(r)

	result, err := h.service.List(r.Context(), orgID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, result)
}

func (h *PublisherAppHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	app, err := h.service.GetByID(r.Context(), orgID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, app)
}

func (h *PublisherAppHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	var req service.UpdatePublisherAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	app, err := h.service.Update(r.Context(), orgID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, app)
}

// parsePagination extracts limit and offset from query params.
// Defaults: limit=20, offset=0. Max limit clamped in service layer.
func parsePagination(r *http.Request) (limit, offset int32) {
	limit = 20
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	return limit, offset
}
