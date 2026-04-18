package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockPublisherAppRepo struct {
	insertFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
	getByIDFn     func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
	getByBundleFn func(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error)
	listByOrgFn   func(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
	updateFn      func(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
}

func (m *mockPublisherAppRepo) Insert(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	return m.insertFn(ctx, app)
}

func (m *mockPublisherAppRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error) {
	return m.getByIDFn(ctx, orgID, id)
}

func (m *mockPublisherAppRepo) GetByBundleID(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error) {
	return m.getByBundleFn(ctx, orgID, bundleID)
}

func (m *mockPublisherAppRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error) {
	return m.listByOrgFn(ctx, orgID, limit, offset)
}

func (m *mockPublisherAppRepo) Update(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	return m.updateFn(ctx, app)
}

func TestPublisherAppService_Create(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name        string
		req         CreatePublisherAppRequest
		bundleCheck func(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error)
		wantErr     bool
	}{
		{
			name:    "valid app",
			req:     CreatePublisherAppRequest{Name: "My App", Platform: "ios", BundleID: "com.example.app"},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     CreatePublisherAppRequest{Name: "", Platform: "ios", BundleID: "com.example.app"},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
		{
			name:    "invalid platform",
			req:     CreatePublisherAppRequest{Name: "My App", Platform: "windows", BundleID: "com.example.app"},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
		{
			name:    "empty bundle_id",
			req:     CreatePublisherAppRequest{Name: "My App", Platform: "android", BundleID: ""},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
		{
			name:    "duplicate bundle_id",
			req:     CreatePublisherAppRequest{Name: "My App", Platform: "ios", BundleID: "com.example.existing"},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return &model.PublisherApp{BundleID: "com.example.existing"}, nil
			},
			wantErr: true,
		},
		{
			name:    "name too long",
			req:     CreatePublisherAppRequest{Name: string(make([]byte, 101)), Platform: "web", BundleID: "com.example.app"},
			bundleCheck: func(_ context.Context, _ uuid.UUID, _ string) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherAppRepo{
				getByBundleFn: tt.bundleCheck,
				insertFn: func(_ context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
					return app, nil
				},
			}
			svc := NewPublisherAppService(repo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), orgID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("Create() returned nil")
				}
				if got.Name != tt.req.Name {
					t.Errorf("Create() name = %v, want %v", got.Name, tt.req.Name)
				}
				if got.Platform != tt.req.Platform {
					t.Errorf("Create() platform = %v, want %v", got.Platform, tt.req.Platform)
				}
				if !got.IsActive {
					t.Error("Create() app should be active by default")
				}
				if got.OrgID != orgID {
					t.Errorf("Create() org_id = %v, want %v", got.OrgID, orgID)
				}
			}
		})
	}
}

func TestPublisherAppService_GetByID(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name    string
		repoFn  func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
		wantErr bool
	}{
		{
			name: "found",
			repoFn: func(_ context.Context, _, id uuid.UUID) (*model.PublisherApp, error) {
				return &model.PublisherApp{ID: id, OrgID: orgID}, nil
			},
			wantErr: false,
		},
		{
			name: "not found",
			repoFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherAppRepo{getByIDFn: tt.repoFn}
			svc := NewPublisherAppService(repo, noop.NewTracerProvider())

			got, err := svc.GetByID(context.Background(), orgID, appID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Error("GetByID() returned nil")
			}
		})
	}
}

func TestPublisherAppService_List(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name       string
		limit      int32
		offset     int32
		repoReturn []model.PublisherApp
		total      int64
		wantLimit  int32
		wantErr    bool
	}{
		{
			name:       "default pagination",
			limit:      0,
			offset:     0,
			repoReturn: []model.PublisherApp{{ID: uuid.New()}},
			total:      1,
			wantLimit:  20,
			wantErr:    false,
		},
		{
			name:       "custom limit",
			limit:      50,
			offset:     10,
			repoReturn: []model.PublisherApp{},
			total:      0,
			wantLimit:  50,
			wantErr:    false,
		},
		{
			name:       "limit clamped to 100",
			limit:      200,
			offset:     0,
			repoReturn: []model.PublisherApp{},
			total:      0,
			wantLimit:  100,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedLimit := int32(0)
			repo := &mockPublisherAppRepo{
				listByOrgFn: func(_ context.Context, _ uuid.UUID, limit, _ int32) ([]model.PublisherApp, int64, error) {
					capturedLimit = limit
					return tt.repoReturn, tt.total, nil
				},
			}
			svc := NewPublisherAppService(repo, noop.NewTracerProvider())

			result, err := svc.List(context.Background(), orgID, tt.limit, tt.offset)
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

func TestPublisherAppService_Update(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	trueBool := true
	newName := "Updated Name"

	tests := []struct {
		name    string
		req     UpdatePublisherAppRequest
		repoFn  func(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
		wantErr bool
	}{
		{
			name: "update name",
			req:  UpdatePublisherAppRequest{Name: &newName},
			repoFn: func(_ context.Context, _, id uuid.UUID) (*model.PublisherApp, error) {
				return &model.PublisherApp{ID: id, OrgID: orgID, Name: "Old Name", IsActive: true}, nil
			},
			wantErr: false,
		},
		{
			name: "update is_active",
			req:  UpdatePublisherAppRequest{IsActive: &trueBool},
			repoFn: func(_ context.Context, _, id uuid.UUID) (*model.PublisherApp, error) {
				return &model.PublisherApp{ID: id, OrgID: orgID, Name: "App", IsActive: false}, nil
			},
			wantErr: false,
		},
		{
			name: "not found",
			req:  UpdatePublisherAppRequest{Name: &newName},
			repoFn: func(_ context.Context, _, _ uuid.UUID) (*model.PublisherApp, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
		{
			name: "empty name",
			req:  UpdatePublisherAppRequest{Name: func() *string { s := ""; return &s }()},
			repoFn: func(_ context.Context, _, id uuid.UUID) (*model.PublisherApp, error) {
				return &model.PublisherApp{ID: id, OrgID: orgID, Name: "App"}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherAppRepo{
				getByIDFn: tt.repoFn,
				updateFn: func(_ context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
					return app, nil
				},
			}
			svc := NewPublisherAppService(repo, noop.NewTracerProvider())

			got, err := svc.Update(context.Background(), orgID, appID, tt.req)
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
