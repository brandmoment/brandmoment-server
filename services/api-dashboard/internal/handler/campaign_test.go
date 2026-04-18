package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockCampaignRepoForHandler implements repository.CampaignRepository for handler tests.
type mockCampaignRepoForHandler struct {
	insertFn       func(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	getByIDFn      func(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error)
	listByOrgFn    func(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error)
	updateFn       func(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	updateStatusFn func(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error)
}

func (m *mockCampaignRepoForHandler) Insert(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	return m.insertFn(ctx, c)
}

func (m *mockCampaignRepoForHandler) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error) {
	return m.getByIDFn(ctx, orgID, id)
}

func (m *mockCampaignRepoForHandler) ListByOrg(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error) {
	return m.listByOrgFn(ctx, orgID, statusFilter, limit, offset)
}

func (m *mockCampaignRepoForHandler) Update(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	return m.updateFn(ctx, c)
}

func (m *mockCampaignRepoForHandler) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error) {
	return m.updateStatusFn(ctx, orgID, id, status, updatedAt)
}

// compile-time check: mockCampaignRepoForHandler satisfies repository.CampaignRepository.
var _ repository.CampaignRepository = (*mockCampaignRepoForHandler)(nil)

func newCampaignHandler(repo *mockCampaignRepoForHandler) *CampaignHandler {
	svc := service.NewCampaignService(repo, noop.NewTracerProvider())
	return NewCampaignHandler(svc)
}

func stubCampaign(orgID uuid.UUID) *model.Campaign {
	return &model.Campaign{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      "Summer 2026",
		Status:    model.StatusDraft,
		Targeting: model.CampaignTargeting{},
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestCampaignHandler_Create covers POST /v1/campaigns.
func TestCampaignHandler_Create(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		body        any
		repo        *mockCampaignRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name: "valid request returns 201",
			body: map[string]any{"name": "Summer 2026", "currency": "USD"},
			repo: &mockCampaignRepoForHandler{
				insertFn: func(_ context.Context, c *model.Campaign) (*model.Campaign, error) {
					return c, nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:        "invalid JSON body returns 400",
			body:        "not-json{",
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:        "empty name returns 400",
			body:        map[string]any{"name": ""},
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "name longer than 200 chars returns 400",
			body:        map[string]any{"name": strings.Repeat("x", 201)},
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "negative budget_cents returns 400",
			body: map[string]any{"name": "Bad Budget", "budget_cents": -1},
			repo: &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "end_date before start_date returns 400",
			body: map[string]any{
				"name":       "Date Error",
				"start_date": "2026-09-01",
				"end_date":   "2026-08-01",
			},
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name: "repo insert error returns 500",
			body: map[string]any{"name": "Fail Campaign"},
			repo: &mockCampaignRepoForHandler{
				insertFn: func(_ context.Context, _ *model.Campaign) (*model.Campaign, error) {
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
			h := newCampaignHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPost, "/v1/campaigns", bytes.NewReader(bodyBytes))
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

// TestCampaignHandler_GetByID covers GET /v1/campaigns/{id}.
func TestCampaignHandler_GetByID(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	tests := []struct {
		name        string
		urlID       string
		getByIDFn   func(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error)
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "found campaign returns 200",
			urlID: campaignID.String(),
			getByIDFn: func(_ context.Context, orgID, id uuid.UUID) (*model.Campaign, error) {
				c := stubCampaign(orgID)
				c.ID = id
				return c, nil
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
			name:  "campaign not found returns 404",
			urlID: campaignID.String(),
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:  "campaign belongs to different org returns 404",
			urlID: campaignID.String(),
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
				// Repository enforces org_id — cross-org access returns ErrNotFound.
				return nil, model.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCampaignRepoForHandler{}
			if tt.getByIDFn != nil {
				repo.getByIDFn = tt.getByIDFn
			}
			h := newCampaignHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/campaigns/"+tt.urlID, nil)
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

// TestCampaignHandler_List covers GET /v1/campaigns with optional status filter.
func TestCampaignHandler_List(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name          string
		query         string
		listByOrgFn   func(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error)
		wantStatus    int
		wantErrCode   string
		checkData     func(t *testing.T, data map[string]any)
	}{
		{
			name:  "returns all campaigns with default pagination",
			query: "",
			listByOrgFn: func(_ context.Context, orgID uuid.UUID, statusFilter *string, _, _ int32) ([]model.Campaign, int64, error) {
				if statusFilter != nil {
					t.Errorf("expected nil statusFilter, got %q", *statusFilter)
				}
				return []model.Campaign{*stubCampaign(orgID)}, 1, nil
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
			name:  "status filter passed to repo",
			query: "?status=active",
			listByOrgFn: func(_ context.Context, orgID uuid.UUID, statusFilter *string, _, _ int32) ([]model.Campaign, int64, error) {
				if statusFilter == nil || *statusFilter != "active" {
					t.Errorf("expected statusFilter=active, got %v", statusFilter)
				}
				c := stubCampaign(orgID)
				c.Status = model.StatusActive
				return []model.Campaign{*c}, 1, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid status filter returns 400",
			query:       "?status=invalid",
			listByOrgFn: nil,
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:  "returns empty list when no campaigns",
			query: "?limit=20&offset=0",
			listByOrgFn: func(_ context.Context, _ uuid.UUID, _ *string, _, _ int32) ([]model.Campaign, int64, error) {
				return []model.Campaign{}, 0, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "repo error returns 500",
			query: "",
			listByOrgFn: func(_ context.Context, _ uuid.UUID, _ *string, _, _ int32) ([]model.Campaign, int64, error) {
				return nil, 0, context.DeadlineExceeded
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCampaignRepoForHandler{}
			if tt.listByOrgFn != nil {
				repo.listByOrgFn = tt.listByOrgFn
			}
			h := newCampaignHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/v1/campaigns"+tt.query, nil)
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

// TestCampaignHandler_Update covers PUT /v1/campaigns/{id}.
func TestCampaignHandler_Update(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	existing := stubCampaign(orgID)
	existing.ID = campaignID

	tests := []struct {
		name        string
		urlID       string
		body        any
		repo        *mockCampaignRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "valid name update returns 200",
			urlID: campaignID.String(),
			body:  map[string]any{"name": "Updated Campaign"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existing, nil
				},
				updateFn: func(_ context.Context, c *model.Campaign) (*model.Campaign, error) {
					return c, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "update budget and currency returns 200",
			urlID: campaignID.String(),
			body:  map[string]any{"budget_cents": int64(100000), "currency": "EUR"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existing, nil
				},
				updateFn: func(_ context.Context, c *model.Campaign) (*model.Campaign, error) {
					return c, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			body:        map[string]any{"name": "Updated"},
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:  "invalid JSON body returns 400",
			urlID: campaignID.String(),
			body:  "not-json{",
			repo:  &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:  "campaign not found returns 404",
			urlID: campaignID.String(),
			body:  map[string]any{"name": "Updated"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:  "empty name in update returns 400",
			urlID: campaignID.String(),
			body:  map[string]any{"name": ""},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existing, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:  "end_date before start_date in update returns 400",
			urlID: campaignID.String(),
			body: map[string]any{
				"start_date": "2026-09-01",
				"end_date":   "2026-08-01",
			},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					c := *existing
					c.StartDate = nil
					c.EndDate = nil
					return &c, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes := marshalBody(t, tt.body)
			h := newCampaignHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPut, "/v1/campaigns/"+tt.urlID, bytes.NewReader(bodyBytes))
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

// TestCampaignHandler_UpdateStatus covers PATCH /v1/campaigns/{id}/status.
// Includes invalid status transition test.
func TestCampaignHandler_UpdateStatus(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	draftCampaign := stubCampaign(orgID)
	draftCampaign.ID = campaignID
	draftCampaign.Status = model.StatusDraft

	activeCampaign := stubCampaign(orgID)
	activeCampaign.ID = campaignID
	activeCampaign.Status = model.StatusActive

	completedCampaign := stubCampaign(orgID)
	completedCampaign.ID = campaignID
	completedCampaign.Status = model.StatusCompleted

	tests := []struct {
		name        string
		urlID       string
		body        any
		repo        *mockCampaignRepoForHandler
		wantStatus  int
		wantErrCode string
	}{
		{
			name:  "draft to active returns 200",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "active"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return draftCampaign, nil
				},
				updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status model.CampaignStatus, _ time.Time) (*model.Campaign, error) {
					c := *draftCampaign
					c.Status = status
					return &c, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "active to paused returns 200",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "paused"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return activeCampaign, nil
				},
				updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status model.CampaignStatus, _ time.Time) (*model.Campaign, error) {
					c := *activeCampaign
					c.Status = status
					return &c, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "active to completed returns 200",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "completed"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return activeCampaign, nil
				},
				updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status model.CampaignStatus, _ time.Time) (*model.Campaign, error) {
					c := *activeCampaign
					c.Status = status
					return &c, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			// Invalid transition: draft → paused (not allowed by state machine).
			name:  "invalid transition draft to paused returns 400",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "paused"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return draftCampaign, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			// Invalid transition: completed → active (terminal state, no transitions allowed).
			name:  "invalid transition completed to active returns 400",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "active"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return completedCampaign, nil
				},
			},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_INPUT",
		},
		{
			name:        "invalid UUID in URL returns 400",
			urlID:       "not-a-uuid",
			body:        map[string]string{"status": "active"},
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_ID",
		},
		{
			name:        "invalid JSON body returns 400",
			urlID:       campaignID.String(),
			body:        "not-json{",
			repo:        &mockCampaignRepoForHandler{},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_BODY",
		},
		{
			name:  "campaign not found returns 404",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "active"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:  "repo update status error returns 500",
			urlID: campaignID.String(),
			body:  map[string]string{"status": "active"},
			repo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return draftCampaign, nil
				},
				updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _ model.CampaignStatus, _ time.Time) (*model.Campaign, error) {
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
			h := newCampaignHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPatch, "/v1/campaigns/"+tt.urlID+"/status", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlID)

			w := httptest.NewRecorder()
			h.UpdateStatus(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}
