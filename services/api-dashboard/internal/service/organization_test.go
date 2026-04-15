package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockOrgRepo struct {
	insertFn  func(ctx context.Context, org *model.Organization) (*model.Organization, error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	listFn    func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
}

func (m *mockOrgRepo) Insert(ctx context.Context, org *model.Organization) (*model.Organization, error) {
	return m.insertFn(ctx, org)
}

func (m *mockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockOrgRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
	return m.listFn(ctx, ids)
}

func TestOrganizationService_Create(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateOrganizationRequest
		wantErr bool
	}{
		{
			name:    "valid organization",
			req:     CreateOrganizationRequest{Type: "publisher", Name: "Acme Corp", Slug: "acme-corp"},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     CreateOrganizationRequest{Type: "publisher", Name: "", Slug: "acme-corp"},
			wantErr: true,
		},
		{
			name:    "empty slug",
			req:     CreateOrganizationRequest{Type: "publisher", Name: "Acme Corp", Slug: ""},
			wantErr: true,
		},
		{
			name:    "empty type",
			req:     CreateOrganizationRequest{Type: "", Name: "Acme Corp", Slug: "acme-corp"},
			wantErr: true,
		},
		{
			name:    "invalid type",
			req:     CreateOrganizationRequest{Type: "invalid", Name: "Acme Corp", Slug: "acme-corp"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepo{
				insertFn: func(ctx context.Context, org *model.Organization) (*model.Organization, error) {
					return org, nil
				},
			}
			svc := NewOrganizationService(repo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("Create() returned nil organization")
			}
			if !tt.wantErr {
				if got.Name != tt.req.Name {
					t.Errorf("Create() name = %v, want %v", got.Name, tt.req.Name)
				}
				if got.Slug != tt.req.Slug {
					t.Errorf("Create() slug = %v, want %v", got.Slug, tt.req.Slug)
				}
				if got.Type != tt.req.Type {
					t.Errorf("Create() type = %v, want %v", got.Type, tt.req.Type)
				}
			}
		})
	}
}

func TestOrganizationService_GetByID(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		repoFn  func(ctx context.Context, id uuid.UUID) (*model.Organization, error)
		wantErr bool
	}{
		{
			name: "found",
			id:   orgID,
			repoFn: func(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
				return &model.Organization{ID: id, Name: "Test", Slug: "test", Type: "publisher"}, nil
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			repoFn: func(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepo{getByIDFn: tt.repoFn}
			svc := NewOrganizationService(repo, noop.NewTracerProvider())

			got, err := svc.GetByID(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("GetByID() returned nil organization")
			}
		})
	}
}

func TestOrganizationService_ListByIDs(t *testing.T) {
	tests := []struct {
		name    string
		ids     []uuid.UUID
		repoFn  func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
		want    int
		wantErr bool
	}{
		{
			name: "returns organizations",
			ids:  []uuid.UUID{uuid.New(), uuid.New()},
			repoFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				orgs := make([]model.Organization, len(ids))
				for i, id := range ids {
					orgs[i] = model.Organization{ID: id, Name: "Org", Type: "publisher"}
				}
				return orgs, nil
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "empty list",
			ids:  []uuid.UUID{},
			repoFn: func(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
				return []model.Organization{}, nil
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepo{listFn: tt.repoFn}
			svc := NewOrganizationService(repo, noop.NewTracerProvider())

			got, err := svc.ListByIDs(context.Background(), tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListByIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("ListByIDs() returned %d orgs, want %d", len(got), tt.want)
			}
		})
	}
}
