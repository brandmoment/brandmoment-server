package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// CampaignRepository defines persistence operations for advertising campaigns.
type CampaignRepository interface {
	// Insert persists a new campaign and returns the stored record.
	Insert(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	// GetByID retrieves a campaign by org and campaign ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error)
	// ListByOrg returns a paginated list of campaigns for the org with an optional status filter and the total count.
	ListByOrg(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error)
	// Update replaces all mutable fields of a campaign and returns the updated record.
	Update(ctx context.Context, c *model.Campaign) (*model.Campaign, error)
	// UpdateStatus transitions a campaign to the given status and returns the updated record.
	UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error)
}

type campaignRepo struct {
	q *db.Queries
}

// NewCampaignRepository constructs a CampaignRepository backed by the given connection pool.
func NewCampaignRepository(pool *pgxpool.Pool) CampaignRepository {
	return &campaignRepo{q: db.New(pool)}
}

func (r *campaignRepo) Insert(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	targeting, err := json.Marshal(c.Targeting)
	if err != nil {
		return nil, fmt.Errorf("marshal targeting: %w", err)
	}

	row, err := r.q.InsertCampaign(ctx, db.InsertCampaignParams{
		ID:          uuidToPgtype(c.ID),
		OrgID:       uuidToPgtype(c.OrgID),
		Name:        c.Name,
		Status:      string(c.Status),
		Targeting:   targeting,
		BudgetCents: int64ToPgtypeInt8(c.BudgetCents),
		Currency:    c.Currency,
		StartDate:   timeToPgtypeDate(c.StartDate),
		EndDate:     timeToPgtypeDate(c.EndDate),
		CreatedAt:   pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert campaign: %w", err)
	}
	return toCampaign(row)
}

func (r *campaignRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error) {
	row, err := r.q.GetCampaignByID(ctx, db.GetCampaignByIDParams{
		OrgID: uuidToPgtype(orgID),
		ID:    uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return toCampaign(row)
}

func (r *campaignRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error) {
	sf := pgtype.Text{}
	if statusFilter != nil {
		sf = pgtype.Text{String: *statusFilter, Valid: true}
	}

	rows, err := r.q.ListCampaignsByOrg(ctx, db.ListCampaignsByOrgParams{
		OrgID:        uuidToPgtype(orgID),
		StatusFilter: sf,
		LimitVal:     limit,
		OffsetVal:    offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	total, err := r.q.CountCampaignsByOrg(ctx, db.CountCampaignsByOrgParams{
		OrgID:        uuidToPgtype(orgID),
		StatusFilter: sf,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count campaigns: %w", err)
	}

	campaigns := make([]model.Campaign, 0, len(rows))
	for _, row := range rows {
		c, err := toCampaign(row)
		if err != nil {
			return nil, 0, err
		}
		campaigns = append(campaigns, *c)
	}
	return campaigns, total, nil
}

func (r *campaignRepo) Update(ctx context.Context, c *model.Campaign) (*model.Campaign, error) {
	targeting, err := json.Marshal(c.Targeting)
	if err != nil {
		return nil, fmt.Errorf("marshal targeting: %w", err)
	}

	row, err := r.q.UpdateCampaign(ctx, db.UpdateCampaignParams{
		Name:        c.Name,
		Targeting:   targeting,
		BudgetCents: int64ToPgtypeInt8(c.BudgetCents),
		Currency:    c.Currency,
		StartDate:   timeToPgtypeDate(c.StartDate),
		EndDate:     timeToPgtypeDate(c.EndDate),
		UpdatedAt:   pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
		OrgID:       uuidToPgtype(c.OrgID),
		ID:          uuidToPgtype(c.ID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	return toCampaign(row)
}

func (r *campaignRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error) {
	row, err := r.q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		Status:    string(status),
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
		OrgID:     uuidToPgtype(orgID),
		ID:        uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update campaign status: %w", err)
	}
	return toCampaign(row)
}

// toCampaign converts a sqlc-generated Campaign row to the domain model.
func toCampaign(row db.Campaign) (*model.Campaign, error) {
	var targeting model.CampaignTargeting
	if len(row.Targeting) > 0 {
		if err := json.Unmarshal(row.Targeting, &targeting); err != nil {
			return nil, fmt.Errorf("unmarshal targeting: %w", err)
		}
	}

	c := &model.Campaign{
		ID:        pgtypeToUUID(row.ID),
		OrgID:     pgtypeToUUID(row.OrgID),
		Name:      row.Name,
		Status:    model.CampaignStatus(row.Status),
		Targeting: targeting,
		Currency:  row.Currency,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.BudgetCents.Valid {
		v := row.BudgetCents.Int64
		c.BudgetCents = &v
	}
	if row.StartDate.Valid {
		t := row.StartDate.Time
		c.StartDate = &t
	}
	if row.EndDate.Valid {
		t := row.EndDate.Time
		c.EndDate = &t
	}

	return c, nil
}

// int64ToPgtypeInt8 converts a nullable int64 pointer to pgtype.Int8.
func int64ToPgtypeInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

// timeToPgtypeDate converts a nullable time.Time pointer to pgtype.Date.
func timeToPgtypeDate(v *time.Time) pgtype.Date {
	if v == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *v, Valid: true}
}
