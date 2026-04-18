package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
)

// mockCampaignRepo implements repository.CampaignRepository with function fields.
type mockCampaignRepo struct {
	insertFn       func(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	getByIDFn      func(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error)
	listByOrgFn    func(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error)
	updateFn       func(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	updateStatusFn func(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error)
}

func (m *mockCampaignRepo) Insert(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	return m.insertFn(ctx, c)
}

func (m *mockCampaignRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error) {
	return m.getByIDFn(ctx, orgID, id)
}

func (m *mockCampaignRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error) {
	return m.listByOrgFn(ctx, orgID, statusFilter, limit, offset)
}

func (m *mockCampaignRepo) Update(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	return m.updateFn(ctx, c)
}

func (m *mockCampaignRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error) {
	return m.updateStatusFn(ctx, orgID, id, status, updatedAt)
}

// Ensure mockCampaignRepo satisfies the interface at compile time.
var _ repository.CampaignRepository = (*mockCampaignRepo)(nil)

func defaultInsertFn(_ context.Context, c *model.Campaign) (*model.Campaign, error) {
	return c, nil
}

func defaultGetByIDFn(orgID uuid.UUID) func(context.Context, uuid.UUID, uuid.UUID) (*model.Campaign, error) {
	return func(_ context.Context, _, id uuid.UUID) (*model.Campaign, error) {
		return &model.Campaign{ID: id, OrgID: orgID, Status: model.StatusDraft}, nil
	}
}

func TestCampaignService_Create(t *testing.T) {
	orgID := uuid.New()
	tomorrow := "2026-05-01"
	yesterday := "2026-04-17"
	today := "2026-04-18"
	negBudget := int64(-1)

	tests := []struct {
		name    string
		req     CreateCampaignRequest
		wantErr bool
		check   func(t *testing.T, c *model.Campaign)
	}{
		{
			name: "valid campaign",
			req:  CreateCampaignRequest{Name: "Summer 2026", StartDate: &today, EndDate: &tomorrow},
			check: func(t *testing.T, c *model.Campaign) {
				if c.Status != model.StatusDraft {
					t.Errorf("expected status draft, got %s", c.Status)
				}
				if c.OrgID != orgID {
					t.Errorf("expected org_id %s, got %s", orgID, c.OrgID)
				}
				if c.Currency != "USD" {
					t.Errorf("expected default currency USD, got %s", c.Currency)
				}
			},
		},
		{
			name:    "empty name",
			req:     CreateCampaignRequest{Name: ""},
			wantErr: true,
		},
		{
			name:    "name too long",
			req:     CreateCampaignRequest{Name: strings.Repeat("a", 201)},
			wantErr: true,
		},
		{
			name:    "end before start",
			req:     CreateCampaignRequest{Name: "Test", StartDate: &tomorrow, EndDate: &yesterday},
			wantErr: true,
		},
		{
			name:    "negative budget",
			req:     CreateCampaignRequest{Name: "Test", BudgetCents: &negBudget},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCampaignRepo{
				insertFn: defaultInsertFn,
			}
			svc := NewCampaignService(repo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), orgID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Fatal("Create() returned nil")
				}
				if tt.check != nil {
					tt.check(t, got)
				}
			}
		})
	}
}

func TestCampaignService_UpdateStatus(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	tests := []struct {
		name          string
		currentStatus model.CampaignStatus
		targetStatus  model.CampaignStatus
		getByIDErr    error
		wantErr       bool
		wantErrIs     error
	}{
		{
			name:          "draft to active",
			currentStatus: model.StatusDraft,
			targetStatus:  model.StatusActive,
		},
		{
			name:          "active to paused",
			currentStatus: model.StatusActive,
			targetStatus:  model.StatusPaused,
		},
		{
			name:          "paused to active",
			currentStatus: model.StatusPaused,
			targetStatus:  model.StatusActive,
		},
		{
			name:          "active to completed",
			currentStatus: model.StatusActive,
			targetStatus:  model.StatusCompleted,
		},
		{
			name:          "paused to completed",
			currentStatus: model.StatusPaused,
			targetStatus:  model.StatusCompleted,
		},
		{
			name:          "completed to active — invalid",
			currentStatus: model.StatusCompleted,
			targetStatus:  model.StatusActive,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:          "draft to paused — invalid",
			currentStatus: model.StatusDraft,
			targetStatus:  model.StatusPaused,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:       "not found",
			getByIDErr: model.ErrNotFound,
			wantErr:    true,
			wantErrIs:  model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCampaignRepo{
				getByIDFn: func(_ context.Context, _, id uuid.UUID) (*model.Campaign, error) {
					if tt.getByIDErr != nil {
						return nil, tt.getByIDErr
					}
					return &model.Campaign{ID: id, OrgID: orgID, Status: tt.currentStatus}, nil
				},
				updateStatusFn: func(_ context.Context, _, id uuid.UUID, status model.CampaignStatus, _ time.Time) (*model.Campaign, error) {
					return &model.Campaign{ID: id, OrgID: orgID, Status: status}, nil
				},
			}
			svc := NewCampaignService(repo, noop.NewTracerProvider())

			got, err := svc.UpdateStatus(context.Background(), orgID, campaignID, UpdateCampaignStatusRequest{Status: tt.targetStatus})
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("UpdateStatus() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
			}
			if !tt.wantErr {
				if got == nil {
					t.Fatal("UpdateStatus() returned nil")
				}
				if got.Status != tt.targetStatus {
					t.Errorf("UpdateStatus() status = %s, want %s", got.Status, tt.targetStatus)
				}
			}
		})
	}
}

func TestCampaignService_List(t *testing.T) {
	orgID := uuid.New()
	activeStatus := "active"
	invalidStatus := "unknown"

	tests := []struct {
		name         string
		statusFilter *string
		limit        int32
		offset       int32
		repoReturn   []model.Campaign
		total        int64
		wantLimit    int32
		wantErr      bool
	}{
		{
			name:       "no filter",
			limit:      20,
			offset:     0,
			repoReturn: []model.Campaign{{ID: uuid.New()}},
			total:      1,
			wantLimit:  20,
		},
		{
			name:         "status filter active",
			statusFilter: &activeStatus,
			limit:        20,
			offset:       0,
			repoReturn:   []model.Campaign{},
			total:        0,
			wantLimit:    20,
		},
		{
			name:         "invalid status filter",
			statusFilter: &invalidStatus,
			limit:        20,
			wantErr:      true,
		},
		{
			name:       "default limit applied",
			limit:      0,
			offset:     0,
			repoReturn: []model.Campaign{},
			total:      0,
			wantLimit:  20,
		},
		{
			name:       "limit clamped to 100",
			limit:      200,
			offset:     0,
			repoReturn: []model.Campaign{},
			total:      0,
			wantLimit:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedLimit := int32(0)
			repo := &mockCampaignRepo{
				listByOrgFn: func(_ context.Context, _ uuid.UUID, _ *string, limit, _ int32) ([]model.Campaign, int64, error) {
					capturedLimit = limit
					return tt.repoReturn, tt.total, nil
				},
			}
			svc := NewCampaignService(repo, noop.NewTracerProvider())

			result, err := svc.List(context.Background(), orgID, tt.statusFilter, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Limit != tt.wantLimit {
					t.Errorf("List() limit = %v, want %v", result.Limit, tt.wantLimit)
				}
				if capturedLimit != tt.wantLimit {
					t.Errorf("List() repo called with limit = %v, want %v", capturedLimit, tt.wantLimit)
				}
				if result.Total != tt.total {
					t.Errorf("List() total = %v, want %v", result.Total, tt.total)
				}
			}
		})
	}
}

func TestCampaignService_Update(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()
	newName := "Updated Name"
	emptyName := ""
	longName := strings.Repeat("x", 201)
	budget := int64(1000)
	negBudget := int64(-1)
	endDate := "2026-08-01"
	startDate := "2026-09-01" // after end date

	tests := []struct {
		name    string
		req     UpdateCampaignRequest
		getErr  error
		wantErr bool
	}{
		{
			name: "update name only",
			req:  UpdateCampaignRequest{Name: &newName},
		},
		{
			name: "update all fields",
			req:  UpdateCampaignRequest{Name: &newName, BudgetCents: &budget, EndDate: &endDate},
		},
		{
			name:    "empty name",
			req:     UpdateCampaignRequest{Name: &emptyName},
			wantErr: true,
		},
		{
			name:    "name too long",
			req:     UpdateCampaignRequest{Name: &longName},
			wantErr: true,
		},
		{
			name:    "negative budget",
			req:     UpdateCampaignRequest{BudgetCents: &negBudget},
			wantErr: true,
		},
		{
			name:    "end before start (set start after existing end)",
			req:     UpdateCampaignRequest{StartDate: &startDate, EndDate: &endDate},
			wantErr: true,
		},
		{
			name:    "not found",
			req:     UpdateCampaignRequest{Name: &newName},
			getErr:  model.ErrNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCampaignRepo{
				getByIDFn: func(_ context.Context, _, id uuid.UUID) (*model.Campaign, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return &model.Campaign{ID: id, OrgID: orgID, Name: "Old Name", Status: model.StatusDraft, Currency: "USD"}, nil
				},
				updateFn: func(_ context.Context, c *model.Campaign) (*model.Campaign, error) {
					return c, nil
				},
			}
			svc := NewCampaignService(repo, noop.NewTracerProvider())

			got, err := svc.Update(context.Background(), orgID, campaignID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("Update() returned nil")
			}
		})
	}
}
