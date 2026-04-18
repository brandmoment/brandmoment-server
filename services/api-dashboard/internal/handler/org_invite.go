package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type OrgInviteHandler struct {
	service *service.OrgInviteService
}

func NewOrgInviteHandler(svc *service.OrgInviteService) *OrgInviteHandler {
	return &OrgInviteHandler{service: svc}
}

func (h *OrgInviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "org id must be a valid UUID")
		return
	}

	var req service.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	invite, err := h.service.Create(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusCreated, invite)
}
