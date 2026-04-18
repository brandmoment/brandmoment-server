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

type APIKeyHandler struct {
	service *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: svc}
}

// apiKeyCreateResponse includes the plaintext key only on creation.
type apiKeyCreateResponse struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	AppID     uuid.UUID  `json:"app_id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"`
	KeyPrefix string     `json:"key_prefix"`
	IsRevoked bool       `json:"is_revoked"`
	CreatedAt string     `json:"created_at"`
}

// apiKeyRevokeResponse carries only id and revoked_at.
type apiKeyRevokeResponse struct {
	ID        uuid.UUID  `json:"id"`
	RevokedAt *string    `json:"revoked_at"`
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appIDStr := chi.URLParam(r, "id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "app id must be a valid UUID")
		return
	}

	var req service.ProvisionAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	result, err := h.service.Provision(r.Context(), orgID, appID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resp := apiKeyCreateResponse{
		ID:        result.Key.ID,
		OrgID:     result.Key.OrgID,
		AppID:     result.Key.AppID,
		Name:      result.Key.Name,
		Key:       result.Plaintext,
		KeyPrefix: result.Key.KeyPrefix,
		IsRevoked: result.Key.IsRevoked,
		CreatedAt: result.Key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	httputil.RespondJSON(w, http.StatusCreated, resp)
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appIDStr := chi.URLParam(r, "id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "app id must be a valid UUID")
		return
	}

	includeRevoked := r.URL.Query().Get("include_revoked") == "true"

	result, err := h.service.ListByApp(r.Context(), orgID, appID, includeRevoked)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, result)
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.OrgIDFromContext(r.Context())

	appIDStr := chi.URLParam(r, "id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "app id must be a valid UUID")
		return
	}

	keyIDStr := chi.URLParam(r, "keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "INVALID_ID", "key id must be a valid UUID")
		return
	}

	revoked, err := h.service.Revoke(r.Context(), orgID, appID, keyID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resp := apiKeyRevokeResponse{ID: revoked.ID}
	if revoked.RevokedAt != nil {
		s := revoked.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.RevokedAt = &s
	}

	httputil.RespondJSON(w, http.StatusOK, resp)
}

