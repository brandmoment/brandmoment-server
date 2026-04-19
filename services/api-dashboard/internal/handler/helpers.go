package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

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
