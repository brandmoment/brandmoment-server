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

// PublisherRuleRepository defines persistence operations for publisher targeting rules scoped to an app.
type PublisherRuleRepository interface {
	// Insert persists a new publisher rule and returns the stored record.
	Insert(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	// GetByID retrieves a publisher rule by org, app, and rule ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error)
	// ListByApp returns a paginated list of publisher rules for the given app together with the total count.
	ListByApp(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error)
	// Update replaces the mutable fields of a publisher rule and returns the updated record.
	Update(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error)
	// Delete permanently removes a publisher rule identified by org, app, and rule ID.
	Delete(ctx context.Context, orgID, appID, id uuid.UUID) error
}

type publisherRuleRepo struct {
	q *db.Queries
}

// NewPublisherRuleRepository constructs a PublisherRuleRepository backed by the given connection pool.
func NewPublisherRuleRepository(pool *pgxpool.Pool) PublisherRuleRepository {
	return &publisherRuleRepo{q: db.New(pool)}
}

func (r *publisherRuleRepo) Insert(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	row, err := r.q.InsertPublisherRule(ctx, db.InsertPublisherRuleParams{
		ID:        uuidToPgtype(rule.ID),
		OrgID:     uuidToPgtype(rule.OrgID),
		AppID:     uuidToPgtype(rule.AppID),
		Type:      rule.Type,
		Config:    []byte(rule.Config),
		IsActive:  rule.IsActive,
		CreatedAt: pgtype.Timestamptz{Time: rule.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: rule.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert publisher rule: %w", err)
	}
	return toPublisherRule(row), nil
}

func (r *publisherRuleRepo) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error) {
	row, err := r.q.GetPublisherRuleByID(ctx, db.GetPublisherRuleByIDParams{
		OrgID: uuidToPgtype(orgID),
		AppID: uuidToPgtype(appID),
		ID:    uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get publisher rule: %w", err)
	}
	return toPublisherRule(row), nil
}

func (r *publisherRuleRepo) ListByApp(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) ([]model.PublisherRule, int64, error) {
	rows, err := r.q.ListPublisherRulesByApp(ctx, db.ListPublisherRulesByAppParams{
		OrgID:     uuidToPgtype(orgID),
		AppID:     uuidToPgtype(appID),
		LimitVal:  limit,
		OffsetVal: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list publisher rules: %w", err)
	}

	total, err := r.q.CountPublisherRulesByApp(ctx, db.CountPublisherRulesByAppParams{
		OrgID: uuidToPgtype(orgID),
		AppID: uuidToPgtype(appID),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count publisher rules: %w", err)
	}

	rules := make([]model.PublisherRule, len(rows))
	for i, row := range rows {
		rules[i] = *toPublisherRule(row)
	}
	return rules, total, nil
}

func (r *publisherRuleRepo) Update(ctx context.Context, rule *model.PublisherRule) (*model.PublisherRule, error) {
	row, err := r.q.UpdatePublisherRule(ctx, db.UpdatePublisherRuleParams{
		Config:    []byte(rule.Config),
		IsActive:  rule.IsActive,
		UpdatedAt: pgtype.Timestamptz{Time: rule.UpdatedAt, Valid: true},
		OrgID:     uuidToPgtype(rule.OrgID),
		AppID:     uuidToPgtype(rule.AppID),
		ID:        uuidToPgtype(rule.ID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update publisher rule: %w", err)
	}
	return toPublisherRule(row), nil
}

func (r *publisherRuleRepo) Delete(ctx context.Context, orgID, appID, id uuid.UUID) error {
	err := r.q.DeletePublisherRule(ctx, db.DeletePublisherRuleParams{
		OrgID: uuidToPgtype(orgID),
		AppID: uuidToPgtype(appID),
		ID:    uuidToPgtype(id),
	})
	if err != nil {
		return fmt.Errorf("delete publisher rule: %w", err)
	}
	return nil
}

func toPublisherRule(row db.PublisherRule) *model.PublisherRule {
	return &model.PublisherRule{
		ID:        pgtypeToUUID(row.ID),
		OrgID:     pgtypeToUUID(row.OrgID),
		AppID:     pgtypeToUUID(row.AppID),
		Type:      row.Type,
		Config:    row.Config,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
