package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockOrgInviteRepo struct {
	insertFn     func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error)
	getByTokenFn func(ctx context.Context, token string) (*model.OrgInvite, error)
}

func (m *mockOrgInviteRepo) Insert(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
	return m.insertFn(ctx, invite)
}

func (m *mockOrgInviteRepo) GetByToken(ctx context.Context, token string) (*model.OrgInvite, error) {
	return m.getByTokenFn(ctx, token)
}

func TestOrgInviteService_Create(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name    string
		req     CreateInviteRequest
		repoFn  func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error)
		wantErr bool
		errIs   error
	}{
		{
			name: "valid invite editor",
			req:  CreateInviteRequest{Email: "alice@example.com", Role: "editor"},
			repoFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return invite, nil
			},
			wantErr: false,
		},
		{
			name: "valid invite admin",
			req:  CreateInviteRequest{Email: "bob@example.com", Role: "admin"},
			repoFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return invite, nil
			},
			wantErr: false,
		},
		{
			name:    "role owner rejected",
			req:     CreateInviteRequest{Email: "owner@example.com", Role: "owner"},
			repoFn:  nil,
			wantErr: true,
			errIs:   model.ErrInvalidInput,
		},
		{
			name:    "invalid role",
			req:     CreateInviteRequest{Email: "super@example.com", Role: "superadmin"},
			repoFn:  nil,
			wantErr: true,
			errIs:   model.ErrInvalidInput,
		},
		{
			name:    "empty email",
			req:     CreateInviteRequest{Email: "", Role: "editor"},
			repoFn:  nil,
			wantErr: true,
			errIs:   model.ErrInvalidInput,
		},
		{
			name: "db error",
			req:  CreateInviteRequest{Email: "fail@example.com", Role: "viewer"},
			repoFn: func(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
				return nil, errors.New("db unavailable")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgInviteRepo{insertFn: tt.repoFn}
			svc := NewOrgInviteService(repo, noop.NewTracerProvider())

			got, err := svc.Create(context.Background(), orgID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errIs != nil {
				if !errors.Is(err, tt.errIs) {
					t.Errorf("Create() error = %v, want errors.Is(%v)", err, tt.errIs)
				}
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("Create() returned nil invite")
					return
				}
				if got.Token == "" {
					t.Error("Create() returned empty token")
				}
				if got.OrgID != orgID {
					t.Errorf("Create() org_id = %v, want %v", got.OrgID, orgID)
				}
				if got.Email != tt.req.Email {
					t.Errorf("Create() email = %v, want %v", got.Email, tt.req.Email)
				}
				if got.Role != tt.req.Role {
					t.Errorf("Create() role = %v, want %v", got.Role, tt.req.Role)
				}
			}
		})
	}
}
