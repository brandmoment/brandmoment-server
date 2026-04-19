package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

// mockCreativeRepoForHandler implements repository.CreativeRepository for handler tests.
type mockCreativeRepoForHandler struct {
	insertFn         func(ctx context.Context, c *model.Creative) (*model.Creative, error)
	getByIDFn        func(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error)
	listByCampaignFn func(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error)
	updateFn         func(ctx context.Context, orgID, campaignID, id uuid.UUID, params repository.UpdateCreativeParams) (*model.Creative, error)
}

func (m *mockCreativeRepoForHandler) Insert(ctx context.Context, c *model.Creative) (*model.Creative, error) {
	return m.insertFn(ctx, c)
}

func (m *mockCreativeRepoForHandler) GetByID(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error) {
	return m.getByIDFn(ctx, orgID, campaignID, id)
}

func (m *mockCreativeRepoForHandler) ListByCampaign(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error) {
	return m.listByCampaignFn(ctx, orgID, campaignID)
}

func (m *mockCreativeRepoForHandler) Update(ctx context.Context, orgID, campaignID, id uuid.UUID, params repository.UpdateCreativeParams) (*model.Creative, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, orgID, campaignID, id, params)
	}
	return nil, nil
}

// compile-time check: mockCreativeRepoForHandler satisfies repository.CreativeRepository.
var _ repository.CreativeRepository = (*mockCreativeRepoForHandler)(nil)

func newCreativeHandler(campaignRepo *mockCampaignRepoForHandler, creativeRepo *mockCreativeRepoForHandler) *CreativeHandler {
	svc := service.NewCreativeService(campaignRepo, creativeRepo, noop.NewTracerProvider())
	return NewCreativeHandler(svc)
}

func stubCreative(orgID, campaignID uuid.UUID) *model.Creative {
	return &model.Creative{
		ID:         uuid.New(),
		OrgID:      orgID,
		CampaignID: campaignID,
		Name:       "Banner 320x50",
		Type:       model.TypeHTML5,
		FileURL:    "s3://bucket/banner.zip",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// withChiCampaignAndCreativeID injects both {id} (campaign) and {creativeId} into the chi route context.
func withChiCampaignAndCreativeID(r *http.Request, campaignID, creativeID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", campaignID)
	rctx.URLParams.Add("creativeId", creativeID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// TestCreativeHandler_Create covers POST /v1/campaigns/{id}/creatives.
func TestCreativeHandler_Create(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	existingCampaign := stubCampaign(orgID)
	existingCampaign.ID = campaignID

	campaignFoundRepo := &mockCampaignRepoForHandler{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
			return existingCampaign, nil
		},
	}
	campaignNotFoundRepo := &mockCampaignRepoForHandler{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
			return nil, model.ErrNotFound
		},
	}

	tests := []struct {
		name         string
		urlID        string
		body         any
		campaignRepo *mockCampaignRepoForHandler
		creativeRepo *mockCreativeRepoForHandler
		wantStatus   int
		wantErrCode  string
	}{
		{
			name:         "valid html5 creative returns 201",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner 320x50", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: campaignFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{
				insertFn: func(_ context.Context, c *model.Creative) (*model.Creative, error) {
					return c, nil
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:         "campaign not found returns 404",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: campaignNotFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusNotFound,
			wantErrCode:  "NOT_FOUND",
		},
		{
			name:         "campaign belongs to different org returns 404",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: campaignNotFoundRepo, // repo enforces org_id — cross-org returns ErrNotFound
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusNotFound,
			wantErrCode:  "NOT_FOUND",
		},
		{
			name:         "invalid campaign UUID in URL returns 400",
			urlID:        "not-a-uuid",
			body:         map[string]any{"name": "Banner", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_ID",
		},
		{
			name:         "invalid JSON body returns 400",
			urlID:        campaignID.String(),
			body:         "not-json{",
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_BODY",
		},
		{
			name:         "empty name returns 400",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_INPUT",
		},
		{
			name:         "invalid type returns 400",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner", "type": "gif", "file_url": "s3://bucket/banner.gif"},
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_INPUT",
		},
		{
			name:         "negative file_size_bytes returns 400",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner", "type": "html5", "file_url": "s3://bucket/banner.zip", "file_size_bytes": int64(-1)},
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_INPUT",
		},
		{
			name:         "repo insert error returns 500",
			urlID:        campaignID.String(),
			body:         map[string]any{"name": "Banner", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo: campaignFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{
				insertFn: func(_ context.Context, _ *model.Creative) (*model.Creative, error) {
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
			h := newCreativeHandler(tt.campaignRepo, tt.creativeRepo)

			req := httptest.NewRequest(http.MethodPost, "/v1/campaigns/"+tt.urlID+"/creatives", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlID)

			w := httptest.NewRecorder()
			h.Create(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestCreativeHandler_ListByCampaign covers GET /v1/campaigns/{id}/creatives.
func TestCreativeHandler_ListByCampaign(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	existingCampaign := stubCampaign(orgID)
	existingCampaign.ID = campaignID

	tests := []struct {
		name         string
		urlID        string
		campaignRepo *mockCampaignRepoForHandler
		creativeRepo *mockCreativeRepoForHandler
		wantStatus   int
		wantErrCode  string
		checkData    func(t *testing.T, data map[string]any)
	}{
		{
			name:  "returns creatives for campaign",
			urlID: campaignID.String(),
			campaignRepo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existingCampaign, nil
				},
			},
			creativeRepo: &mockCreativeRepoForHandler{
				listByCampaignFn: func(_ context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error) {
					return []model.Creative{*stubCreative(orgID, campaignID)}, 1, nil
				},
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
				total, ok := data["total"].(float64)
				if !ok {
					t.Fatalf("total field missing: %+v", data)
				}
				if int(total) != 1 {
					t.Errorf("total = %v, want 1", total)
				}
			},
		},
		{
			name:  "returns empty list when no creatives",
			urlID: campaignID.String(),
			campaignRepo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existingCampaign, nil
				},
			},
			creativeRepo: &mockCreativeRepoForHandler{
				listByCampaignFn: func(_ context.Context, _, _ uuid.UUID) ([]model.Creative, int64, error) {
					return []model.Creative{}, 0, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "campaign not found returns 404",
			urlID: campaignID.String(),
			campaignRepo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return nil, model.ErrNotFound
				},
			},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusNotFound,
			wantErrCode:  "NOT_FOUND",
		},
		{
			name:         "invalid campaign UUID in URL returns 400",
			urlID:        "not-a-uuid",
			campaignRepo: &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{},
			wantStatus:   http.StatusBadRequest,
			wantErrCode:  "INVALID_ID",
		},
		{
			name:  "repo list error returns 500",
			urlID: campaignID.String(),
			campaignRepo: &mockCampaignRepoForHandler{
				getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
					return existingCampaign, nil
				},
			},
			creativeRepo: &mockCreativeRepoForHandler{
				listByCampaignFn: func(_ context.Context, _, _ uuid.UUID) ([]model.Creative, int64, error) {
					return nil, 0, context.DeadlineExceeded
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newCreativeHandler(tt.campaignRepo, tt.creativeRepo)

			req := httptest.NewRequest(http.MethodGet, "/v1/campaigns/"+tt.urlID+"/creatives", nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiID(req, tt.urlID)

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

// TestCreativeHandler_Update covers PUT /v1/campaigns/{id}/creatives/{creativeId}.
func TestCreativeHandler_Update(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()
	creativeID := uuid.New()

	existingCreative := stubCreative(orgID, campaignID)
	existingCreative.ID = creativeID

	validBody := map[string]any{
		"name":     "Updated Banner",
		"type":     "html5",
		"file_url": "s3://bucket/banner.zip",
	}

	tests := []struct {
		name          string
		urlCampaignID string
		urlCreativeID string
		body          any
		campaignRepo  *mockCampaignRepoForHandler
		creativeRepo  *mockCreativeRepoForHandler
		wantStatus    int
		wantErrCode   string
	}{
		{
			name:          "success returns 200",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			body:          validBody,
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{
				updateFn: func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
					return existingCreative, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "creative not found returns 404",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			body:          validBody,
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{
				updateFn: func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:          "invalid campaign UUID returns 400",
			urlCampaignID: "not-a-uuid",
			urlCreativeID: creativeID.String(),
			body:          validBody,
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_ID",
		},
		{
			name:          "invalid creative UUID returns 400",
			urlCampaignID: campaignID.String(),
			urlCreativeID: "not-a-uuid",
			body:          validBody,
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_ID",
		},
		{
			name:          "invalid JSON body returns 400",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			body:          "not-json{",
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_BODY",
		},
		{
			name:          "empty name returns 400",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			body:          map[string]any{"name": "", "type": "html5", "file_url": "s3://bucket/banner.zip"},
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_INPUT",
		},
		{
			name:          "repo error returns 500",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			body:          validBody,
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo: &mockCreativeRepoForHandler{
				updateFn: func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
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
			h := newCreativeHandler(tt.campaignRepo, tt.creativeRepo)

			url := "/v1/campaigns/" + tt.urlCampaignID + "/creatives/" + tt.urlCreativeID
			req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = injectAuthContext(req, orgID, "editor", []uuid.UUID{orgID})
			req = withChiCampaignAndCreativeID(req, tt.urlCampaignID, tt.urlCreativeID)

			w := httptest.NewRecorder()
			h.Update(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}

// TestCreativeHandler_GetByID covers GET /v1/campaigns/{id}/creatives/{creativeId}.
func TestCreativeHandler_GetByID(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()
	creativeID := uuid.New()

	existingCampaign := stubCampaign(orgID)
	existingCampaign.ID = campaignID

	existingCreative := stubCreative(orgID, campaignID)
	existingCreative.ID = creativeID

	campaignFoundRepo := &mockCampaignRepoForHandler{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
			return existingCampaign, nil
		},
	}

	tests := []struct {
		name           string
		urlCampaignID  string
		urlCreativeID  string
		campaignRepo   *mockCampaignRepoForHandler
		creativeRepo   *mockCreativeRepoForHandler
		wantStatus     int
		wantErrCode    string
	}{
		{
			name:          "found creative returns 200",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			campaignRepo:  campaignFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.Creative, error) {
					return existingCreative, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "creative not found returns 404",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			campaignRepo:  campaignFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.Creative, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:          "creative belongs to different org returns 404",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			campaignRepo:  campaignFoundRepo,
			// repo enforces org_id filter — cross-org lookup returns ErrNotFound
			creativeRepo: &mockCreativeRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.Creative, error) {
					return nil, model.ErrNotFound
				},
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "NOT_FOUND",
		},
		{
			name:          "invalid campaign UUID in URL returns 400",
			urlCampaignID: "not-a-uuid",
			urlCreativeID: creativeID.String(),
			campaignRepo:  &mockCampaignRepoForHandler{},
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_ID",
		},
		{
			name:          "invalid creative UUID in URL returns 400",
			urlCampaignID: campaignID.String(),
			urlCreativeID: "not-a-uuid",
			campaignRepo:  campaignFoundRepo,
			creativeRepo:  &mockCreativeRepoForHandler{},
			wantStatus:    http.StatusBadRequest,
			wantErrCode:   "INVALID_ID",
		},
		{
			name:          "repo error returns 500",
			urlCampaignID: campaignID.String(),
			urlCreativeID: creativeID.String(),
			campaignRepo:  campaignFoundRepo,
			creativeRepo: &mockCreativeRepoForHandler{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.Creative, error) {
					return nil, context.DeadlineExceeded
				},
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newCreativeHandler(tt.campaignRepo, tt.creativeRepo)

			url := "/v1/campaigns/" + tt.urlCampaignID + "/creatives/" + tt.urlCreativeID
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = injectAuthContext(req, orgID, "viewer", []uuid.UUID{orgID})
			req = withChiCampaignAndCreativeID(req, tt.urlCampaignID, tt.urlCreativeID)

			w := httptest.NewRecorder()
			h.GetByID(w, req)

			assertStatus(t, w, tt.wantStatus)
			if tt.wantErrCode != "" {
				assertErrorCode(t, w, tt.wantErrCode)
			}
		})
	}
}
