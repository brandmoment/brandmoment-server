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

// PublisherAppRepository defines persistence operations for publisher mobile applications.
type PublisherAppRepository interface {
	// Insert persists a new publisher app and returns the stored record.
	Insert(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
	// GetByID retrieves a publisher app by org and app ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error)
	// GetByBundleID retrieves a publisher app by org and bundle identifier, returning ErrNotFound when absent.
	GetByBundleID(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error)
	// ListByOrg returns a paginated list of publisher apps for the org together with the total count.
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error)
	// Update replaces the mutable fields of a publisher app and returns the updated record.
	Update(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error)
}

type publisherAppRepo struct {
	q *db.Queries
}

// NewPublisherAppRepository constructs a PublisherAppRepository backed by the given connection pool.
func NewPublisherAppRepository(pool *pgxpool.Pool) PublisherAppRepository {
	return &publisherAppRepo{q: db.New(pool)}
}

func (r *publisherAppRepo) Insert(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	row, err := r.q.InsertPublisherApp(ctx, db.InsertPublisherAppParams{
		ID:        uuidToPgtype(app.ID),
		OrgID:     uuidToPgtype(app.OrgID),
		Name:      app.Name,
		Platform:  app.Platform,
		BundleID:  app.BundleID,
		IsActive:  app.IsActive,
		CreatedAt: pgtype.Timestamptz{Time: app.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: app.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert publisher app: %w", err)
	}
	return toPublisherApp(row), nil
}

func (r *publisherAppRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error) {
	row, err := r.q.GetPublisherAppByID(ctx, db.GetPublisherAppByIDParams{
		OrgID: uuidToPgtype(orgID),
		ID:    uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get publisher app: %w", err)
	}
	return toPublisherApp(row), nil
}

func (r *publisherAppRepo) GetByBundleID(ctx context.Context, orgID uuid.UUID, bundleID string) (*model.PublisherApp, error) {
	row, err := r.q.GetPublisherAppByBundleID(ctx, db.GetPublisherAppByBundleIDParams{
		OrgID:    uuidToPgtype(orgID),
		BundleID: bundleID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get publisher app by bundle id: %w", err)
	}
	return toPublisherApp(row), nil
}

func (r *publisherAppRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.PublisherApp, int64, error) {
	rows, err := r.q.ListPublisherAppsByOrg(ctx, db.ListPublisherAppsByOrgParams{
		OrgID:     uuidToPgtype(orgID),
		LimitVal:  limit,
		OffsetVal: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list publisher apps: %w", err)
	}

	total, err := r.q.CountPublisherAppsByOrg(ctx, uuidToPgtype(orgID))
	if err != nil {
		return nil, 0, fmt.Errorf("count publisher apps: %w", err)
	}

	apps := make([]model.PublisherApp, len(rows))
	for i, row := range rows {
		apps[i] = *toPublisherApp(row)
	}
	return apps, total, nil
}

func (r *publisherAppRepo) Update(ctx context.Context, app *model.PublisherApp) (*model.PublisherApp, error) {
	row, err := r.q.UpdatePublisherApp(ctx, db.UpdatePublisherAppParams{
		Name:      app.Name,
		IsActive:  app.IsActive,
		UpdatedAt: pgtype.Timestamptz{Time: app.UpdatedAt, Valid: true},
		OrgID:     uuidToPgtype(app.OrgID),
		ID:        uuidToPgtype(app.ID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update publisher app: %w", err)
	}
	return toPublisherApp(row), nil
}

func toPublisherApp(row db.PublisherApp) *model.PublisherApp {
	return &model.PublisherApp{
		ID:        pgtypeToUUID(row.ID),
		OrgID:     pgtypeToUUID(row.OrgID),
		Name:      row.Name,
		Platform:  row.Platform,
		BundleID:  row.BundleID,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
