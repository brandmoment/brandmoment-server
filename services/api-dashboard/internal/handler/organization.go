package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type OrganizationHandler struct {
	service *service.OrganizationService
}

func NewOrganizationHandler(svc *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: svc}
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	org, err := h.service.Create(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, org)
}

func (h *OrganizationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "id must be a valid UUID")
		return
	}

	orgIDs := middleware.OrgIDsFromContext(r.Context())
	if !slices.Contains(orgIDs, id) {
		httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", "you are not a member of this organization")
		return
	}

	org, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, org)
}

func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	orgIDs := middleware.OrgIDsFromContext(r.Context())

	orgs, err := h.service.ListByIDs(r.Context(), orgIDs)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, orgs)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		httputil.RespondError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, model.ErrInvalidInput):
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, model.ErrUnauthorized):
		httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	default:
		httputil.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
