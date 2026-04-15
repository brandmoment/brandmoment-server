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

type CreateOrganizationRequest struct {
	Type model.OrgType `json:"type"`
	Name string        `json:"name"`
	Slug string        `json:"slug"`
}

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

func (s *OrganizationService) Create(ctx context.Context, orgID uuid.UUID, req CreateOrganizationRequest) (*model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.Create")
	defer span.End()

	if !req.Type.Valid() {
		return nil, model.ErrInvalidInput
	}
	if req.Name == "" || req.Slug == "" {
		return nil, model.ErrInvalidInput
	}

	now := time.Now()
	org := &model.Organization{
		ID:        uuid.New(),
		Type:      req.Type,
		Name:      req.Name,
		Slug:      req.Slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	slog.InfoContext(ctx, "creating organization",
		slog.String("name", req.Name),
		slog.String("slug", req.Slug),
		slog.String("type", string(req.Type)),
		slog.String("org_id", orgID.String()),
	)

	if err := s.repo.Insert(ctx, org); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert organization: %w", err)
	}

	return org, nil
}

func (s *OrganizationService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.GetByID")
	defer span.End()

	slog.DebugContext(ctx, "getting organization",
		slog.String("id", id.String()),
		slog.String("org_id", orgID.String()),
	)

	org, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get organization: %w", err)
	}

	return org, nil
}

func (s *OrganizationService) List(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]model.Organization, error) {
	ctx, span := s.tracer.Start(ctx, "OrganizationService.List")
	defer span.End()

	slog.DebugContext(ctx, "listing organizations",
		slog.String("org_id", orgID.String()),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	orgs, err := s.repo.ListByOrg(ctx, orgID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return orgs, nil
}
