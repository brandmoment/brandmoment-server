package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// TestToPublisherRule validates db.PublisherRule → model.PublisherRule conversion.
func TestToPublisherRule(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	appID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	blocklistConfig, _ := json.Marshal(model.BlocklistConfig{
		Domains:   []string{"spam.com"},
		BundleIDs: []string{"com.spam.app"},
	})
	freqCapConfig, _ := json.Marshal(model.FrequencyCapConfig{
		MaxImpressions: 5,
		WindowSeconds:  3600,
	})

	tests := []struct {
		name string
		row  db.PublisherRule
	}{
		{
			name: "active blocklist rule",
			row: db.PublisherRule{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Type:      model.RuleTypeBlocklist,
				Config:    blocklistConfig,
				IsActive:  true,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "inactive frequency cap rule",
			row: db.PublisherRule{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Type:      model.RuleTypeFrequencyCap,
				Config:    freqCapConfig,
				IsActive:  false,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "rule with empty config",
			row: db.PublisherRule{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				AppID:     uuidToPgtype(appID),
				Type:      model.RuleTypeGeoFilter,
				Config:    []byte(`{}`),
				IsActive:  true,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPublisherRule(tt.row)

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch")
			}
			if got.AppID != pgtypeToUUID(tt.row.AppID) {
				t.Errorf("AppID mismatch")
			}
			if got.Type != tt.row.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.row.Type)
			}
			if string(got.Config) != string(tt.row.Config) {
				t.Errorf("Config = %s, want %s", got.Config, tt.row.Config)
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

// TestPublisherRuleRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestPublisherRuleRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestPublisherRuleRepo_GetByID_DBError verifies generic errors are wrapped.
func TestPublisherRuleRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestPublisherRuleRepo_Update_NotFound verifies pgx.ErrNoRows → model.ErrNotFound in Update.
func TestPublisherRuleRepo_Update_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	now := time.Now()
	config, _ := json.Marshal(model.BlocklistConfig{Domains: []string{"bad.com"}})
	_, err := repo.Update(context.Background(), &model.PublisherRule{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		AppID:     uuid.New(),
		Type:      model.RuleTypeBlocklist,
		Config:    config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestPublisherRuleRepo_Delete_DBError verifies delete errors are returned.
func TestPublisherRuleRepo_Delete_DBError(t *testing.T) {
	mock := &mockDBTX{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errSentinel("delete failed")
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	err := repo.Delete(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestPublisherRuleRepo_Insert_DBError verifies insert errors are returned.
func TestPublisherRuleRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	now := time.Now()
	config, _ := json.Marshal(model.BlocklistConfig{Domains: []string{"bad.com"}})
	_, err := repo.Insert(context.Background(), &model.PublisherRule{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		AppID:     uuid.New(),
		Type:      model.RuleTypeBlocklist,
		Config:    config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestPublisherRuleRepo_ListByApp_DBError verifies list query errors are returned.
func TestPublisherRuleRepo_ListByApp_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &publisherRuleRepo{q: db.New(mock)}
	_, _, err := repo.ListByApp(context.Background(), uuid.New(), uuid.New(), 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
