package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

const handlerTestSecret = "handler-test-secret"

// mockOrgRepoForHandler satisfies repository.OrganizationRepository via the same interface
// that service.OrganizationService accepts, letting handler tests avoid a real DB.
type mockOrgRepoForHandler struct {
	insertFn  func(ctx context.Context, org *model.Organization) (*model.Organization, error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	listFn    func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
}

func (m *mockOrgRepoForHandler) Insert(ctx context.Context, org *model.Organization) (*model.Organization, error) {
	return m.insertFn(ctx, org)
}

func (m *mockOrgRepoForHandler) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockOrgRepoForHandler) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
	return m.listFn(ctx, ids)
}

func newHandlerWithRepo(repo *mockOrgRepoForHandler) *OrganizationHandler {
	svc := service.NewOrganizationService(repo, noop.NewTracerProvider())
	return NewOrganizationHandler(svc)
}

// makeJWTForOrgs creates a signed JWT with the given org memberships.
func makeJWTForOrgs(t *testing.T, orgs []middleware.OrgClaim) string {
	t.Helper()
	claims := &middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Orgs: orgs,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(handlerTestSecret))
	if err != nil {
		t.Fatalf("makeJWTForOrgs: %v", err)
	}
	return signed
}

func decodeRespBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("decodeRespBody: %v", err)
	}
	return m
}

func withChiID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// applyAuth wraps a handler with ValidateJWT, injects the Bearer token and X-Org-ID header.
func applyAuth(t *testing.T, h http.Handler, token, xOrgID string) http.Handler {
	t.Helper()
	auth := middleware.NewAuth(handlerTestSecret)
	return auth.ValidateJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
}

// TestOrganizationHandler_Create covers Create endpoint.
func TestOrganizationHandler_Create(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		body        any
		insertFn    func(ctx context.Context, org *model.Organization) (*model.Organization, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name: "valid request returns 201",
			body: map[string]string{"type": "publisher", "name": "Acme Corp", "slug": "acme-corp"},
			insertFn: func(ctx context.Context, org *model.Organization) (*model.Organization, error) {
				return org, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "invalid JSON body returns 400",
			body:        "not-json{",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:        "empty name returns 400",
			body:        map[string]string{"type": "publisher", "name": "", "slug": "acme"},
			insertFn:    func(ctx context.Context, org *model.Organization) (*model.Organization, error) { return org, nil },
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "invalid org type returns 400",
			body:        map[string]string{"type": "unknown", "name": "X", "slug": "x"},
			insertFn:    func(ctx context.Context, org *model.Organization) (*model.Organization, error) { return org, nil },
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "repo returns ErrNotFound propagated as 404",
			body: map[string]string{"type": "brand", "name": "Brand X", "slug": "brand-x"},
			insertFn: func(ctx context.Context, org *model.Organization) (*model.Organization, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
			}

			repo := &mockOrgRepoForHandler{}
			if tt.insertFn != nil {
				repo.insertFn = tt.insertFn
			}
			h := newHandlerWithRepo(repo)

			// Build request with auth context (Create does not use orgIDs from context, so any valid JWT is fine)
			token := makeJWTForOrgs(t, []middleware.OrgClaim{{OrgID: orgID.String(), Role: "owner"}})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Org-ID", orgID.String())

			w := httptest.NewRecorder()
			applyAuth(t, http.HandlerFunc(h.Create), token, orgID.String()).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Create() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrCode != "" {
				resp := decodeRespBody(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("Create() error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrCode {
					t.Errorf("Create() error.code = %q, want %q", gotCode, tt.wantErrCode)
				}
			}
		})
	}
}

// TestOrganizationHandler_GetByID covers GetByID endpoint.
func TestOrganizationHandler_GetByID(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	tests := []struct {
		name        string
		urlID       string
		memberOrg   uuid.UUID // org that the JWT contains
		xOrgID      uuid.UUID // active org for request
		getByIDFn   func(ctx context.Context, id uuid.UUID) (*model.Organization, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:      "found organization returns 200",
			urlID:     orgID.String(),
			memberOrg: orgID,
			xOrgID:    orgID,
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
				return &model.Organization{ID: id, Type: "publisher", Name: "Acme", Slug: "acme"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			memberOrg:   orgID,
			xOrgID:      orgID,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:      "org not in user memberships returns 403",
			urlID:     otherOrgID.String(),
			memberOrg: orgID,
			xOrgID:    orgID,
			// memberOrg is orgID but we request otherOrgID — JWT has only orgID
			wantStatus:  http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name:      "org found in memberships but not in repo returns 404",
			urlID:     orgID.String(),
			memberOrg: orgID,
			xOrgID:    orgID,
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepoForHandler{}
			if tt.getByIDFn != nil {
				repo.getByIDFn = tt.getByIDFn
			}
			h := newHandlerWithRepo(repo)

			token := makeJWTForOrgs(t, []middleware.OrgClaim{{OrgID: tt.memberOrg.String(), Role: "viewer"}})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+tt.urlID, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Org-ID", tt.xOrgID.String())
			req = withChiID(req, tt.urlID)

			// Chain: ValidateJWT (sets context) → GetByID handler
			auth := middleware.NewAuth(handlerTestSecret)
			chain := auth.ValidateJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// re-attach chi context after middleware replaces r's context
				r = withChiID(r, tt.urlID)
				h.GetByID(w, r)
			}))

			w := httptest.NewRecorder()
			chain.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("GetByID() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrCode != "" {
				resp := decodeRespBody(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("GetByID() error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrCode {
					t.Errorf("GetByID() error.code = %q, want %q", gotCode, tt.wantErrCode)
				}
			}
		})
	}
}

// TestOrganizationHandler_List covers List endpoint.
func TestOrganizationHandler_List(t *testing.T) {
	org1 := uuid.New()
	org2 := uuid.New()

	tests := []struct {
		name        string
		memberOrgs  []middleware.OrgClaim
		xOrgID      uuid.UUID
		listFn      func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:       "returns organizations for user memberships",
			memberOrgs: []middleware.OrgClaim{{OrgID: org1.String(), Role: "owner"}, {OrgID: org2.String(), Role: "viewer"}},
			xOrgID:     org1,
			listFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				orgs := make([]model.Organization, len(ids))
				for i, id := range ids {
					orgs[i] = model.Organization{ID: id, Type: "publisher", Name: "Org", Slug: "org"}
				}
				return orgs, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "single membership returns one organization",
			memberOrgs: []middleware.OrgClaim{{OrgID: org1.String(), Role: "editor"}},
			xOrgID:     org1,
			listFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				return []model.Organization{{ID: ids[0], Type: "brand", Name: "BrandX", Slug: "brandx"}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "repo returns ErrInvalidInput propagated as 400",
			memberOrgs: []middleware.OrgClaim{{OrgID: org1.String(), Role: "owner"}},
			xOrgID:     org1,
			listFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				return nil, model.ErrInvalidInput
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepoForHandler{listFn: tt.listFn}
			h := newHandlerWithRepo(repo)

			token := makeJWTForOrgs(t, tt.memberOrgs)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Org-ID", tt.xOrgID.String())

			auth := middleware.NewAuth(handlerTestSecret)
			chain := auth.ValidateJWT(http.HandlerFunc(h.List))

			w := httptest.NewRecorder()
			chain.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("List() status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrCode != "" {
				resp := decodeRespBody(t, w)
				errField, ok := resp["error"].(map[string]any)
				if !ok {
					t.Fatalf("List() error field missing: %+v", resp)
				}
				gotCode, _ := errField["code"].(string)
				if gotCode != tt.wantErrCode {
					t.Errorf("List() error.code = %q, want %q", gotCode, tt.wantErrCode)
				}
			}
		})
	}
}

// TestHandleServiceError covers the error-mapping helper.
func TestHandleServiceError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantErrCode string
	}{
		{
			name:        "ErrNotFound maps to 404",
			err:         model.ErrNotFound,
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:        "ErrInvalidInput maps to 400",
			err:         model.ErrInvalidInput,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "ErrUnauthorized maps to 401",
			err:         model.ErrUnauthorized,
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "UNAUTHORIZED",
		},
		{
			name:        "unknown error maps to 500",
			err:         context.DeadlineExceeded,
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleServiceError(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("handleServiceError() status = %d, want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			errField, ok := resp["error"].(map[string]any)
			if !ok {
				t.Fatalf("error field missing: %+v", resp)
			}
			gotCode, _ := errField["code"].(string)
			if gotCode != tt.wantErrCode {
				t.Errorf("handleServiceError() error.code = %q, want %q", gotCode, tt.wantErrCode)
			}
		})
	}
}
