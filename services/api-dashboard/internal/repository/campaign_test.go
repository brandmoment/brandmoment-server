package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// TestToCampaign validates db.Campaign → model.Campaign conversion.
func TestToCampaign(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)
	budget := int64(50000)
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	targeting := model.CampaignTargeting{
		Geo:       []string{"US", "CA"},
		Platforms: []string{"ios"},
	}
	targetingJSON, _ := json.Marshal(targeting)

	tests := []struct {
		name       string
		row        db.Campaign
		wantBudget bool
		wantDates  bool
		wantErr    bool
	}{
		{
			name: "full campaign with all optional fields",
			row: db.Campaign{
				ID:          uuidToPgtype(id),
				OrgID:       uuidToPgtype(orgID),
				Name:        "Summer Campaign",
				Status:      "active",
				Targeting:   targetingJSON,
				BudgetCents: pgtype.Int8{Int64: budget, Valid: true},
				Currency:    "USD",
				StartDate:   pgtype.Date{Time: startDate, Valid: true},
				EndDate:     pgtype.Date{Time: endDate, Valid: true},
				CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantBudget: true,
			wantDates:  true,
		},
		{
			name: "minimal campaign with no optional fields",
			row: db.Campaign{
				ID:          uuidToPgtype(id),
				OrgID:       uuidToPgtype(orgID),
				Name:        "Minimal",
				Status:      "draft",
				Targeting:   []byte(`{}`),
				BudgetCents: pgtype.Int8{Valid: false},
				Currency:    "EUR",
				StartDate:   pgtype.Date{Valid: false},
				EndDate:     pgtype.Date{Valid: false},
				CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantBudget: false,
			wantDates:  false,
		},
		{
			name: "nil targeting bytes yields empty targeting",
			row: db.Campaign{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				Name:      "No Targeting",
				Status:    "draft",
				Targeting: nil,
				Currency:  "USD",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		{
			name: "invalid targeting JSON returns error",
			row: db.Campaign{
				ID:        uuidToPgtype(id),
				OrgID:     uuidToPgtype(orgID),
				Name:      "Bad JSON",
				Status:    "draft",
				Targeting: []byte(`not-json`),
				Currency:  "USD",
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCampaign(tt.row)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid targeting JSON, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.ID != pgtypeToUUID(tt.row.ID) {
				t.Errorf("ID mismatch")
			}
			if got.OrgID != pgtypeToUUID(tt.row.OrgID) {
				t.Errorf("OrgID mismatch")
			}
			if got.Name != tt.row.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.row.Name)
			}
			if string(got.Status) != tt.row.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.row.Status)
			}
			if tt.wantBudget {
				if got.BudgetCents == nil {
					t.Error("expected non-nil BudgetCents")
				} else if *got.BudgetCents != budget {
					t.Errorf("BudgetCents = %d, want %d", *got.BudgetCents, budget)
				}
			} else if got.BudgetCents != nil {
				t.Errorf("expected nil BudgetCents, got %d", *got.BudgetCents)
			}

			if tt.wantDates {
				if got.StartDate == nil || got.EndDate == nil {
					t.Error("expected non-nil start/end dates")
				}
			} else {
				if got.StartDate != nil {
					t.Errorf("expected nil StartDate, got %v", got.StartDate)
				}
				if got.EndDate != nil {
					t.Errorf("expected nil EndDate, got %v", got.EndDate)
				}
			}
		})
	}
}

// TestCampaignRepo_GetByID_NotFound verifies pgx.ErrNoRows → model.ErrNotFound.
func TestCampaignRepo_GetByID_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestCampaignRepo_GetByID_DBError verifies generic errors are wrapped.
func TestCampaignRepo_GetByID_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("db error")}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	_, err := repo.GetByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == model.ErrNotFound {
		t.Error("should not map generic error to ErrNotFound")
	}
}

// TestCampaignRepo_Update_NotFound verifies pgx.ErrNoRows → model.ErrNotFound in Update.
func TestCampaignRepo_Update_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Update(context.Background(), &model.Campaign{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "test",
		Status:    model.StatusDraft,
		Currency:  "USD",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestCampaignRepo_UpdateStatus_NotFound verifies pgx.ErrNoRows → model.ErrNotFound in UpdateStatus.
func TestCampaignRepo_UpdateStatus_NotFound(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: pgx.ErrNoRows}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	_, err := repo.UpdateStatus(context.Background(), uuid.New(), uuid.New(), model.StatusActive, time.Now())
	if err != model.ErrNotFound {
		t.Errorf("got %v, want model.ErrNotFound", err)
	}
}

// TestCampaignRepo_Insert_DBError verifies insert errors are returned.
func TestCampaignRepo_Insert_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &errRow{err: errSentinel("insert failed")}
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	now := time.Now()
	_, err := repo.Insert(context.Background(), &model.Campaign{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Name:      "test",
		Status:    model.StatusDraft,
		Currency:  "USD",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestCampaignRepo_ListByOrg_DBError verifies list query errors are returned.
func TestCampaignRepo_ListByOrg_DBError(t *testing.T) {
	mock := &mockDBTX{
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errSentinel("list failed")
		},
	}
	repo := &campaignRepo{q: db.New(mock)}
	_, _, err := repo.ListByOrg(context.Background(), uuid.New(), nil, 10, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
