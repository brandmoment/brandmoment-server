package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           any
		wantStatus     int
		wantContentType string
		wantData       any
	}{
		{
			name:            "200 with map data",
			status:          http.StatusOK,
			data:            map[string]string{"key": "value"},
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantData:        map[string]any{"key": "value"},
		},
		{
			name:            "201 created with struct",
			status:          http.StatusCreated,
			data:            map[string]string{"id": "abc123"},
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
			wantData:        map[string]any{"id": "abc123"},
		},
		{
			name:            "200 with nil data",
			status:          http.StatusOK,
			data:            nil,
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantData:        nil,
		},
		{
			name:            "200 with slice data",
			status:          http.StatusOK,
			data:            []string{"a", "b", "c"},
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantData:        []any{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RespondJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("RespondJSON() status = %d, want %d", w.Code, tt.wantStatus)
			}

			ct := w.Header().Get("Content-Type")
			if ct != tt.wantContentType {
				t.Errorf("RespondJSON() Content-Type = %q, want %q", ct, tt.wantContentType)
			}

			var resp Response
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("RespondJSON() failed to decode response body: %v", err)
			}

			if resp.Error != nil {
				t.Errorf("RespondJSON() unexpected error field: %+v", resp.Error)
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		code            string
		message         string
		wantStatus      int
		wantCode        string
		wantMessage     string
		wantContentType string
	}{
		{
			name:            "400 bad request",
			status:          http.StatusBadRequest,
			code:            "INVALID_BODY",
			message:         "failed to decode request",
			wantStatus:      http.StatusBadRequest,
			wantCode:        "INVALID_BODY",
			wantMessage:     "failed to decode request",
			wantContentType: "application/json",
		},
		{
			name:            "401 unauthorized",
			status:          http.StatusUnauthorized,
			code:            "UNAUTHORIZED",
			message:         "missing token",
			wantStatus:      http.StatusUnauthorized,
			wantCode:        "UNAUTHORIZED",
			wantMessage:     "missing token",
			wantContentType: "application/json",
		},
		{
			name:            "403 forbidden",
			status:          http.StatusForbidden,
			code:            "FORBIDDEN",
			message:         "insufficient permissions",
			wantStatus:      http.StatusForbidden,
			wantCode:        "FORBIDDEN",
			wantMessage:     "insufficient permissions",
			wantContentType: "application/json",
		},
		{
			name:            "404 not found",
			status:          http.StatusNotFound,
			code:            "NOT_FOUND",
			message:         "resource does not exist",
			wantStatus:      http.StatusNotFound,
			wantCode:        "NOT_FOUND",
			wantMessage:     "resource does not exist",
			wantContentType: "application/json",
		},
		{
			name:            "500 internal error",
			status:          http.StatusInternalServerError,
			code:            "INTERNAL_ERROR",
			message:         "internal server error",
			wantStatus:      http.StatusInternalServerError,
			wantCode:        "INTERNAL_ERROR",
			wantMessage:     "internal server error",
			wantContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RespondError(w, tt.status, tt.code, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("RespondError() status = %d, want %d", w.Code, tt.wantStatus)
			}

			ct := w.Header().Get("Content-Type")
			if ct != tt.wantContentType {
				t.Errorf("RespondError() Content-Type = %q, want %q", ct, tt.wantContentType)
			}

			var resp Response
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("RespondError() failed to decode response body: %v", err)
			}

			if resp.Data != nil {
				t.Errorf("RespondError() unexpected data field: %+v", resp.Data)
			}

			if resp.Error == nil {
				t.Fatal("RespondError() error field is nil")
			}

			if resp.Error.Code != tt.wantCode {
				t.Errorf("RespondError() error.code = %q, want %q", resp.Error.Code, tt.wantCode)
			}

			if resp.Error.Message != tt.wantMessage {
				t.Errorf("RespondError() error.message = %q, want %q", resp.Error.Message, tt.wantMessage)
			}
		})
	}
}

func TestRespondJSON_DataFieldPresent(t *testing.T) {
	w := httptest.NewRecorder()
	RespondJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Data == nil {
		t.Error("RespondJSON() data field is nil, want non-nil")
	}
}

func TestRespondError_NoDataField(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusBadRequest, "ERR", "something bad")

	var raw map[string]any
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if _, ok := raw["data"]; ok {
		t.Error("RespondError() response contains unexpected 'data' field")
	}

	if _, ok := raw["error"]; !ok {
		t.Error("RespondError() response missing 'error' field")
	}
}
