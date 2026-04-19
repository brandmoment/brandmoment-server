package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brandmoment/brandmoment-server/packages/shared-domain/db"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// OrgInviteRepository defines persistence operations for organization invitations.
type OrgInviteRepository interface {
	// Insert persists a new org invite and returns the stored record.
	Insert(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error)
	// GetByToken retrieves an invite by its unique token, returning ErrNotFound when absent.
	GetByToken(ctx context.Context, token string) (*model.OrgInvite, error)
}

type orgInviteRepo struct {
	q *db.Queries
}

// NewOrgInviteRepository constructs an OrgInviteRepository backed by the given connection pool.
func NewOrgInviteRepository(pool *pgxpool.Pool) OrgInviteRepository {
	return &orgInviteRepo{q: db.New(pool)}
}

func (r *orgInviteRepo) Insert(ctx context.Context, invite *model.OrgInvite) (*model.OrgInvite, error) {
	row, err := r.q.InsertOrgInvite(ctx, db.InsertOrgInviteParams{
		ID:        uuidToPgtype(invite.ID),
		OrgID:     uuidToPgtype(invite.OrgID),
		Email:     invite.Email,
		Role:      invite.Role,
		Token:     invite.Token,
		ExpiresAt: pgtype.Timestamptz{Time: invite.ExpiresAt, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: invite.CreatedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert org invite: %w", err)
	}
	return toOrgInvite(row), nil
}

func (r *orgInviteRepo) GetByToken(ctx context.Context, token string) (*model.OrgInvite, error) {
	row, err := r.q.GetOrgInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get org invite by token: %w", err)
	}
	return toOrgInvite(row), nil
}

func toOrgInvite(row db.OrgInvite) *model.OrgInvite {
	inv := &model.OrgInvite{
		ID:        pgtypeToUUID(row.ID),
		OrgID:     pgtypeToUUID(row.OrgID),
		Email:     row.Email,
		Role:      row.Role,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.AcceptedAt.Valid {
		t := row.AcceptedAt.Time
		inv.AcceptedAt = &t
	}
	return inv
}
