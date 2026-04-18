package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func decodeErrorBody(t *testing.T, body *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body.Body).Decode(&resp); err != nil {
		t.Fatalf("decodeErrorBody: failed to decode body: %v", err)
	}
	return resp
}

// newAuthForTest creates an Auth with a nil JWKS — only usable for tests
// that don't invoke JWT parsing (e.g., RequireRole tests, context helper tests).
func newAuthForTest() *Auth {
	return &Auth{}
}

// TestAuth_ValidateJWT_HeaderChecks verifies that ValidateJWT rejects requests
// before touching the JWKS (missing/malformed Authorization header).
// These cases fail before network I/O so no real JWKS URL is needed.
func TestAuth_ValidateJWT_HeaderChecks(t *testing.T) {
	// Auth with a nil JWKS keyfunc — any token that reaches ParseWithClaims
	// will panic, but the header checks fire first for these cases.
	auth := newAuthForTest()

	tests := []struct {
		name          string
		authHeader    string
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "missing authorization header",
			authHeader:    "",
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
		{
			name:          "authorization header without Bearer prefix",
			authHeader:    "Basic sometoken",
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("X-Org-ID", uuid.New().String())

			w := httptest.NewRecorder()
			auth.ValidateJWT(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("ValidateJWT() status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantErrorCode != "" {
				resp := decodeErrorBody(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrorCode {
					t.Errorf("ValidateJWT() error.code = %q, want %q", gotCode, tt.wantErrorCode)
				}
			}
		})
	}
}

func TestAuth_RequireRole(t *testing.T) {
	auth := newAuthForTest()

	tests := []struct {
		name          string
		ctxRole       string
		allowedRoles  []string
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:         "exact role match — owner",
			ctxRole:      "owner",
			allowedRoles: []string{"owner", "admin"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "exact role match — admin",
			ctxRole:      "admin",
			allowedRoles: []string{"owner", "admin", "editor"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "exact role match — editor",
			ctxRole:      "editor",
			allowedRoles: []string{"editor"},
			wantStatus:   http.StatusOK,
		},
		{
			name:          "role not in allowed list — viewer",
			ctxRole:       "viewer",
			allowedRoles:  []string{"owner", "admin", "editor"},
			wantStatus:    http.StatusForbidden,
			wantErrorCode: "FORBIDDEN",
		},
		{
			name:          "empty context role",
			ctxRole:       "",
			allowedRoles:  []string{"owner"},
			wantStatus:    http.StatusForbidden,
			wantErrorCode: "FORBIDDEN",
		},
		{
			name:          "no allowed roles specified",
			ctxRole:       "owner",
			allowedRoles:  []string{},
			wantStatus:    http.StatusForbidden,
			wantErrorCode: "FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), ctxRole, tt.ctxRole)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			auth.RequireRole(tt.allowedRoles...)(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("RequireRole() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrorCode != "" {
				resp := decodeErrorBody(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("RequireRole() error field missing or wrong type: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrorCode {
					t.Errorf("RequireRole() error.code = %q, want %q", gotCode, tt.wantErrorCode)
				}
			}
		})
	}
}

func TestContextHelpers_ZeroValues(t *testing.T) {
	ctx := context.Background()

	t.Run("OrgIDFromContext returns zero UUID when not set", func(t *testing.T) {
		got := OrgIDFromContext(ctx)
		if got != (uuid.UUID{}) {
			t.Errorf("OrgIDFromContext() = %v, want zero UUID", got)
		}
	})

	t.Run("RoleFromContext returns empty string when not set", func(t *testing.T) {
		got := RoleFromContext(ctx)
		if got != "" {
			t.Errorf("RoleFromContext() = %q, want empty string", got)
		}
	})

	t.Run("OrgIDsFromContext returns nil when not set", func(t *testing.T) {
		got := OrgIDsFromContext(ctx)
		if got != nil {
			t.Errorf("OrgIDsFromContext() = %v, want nil", got)
		}
	})

	t.Run("UserIDFromContext returns zero UUID when not set", func(t *testing.T) {
		got := UserIDFromContext(ctx)
		if got != (uuid.UUID{}) {
			t.Errorf("UserIDFromContext() = %v, want zero UUID", got)
		}
	})
}

func TestContextHelpers_SetValues(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	orgClaims := []OrgClaim{{OrgID: orgID.String(), Role: "editor"}}

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxOrgID, orgID)
	ctx = context.WithValue(ctx, ctxRole, "editor")
	ctx = context.WithValue(ctx, ctxOrgIDs, []uuid.UUID{orgID})
	ctx = context.WithValue(ctx, ctxOrgClaims, orgClaims)

	if got := OrgIDFromContext(ctx); got != orgID {
		t.Errorf("OrgIDFromContext() = %v, want %v", got, orgID)
	}
	if got := RoleFromContext(ctx); got != "editor" {
		t.Errorf("RoleFromContext() = %q, want %q", got, "editor")
	}
	if got := UserIDFromContext(ctx); got != userID {
		t.Errorf("UserIDFromContext() = %v, want %v", got, userID)
	}
	if got := OrgClaimsFromContext(ctx); len(got) != 1 || got[0].OrgID != orgID.String() {
		t.Errorf("OrgClaimsFromContext() = %v, want %v", got, orgClaims)
	}
}
