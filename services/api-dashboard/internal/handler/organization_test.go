package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)


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

// injectAuthContext sets the middleware context values that ValidateJWT normally sets.
// Used in handler tests to avoid needing a real JWKS endpoint.
func injectAuthContext(r *http.Request, orgID uuid.UUID, role string, orgIDs []uuid.UUID) *http.Request {
	ctx := middleware.InjectTestContext(r.Context(), uuid.New(), orgID, role, orgIDs)
	return r.WithContext(ctx)
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

			req := httptest.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "owner", []uuid.UUID{orgID})

			w := httptest.NewRecorder()
			h.Create(w, req)

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
		memberOrgs  []uuid.UUID
		xOrgID      uuid.UUID
		getByIDFn   func(ctx context.Context, id uuid.UUID) (*model.Organization, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:       "found organization returns 200",
			urlID:      orgID.String(),
			memberOrgs: []uuid.UUID{orgID},
			xOrgID:     orgID,
			getByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
				return &model.Organization{ID: id, Type: "publisher", Name: "Acme", Slug: "acme"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			memberOrgs:  []uuid.UUID{orgID},
			xOrgID:      orgID,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:       "org not in user memberships returns 403",
			urlID:      otherOrgID.String(),
			memberOrgs: []uuid.UUID{orgID}, // JWT has only orgID, but we request otherOrgID
			xOrgID:     orgID,
			wantStatus:  http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name:       "org found in memberships but not in repo returns 404",
			urlID:      orgID.String(),
			memberOrgs: []uuid.UUID{orgID},
			xOrgID:     orgID,
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

			req := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+tt.urlID, nil)
			req = injectAuthContext(req, tt.xOrgID, "viewer", tt.memberOrgs)
			req = withChiID(req, tt.urlID)

			w := httptest.NewRecorder()
			h.GetByID(w, req)

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
		memberOrgs  []uuid.UUID
		xOrgID      uuid.UUID
		listFn      func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:       "returns organizations for user memberships",
			memberOrgs: []uuid.UUID{org1, org2},
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
			memberOrgs: []uuid.UUID{org1},
			xOrgID:     org1,
			listFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				return []model.Organization{{ID: ids[0], Type: "brand", Name: "BrandX", Slug: "brandx"}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "repo returns ErrInvalidInput propagated as 400",
			memberOrgs: []uuid.UUID{org1},
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

			req := httptest.NewRequest(http.MethodGet, "/v1/organizations", nil)
			req = injectAuthContext(req, tt.xOrgID, "owner", tt.memberOrgs)

			w := httptest.NewRecorder()
			h.List(w, req)

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
