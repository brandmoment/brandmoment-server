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

type PublisherAppService struct {
	repo   repository.PublisherAppRepository
	tracer trace.Tracer
}

func NewPublisherAppService(repo repository.PublisherAppRepository, tp trace.TracerProvider) *PublisherAppService {
	return &PublisherAppService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreatePublisherAppRequest struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	BundleID string `json:"bundle_id"`
}

type UpdatePublisherAppRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type PublisherAppListResult struct {
	Items  []model.PublisherApp `json:"items"`
	Total  int64                `json:"total"`
	Limit  int32                `json:"limit"`
	Offset int32                `json:"offset"`
}

func (s *PublisherAppService) Create(ctx context.Context, orgID uuid.UUID, req CreatePublisherAppRequest) (*model.PublisherApp, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherAppService.Create")
	defer span.End()

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}
	if len(req.Name) > 100 {
		return nil, fmt.Errorf("%w: name must be 100 characters or less", model.ErrInvalidInput)
	}
	if req.Platform != "ios" && req.Platform != "android" && req.Platform != "web" {
		return nil, fmt.Errorf("%w: platform must be ios, android, or web", model.ErrInvalidInput)
	}
	if req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundle_id is required", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "creating publisher app",
		slog.String("org_id", orgID.String()),
		slog.String("name", req.Name),
		slog.String("platform", req.Platform),
	)

	// Enforce bundle_id uniqueness per org.
	_, err := s.repo.GetByBundleID(ctx, orgID, req.BundleID)
	if err == nil {
		return nil, fmt.Errorf("%w: bundle_id already exists for this org", model.ErrInvalidInput)
	}
	if err != model.ErrNotFound {
		span.RecordError(err)
		return nil, fmt.Errorf("check bundle_id uniqueness: %w", err)
	}

	now := time.Now()
	app := &model.PublisherApp{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      req.Name,
		Platform:  req.Platform,
		BundleID:  req.BundleID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.repo.Insert(ctx, app)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert publisher app: %w", err)
	}
	return created, nil
}

func (s *PublisherAppService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.PublisherApp, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherAppService.GetByID")
	defer span.End()

	slog.InfoContext(ctx, "getting publisher app",
		slog.String("org_id", orgID.String()),
		slog.String("id", id.String()),
	)

	app, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return app, nil
}

func (s *PublisherAppService) List(ctx context.Context, orgID uuid.UUID, limit, offset int32) (*PublisherAppListResult, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherAppService.List")
	defer span.End()

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	slog.InfoContext(ctx, "listing publisher apps",
		slog.String("org_id", orgID.String()),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	apps, total, err := s.repo.ListByOrg(ctx, orgID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list publisher apps: %w", err)
	}

	return &PublisherAppListResult{
		Items:  apps,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *PublisherAppService) Update(ctx context.Context, orgID, id uuid.UUID, req UpdatePublisherAppRequest) (*model.PublisherApp, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherAppService.Update")
	defer span.End()

	slog.InfoContext(ctx, "updating publisher app",
		slog.String("org_id", orgID.String()),
		slog.String("id", id.String()),
	)

	existing, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", model.ErrInvalidInput)
		}
		if len(*req.Name) > 100 {
			return nil, fmt.Errorf("%w: name must be 100 characters or less", model.ErrInvalidInput)
		}
		existing.Name = *req.Name
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	existing.UpdatedAt = time.Now()

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update publisher app: %w", err)
	}
	return updated, nil
}
