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

// TestToPublisherApp validates db.PublisherApp → model.PublisherApp conversion.
func TestToPublisherApp(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	tests := []struct {
		name string
		row  db.PublisherApp
	}{
		{
			name: "active iOS app",
			row: db.PublisherApp{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				Name:      "My iOS App",
				Platform:  "ios",
				BundleID:  "com.example.myapp",
				IsActive:  true,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "inactive Android app",
			row: db.PublisherApp{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				Name:      "My Android App",
				Platform:  "android",
				BundleID:  "com.example.myapp.android",
				IsActive:  false,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "web platform",
			row: db.PublisherApp{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				Name:      "My Web App",
				Platform:  "web",
				BundleID:  "example.com",
				IsActive:  true,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPublisherApp(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch")
			}
			if got.Name != tt.row.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.row.Name)
			}
			if got.Platform != tt.row.Platform {
				t.Errorf("Platform = %q, want %q", got.Platform, tt.row.Platform)
			}
			if got.BundleID != tt.row.BundleID {
				t.Errorf("BundleID = %q, want %q", got.BundleID, tt.row.BundleID)
			}
			if got.IsActive != tt.row.IsActive {
				t.Errorf("IsActive = %v, want %v", got.IsActive, tt.row.IsActive)
			}
			if !got.CreatedAt.Equal(tt.row.CreatedAt.Time) {
				t.Errorf("CreatedAt mismatch")
			}
			if !got.UpdatedAt.Equal(tt.row.UpdatedAt.Time) {
				t.Errorf("UpdatedAt mismatch")
			}
		})
	}
}

// TestPublisherAppRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestPublisherAppRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestPublisherAppRepo_GetByBundleID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestPublisherAppRepo_GetByBundleID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	_, err := repo.GetByBundleID(context.Background(), uuid.New(), "com.example.app")
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestPublisherAppRepo_GetByBundleID_DBError verifies generic DB errors are wrapped.
func TestPublisherAppRepo_GetByBundleID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	_, err := repo.GetByBundleID(context.Background(), uuid.New(), "com.example.app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestPublisherAppRepo_Update_NotFound verifies pgx.ErrNoRows → model.ErrNotFound in Update.
func TestPublisherAppRepo_Update_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Update(context.Background(), &model.PublisherApp{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "Updated App",
		Platform:  "ios",
		BundleID:  "com.example.app",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestPublisherAppRepo_Insert_DBError verifies insert errors are returned.
func TestPublisherAppRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.PublisherApp{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "New App",
		Platform:  "android",
		BundleID:  "com.example.new",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestPublisherAppRepo_ListByOrg_DBError verifies list query errors are returned.
func TestPublisherAppRepo_ListByOrg_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &publisherAppRepo{q: db.New(mock)}
	_, _, err := repo.ListByOrg(context.Background(), uuid.New(), 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
