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

// OrganizationRepository defines persistence operations for organizations.
type OrganizationRepository interface {
	// Insert persists a new organization and returns the stored record.
	Insert(ctx context.Context, org *model.Organization) (*model.Organization, error)
	// GetByID retrieves an organization by its ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	// ListByIDs returns all organizations whose IDs are in the provided slice.
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error)
}

type organizationRepo struct {
	q *db.Queries
}

// NewOrganizationRepository constructs an OrganizationRepository backed by the given connection pool.
func NewOrganizationRepository(pool *pgxpool.Pool) OrganizationRepository {
	return &organizationRepo{q: db.New(pool)}
}

func (r *organizationRepo) Insert(ctx context.Context, org *model.Organization) (*model.Organization, error) {
	row, err := r.q.InsertOrganization(ctx, db.InsertOrganizationParams{
		ID:        uuidToPgtype(org.ID),
		Type:      org.Type,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: pgtype.Timestamptz{Time: org.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: org.UpdatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert organization: %w", err)
	}
	return toOrganization(row), nil
}

func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	row, err := r.q.GetOrganizationByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return toOrganization(row), nil
}

func (r *organizationRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = uuidToPgtype(id)
	}
	rows, err := r.q.ListOrganizationsByIDs(ctx, pgIDs)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	orgs := make([]model.Organization, len(rows))
	for i, row := range rows {
		orgs[i] = *toOrganization(row)
	}
	return orgs, nil
}

func toOrganization(row db.Organization) *model.Organization {
	return &model.Organization{
		ID:        pgtypeToUUID(row.ID),
		Type:      row.Type,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
