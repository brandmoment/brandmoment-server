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

type OrganizationService struct {
	repo   repository.OrganizationRepository
	tracer trace.Tracer
}

func NewOrganizationService(repo repository.OrganizationRepository, tp trace.TracerProvider) *OrganizationService {
	return &OrganizationService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreateOrganizationRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *OrganizationService) Create(ctx context.Context, req CreateOrganizationRequest) (*model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.Create")
	defer span.End()

	if req.Name == "" || req.Slug == "" || req.Type == "" {
		return nil, fmt.Errorf("%w: name, slug, and type are required", model.ErrInvalidInput)
	}

	if req.Type != "admin" && req.Type != "publisher" && req.Type != "brand" {
		return nil, fmt.Errorf("%w: type must be admin, publisher, or brand", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "creating organization",
		slog.String("name", req.Name),
		slog.String("slug", req.Slug),
		slog.String("type", req.Type),
	)

	now := time.Now()
	org := &model.Organization{
		ID:        uuid.New(),
		Type:      req.Type,
		Name:      req.Name,
		Slug:      req.Slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.repo.Insert(ctx, org)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert organization: %w", err)
	}
	return created, nil
}

func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.GetByID")
	defer span.End()

	slog.InfoContext(ctx, "getting organization",
		slog.String("id", id.String()),
	)

	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.ListByIDs")
	defer span.End()

	slog.InfoContext(ctx, "listing organizations",
		slog.Int("count", len(ids)),
	)

	orgs, err := s.repo.ListByIDs(ctx, ids)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	return orgs, nil
}
