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

// TestToUser validates db.User → model.User conversion.
func TestToUser(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	tests := []struct {
		name string
		row  db.User
	}{
		{
			name: "standard user",
			row: db.User{
				ID:        uuidToPgtype(id),
				Email:     "alice@example.com",
				Name:      "Alice Smith",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "user with empty name",
			row: db.User{
				ID:        uuidToPgtype(id),
				Email:     "noname@example.com",
				Name:      "",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "user with zero timestamp",
			row: db.User{
				ID:        uuidToPgtype(id),
				Email:     "zero@example.com",
				Name:      "Zero Time",
				CreatedAt: pgtype.Timestamptz{Valid: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toUser(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.Email != tt.row.Email {
				t.Errorf("Email = %q, want %q", got.Email, tt.row.Email)
			}
			if got.Name != tt.row.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.row.Name)
			}
			if !got.CreatedAt.Equal(tt.row.CreatedAt.Time) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.row.CreatedAt.Time)
			}
		})
	}
}

// TestUserRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestUserRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &userRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestUserRepo_GetByID_DBError verifies generic errors are wrapped.
func TestUserRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &userRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestUserRepo_Upsert_DBError verifies upsert errors are returned.
func TestUserRepo_Upsert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("upsert failed")}
		},
	}
	repo := &userRepo{q: db.New(mock)}
	_, err := repo.Upsert(context.Background(), uuid.New(), "test@example.com", "Test User", time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestUserRepo_Upsert_GetByID_NotFound_Independence verifies GetByID ErrNoRows and Upsert are distinct paths.
func TestUserRepo_Upsert_GetByID_Independence(t *testing.T) {
	// ErrNoRows from Upsert should NOT be treated as model.ErrNotFound (it wraps differently).
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &userRepo{q: db.New(mock)}

	// GetByID should map ErrNoRows to ErrNotFound.
	_, getErr := repo.GetByID(context.Background(), uuid.New())
	if getErr != model.ErrNotFound {
		t.Errorf("GetByID: got %v, want model.ErrNotFound", getErr)
	}

	// Upsert does NOT map ErrNoRows — it wraps it generically.
	// (If upsert returns ErrNoRows, it's a real DB error; the user record should always exist after upsert.)
	_, upsertErr := repo.Upsert(context.Background(), uuid.New(), "x@x.com", "X", time.Now())
	if upsertErr == nil {
		t.Fatal("expected error from Upsert with ErrNoRows scan, got nil")
	}
}
