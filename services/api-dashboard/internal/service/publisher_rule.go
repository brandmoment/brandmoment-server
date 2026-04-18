package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
)

type PublisherRuleService struct {
	repo   repository.PublisherRuleRepository
	tracer trace.Tracer
}

func NewPublisherRuleService(repo repository.PublisherRuleRepository, tp trace.TracerProvider) *PublisherRuleService {
	return &PublisherRuleService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreatePublisherRuleRequest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type UpdatePublisherRuleRequest struct {
	Config   json.RawMessage `json:"config"`
	IsActive *bool           `json:"is_active"`
}

type PublisherRuleListResult struct {
	Items  []model.PublisherRule `json:"items"`
	Total  int64                 `json:"total"`
	Limit  int32                 `json:"limit"`
	Offset int32                 `json:"offset"`
}

func (s *PublisherRuleService) Create(ctx context.Context, orgID, appID uuid.UUID, req CreatePublisherRuleRequest) (*model.PublisherRule, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherRuleService.Create")
	defer span.End()

	if err := validateRuleType(req.Type); err != nil {
		return nil, err
	}
	if err := validateRuleConfig(req.Type, req.Config); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "creating publisher rule",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("type", req.Type),
	)

	now := time.Now()
	rule := &model.PublisherRule{
		ID:        uuid.New(),
		OrgID:     orgID,
		AppID:     appID,
		Type:      req.Type,
		Config:    req.Config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.repo.Insert(ctx, rule)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert publisher rule: %w", err)
	}
	return created, nil
}

func (s *PublisherRuleService) GetByID(ctx context.Context, orgID, appID, id uuid.UUID) (*model.PublisherRule, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherRuleService.GetByID")
	defer span.End()

	slog.InfoContext(ctx, "getting publisher rule",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("id", id.String()),
	)

	rule, err := s.repo.GetByID(ctx, orgID, appID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return rule, nil
}

func (s *PublisherRuleService) List(ctx context.Context, orgID, appID uuid.UUID, limit, offset int32) (*PublisherRuleListResult, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherRuleService.List")
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

	slog.InfoContext(ctx, "listing publisher rules",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	rules, total, err := s.repo.ListByApp(ctx, orgID, appID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list publisher rules: %w", err)
	}

	return &PublisherRuleListResult{
		Items:  rules,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *PublisherRuleService) Update(ctx context.Context, orgID, appID, id uuid.UUID, req UpdatePublisherRuleRequest) (*model.PublisherRule, error) {
	ctx, span := s.tracer.Start(ctx, "PublisherRuleService.Update")
	defer span.End()

	slog.InfoContext(ctx, "updating publisher rule",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("id", id.String()),
	)

	existing, err := s.repo.GetByID(ctx, orgID, appID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if req.Config != nil {
		if err := validateRuleConfig(existing.Type, req.Config); err != nil {
			return nil, err
		}
		existing.Config = req.Config
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	existing.UpdatedAt = time.Now()

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update publisher rule: %w", err)
	}
	return updated, nil
}

func (s *PublisherRuleService) Delete(ctx context.Context, orgID, appID, id uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "PublisherRuleService.Delete")
	defer span.End()

	slog.InfoContext(ctx, "deleting publisher rule",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("id", id.String()),
	)

	// Verify existence before delete so we return 404 on missing rule.
	if _, err := s.repo.GetByID(ctx, orgID, appID, id); err != nil {
		span.RecordError(err)
		return err
	}

	if err := s.repo.Delete(ctx, orgID, appID, id); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete publisher rule: %w", err)
	}
	return nil
}

// validateRuleType checks that the type is one of the 5 supported values.
func validateRuleType(ruleType string) error {
	switch ruleType {
	case model.RuleTypeBlocklist,
		model.RuleTypeAllowlist,
		model.RuleTypeFrequencyCap,
		model.RuleTypeGeoFilter,
		model.RuleTypePlatformFilter:
		return nil
	default:
		return fmt.Errorf("%w: unknown rule type %q; must be one of blocklist, allowlist, frequency_cap, geo_filter, platform_filter", model.ErrInvalidInput, ruleType)
	}
}

// validateRuleConfig deserializes config into the appropriate struct for the given type.
func validateRuleConfig(ruleType string, config json.RawMessage) error {
	if len(config) == 0 {
		return fmt.Errorf("%w: config is required", model.ErrInvalidInput)
	}

	switch ruleType {
	case model.RuleTypeBlocklist:
		var cfg model.BlocklistConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("%w: invalid blocklist config: %s", model.ErrInvalidInput, err)
		}
	case model.RuleTypeAllowlist:
		var cfg model.AllowlistConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("%w: invalid allowlist config: %s", model.ErrInvalidInput, err)
		}
	case model.RuleTypeFrequencyCap:
		var cfg model.FrequencyCapConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("%w: invalid frequency_cap config: %s", model.ErrInvalidInput, err)
		}
		if cfg.MaxImpressions <= 0 {
			return fmt.Errorf("%w: frequency_cap requires max_impressions > 0", model.ErrInvalidInput)
		}
		if cfg.WindowSeconds <= 0 {
			return fmt.Errorf("%w: frequency_cap requires window_seconds > 0", model.ErrInvalidInput)
		}
	case model.RuleTypeGeoFilter:
		var cfg model.GeoFilterConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("%w: invalid geo_filter config: %s", model.ErrInvalidInput, err)
		}
		if cfg.Mode != "include" && cfg.Mode != "exclude" {
			return fmt.Errorf("%w: geo_filter mode must be include or exclude", model.ErrInvalidInput)
		}
		if len(cfg.CountryCodes) == 0 {
			return fmt.Errorf("%w: geo_filter requires at least one country_code", model.ErrInvalidInput)
		}
	case model.RuleTypePlatformFilter:
		var cfg model.PlatformFilterConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return fmt.Errorf("%w: invalid platform_filter config: %s", model.ErrInvalidInput, err)
		}
		if cfg.Mode != "include" && cfg.Mode != "exclude" {
			return fmt.Errorf("%w: platform_filter mode must be include or exclude", model.ErrInvalidInput)
		}
		if len(cfg.Platforms) == 0 {
			return fmt.Errorf("%w: platform_filter requires at least one platform", model.ErrInvalidInput)
		}
	}
	return nil
}
