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

// TestToOrgInvite validates db.OrgInvite → model.OrgInvite conversion.
func TestToOrgInvite(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)
	expires := now.Add(48 * time.Hour)
	accepted := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		row          db.OrgInvite
		wantAccepted bool
	}{
		{
			name: "pending invite with no accepted_at",
			row: db.OrgInvite{
				ID:         uuidToPgtype(id),
				OrgID:      uuidToPgtype(orgID),
				Email:      "user@example.com",
				Role:       "editor",
				Token:      "tok_abc123",
				ExpiresAt:  pgtype.Timestamptz{Time: expires, Valid: true},
				AcceptedAt: pgtype.Timestamptz{Valid: false},
				CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantAccepted: false,
		},
		{
			name: "accepted invite with accepted_at set",
			row: db.OrgInvite{
				ID:         uuidToPgtype(id),
				OrgID:      uuidToPgtype(orgID),
				Email:      "member@example.com",
				Role:       "viewer",
				Token:      "tok_def456",
				ExpiresAt:  pgtype.Timestamptz{Time: expires, Valid: true},
				AcceptedAt: pgtype.Timestamptz{Time: accepted, Valid: true},
				CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantAccepted: true,
		},
		{
			name: "owner role invite",
			row: db.OrgInvite{
				ID:         uuidToPgtype(id),
				OrgID:      uuidToPgtype(orgID),
				Email:      "owner@example.com",
				Role:       "owner",
				Token:      "tok_ghi789",
				ExpiresAt:  pgtype.Timestamptz{Time: expires, Valid: true},
				AcceptedAt: pgtype.Timestamptz{Valid: false},
				CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantAccepted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toOrgInvite(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch")
			}
			if got.Email != tt.row.Email {
				t.Errorf("Email = %q, want %q", got.Email, tt.row.Email)
			}
			if got.Role != tt.row.Role {
				t.Errorf("Role = %q, want %q", got.Role, tt.row.Role)
			}
			if got.Token != tt.row.Token {
				t.Errorf("Token = %q, want %q", got.Token, tt.row.Token)
			}
			if !got.ExpiresAt.Equal(tt.row.ExpiresAt.Time) {
				t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, tt.row.ExpiresAt.Time)
			}
			if !got.CreatedAt.Equal(tt.row.CreatedAt.Time) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.row.CreatedAt.Time)
			}

			if tt.wantAccepted {
				if got.AcceptedAt == nil {
					t.Error("expected non-nil AcceptedAt")
				} else if !got.AcceptedAt.Equal(accepted) {
					t.Errorf("AcceptedAt = %v, want %v", got.AcceptedAt, accepted)
				}
			} else if got.AcceptedAt != nil {
				t.Errorf("expected nil AcceptedAt, got %v", got.AcceptedAt)
			}
		})
	}
}

// TestOrgInviteRepo_GetByToken_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestOrgInviteRepo_GetByToken_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &orgInviteRepo{q: db.New(mock)}
	_, err := repo.GetByToken(context.Background(), "nonexistent-token")
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestOrgInviteRepo_GetByToken_DBError verifies generic errors are wrapped.
func TestOrgInviteRepo_GetByToken_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &orgInviteRepo{q: db.New(mock)}
	_, err := repo.GetByToken(context.Background(), "some-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestOrgInviteRepo_Insert_DBError verifies insert errors are returned.
func TestOrgInviteRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &orgInviteRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.OrgInvite{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "test@example.com",
		Role:      "viewer",
		Token:     "tok_test",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
