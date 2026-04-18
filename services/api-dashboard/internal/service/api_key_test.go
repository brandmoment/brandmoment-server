package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type mockAPIKeyRepo struct {
	insertFn   func(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
	getByIDFn  func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error)
	listFn     func(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error)
	revokeFn   func(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error)
}

func (m *mockAPIKeyRepo) Insert(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	return m.insertFn(ctx, key)
}

func (m *mockAPIKeyRepo) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error) {
	return m.getByIDFn(ctx, orgID, appID, id)
}

func (m *mockAPIKeyRepo) ListByApp(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
	return m.listFn(ctx, orgID, appID, activeOnly)
}

func (m *mockAPIKeyRepo) Revoke(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error) {
	return m.revokeFn(ctx, orgID, appID, id, revokedAt)
}

func TestAPIKeyService_Provision(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	tests := []struct {
		name    string
		req     ProvisionAPIKeyRequest
		wantErr bool
	}{
		{
			name:    "valid provision",
			req:     ProvisionAPIKeyRequest{Name: "Production"},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     ProvisionAPIKeyRequest{Name: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAPIKeyRepo{
				insertFn: func(_ context.Context, key *model.APIKey) (*model.APIKey, error) {
					return key, nil
				},
			}
			svc := NewAPIKeyService(repo, noop.NewTracerProvider())

			result, err := svc.Provision(context.Background(), orgID, appID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provision() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Verify plaintext has bm_ prefix.
			if !strings.HasPrefix(result.Plaintext, "bm_") {
				t.Errorf("Provision() plaintext does not start with bm_: %v", result.Plaintext)
			}

			// Verify key_prefix = first 8 chars of plaintext.
			if result.Key.KeyPrefix != result.Plaintext[:8] {
				t.Errorf("Provision() key_prefix = %v, want first 8 chars of plaintext %v", result.Key.KeyPrefix, result.Plaintext[:8])
			}

			// Verify key_hash = sha256 of plaintext (never equal to plaintext).
			expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(result.Plaintext)))
			if result.Key.KeyHash != expectedHash {
				t.Errorf("Provision() key_hash = %v, want %v", result.Key.KeyHash, expectedHash)
			}
			if result.Key.KeyHash == result.Plaintext {
				t.Error("Provision() key_hash must not equal plaintext")
			}

			// Verify key is not revoked.
			if result.Key.IsRevoked {
				t.Error("Provision() new key should not be revoked")
			}

			// Verify org_id and app_id.
			if result.Key.OrgID != orgID {
				t.Errorf("Provision() org_id = %v, want %v", result.Key.OrgID, orgID)
			}
			if result.Key.AppID != appID {
				t.Errorf("Provision() app_id = %v, want %v", result.Key.AppID, appID)
			}
		})
	}
}

func TestAPIKeyService_ListByApp(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()

	activeKey := model.APIKey{ID: uuid.New(), IsRevoked: false}
	revokedKey := model.APIKey{ID: uuid.New(), IsRevoked: true}

	tests := []struct {
		name           string
		includeRevoked bool
		repoReturn     []model.APIKey
		wantCount      int
	}{
		{
			name:           "active only (default)",
			includeRevoked: false,
			repoReturn:     []model.APIKey{activeKey},
			wantCount:      1,
		},
		{
			name:           "include revoked",
			includeRevoked: true,
			repoReturn:     []model.APIKey{activeKey, revokedKey},
			wantCount:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedActiveOnly := false
			repo := &mockAPIKeyRepo{
				listFn: func(_ context.Context, _, _ uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
					capturedActiveOnly = activeOnly
					return tt.repoReturn, nil
				},
			}
			svc := NewAPIKeyService(repo, noop.NewTracerProvider())

			result, err := svc.ListByApp(context.Background(), orgID, appID, tt.includeRevoked)
			if err != nil {
				t.Errorf("ListByApp() error = %v", err)
				return
			}
			if len(result.Items) != tt.wantCount {
				t.Errorf("ListByApp() items count = %v, want %v", len(result.Items), tt.wantCount)
			}

			// Verify activeOnly flag is passed correctly (inverted from includeRevoked).
			wantActiveOnly := !tt.includeRevoked
			if capturedActiveOnly != wantActiveOnly {
				t.Errorf("ListByApp() passed activeOnly = %v, want %v", capturedActiveOnly, wantActiveOnly)
			}
		})
	}
}

func TestAPIKeyService_Revoke(t *testing.T) {
	orgID := uuid.New()
	appID := uuid.New()
	keyID := uuid.New()

	tests := []struct {
		name       string
		getByIDFn  func(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error)
		wantErr    bool
	}{
		{
			name: "revoke active key",
			getByIDFn: func(_ context.Context, _, _, id uuid.UUID) (*model.APIKey, error) {
				return &model.APIKey{ID: id, IsRevoked: false}, nil
			},
			wantErr: false,
		},
		{
			name: "already revoked",
			getByIDFn: func(_ context.Context, _, _, id uuid.UUID) (*model.APIKey, error) {
				return &model.APIKey{ID: id, IsRevoked: true}, nil
			},
			wantErr: true,
		},
		{
			name: "key not found",
			getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*model.APIKey, error) {
				return nil, model.ErrNotFound
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAPIKeyRepo{
				getByIDFn: tt.getByIDFn,
				revokeFn: func(_ context.Context, _, _, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error) {
					now := time.Now()
					return &model.APIKey{ID: id, IsRevoked: true, RevokedAt: &now}, nil
				},
			}
			svc := NewAPIKeyService(repo, noop.NewTracerProvider())

			got, err := svc.Revoke(context.Background(), orgID, appID, keyID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Revoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !got.IsRevoked {
					t.Error("Revoke() key should be revoked")
				}
				if got.RevokedAt == nil {
					t.Error("Revoke() revoked_at should not be nil")
				}
			}
		})
	}
}
