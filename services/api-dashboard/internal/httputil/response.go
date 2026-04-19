package httputil

import (
	"encoding/json"
	"net/http"
)

// Response is the standard JSON envelope returned by all API endpoints.
type Response struct {
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
}

// ErrorBody carries a machine-readable error code and a human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondJSON writes a successful JSON response with the given HTTP status code and data payload.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

// RespondError writes a JSON error response with the given HTTP status code, error code, and message.
func RespondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: &ErrorBody{Code: code, Message: message}})
}
