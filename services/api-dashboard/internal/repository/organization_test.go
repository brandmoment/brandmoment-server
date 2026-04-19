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

// TestToOrganization validates db.Organization → model.Organization conversion.
func TestToOrganization(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	tests := []struct {
		name    string
		row     db.Organization
		wantOrg model.Organization
	}{
		{
			name: "publisher org",
			row: db.Organization{
				ID:        uuidToPgtype(id),
				Type:      "publisher",
				Name:      "Acme Media",
				Slug:      "acme-media",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantOrg: model.Organization{
				ID:        id,
				Type:      "publisher",
				Name:      "Acme Media",
				Slug:      "acme-media",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "admin org",
			row: db.Organization{
				ID:        uuidToPgtype(id),
				Type:      "admin",
				Name:      "BrandMoment HQ",
				Slug:      "brandmoment-hq",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantOrg: model.Organization{
				ID:        id,
				Type:      "admin",
				Name:      "BrandMoment HQ",
				Slug:      "brandmoment-hq",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "brand org",
			row: db.Organization{
				ID:        uuidToPgtype(id),
				Type:      "brand",
				Name:      "Widgets Co",
				Slug:      "widgets-co",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantOrg: model.Organization{
				ID:        id,
				Type:      "brand",
				Name:      "Widgets Co",
				Slug:      "widgets-co",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toOrganization(tt.row)

			if got.ID != tt.wantOrg.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.wantOrg.ID)
			}
			if got.Type != tt.wantOrg.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantOrg.Type)
			}
			if got.Name != tt.wantOrg.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantOrg.Name)
			}
			if got.Slug != tt.wantOrg.Slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.wantOrg.Slug)
			}
			if !got.CreatedAt.Equal(tt.wantOrg.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.wantOrg.CreatedAt)
			}
			if !got.UpdatedAt.Equal(tt.wantOrg.UpdatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, tt.wantOrg.UpdatedAt)
			}
		})
	}
}

// TestOrganizationRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestOrganizationRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestOrganizationRepo_GetByID_DBError verifies generic errors are wrapped.
func TestOrganizationRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestOrganizationRepo_Insert_DBError verifies insert errors are returned.
func TestOrganizationRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.Organization{
		ID:        uuid.New(),
		Type:      "publisher",
		Name:      "Test Org",
		Slug:      "test-org",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestOrganizationRepo_ListByIDs_DBError verifies list query errors are returned.
func TestOrganizationRepo_ListByIDs_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	_, err := repo.ListByIDs(context.Background(), []uuid.UUID{uuid.New(), uuid.New()})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestOrganizationRepo_ListByIDs_Empty verifies empty IDs slice returns empty slice without error.
func TestOrganizationRepo_ListByIDs_Empty(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return &emptyRows{}, nil
		},
	}
	repo := &organizationRepo{q: db.New(mock)}
	orgs, err := repo.ListByIDs(context.Background(), []uuid.UUID{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orgs) != 0 {
		t.Errorf("expected empty slice, got %d orgs", len(orgs))
	}
}
