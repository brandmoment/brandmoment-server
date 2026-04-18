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

type CampaignService struct {
	repo   repository.CampaignRepository
	tracer trace.Tracer
}

func NewCampaignService(repo repository.CampaignRepository, tp trace.TracerProvider) *CampaignService {
	return &CampaignService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreateCampaignRequest struct {
	Name        string                   `json:"name"`
	Targeting   model.CampaignTargeting  `json:"targeting"`
	BudgetCents *int64                   `json:"budget_cents"`
	Currency    string                   `json:"currency"`
	StartDate   *string                  `json:"start_date"`
	EndDate     *string                  `json:"end_date"`
}

type UpdateCampaignRequest struct {
	Name        *string                  `json:"name"`
	Targeting   *model.CampaignTargeting `json:"targeting"`
	BudgetCents *int64                   `json:"budget_cents"`
	Currency    *string                  `json:"currency"`
	StartDate   *string                  `json:"start_date"`
	EndDate     *string                  `json:"end_date"`
}

type UpdateCampaignStatusRequest struct {
	Status model.CampaignStatus `json:"status"`
}

type CampaignListResult struct {
	Items  []model.Campaign `json:"items"`
	Total  int64            `json:"total"`
	Limit  int32            `json:"limit"`
	Offset int32            `json:"offset"`
}

const dateLayout = "2006-01-02"

func (s *CampaignService) Create(ctx context.Context, orgID uuid.UUID, req CreateCampaignRequest) (*model.Campaign, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.Create")
	defer span.End()

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}
	if len(req.Name) > 200 {
		return nil, fmt.Errorf("%w: name must be 200 characters or less", model.ErrInvalidInput)
	}
	if req.BudgetCents != nil && *req.BudgetCents < 0 {
		return nil, fmt.Errorf("%w: budget_cents must be non-negative", model.ErrInvalidInput)
	}

	var startDate, endDate *time.Time
	if req.StartDate != nil {
		t, err := time.Parse(dateLayout, *req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("%w: start_date must be in YYYY-MM-DD format", model.ErrInvalidInput)
		}
		startDate = &t
	}
	if req.EndDate != nil {
		t, err := time.Parse(dateLayout, *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%w: end_date must be in YYYY-MM-DD format", model.ErrInvalidInput)
		}
		endDate = &t
	}
	if startDate != nil && endDate != nil && !endDate.After(*startDate) {
		return nil, fmt.Errorf("%w: end_date must be after start_date", model.ErrInvalidInput)
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	slog.InfoContext(ctx, "creating campaign",
		slog.String("org_id", orgID.String()),
		slog.String("name", req.Name),
	)

	now := time.Now()
	c := &model.Campaign{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        req.Name,
		Status:      model.StatusDraft,
		Targeting:   req.Targeting,
		BudgetCents: req.BudgetCents,
		Currency:    currency,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.repo.Insert(ctx, c)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert campaign: %w", err)
	}
	return created, nil
}

func (s *CampaignService) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Campaign, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.GetByID")
	defer span.End()

	slog.InfoContext(ctx, "getting campaign",
		slog.String("org_id", orgID.String()),
		slog.String("id", id.String()),
	)

	c, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return c, nil
}

func (s *CampaignService) List(ctx context.Context, orgID uuid.UUID, statusFilter *string, limit, offset int32) (*CampaignListResult, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.List")
	defer span.End()

	if statusFilter != nil {
		switch model.CampaignStatus(*statusFilter) {
		case model.StatusDraft, model.StatusActive, model.StatusPaused, model.StatusCompleted:
			// valid
		default:
			return nil, fmt.Errorf("%w: status must be one of draft, active, paused, completed", model.ErrInvalidInput)
		}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	slog.InfoContext(ctx, "listing campaigns",
		slog.String("org_id", orgID.String()),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	campaigns, total, err := s.repo.ListByOrg(ctx, orgID, statusFilter, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list campaigns: %w", err)
	}

	return &CampaignListResult{
		Items:  campaigns,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *CampaignService) Update(ctx context.Context, orgID, id uuid.UUID, req UpdateCampaignRequest) (*model.Campaign, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.Update")
	defer span.End()

	slog.InfoContext(ctx, "updating campaign",
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
		if len(*req.Name) > 200 {
			return nil, fmt.Errorf("%w: name must be 200 characters or less", model.ErrInvalidInput)
		}
		existing.Name = *req.Name
	}
	if req.Targeting != nil {
		existing.Targeting = *req.Targeting
	}
	if req.BudgetCents != nil {
		if *req.BudgetCents < 0 {
			return nil, fmt.Errorf("%w: budget_cents must be non-negative", model.ErrInvalidInput)
		}
		existing.BudgetCents = req.BudgetCents
	}
	if req.Currency != nil {
		existing.Currency = *req.Currency
	}

	var newStart, newEnd *time.Time
	if req.StartDate != nil {
		t, err := time.Parse(dateLayout, *req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("%w: start_date must be in YYYY-MM-DD format", model.ErrInvalidInput)
		}
		newStart = &t
		existing.StartDate = newStart
	}
	if req.EndDate != nil {
		t, err := time.Parse(dateLayout, *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%w: end_date must be in YYYY-MM-DD format", model.ErrInvalidInput)
		}
		newEnd = &t
		existing.EndDate = newEnd
	}

	// Re-validate date order after applying updates.
	if existing.StartDate != nil && existing.EndDate != nil && !existing.EndDate.After(*existing.StartDate) {
		return nil, fmt.Errorf("%w: end_date must be after start_date", model.ErrInvalidInput)
	}

	existing.UpdatedAt = time.Now()

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	return updated, nil
}

func (s *CampaignService) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, req UpdateCampaignStatusRequest) (*model.Campaign, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.UpdateStatus")
	defer span.End()

	slog.InfoContext(ctx, "updating campaign status",
		slog.String("org_id", orgID.String()),
		slog.String("id", id.String()),
		slog.String("target_status", string(req.Status)),
	)

	existing, err := s.repo.GetByID(ctx, orgID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	allowed, ok := model.ValidTransitions[existing.Status]
	if !ok {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s (INVALID_TRANSITION)", model.ErrInvalidInput, existing.Status, req.Status)
	}

	valid := false
	for _, next := range allowed {
		if next == req.Status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s (INVALID_TRANSITION)", model.ErrInvalidInput, existing.Status, req.Status)
	}

	slog.InfoContext(ctx, "campaign status transition",
		slog.String("org_id", orgID.String()),
		slog.String("id", id.String()),
		slog.String("from_status", string(existing.Status)),
		slog.String("to_status", string(req.Status)),
	)

	updated, err := s.repo.UpdateStatus(ctx, orgID, id, req.Status, time.Now())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update campaign status: %w", err)
	}
	return updated, nil
}
