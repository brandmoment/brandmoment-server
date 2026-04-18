package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
)

type UserService struct {
	repo   repository.UserRepository
	tracer trace.Tracer
}

func NewUserService(repo repository.UserRepository, tp trace.TracerProvider) *UserService {
	return &UserService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

// GetMe fetches the user record by ID from the users table.
// The orgs list is resolved at the handler layer from JWT claims.
func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.GetMe")
	defer span.End()

	slog.InfoContext(ctx, "getting current user",
		slog.String("user_id", userID.String()),
	)

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return user, nil
}

// UpsertUser creates or updates the user record.
// Called on first login to ensure the platform user record exists.
func (s *UserService) UpsertUser(ctx context.Context, id uuid.UUID, email, name string) (*model.User, error) {
	ctx, span := s.tracer.Start(ctx, "UserService.UpsertUser")
	defer span.End()

	if email == "" {
		return nil, fmt.Errorf("%w: email is required", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "upserting user",
		slog.String("user_id", id.String()),
		slog.String("email", email),
	)

	user, err := s.repo.Upsert(ctx, id, email, name, time.Now())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}
