package handler

import (
	"net/http"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
)

type HealthHandler struct{}

// NewHealthHandler returns a HealthHandler for the liveness check endpoint.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
