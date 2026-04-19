package httputil

import (
	"encoding/json"
	"net/http"
)

// Response is the standard response structure for API responses.
  type Response struct {
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
}

// ErrorBody contains error details for API responses.
  type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondJSON encodes and sends a JSON response with data.
  func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

// RespondError encodes and sends a JSON error response with code and message.
  func RespondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: &ErrorBody{Code: code, Message: message}})
}
