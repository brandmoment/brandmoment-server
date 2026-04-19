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

// mockCreativeRepo implements repository.CreativeRepository with function fields.
type mockCreativeRepo struct {
	insertFn         func(ctx context.Context, c *model.Creative) (*model.Creative, error)
	getByIDFn        func(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error)
	listByCampaignFn func(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error)
	updateFn         func(ctx context.Context, orgID, campaignID, id uuid.UUID, params repository.UpdateCreativeParams) (*model.Creative, error)
}

func (m *mockCreativeRepo) Insert(ctx context.Context, c *model.Creative) (*model.Creative, error) {
	return m.insertFn(ctx, c)
}

func (m *mockCreativeRepo) GetByID(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error) {
	return m.getByIDFn(ctx, orgID, campaignID, id)
}

func (m *mockCreativeRepo) ListByCampaign(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error) {
	return m.listByCampaignFn(ctx, orgID, campaignID)
}

func (m *mockCreativeRepo) Update(ctx context.Context, orgID, campaignID, id uuid.UUID, params repository.UpdateCreativeParams) (*model.Creative, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, orgID, campaignID, id, params)
	}
	return nil, nil
}

// Ensure mockCreativeRepo satisfies the interface at compile time.
var _ repository.CreativeRepository = (*mockCreativeRepo)(nil)

func TestCreativeService_Create(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()
	fileSize := int64(204800)
	negFileSize := int64(-1)
	zeroFileSize := int64(0)

	campaignFound := func(_ context.Context, _, id uuid.UUID) (*model.Campaign, error) {
		return &model.Campaign{ID: id, OrgID: orgID, Status: model.StatusDraft}, nil
	}
	campaignNotFound := func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
		return nil, model.ErrNotFound
	}

	tests := []struct {
		name          string
		req           CreateCreativeRequest
		campaignGetFn func(context.Context, uuid.UUID, uuid.UUID) (*model.Campaign, error)
		wantErr       bool
		wantErrIs     error
		check         func(t *testing.T, c *model.Creative)
	}{
		{
			name:          "valid html5 creative",
			req:           CreateCreativeRequest{Name: "Banner 320x50", Type: "html5", FileURL: "s3://bucket/file.zip", FileSizeBytes: &fileSize},
			campaignGetFn: campaignFound,
			check: func(t *testing.T, c *model.Creative) {
				if !c.IsActive {
					t.Error("expected is_active=true")
				}
				if c.OrgID != orgID {
					t.Errorf("expected org_id %s, got %s", orgID, c.OrgID)
				}
				if c.CampaignID != campaignID {
					t.Errorf("expected campaign_id %s, got %s", campaignID, c.CampaignID)
				}
				if c.Type != model.TypeHTML5 {
					t.Errorf("expected type html5, got %s", c.Type)
				}
			},
		},
		{
			name:          "campaign not found",
			req:           CreateCreativeRequest{Name: "Banner", Type: "html5", FileURL: "s3://bucket/file.zip"},
			campaignGetFn: campaignNotFound,
			wantErr:       true,
			wantErrIs:     model.ErrNotFound,
		},
		{
			name:          "empty name",
			req:           CreateCreativeRequest{Name: "", Type: "html5", FileURL: "s3://bucket/file.zip"},
			campaignGetFn: campaignFound,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:          "name too long",
			req:           CreateCreativeRequest{Name: strings.Repeat("a", 201), Type: "html5", FileURL: "s3://bucket/file.zip"},
			campaignGetFn: campaignFound,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:          "invalid type",
			req:           CreateCreativeRequest{Name: "Banner", Type: "gif", FileURL: "s3://bucket/file.zip"},
			campaignGetFn: campaignFound,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:          "negative file_size_bytes",
			req:           CreateCreativeRequest{Name: "Banner", Type: "image", FileURL: "s3://bucket/file.zip", FileSizeBytes: &negFileSize},
			campaignGetFn: campaignFound,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
		{
			name:          "zero file_size_bytes",
			req:           CreateCreativeRequest{Name: "Banner", Type: "image", FileURL: "s3://bucket/file.zip", FileSizeBytes: &zeroFileSize},
			campaignGetFn: campaignFound,
			wantErr:       true,
			wantErrIs:     model.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaignRepo := &mockCampaignRepo{
				getByIDFn: tt.campaignGetFn,
			}
			creativeRepo := &mockCreativeRepo{
				insertFn: func(_ context.Context, c *model.Creative) (*model.Creative, error) {
					return c, nil
				},
			}
			svc := NewCreativeService(campaignRepo, creativeRepo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), orgID, campaignID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Create() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
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

func TestCreativeService_ListByCampaign(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	campaignFound := func(_ context.Context, _, id uuid.UUID) (*model.Campaign, error) {
		return &model.Campaign{ID: id, OrgID: orgID, Status: model.StatusActive}, nil
	}
	campaignNotFound := func(_ context.Context, _, _ uuid.UUID) (*model.Campaign, error) {
		return nil, model.ErrNotFound
	}

	tests := []struct {
		name          string
		campaignGetFn func(context.Context, uuid.UUID, uuid.UUID) (*model.Campaign, error)
		listReturn    []model.Creative
		total         int64
		wantErr       bool
		wantErrIs     error
	}{
		{
			name:          "returns creatives",
			campaignGetFn: campaignFound,
			listReturn:    []model.Creative{{ID: uuid.New(), CampaignID: campaignID, OrgID: orgID}},
			total:         1,
		},
		{
			name:          "empty list",
			campaignGetFn: campaignFound,
			listReturn:    []model.Creative{},
			total:         0,
		},
		{
			name:          "campaign not found",
			campaignGetFn: campaignNotFound,
			wantErr:       true,
			wantErrIs:     model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaignRepo := &mockCampaignRepo{
				getByIDFn: tt.campaignGetFn,
			}
			creativeRepo := &mockCreativeRepo{
				listByCampaignFn: func(_ context.Context, _, _ uuid.UUID) ([]model.Creative, int64, error) {
					return tt.listReturn, tt.total, nil
				},
			}
			svc := NewCreativeService(campaignRepo, creativeRepo, noop.NewTracerProvider())

			result, err := svc.ListByCampaign(context.Background(), orgID, campaignID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListByCampaign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("ListByCampaign() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
			}
			if !tt.wantErr {
				if result == nil {
					t.Fatal("ListByCampaign() returned nil")
				}
				if result.Total != tt.total {
					t.Errorf("ListByCampaign() total = %v, want %v", result.Total, tt.total)
				}
				if len(result.Items) != len(tt.listReturn) {
					t.Errorf("ListByCampaign() items count = %v, want %v", len(result.Items), len(tt.listReturn))
				}
			}
		})
	}
}

func TestCreativeService_Update(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()
	creativeID := uuid.New()

	existingCreative := &model.Creative{
		ID:         creativeID,
		OrgID:      orgID,
		CampaignID: campaignID,
		Name:       "Old Banner",
		Type:       model.TypeHTML5,
		FileURL:    "s3://bucket/old.zip",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	successUpdateFn := func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
		return existingCreative, nil
	}
	notFoundUpdateFn := func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
		return nil, model.ErrNotFound
	}
	repoErrUpdateFn := func(_ context.Context, _, _, _ uuid.UUID, _ repository.UpdateCreativeParams) (*model.Creative, error) {
		return nil, context.DeadlineExceeded
	}

	tests := []struct {
		name      string
		req       UpdateCreativeRequest
		updateFn  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, repository.UpdateCreativeParams) (*model.Creative, error)
		wantErr   bool
		wantErrIs error
		check     func(t *testing.T, c *model.Creative)
	}{
		{
			name:     "success",
			req:      UpdateCreativeRequest{Name: "Updated Banner", Type: "html5", FileURL: "s3://bucket/new.zip", IsActive: true},
			updateFn: successUpdateFn,
			check: func(t *testing.T, c *model.Creative) {
				t.Helper()
				if c == nil {
					t.Fatal("Update() returned nil creative")
				}
				if c.ID != creativeID {
					t.Errorf("Update() creative ID = %v, want %v", c.ID, creativeID)
				}
			},
		},
		{
			name:      "empty name",
			req:       UpdateCreativeRequest{Name: "", Type: "html5", FileURL: "s3://bucket/new.zip"},
			updateFn:  successUpdateFn,
			wantErr:   true,
			wantErrIs: model.ErrInvalidInput,
		},
		{
			name:      "name too long",
			req:       UpdateCreativeRequest{Name: strings.Repeat("x", 201), Type: "html5", FileURL: "s3://bucket/new.zip"},
			updateFn:  successUpdateFn,
			wantErr:   true,
			wantErrIs: model.ErrInvalidInput,
		},
		{
			name:      "invalid type",
			req:       UpdateCreativeRequest{Name: "Banner", Type: "gif", FileURL: "s3://bucket/new.zip"},
			updateFn:  successUpdateFn,
			wantErr:   true,
			wantErrIs: model.ErrInvalidInput,
		},
		{
			name:      "not found from repo",
			req:       UpdateCreativeRequest{Name: "Banner", Type: "image", FileURL: "s3://bucket/new.zip"},
			updateFn:  notFoundUpdateFn,
			wantErr:   true,
			wantErrIs: model.ErrNotFound,
		},
		{
			name:     "repo error propagated",
			req:      UpdateCreativeRequest{Name: "Banner", Type: "video", FileURL: "s3://bucket/new.mp4"},
			updateFn: repoErrUpdateFn,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaignRepo := &mockCampaignRepo{}
			creativeRepo := &mockCreativeRepo{
				updateFn: tt.updateFn,
			}
			svc := NewCreativeService(campaignRepo, creativeRepo, noop.NewTracerProvider())

			got, err := svc.Update(context.Background(), orgID, campaignID, creativeID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Update() error = %v, want errors.Is(%v)", err, tt.wantErrIs)
			}
			if !tt.wantErr {
				if got == nil {
					t.Fatal("Update() returned nil")
				}
				if tt.check != nil {
					tt.check(t, got)
				}
			}
		})
	}
}

// Compile-time check that unused field in mock doesn't break the test file.
var _ = time.Now
