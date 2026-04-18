package repository

import (
	"context"
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

type APIKeyRepository interface {
	Insert(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
	GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error)
	ListByApp(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error)
	Revoke(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error)
}

type apiKeyRepo struct {
	q *db.Queries
}

func NewAPIKeyRepository(pool *pgxpool.Pool) APIKeyRepository {
	return &apiKeyRepo{q: db.New(pool)}
}

func (r *apiKeyRepo) Insert(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	row, err := r.q.InsertAPIKey(ctx, db.InsertAPIKeyParams{
		ID:        uuidToPgtype(key.ID),
		OrgID:     uuidToPgtype(key.OrgID),
		AppID:     uuidToPgtype(key.AppID),
		Name:      key.Name,
		KeyHash:   key.KeyHash,
		KeyPrefix: key.KeyPrefix,
		CreatedAt: pgtype.Timestamptz{Time: key.CreatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}
	return toAPIKey(row), nil
}

func (r *apiKeyRepo) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error) {
	row, err := r.q.GetAPIKeyByID(ctx, db.GetAPIKeyByIDParams{
		OrgID: uuidToPgtype(orgID),
		AppID: uuidToPgtype(appID),
		ID:    uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return toAPIKey(row), nil
}

func (r *apiKeyRepo) ListByApp(ctx context.Context, orgID, appID uuid.UUID, activeOnly bool) ([]model.APIKey, error) {
	rows, err := r.q.ListAPIKeysByApp(ctx, db.ListAPIKeysByAppParams{
		OrgID:      uuidToPgtype(orgID),
		AppID:      uuidToPgtype(appID),
		ActiveOnly: activeOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	keys := make([]model.APIKey, len(rows))
	for i, row := range rows {
		keys[i] = *toAPIKey(row)
	}
	return keys, nil
}

func (r *apiKeyRepo) Revoke(ctx context.Context, orgID, appID, id uuid.UUID, revokedAt time.Time) (*model.APIKey, error) {
	row, err := r.q.RevokeAPIKey(ctx, db.RevokeAPIKeyParams{
		RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
		OrgID:     uuidToPgtype(orgID),
		AppID:     uuidToPgtype(appID),
		ID:        uuidToPgtype(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("revoke api key: %w", err)
	}
	return toAPIKey(row), nil
}

func toAPIKey(row db.ApiKey) *model.APIKey {
	k := &model.APIKey{
		ID:        pgtypeToUUID(row.ID),
		OrgID:     pgtypeToUUID(row.OrgID),
		AppID:     pgtypeToUUID(row.AppID),
		Name:      row.Name,
		KeyHash:   row.KeyHash,
		KeyPrefix: row.KeyPrefix,
		IsRevoked: row.IsRevoked,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.RevokedAt.Valid {
		t := row.RevokedAt.Time
		k.RevokedAt = &t
	}
	return k
}
