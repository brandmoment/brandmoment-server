package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

type OrganizationRepository interface {
	Insert(ctx context.Context, org *model.Organization) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Organization, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.Organization, error)
}

type organizationRepo struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) OrganizationRepository {
	return &organizationRepo{pool: pool}
}

func (r *organizationRepo) Insert(ctx context.Context, org *model.Organization) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO organizations (id, type, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		org.ID, org.Type, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

func (r *organizationRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Organization, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, type, name, slug, created_at, updated_at
		 FROM organizations
		 WHERE id = $1 AND id = $2`,
		orgID, id,
	)

	var org model.Organization
	err := row.Scan(&org.ID, &org.Type, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &org, nil
}

func (r *organizationRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.Organization, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, slug, created_at, updated_at
		 FROM organizations
		 WHERE id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []model.Organization
	for rows.Next() {
		var org model.Organization
		if err := rows.Scan(&org.ID, &org.Type, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}
