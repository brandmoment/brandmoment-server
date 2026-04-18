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
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockPublisherAppRepoForHandler implements repository.PublisherAppRepository for handler tests.
type mockPublisherAppRepoForHandler struct {
	insertFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
	getByIDFn     func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
	getByBundleFn func(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error)
	listByOrgFn   func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
	updateFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
}

func (m *mockPublisherAppRepoForHandler) Insert(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	return m.insertFn(ctx, app)
}

func (m *mockPublisherAppRepoForHandler) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error) {
	return m.getByIDFn(ctx, orgID, id)
}

func (m *mockPublisherAppRepoForHandler) GetByBundleID(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error) {
	return m.getByBundleFn(ctx, orgID, bundleID)
}

func (m *mockPublisherAppRepoForHandler) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error) {
	return m.listByOrgFn(ctx, orgID, limit, offset)
}

func (m *mockPublisherAppRepoForHandler) Update(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	return m.updateFn(ctx, app)
}

// compile-time check: mockPublisherAppRepoForHandler satisfies repository.PublisherAppRepository.
var _ repository.PublisherAppRepository = (*mockPublisherAppRepoForHandler)(nil)

func newPublisherAppHandler(repo *mockPublisherAppRepoForHandler) *PublisherAppHandler {
	svc := service.NewPublisherAppService(repo, noop.NewTracerProvider())
	return NewPublisherAppHandler(svc)
}

// stubPublisherApp returns a minimal valid PublisherApp for use in stub responses.
func stubPublisherApp(orgID uuid.UUID) *model.PublisherApp {
	return &model.PublisherApp{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      "My App",
		Platform:  "ios",
		BundleID:  "com.example.app",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestPublisherAppHandler_Create covers POST /v1/publisher-apps.
func TestPublisherAppHandler_Create(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		body        any
		repo        *mockPublisherAppRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name: "valid request returns 201",
			body: map[string]string{"name": "My App", "platform": "ios", "bundle_id": "com.example.app"},
			repo: &mockPublisherAppRepoForHandler{
				getByBundleFn: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
					return nil, model.ErrNotFound
				},
				insertFn: func(_ context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
					return app, nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "invalid JSON body returns 400",
			body:        "not-json{",
			repo:        &mockPublisherAppRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name: "empty name returns 400",
			body: map[string]string{"name": "", "platform": "ios", "bundle_id": "com.example.app"},
			repo: &mockPublisherAppRepoForHandler{
				getByBundleFn: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "invalid platform returns 400",
			body: map[string]string{"name": "My App", "platform": "windows", "bundle_id": "com.example.app"},
			repo: &mockPublisherAppRepoForHandler{
				getByBundleFn: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "duplicate bundle_id returns 400",
			body: map[string]string{"name": "My App", "platform": "android", "bundle_id": "com.example.app"},
			repo: &mockPublisherAppRepoForHandler{
				getByBundleFn: func(_ context.Context, orgID uuid.UUID, _ string) (*model.PublisherApp, error) {
					// Bundle ID already exists — return an app.
					return stubPublisherApp(orgID), nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "repo insert error returns 500",
			body: map[string]string{"name": "My App", "platform": "web", "bundle_id": "com.example.web"},
			repo: &mockPublisherAppRepoForHandler{
				getByBundleFn: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
					return nil, model.ErrNotFound
				},
				insertFn: func(_ context.Context, _ *model.PublisherApp) (*model.PublisherApp, error) {
					return nil, context.DeadlineExceeded
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			h := newPublisherAppHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPost, "/v1/publisher-apps", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})

			w := httptest.NewRecorder()
			h.Create(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestPublisherAppHandler_GetByID covers GET /v1/publisher-apps/{id}.
func TestPublisherAppHandler_GetByID(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name        string
		urlID       string
		getByIDFn   func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "found app returns 200",
			urlID: appID.String(),
			getByIDFn: func(_ context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error) {
				return &model.PublisherApp{ID: id, OrgID: orgID, Name: "My App", Platform: "ios", BundleID: "com.example.app", IsActive: true}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:  "app not found returns 404",
			urlID: appID.String(),
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:  "app belongs to different org returns 404",
			urlID: appID.String(),
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
				// Repository enforces org_id filter — cross-org access returns ErrNotFound.
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherAppRepoForHandler{}
			if tt.getByIDFn != nil {
				repo.getByIDFn = tt.getByIDFn
			}
			h := newPublisherAppHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/publisher-apps/"+tt.urlID, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlID)

			w := httptest.NewRecorder()
			h.GetByID(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestPublisherAppHandler_List covers GET /v1/publisher-apps.
func TestPublisherAppHandler_List(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		query       string
		listByOrgFn func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:  "returns list with default pagination",
			query: "",
			listByOrgFn: func(_ context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error) {
				return []model.PublisherApp{*stubPublisherApp(orgID)}, 1, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				items, ok := data["items"].([]any)
				if !ok {
					t.Fatalf("items field missing or wrong type: %+v", data)
				}
				if len(items) != 1 {
					t.Errorf("items count = %d, want 1", len(items))
				}
			},
		},
		{
			name:  "returns empty list when no apps",
			query: "?limit=20&offset=0",
			listByOrgFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]model.PublisherApp, int64, error) {
				return []model.PublisherApp{}, 0, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "repo error returns 500",
			query: "",
			listByOrgFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]model.PublisherApp, int64, error) {
				return nil, 0, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherAppRepoForHandler{listByOrgFn: tt.listByOrgFn}
			h := newPublisherAppHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/publisher-apps"+tt.query, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})

			w := httptest.NewRecorder()
			h.List(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
				return
			}

			if tt.checkData != nil {
				resp := decodeRespBody(t, w)
				data, ok := resp["data"].(map[string]any)
				if !ok {
					t.Fatalf("data field missing: %+v", resp)
				}
				tt.checkData(t, data)
			}
		})
	}
}

// TestPublisherAppHandler_Update covers PUT /v1/publisher-apps/{id}.
func TestPublisherAppHandler_Update(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	existing := stubPublisherApp(orgID)
	existing.ID = appID

	tests := []struct {
		name        string
		urlID       string
		body        any
		repo        *mockPublisherAppRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "valid name update returns 200",
			urlID: appID.String(),
			body:  map[string]string{"name": "Updated App"},
			repo: &mockPublisherAppRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
					return existing, nil
				},
				updateFn: func(_ context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
					return app, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "valid is_active update returns 200",
			urlID: appID.String(),
			body:  map[string]any{"is_active": false},
			repo: &mockPublisherAppRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
					return existing, nil
				},
				updateFn: func(_ context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
					return app, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			body:        map[string]string{"name": "Updated"},
			repo:        &mockPublisherAppRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:  "invalid JSON body returns 400",
			urlID: appID.String(),
			body:  "not-json{",
			repo:  &mockPublisherAppRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:  "app not found returns 404",
			urlID: appID.String(),
			body:  map[string]string{"name": "Updated"},
			repo: &mockPublisherAppRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:  "empty name in update returns 400",
			urlID: appID.String(),
			body:  map[string]string{"name": ""},
			repo: &mockPublisherAppRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
					return existing, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			h := newPublisherAppHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPut, "/v1/publisher-apps/"+tt.urlID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlID)

			w := httptest.NewRecorder()
			h.Update(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// marshalBody converts body to JSON bytes; string bodies are returned as-is.
func marshalBody(t *testing.T, body any) []byte {
	t.Helper()
	switch v := body.(type) {
	case string:
		return []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		return b
	}
}

// assertStatus checks the HTTP response code.
func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("status = %d, want %d; body = %s", w.Code, want, w.Body.String())
	}
}

// assertErrorCode decodes the response body and checks error.code.
func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	resp := decodeRespBody(t, w)
	errField, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field missing in response: %+v", resp)
	}
	got, _ := errField["code"].(string)
	if got != want {
		t.Errorf("error.code = %q, want %q", got, want)
	}
}

// withChiAppAndRuleID injects both {id} (app) and {ruleId} into the chi route context.
func withChiAppAndRuleID(r *http.Request, appID, ruleID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", appID)
	rctx.URLParams.Add("ruleId", ruleID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withChiAppAndKeyID injects both {id} (app) and {keyId} into the chi route context.
func withChiAppAndKeyID(r *http.Request, appID, keyID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", appID)
	rctx.URLParams.Add("keyId", keyID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// injectMiddlewareContext sets context values using middleware.InjectTestContext.
// Declared as local alias to keep test files self-contained within the package.
func injectMiddlewareContext(r *http.Request, orgID uuid.UUID, role string) *http.Request {
	ctx := middleware.InjectTestContext(r.Context(), uuid.New(), orgID, role, []uuid.UUID{orgID})
	return r.WithContext(ctx)
}
