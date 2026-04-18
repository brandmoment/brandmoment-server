package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-key-for-unit-tests"

func makeToken(t *testing.T, orgs []OrgClaim) string {
	t.Helper()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Orgs: orgs,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeToken: failed to sign: %v", err)
	}
	return signed
}

func makeExpiredToken(t *testing.T, orgs []OrgClaim) string {
	t.Helper()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		Orgs: orgs,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeExpiredToken: failed to sign: %v", err)
	}
	return signed
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func decodeError(t *testing.T, body *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(body.Body).Decode(&resp); err != nil {
		t.Fatalf("decodeError: failed to decode body: %v", err)
	}
	return resp
}

func TestAuth_ValidateJWT(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	auth := NewAuth(testSecret)

	tests := []struct {
		name           string
		authHeader     string
		xOrgIDHeader   string
		wantStatus     int
		wantErrorCode  string
	}{
		{
			name:          "valid token and org membership",
			authHeader:    "Bearer " + makeToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "owner"}}),
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusOK,
		},
		{
			name:          "missing authorization header",
			authHeader:    "",
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
		{
			name:          "authorization header without Bearer prefix",
			authHeader:    makeToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "owner"}}),
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
		{
			name:          "invalid token signature",
			authHeader:    "Bearer invalid.token.here",
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
		{
			name:          "expired token",
			authHeader:    "Bearer " + makeExpiredToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "owner"}}),
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "UNAUTHORIZED",
		},
		{
			name:          "missing X-Org-ID header",
			authHeader:    "Bearer " + makeToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "owner"}}),
			xOrgIDHeader:  "",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "MISSING_ORG_ID",
		},
		{
			name:          "X-Org-ID not a valid UUID",
			authHeader:    "Bearer " + makeToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "owner"}}),
			xOrgIDHeader:  "not-a-uuid",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "INVALID_ORG_ID",
		},
		{
			name:          "org_id not in user memberships",
			authHeader:    "Bearer " + makeToken(t, []OrgClaim{{OrgID: otherOrgID.String(), Role: "viewer"}}),
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusForbidden,
			wantErrorCode: "FORBIDDEN",
		},
		{
			name:          "user is member of multiple orgs, correct org selected",
			authHeader:    "Bearer " + makeToken(t, []OrgClaim{{OrgID: otherOrgID.String(), Role: "viewer"}, {OrgID: orgID.String(), Role: "editor"}}),
			xOrgIDHeader:  orgID.String(),
			wantStatus:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.xOrgIDHeader != "" {
				req.Header.Set("X-Org-ID", tt.xOrgIDHeader)
			}

			w := httptest.NewRecorder()
			auth.ValidateJWT(http.HandlerFunc(okHandler)).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("ValidateJWT() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrorCode != "" {
				resp := decodeError(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("ValidateJWT() error field missing or wrong type: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrorCode {
					t.Errorf("ValidateJWT() error.code = %q, want %q", gotCode, tt.wantErrorCode)
				}
			}
		})
	}
}

func TestAuth_ValidateJWT_SetsContextValues(t *testing.T) {
	orgID := uuid.New()
	auth := NewAuth(testSecret)
	token := makeToken(t, []OrgClaim{{OrgID: orgID.String(), Role: "editor"}})

	var capturedCtx context.Context
	capture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Org-ID", orgID.String())

	w := httptest.NewRecorder()
	auth.ValidateJWT(capture).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	gotOrgID := OrgIDFromContext(capturedCtx)
	if gotOrgID != orgID {
		t.Errorf("OrgIDFromContext() = %v, want %v", gotOrgID, orgID)
	}

	gotRole := RoleFromContext(capturedCtx)
	if gotRole != "editor" {
		t.Errorf("RoleFromContext() = %q, want %q", gotRole, "editor")
	}

	gotOrgIDs := OrgIDsFromContext(capturedCtx)
	if len(gotOrgIDs) != 1 || gotOrgIDs[0] != orgID {
		t.Errorf("OrgIDsFromContext() = %v, want [%v]", gotOrgIDs, orgID)
	}
}

func TestAuth_RequireRole(t *testing.T) {
	auth := NewAuth(testSecret)

	tests := []struct {
		name           string
		ctxRole        string
		allowedRoles   []string
		wantStatus     int
		wantErrorCode  string
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
				resp := decodeError(t, w)
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
}
