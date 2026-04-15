package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type Response struct {
	Data  any        `json:"data"`
	Error *ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Error: &ErrorBody{Code: code, Message: message},
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, model.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid input")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
