package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// CreativeRepository defines persistence operations for ad creatives within a campaign.
type CreativeRepository interface {
	// Insert persists a new creative and returns the stored record.
	Insert(ctx context.Context, c *model.Creative) (*model.Creative, error)
	// GetByID retrieves a creative by org, campaign, and creative ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error)
	// ListByCampaign returns all creatives for the given campaign together with the total count.
	ListByCampaign(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error)
	// Update replaces the mutable fields of a creative and returns the updated record.
	Update(ctx context.Context, orgID, campaignID, id uuid.UUID, params UpdateCreativeParams) (*model.Creative, error)
}

// UpdateCreativeParams holds the fields that may be changed on an existing creative.
type UpdateCreativeParams struct {
	Name          string
	Type          string
	FileURL       string
	FileSizeBytes *int64
	PreviewURL    *string
	IsActive      bool
}

type creativeRepo struct {
	q *db.Queries
}

// NewCreativeRepository constructs a CreativeRepository backed by the given connection pool.
func NewCreativeRepository(pool *pgxpool.Pool) CreativeRepository {
	return &creativeRepo{q: db.New(pool)}
}

func (r *creativeRepo) Insert(ctx context.Context, c *model.Creative) (*model.Creative, error) {
	row, err := r.q.InsertCreative(ctx, db.InsertCreativeParams{
		ID:            uuidToPgtype(c.ID),
		OrgID:         uuidToPgtype(c.OrgID),
		CampaignID:    uuidToPgtype(c.CampaignID),
		Name:          c.Name,
		Type:          string(c.Type),
		FileUrl:       c.FileURL,
		FileSizeBytes: int64ToPgtypeInt8(c.FileSizeBytes),
		PreviewUrl:    stringToPgtypeText(c.PreviewURL),
		IsActive:      c.IsActive,
		CreatedAt:     pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: c.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert creative: %w", err)
	}
	return toCreative(row), nil
}

func (r *creativeRepo) GetByID(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error) {
	row, err := r.q.GetCreativeByID(ctx, db.GetCreativeByIDParams{
		OrgID:      uuidToPgtype(orgID),
		CampaignID: uuidToPgtype(campaignID),
		ID:         uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get creative: %w", err)
	}
	return toCreative(row), nil
}

func (r *creativeRepo) ListByCampaign(ctx context.Context, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error) {
	rows, err := r.q.ListCreativesByCampaign(ctx, db.ListCreativesByCampaignParams{
		OrgID:      uuidToPgtype(orgID),
		CampaignID: uuidToPgtype(campaignID),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list creatives: %w", err)
	}

	total, err := r.q.CountCreativesByCampaign(ctx, db.CountCreativesByCampaignParams{
		OrgID:      uuidToPgtype(orgID),
		CampaignID: uuidToPgtype(campaignID),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count creatives: %w", err)
	}

	creatives := make([]model.Creative, len(rows))
	for i, row := range rows {
		creatives[i] = *toCreative(row)
	}
	return creatives, total, nil
}

func (r *creativeRepo) Update(ctx context.Context, orgID, campaignID, id uuid.UUID, params UpdateCreativeParams) (*model.Creative, error) {
	row, err := r.q.UpdateCreative(ctx, db.UpdateCreativeParams{
		Name:          params.Name,
		Type:          params.Type,
		FileUrl:       params.FileURL,
		FileSizeBytes: int64ToPgtypeInt8(params.FileSizeBytes),
		PreviewUrl:    stringToPgtypeText(params.PreviewURL),
		IsActive:      params.IsActive,
		OrgID:         uuidToPgtype(orgID),
		CampaignID:    uuidToPgtype(campaignID),
		ID:            uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update creative: %w", err)
	}
	return toCreative(row), nil
}

// toCreative converts a sqlc-generated Creative row to the domain model.
func toCreative(row db.Creative) *model.Creative {
	c := &model.Creative{
		ID:         pgtypeToUUID(row.ID),
		OrgID:      pgtypeToUUID(row.OrgID),
		CampaignID: pgtypeToUUID(row.CampaignID),
		Name:       row.Name,
		Type:       model.CreativeType(row.Type),
		FileURL:    row.FileUrl,
		IsActive:   row.IsActive,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}

	if row.FileSizeBytes.Valid {
		v := row.FileSizeBytes.Int64
		c.FileSizeBytes = &v
	}
	if row.PreviewUrl.Valid {
		v := row.PreviewUrl.String
		c.PreviewURL = &v
	}

	return c
}

// stringToPgtypeText converts a nullable string pointer to pgtype.Text.
func stringToPgtypeText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
