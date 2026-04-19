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

// TestToCreative validates db.Creative → model.Creative conversion.
func TestToCreative(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	campaignID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)
	size := int64(204800)
	previewURL := "https://cdn.example.com/preview.jpg"

	tests := []struct {
		name          string
		row           db.Creative
		wantFileSize  bool
		wantPreviewURL bool
	}{
		{
			name: "full creative with all optional fields",
			row: db.Creative{
				ID:            uuidToPgtype(id),
				OrgID:         uuidToPgtype(orgID),
				CampaignID:    uuidToPgtype(campaignID),
				Name:          "Banner Ad",
				Type:          "image",
				FileUrl:       "https://cdn.example.com/banner.png",
				FileSizeBytes: pgtype.Int8{Int64: size, Valid: true},
				PreviewUrl:    pgtype.Text{String: previewURL, Valid: true},
				IsActive:      true,
				CreatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantFileSize:  true,
			wantPreviewURL: true,
		},
		{
			name: "creative with no optional fields",
			row: db.Creative{
				ID:            uuidToPgtype(id),
				OrgID:         uuidToPgtype(orgID),
				CampaignID:    uuidToPgtype(campaignID),
				Name:          "Video Ad",
				Type:          "video",
				FileUrl:       "https://cdn.example.com/ad.mp4",
				FileSizeBytes: pgtype.Int8{Valid: false},
				PreviewUrl:    pgtype.Text{Valid: false},
				IsActive:      false,
				CreatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantFileSize:  false,
			wantPreviewURL: false,
		},
		{
			name: "inactive creative",
			row: db.Creative{
				ID:            uuidToPgtype(id),
				OrgID:         uuidToPgtype(orgID),
				CampaignID:    uuidToPgtype(campaignID),
				Name:          "Inactive",
				Type:          "image",
				FileUrl:       "https://cdn.example.com/inactive.png",
				FileSizeBytes: pgtype.Int8{Valid: false},
				PreviewUrl:    pgtype.Text{Valid: false},
				IsActive:      false,
				CreatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantFileSize:  false,
			wantPreviewURL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCreative(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch")
			}
			if got.CampaignID != pgtypeToUUID(tt.row.CampaignID) {
				t.Errorf("CampaignID mismatch")
			}
			if got.Name != tt.row.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.row.Name)
			}
			if string(got.Type) != tt.row.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.row.Type)
			}
			if got.FileURL != tt.row.FileUrl {
				t.Errorf("FileURL = %q, want %q", got.FileURL, tt.row.FileUrl)
			}
			if got.IsActive != tt.row.IsActive {
				t.Errorf("IsActive = %v, want %v", got.IsActive, tt.row.IsActive)
			}

			if tt.wantFileSize {
				if got.FileSizeBytes == nil {
					t.Error("expected non-nil FileSizeBytes")
				} else if *got.FileSizeBytes != size {
					t.Errorf("FileSizeBytes = %d, want %d", *got.FileSizeBytes, size)
				}
			} else if got.FileSizeBytes != nil {
				t.Errorf("expected nil FileSizeBytes, got %d", *got.FileSizeBytes)
			}

			if tt.wantPreviewURL {
				if got.PreviewURL == nil {
					t.Error("expected non-nil PreviewURL")
				} else if *got.PreviewURL != previewURL {
					t.Errorf("PreviewURL = %q, want %q", *got.PreviewURL, previewURL)
				}
			} else if got.PreviewURL != nil {
				t.Errorf("expected nil PreviewURL, got %q", *got.PreviewURL)
			}
		})
	}
}

// TestCreativeRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestCreativeRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestCreativeRepo_GetByID_DBError verifies generic errors are wrapped.
func TestCreativeRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestCreativeRepo_Update_NotFound verifies pgx.ErrNoRows → model.ErrNotFound in Update.
func TestCreativeRepo_Update_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	_, err := repo.Update(context.Background(), uuid.New(), uuid.New(), uuid.New(), UpdateCreativeParams{
		Name:     "Updated",
		Type:     "image",
		FileURL:  "https://cdn.example.com/new.png",
		IsActive: true,
	})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestCreativeRepo_Insert_DBError verifies insert errors are returned.
func TestCreativeRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.Creative{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		CampaignID: uuid.New(),
		Name:       "test creative",
		Type:       "image",
		FileURL:    "https://cdn.example.com/test.png",
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestCreativeRepo_ListByCampaign_DBError verifies list query errors are returned.
func TestCreativeRepo_ListByCampaign_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &creativeRepo{q: db.New(mock)}
	_, _, err := repo.ListByCampaign(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
