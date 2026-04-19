package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type MeResponse struct {
	ID        string               `json:"id"`
	Email     string               `json:"email"`
	Name      string               `json:"name"`
	CreatedAt string               `json:"created_at"`
	Orgs      []orgMembershipEntry `json:"orgs"`
}

type orgMembershipEntry struct {
	OrgID uuid.UUID `json:"org_id"`
	Role  string    `json:"role"`
}

type UserHandler struct {
	service *service.UserService
}

// NewUserHandler returns a new UserHandler instance.
  func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserIDFromContext(ctx)

	user, err := h.service.GetMe(ctx, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	orgClaims := middleware.OrgClaimsFromContext(ctx)
	orgs := make([]orgMembershipEntry, 0, len(orgClaims))
	for _, oc := range orgClaims {
		orgID, err := uuid.Parse(oc.OrgID)
		if err != nil {
			continue
		}
		orgs = append(orgs, orgMembershipEntry{OrgID: orgID, Role: oc.Role})
	}

	httputil.RespondJSON(w, http.StatusOK, MeResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Orgs:      orgs,
	})
}
