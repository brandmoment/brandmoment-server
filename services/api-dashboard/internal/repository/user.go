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

// UserRepository defines persistence operations for user accounts.
type UserRepository interface {
	// GetByID retrieves a user by their ID, returning ErrNotFound when absent.
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	// Upsert inserts or updates a user record and returns the stored record.
	Upsert(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error)
}

type userRepo struct {
	q *db.Queries
}

// NewUserRepository constructs a UserRepository backed by the given connection pool.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{q: db.New(pool)}
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return toUser(row), nil
}

func (r *userRepo) Upsert(ctx context.Context, id uuid.UUID, email, name string, createdAt time.Time) (*model.User, error) {
	row, err := r.q.UpsertUser(ctx, db.UpsertUserParams{
		ID:        uuidToPgtype(id),
		Email:     email,
		Name:      name,
		CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return toUser(row), nil
}

func toUser(row db.User) *model.User {
	return &model.User{
		ID:        pgtypeToUUID(row.ID),
		Email:     row.Email,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
}
