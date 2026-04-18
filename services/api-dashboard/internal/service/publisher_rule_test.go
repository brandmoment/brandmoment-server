package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockPublisherRuleRepo struct {
	insertFn   func(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	getByIDFn  func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error)
	listFn     func(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error)
	updateFn   func(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	deleteFn   func(ctx context.Context, orgID, appID, id uuid.UUID) error
}

func (m *mockPublisherRuleRepo) Insert(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	return m.insertFn(ctx, rule)
}

func (m *mockPublisherRuleRepo) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error) {
	return m.getByIDFn(ctx, orgID, appID, id)
}

func (m *mockPublisherRuleRepo) ListByApp(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error) {
	return m.listFn(ctx, orgID, appID, limit, offset)
}

func (m *mockPublisherRuleRepo) Update(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	return m.updateFn(ctx, rule)
}

func (m *mockPublisherRuleRepo) Delete(ctx context.Context, orgID, appID, id uuid.UUID) error {
	return m.deleteFn(ctx, orgID, appID, id)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestPublisherRuleService_Create(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name    string
		req     CreatePublisherRuleRequest
		wantErr bool
	}{
		{
			name: "valid blocklist",
			req:  CreatePublisherRuleRequest{Type: "blocklist", Config: mustJSON(model.BlocklistConfig{Domains: []string{"bad.com"}})},
			wantErr: false,
		},
		{
			name: "valid allowlist",
			req:  CreatePublisherRuleRequest{Type: "allowlist", Config: mustJSON(model.AllowlistConfig{BundleIDs: []string{"com.ok.app"}})},
			wantErr: false,
		},
		{
			name: "valid frequency_cap",
			req:  CreatePublisherRuleRequest{Type: "frequency_cap", Config: mustJSON(model.FrequencyCapConfig{MaxImpressions: 5, WindowSeconds: 3600})},
			wantErr: false,
		},
		{
			name: "valid geo_filter",
			req:  CreatePublisherRuleRequest{Type: "geo_filter", Config: mustJSON(model.GeoFilterConfig{Mode: "exclude", CountryCodes: []string{"CN"}})},
			wantErr: false,
		},
		{
			name: "valid platform_filter",
			req:  CreatePublisherRuleRequest{Type: "platform_filter", Config: mustJSON(model.PlatformFilterConfig{Mode: "include", Platforms: []string{"ios"}})},
			wantErr: false,
		},
		{
			name:    "unknown rule type",
			req:     CreatePublisherRuleRequest{Type: "unknown", Config: mustJSON(map[string]string{})},
			wantErr: true,
		},
		{
			name:    "frequency_cap missing max_impressions",
			req:     CreatePublisherRuleRequest{Type: "frequency_cap", Config: mustJSON(map[string]int{"window_seconds": 3600})},
			wantErr: true,
		},
		{
			name:    "geo_filter invalid mode",
			req:     CreatePublisherRuleRequest{Type: "geo_filter", Config: mustJSON(model.GeoFilterConfig{Mode: "maybe", CountryCodes: []string{"US"}})},
			wantErr: true,
		},
		{
			name:    "geo_filter empty country_codes",
			req:     CreatePublisherRuleRequest{Type: "geo_filter", Config: mustJSON(model.GeoFilterConfig{Mode: "include", CountryCodes: []string{}})},
			wantErr: true,
		},
		{
			name:    "platform_filter invalid mode",
			req:     CreatePublisherRuleRequest{Type: "platform_filter", Config: mustJSON(model.PlatformFilterConfig{Mode: "all", Platforms: []string{"ios"}})},
			wantErr: true,
		},
		{
			name:    "platform_filter empty platforms",
			req:     CreatePublisherRuleRequest{Type: "platform_filter", Config: mustJSON(model.PlatformFilterConfig{Mode: "include", Platforms: []string{}})},
			wantErr: true,
		},
		{
			name:    "empty config",
			req:     CreatePublisherRuleRequest{Type: "blocklist", Config: json.RawMessage(nil)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherRuleRepo{
				insertFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
					return rule, nil
				},
			}
			svc := NewPublisherRuleService(repo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), orgID, appID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("Create() returned nil")
				}
				if got.Type != tt.req.Type {
					t.Errorf("Create() type = %v, want %v", got.Type, tt.req.Type)
				}
				if !got.IsActive {
					t.Error("Create() rule should be active by default")
				}
			}
		})
	}
}

func TestPublisherRuleService_List(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name      string
		limit     int32
		offset    int32
		wantLimit int32
		wantErr   bool
	}{
		{
			name:      "default limit",
			limit:     0,
			offset:    0,
			wantLimit: 20,
			wantErr:   false,
		},
		{
			name:      "limit clamped to 100",
			limit:     500,
			offset:    0,
			wantLimit: 100,
			wantErr:   false,
		},
		{
			name:      "custom limit",
			limit:     30,
			offset:    5,
			wantLimit: 30,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedLimit := int32(0)
			repo := &mockPublisherRuleRepo{
				listFn: func(_ context.Context, _, _ uuid.UUID, limit, _ int32) ([]model.PublisherRule, int64, error) {
					capturedLimit = limit
					return []model.PublisherRule{}, 0, nil
				},
			}
			svc := NewPublisherRuleService(repo, noop.NewTracerProvider())

			result, err := svc.List(context.Background(), orgID, appID, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Limit != tt.wantLimit {
					t.Errorf("List() result.Limit = %v, want %v", result.Limit, tt.wantLimit)
				}
				if capturedLimit != tt.wantLimit {
					t.Errorf("List() repo limit = %v, want %v", capturedLimit, tt.wantLimit)
				}
			}
		})
	}
}

func TestPublisherRuleService_Update(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	ruleID := uuid.New()

	trueBool := true

	tests := []struct {
		name      string
		req       UpdatePublisherRuleRequest
		existRule *model.PublisherRule
		repoErr   error
		wantErr   bool
	}{
		{
			name: "update config",
			req:  UpdatePublisherRuleRequest{Config: mustJSON(model.GeoFilterConfig{Mode: "include", CountryCodes: []string{"US"}})},
			existRule: &model.PublisherRule{
				ID:    ruleID,
				Type:  "geo_filter",
				Config: mustJSON(model.GeoFilterConfig{Mode: "exclude", CountryCodes: []string{"CN"}}),
				IsActive: true,
			},
			wantErr: false,
		},
		{
			name: "update is_active",
			req:  UpdatePublisherRuleRequest{IsActive: &trueBool},
			existRule: &model.PublisherRule{
				ID:    ruleID,
				Type:  "blocklist",
				Config: mustJSON(model.BlocklistConfig{Domains: []string{"bad.com"}}),
				IsActive: false,
			},
			wantErr: false,
		},
		{
			name:    "not found",
			req:     UpdatePublisherRuleRequest{IsActive: &trueBool},
			repoErr: model.ErrNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherRuleRepo{
				getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return tt.existRule, nil
				},
				updateFn: func(_ context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
					return rule, nil
				},
			}
			svc := NewPublisherRuleService(repo, noop.NewTracerProvider())

			got, err := svc.Update(context.Background(), orgID, appID, ruleID, tt.req)
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

func TestPublisherRuleService_Delete(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	ruleID := uuid.New()

	tests := []struct {
		name      string
		getByIDFn func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error)
		wantErr   bool
	}{
		{
			name: "delete existing rule",
			getByIDFn: func(_ context.Context, _, _, id uuid.UUID) (*model.PublisherRule, error) {
				return &model.PublisherRule{ID: id}, nil
			},
			wantErr: false,
		},
		{
			name: "rule not found",
			getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.PublisherRule, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockPublisherRuleRepo{
				getByIDFn: tt.getByIDFn,
				deleteFn: func(_ context.Context, _, _, _ uuid.UUID) error {
					return nil
				},
			}
			svc := NewPublisherRuleService(repo, noop.NewTracerProvider())

			err := svc.Delete(context.Background(), orgID, appID, ruleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
