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

type CreativeService struct {
	campaignRepo repository.CampaignRepository
	creativeRepo repository.CreativeRepository
	tracer       trace.Tracer
}

func NewCreativeService(campaignRepo repository.CampaignRepository, creativeRepo repository.CreativeRepository, tp trace.TracerProvider) *CreativeService {
	return &CreativeService{
		campaignRepo: campaignRepo,
		creativeRepo: creativeRepo,
		tracer:       tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreateCreativeRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	FileURL       string `json:"file_url"`
	FileSizeBytes *int64 `json:"file_size_bytes"`
	PreviewURL    *string `json:"preview_url"`
}

type CreativeListResult struct {
	Items []model.Creative `json:"items"`
	Total int64            `json:"total"`
}

var validCreativeTypes = map[string]bool{
	string(model.TypeHTML5): true,
	string(model.TypeImage): true,
	string(model.TypeVideo): true,
}

func (s *CreativeService) Create(ctx context.Context, orgID, campaignID uuid.UUID, req CreateCreativeRequest) (*model.Creative, error) {
	ctx, span := s.tracer.Start(ctx, "CreativeService.Create")
	defer span.End()

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}
	if len(req.Name) > 200 {
		return nil, fmt.Errorf("%w: name must be 200 characters or less", model.ErrInvalidInput)
	}
	if !validCreativeTypes[req.Type] {
		return nil, fmt.Errorf("%w: type must be one of html5, image, video", model.ErrInvalidInput)
	}
	if req.FileSizeBytes != nil && *req.FileSizeBytes <= 0 {
		return nil, fmt.Errorf("%w: file_size_bytes must be a positive integer", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "creating creative",
		slog.String("org_id", orgID.String()),
		slog.String("campaign_id", campaignID.String()),
		slog.String("name", req.Name),
	)

	// Verify the campaign belongs to the org before inserting (prevents cross-org injection).
	if _, err := s.campaignRepo.GetByID(ctx, orgID, campaignID); err != nil {
		span.RecordError(err)
		return nil, err
	}

	now := time.Now()
	c := &model.Creative{
		ID:            uuid.New(),
		OrgID:         orgID,
		CampaignID:    campaignID,
		Name:          req.Name,
		Type:          model.CreativeType(req.Type),
		FileURL:       req.FileURL,
		FileSizeBytes: req.FileSizeBytes,
		PreviewURL:    req.PreviewURL,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	created, err := s.creativeRepo.Insert(ctx, c)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert creative: %w", err)
	}
	return created, nil
}

func (s *CreativeService) GetByID(ctx context.Context, orgID, campaignID, id uuid.UUID) (*model.Creative, error) {
	ctx, span := s.tracer.Start(ctx, "CreativeService.GetByID")
	defer span.End()

	slog.InfoContext(ctx, "getting creative",
		slog.String("org_id", orgID.String()),
		slog.String("campaign_id", campaignID.String()),
		slog.String("id", id.String()),
	)

	c, err := s.creativeRepo.GetByID(ctx, orgID, campaignID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return c, nil
}

func (s *CreativeService) ListByCampaign(ctx context.Context, orgID, campaignID uuid.UUID) (*CreativeListResult, error) {
	ctx, span := s.tracer.Start(ctx, "CreativeService.ListByCampaign")
	defer span.End()

	slog.InfoContext(ctx, "listing creatives",
		slog.String("org_id", orgID.String()),
		slog.String("campaign_id", campaignID.String()),
	)

	// Verify the campaign belongs to the org.
	if _, err := s.campaignRepo.GetByID(ctx, orgID, campaignID); err != nil {
		span.RecordError(err)
		return nil, err
	}

	creatives, total, err := s.creativeRepo.ListByCampaign(ctx, orgID, campaignID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list creatives: %w", err)
	}

	return &CreativeListResult{
		Items: creatives,
		Total: total,
	}, nil
}
