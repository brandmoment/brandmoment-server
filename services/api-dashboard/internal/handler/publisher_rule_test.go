package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockPublisherRuleRepoForHandler implements repository.PublisherRuleRepository for handler tests.
type mockPublisherRuleRepoForHandler struct {
	insertFn   func(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	getByIDFn  func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error)
	listFn     func(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error)
	updateFn   func(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	deleteFn   func(ctx context.Context, orgID, appID, id uuid.UUID) error
}

func (m *mockPublisherRuleRepoForHandler) Insert(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	return m.insertFn(ctx, rule)
}

func (m *mockPublisherRuleRepoForHandler) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error) {
	return m.getByIDFn(ctx, orgID, appID, id)
}

func (m *mockPublisherRuleRepoForHandler) ListByApp(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error) {
	return m.listFn(ctx, orgID, appID, limit, offset)
}

func (m *mockPublisherRuleRepoForHandler) Update(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	return m.updateFn(ctx, rule)
}

func (m *mockPublisherRuleRepoForHandler) Delete(ctx context.Context, orgID, appID, id uuid.UUID) error {
	return m.deleteFn(ctx, orgID, appID, id)
}

// compile-time check: mockPublisherRuleRepoForHandler satisfies repository.PublisherRuleRepository.
var _ repository.PublisherRuleRepository = (*mockPublisherRuleRepoForHandler)(nil)

func newPublisherRuleHandler(repo *mockPublisherRuleRepoForHandler) *PublisherRuleHandler {
	svc := service.NewPublisherRuleService(repo, noop.NewTracerProvider())
	return NewPublisherRuleHandler(svc)
}

func stubPublisherRule(orgID, appID uuid.UUID, ruleType string, config json.RawMessage) *model.PublisherRule {
	return &model.PublisherRule{
		ID:        uuid.New(),
		OrgID:     orgID,
		AppID:     appID,
		Type:      ruleType,
		Config:    config,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestPublisherRuleHandler_Create covers POST /v1/publisher-apps/{id}/rules.
func TestPublisherRuleHandler_Create(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	validBlocklistConfig := json.RawMessage(`{"domains":["bad.com"],"bundle_ids":[]}`)
	validFreqCapConfig := json.RawMessage(`{"max_impressions":10,"window_seconds":3600}`)
	validGeoFilterConfig := json.RawMessage(`{"mode":"exclude","country_codes":["CN"]}`)

	tests := []struct {
		name        string
		urlAppID    string
		body        any
		insertFn    func(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:     "valid blocklist rule returns 201",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "blocklist", "config": map[string]any{"domains": []string{"bad.com"}, "bundle_ids": []string{}}},
			insertFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
				return rule, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "valid frequency_cap rule returns 201",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "frequency_cap", "config": map[string]any{"max_impressions": 10, "window_seconds": 3600}},
			insertFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
				return rule, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "valid geo_filter rule returns 201",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "geo_filter", "config": map[string]any{"mode": "exclude", "country_codes": []string{"CN"}}},
			insertFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
				return rule, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "invalid app UUID in URL returns 400",
			urlAppID:    "not-a-uuid",
			body:        map[string]any{"type": "blocklist", "config": map[string]any{"domains": []string{"bad.com"}, "bundle_ids": []string{}}},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid JSON body returns 400",
			urlAppID:    appID.String(),
			body:        "not-json{",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:        "unknown rule type returns 400",
			urlAppID:    appID.String(),
			body:        map[string]any{"type": "unknown_type", "config": map[string]any{}},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "frequency_cap missing max_impressions returns 400",
			urlAppID:    appID.String(),
			body:        map[string]any{"type": "frequency_cap", "config": map[string]any{"window_seconds": 3600}},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "geo_filter invalid mode returns 400",
			urlAppID:    appID.String(),
			body:        map[string]any{"type": "geo_filter", "config": map[string]any{"mode": "unknown", "country_codes": []string{"US"}}},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "geo_filter empty country_codes returns 400",
			urlAppID:    appID.String(),
			body:        map[string]any{"type": "geo_filter", "config": map[string]any{"mode": "include", "country_codes": []string{}}},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:     "repo insert error returns 500",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "blocklist", "config": map[string]any{"domains": []string{"bad.com"}, "bundle_ids": []string{}}},
			insertFn: func(_ context.Context, _ *model.PublisherRule) (*model.PublisherRule, error) {
				return nil, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
		// Keep these here so callers exercise the stubPublisherRule helper.
		{
			name:     "blocklist config reference",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "blocklist", "config": map[string]any{"domains": []string{"bad.com"}, "bundle_ids": []string{}}},
			insertFn: func(_ context.Context, _ *model.PublisherRule) (*model.PublisherRule, error) {
				return stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, validBlocklistConfig), nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "frequency_cap config reference",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "frequency_cap", "config": map[string]any{"max_impressions": 5, "window_seconds": 60}},
			insertFn: func(_ context.Context, _ *model.PublisherRule) (*model.PublisherRule, error) {
				return stubPublisherRule(orgID, appID, model.RuleTypeFrequencyCap, validFreqCapConfig), nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:     "geo_filter config reference",
			urlAppID: appID.String(),
			body:     map[string]any{"type": "geo_filter", "config": map[string]any{"mode": "include", "country_codes": []string{"US"}}},
			insertFn: func(_ context.Context, _ *model.PublisherRule) (*model.PublisherRule, error) {
				return stubPublisherRule(orgID, appID, model.RuleTypeGeoFilter, validGeoFilterConfig), nil
			},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			repo := &mockPublisherRuleRepoForHandler{}
			if tt.insertFn != nil {
				repo.insertFn = tt.insertFn
			}
			h := newPublisherRuleHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/v1/publisher-apps/"+tt.urlAppID+"/rules", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlAppID)

			w := httptest.NewRecorder()
			h.Create(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestPublisherRuleHandler_GetByID covers GET /v1/publisher-apps/{id}/rules/{ruleId}.
func TestPublisherRuleHandler_GetByID(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	ruleID := uuid.New()
	config := json.RawMessage(`{"domains":["bad.com"],"bundle_ids":[]}`)

	tests := []struct {
		name        string
		urlAppID    string
		urlRuleID   string
		getByIDFn   func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:      "found rule returns 200",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			getByIDFn: func(_ context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error) {
				return stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, config), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid app UUID returns 400",
			urlAppID:    "not-a-uuid",
			urlRuleID:   ruleID.String(),
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid rule UUID returns 400",
			urlAppID:    appID.String(),
			urlRuleID:   "not-a-uuid",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:      "rule not found returns 404",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherRuleRepoForHandler{}
			if tt.getByIDFn != nil {
				repo.getByIDFn = tt.getByIDFn
			}
			h := newPublisherRuleHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/publisher-apps/"+tt.urlAppID+"/rules/"+tt.urlRuleID, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiAppAndRuleID(req, tt.urlAppID, tt.urlRuleID)

			w := httptest.NewRecorder()
			h.GetByID(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestPublisherRuleHandler_ListByApp covers GET /v1/publisher-apps/{id}/rules.
func TestPublisherRuleHandler_ListByApp(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	config := json.RawMessage(`{"domains":["bad.com"],"bundle_ids":[]}`)

	tests := []struct {
		name        string
		urlAppID    string
		query       string
		listFn      func(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error)
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:     "returns rules with default pagination",
			urlAppID: appID.String(),
			query:    "",
			listFn: func(_ context.Context, _, _ uuid.UUID, _, _ int32) ([]model.PublisherRule, int64, error) {
				return []model.PublisherRule{*stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, config)}, 1, nil
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				items, ok := data["items"].([]any)
				if !ok {
					t.Fatalf("items missing: %+v", data)
				}
				if len(items) != 1 {
					t.Errorf("items count = %d, want 1", len(items))
				}
			},
		},
		{
			name:     "returns empty list when no rules",
			urlAppID: appID.String(),
			query:    "?limit=50&offset=0",
			listFn: func(_ context.Context, _, _ uuid.UUID, _, _ int32) ([]model.PublisherRule, int64, error) {
				return []model.PublisherRule{}, 0, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid app UUID returns 400",
			urlAppID:    "not-a-uuid",
			query:       "",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:     "repo error returns 500",
			urlAppID: appID.String(),
			query:    "",
			listFn: func(_ context.Context, _, _ uuid.UUID, _, _ int32) ([]model.PublisherRule, int64, error) {
				return nil, 0, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherRuleRepoForHandler{}
			if tt.listFn != nil {
				repo.listFn = tt.listFn
			}
			h := newPublisherRuleHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/publisher-apps/"+tt.urlAppID+"/rules"+tt.query, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlAppID)

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

// TestPublisherRuleHandler_Update covers PUT /v1/publisher-apps/{id}/rules/{ruleId}.
func TestPublisherRuleHandler_Update(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	ruleID := uuid.New()
	config := json.RawMessage(`{"domains":["bad.com"],"bundle_ids":[]}`)

	existingRule := stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, config)
	existingRule.ID = ruleID

	tests := []struct {
		name        string
		urlAppID    string
		urlRuleID   string
		body        any
		repo        *mockPublisherRuleRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name:      "valid config update returns 200",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			body:      map[string]any{"config": map[string]any{"domains": []string{"updated.com"}, "bundle_ids": []string{}}},
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return existingRule, nil
				},
				updateFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
					return rule, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "valid is_active toggle returns 200",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			body:      map[string]any{"is_active": false},
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return existingRule, nil
				},
				updateFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
					return rule, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid app UUID returns 400",
			urlAppID:    "not-a-uuid",
			urlRuleID:   ruleID.String(),
			body:        map[string]any{"is_active": false},
			repo:        &mockPublisherRuleRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid rule UUID returns 400",
			urlAppID:    appID.String(),
			urlRuleID:   "not-a-uuid",
			body:        map[string]any{"is_active": false},
			repo:        &mockPublisherRuleRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:      "invalid JSON body returns 400",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			body:      "not-json{",
			repo:      &mockPublisherRuleRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:      "rule not found returns 404",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			body:      map[string]any{"is_active": false},
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			h := newPublisherRuleHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPut, "/v1/publisher-apps/"+tt.urlAppID+"/rules/"+tt.urlRuleID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiAppAndRuleID(req, tt.urlAppID, tt.urlRuleID)

			w := httptest.NewRecorder()
			h.Update(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestPublisherRuleHandler_Delete covers DELETE /v1/publisher-apps/{id}/rules/{ruleId}.
func TestPublisherRuleHandler_Delete(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	ruleID := uuid.New()
	config := json.RawMessage(`{"domains":["bad.com"],"bundle_ids":[]}`)

	tests := []struct {
		name        string
		urlAppID    string
		urlRuleID   string
		repo        *mockPublisherRuleRepoForHandler
		wantStatus  int
		wantErrCode string
		checkData   func(t *testing.T, data map[string]any)
	}{
		{
			name:      "valid delete returns 200 with id",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, config), nil
				},
				deleteFn: func(_ context.Context, _, _, _ uuid.UUID) error {
					return nil
				},
			},
			wantStatus: http.StatusOK,
			checkData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if _, ok := data["id"]; !ok {
					t.Error("delete response must include 'id' field")
				}
			},
		},
		{
			name:        "invalid app UUID returns 400",
			urlAppID:    "not-a-uuid",
			urlRuleID:   ruleID.String(),
			repo:        &mockPublisherRuleRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid rule UUID returns 400",
			urlAppID:    appID.String(),
			urlRuleID:   "not-a-uuid",
			repo:        &mockPublisherRuleRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:      "rule not found returns 404",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:      "repo delete error returns 500",
			urlAppID:  appID.String(),
			urlRuleID: ruleID.String(),
			repo: &mockPublisherRuleRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					return stubPublisherRule(orgID, appID, model.RuleTypeBlocklist, config), nil
				},
				deleteFn: func(_ context.Context, _, _, _ uuid.UUID) error {
					return context.DeadlineExceeded
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPublisherRuleHandler(tt.repo)

			req := httptest.NewRequest(http.MethodDelete, "/v1/publisher-apps/"+tt.urlAppID+"/rules/"+tt.urlRuleID, nil)
			req = injectAuthContext(req, orgID, "admin", []uuid.UUID{orgID})
			req = withChiAppAndRuleID(req, tt.urlAppID, tt.urlRuleID)

			w := httptest.NewRecorder()
			h.Delete(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
				return
			}

			if tt.checkData != nil {
				var resp map[string]any
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				data, ok := resp["data"].(map[string]any)
				if !ok {
					t.Fatalf("data field missing: %+v", resp)
				}
				tt.checkData(t, data)
			}
		})
	}
}
