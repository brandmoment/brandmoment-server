package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// TestToAPIKey validates the converter from db.ApiKey to model.APIKey.
func TestToAPIKey(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	appID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)
	revokedAt := now.Add(time.Hour)

	tests := []struct {
		name      string
		row       db.ApiKey
		wantRevAt bool
	}{
		{
			name: "active key with no revoked_at",
			row: db.ApiKey{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Name:      "test-key",
				KeyHash:   "hash123",
				KeyPrefix: "bm_abc",
				IsRevoked: false,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				RevokedAt: pgtype.Timestamptz{Valid: false},
			},
			wantRevAt: false,
		},
		{
			name: "revoked key with revoked_at set",
			row: db.ApiKey{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Name:      "old-key",
				KeyHash:   "hash456",
				KeyPrefix: "bm_xyz",
				IsRevoked: true,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
			},
			wantRevAt: true,
		},
		{
			name: "zero timestamps",
			row: db.ApiKey{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Name:      "zero-key",
				KeyHash:   "",
				KeyPrefix: "",
				IsRevoked: false,
				CreatedAt: pgtype.Timestamptz{Valid: false},
				RevokedAt: pgtype.Timestamptz{Valid: false},
			},
			wantRevAt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toAPIKey(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch: got %v", got.ID)
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch: got %v", got.OrgID)
			}
			if got.AppID != pgtypeToUUID(tt.row.AppID) {
				t.Errorf("AppID mismatch: got %v", got.AppID)
			}
			if got.Name != tt.row.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.row.Name)
			}
			if got.KeyHash != tt.row.KeyHash {
				t.Errorf("KeyHash = %q, want %q", got.KeyHash, tt.row.KeyHash)
			}
			if got.KeyPrefix != tt.row.KeyPrefix {
				t.Errorf("KeyPrefix = %q, want %q", got.KeyPrefix, tt.row.KeyPrefix)
			}
			if got.IsRevoked != tt.row.IsRevoked {
				t.Errorf("IsRevoked = %v, want %v", got.IsRevoked, tt.row.IsRevoked)
			}
			if tt.wantRevAt && got.RevokedAt == nil {
				t.Error("expected non-nil RevokedAt")
			}
			if !tt.wantRevAt && got.RevokedAt != nil {
				t.Errorf("expected nil RevokedAt, got %v", got.RevokedAt)
			}
			if tt.wantRevAt && got.RevokedAt != nil && !got.RevokedAt.Equal(revokedAt) {
				t.Errorf("RevokedAt = %v, want %v", got.RevokedAt, revokedAt)
			}
		})
	}
}

// TestAPIKeyRepo_GetByID_NotFound verifies pgx.ErrNoRows is mapped to model.ErrNotFound.
func TestAPIKeyRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("GetByID with ErrNoRows: got %v, want model.ErrNotFound", err)
	}
}

// TestAPIKeyRepo_GetByID_DBError verifies a generic DB error is wrapped and propagated.
func TestAPIKeyRepo_GetByID_DBError(t *testing.T) {
	dbErr := errSentinel("db error")
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: dbErr}
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("generic DB error should not be mapped to ErrNotFound")
	}
}

// TestAPIKeyRepo_Revoke_NotFound verifies pgx.ErrNoRows is mapped to model.ErrNotFound in Revoke.
func TestAPIKeyRepo_Revoke_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.Revoke(context.Background(), uuid.New(), uuid.New(), uuid.New(), time.Now())
	if err != model.ErrNotFound {
		t.Errorf("Revoke with ErrNoRows: got %v, want model.ErrNotFound", err)
	}
}

// TestAPIKeyRepo_Insert_DBError verifies insert error is wrapped and returned.
func TestAPIKeyRepo_Insert_DBError(t *testing.T) {
	dbErr := errSentinel("insert failed")
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: dbErr}
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.Insert(context.Background(), &model.APIKey{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		AppID:     uuid.New(),
		Name:      "key",
		KeyHash:   "hash",
		KeyPrefix: "bm_",
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestAPIKeyRepo_ListByApp_DBError verifies list error is wrapped and returned.
func TestAPIKeyRepo_ListByApp_DBError(t *testing.T) {
	dbErr := errSentinel("list failed")
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, dbErr
		},
	}
	repo := &apiKeyRepo{q: db.New(mock)}
	_, err := repo.ListByApp(context.Background(), uuid.New(), uuid.New(), true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// errSentinel is a simple error type for tests.
type errSentinel string

func (e errSentinel) Error() string { return string(e) }
