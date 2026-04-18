package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
)

// APIKeyProvisionResult carries the provisioned key and the plaintext.
// The plaintext is returned exactly once and must never be stored or logged.
type APIKeyProvisionResult struct {
	Key      *model.APIKey
	Plaintext string
}

type APIKeyListResult struct {
	Items []model.APIKey `json:"items"`
}

type APIKeyService struct {
	repo   repository.APIKeyRepository
	tracer trace.Tracer
}

func NewAPIKeyService(repo repository.APIKeyRepository, tp trace.TracerProvider) *APIKeyService {
	return &APIKeyService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type ProvisionAPIKeyRequest struct {
	Name string `json:"name"`
}

func (s *APIKeyService) Provision(ctx context.Context, orgID, appID uuid.UUID, req ProvisionAPIKeyRequest) (*APIKeyProvisionResult, error) {
	ctx, span := s.tracer.Start(ctx, "APIKeyService.Provision")
	defer span.End()

	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "provisioning api key",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("name", req.Name),
	)

	// Generate 32 random bytes encoded as hex, prefixed with "bm_".
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("generate key bytes: %w", err)
	}
	plaintext := "bm_" + hex.EncodeToString(rawBytes)

	// key_prefix: first 8 chars of plaintext (includes "bm_" prefix).
	keyPrefix := plaintext[:8]

	// key_hash: hex-encoded SHA-256 of plaintext.
	sum := sha256.Sum256([]byte(plaintext))
	keyHash := fmt.Sprintf("%x", sum)

	now := time.Now()
	apiKey := &model.APIKey{
		ID:        uuid.New(),
		OrgID:     orgID,
		AppID:     appID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		IsRevoked: false,
		CreatedAt: now,
	}

	created, err := s.repo.Insert(ctx, apiKey)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert api key: %w", err)
	}

	return &APIKeyProvisionResult{
		Key:       created,
		Plaintext: plaintext,
	}, nil
}

func (s *APIKeyService) ListByApp(ctx context.Context, orgID, appID uuid.UUID, includeRevoked bool) (*APIKeyListResult, error) {
	ctx, span := s.tracer.Start(ctx, "APIKeyService.ListByApp")
	defer span.End()

	slog.InfoContext(ctx, "listing api keys",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.Bool("include_revoked", includeRevoked),
	)

	activeOnly := !includeRevoked
	keys, err := s.repo.ListByApp(ctx, orgID, appID, activeOnly)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	return &APIKeyListResult{Items: keys}, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, orgID, appID, id uuid.UUID) (*model.APIKey, error) {
	ctx, span := s.tracer.Start(ctx, "APIKeyService.Revoke")
	defer span.End()

	slog.InfoContext(ctx, "revoking api key",
		slog.String("org_id", orgID.String()),
		slog.String("app_id", appID.String()),
		slog.String("id", id.String()),
	)

	existing, err := s.repo.GetByID(ctx, orgID, appID, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if existing.IsRevoked {
		return nil, fmt.Errorf("%w: key already revoked", model.ErrInvalidInput)
	}

	revoked, err := s.repo.Revoke(ctx, orgID, appID, id, time.Now())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("revoke api key: %w", err)
	}
	return revoked, nil
}
