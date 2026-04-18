package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/repository"
)

var validInviteRoles = map[string]bool{
	"admin":  true,
	"editor": true,
	"viewer": true,
}

const inviteTTL = 7 * 24 * time.Hour

type OrgInviteService struct {
	repo   repository.OrgInviteRepository
	tracer trace.Tracer
}

func NewOrgInviteService(repo repository.OrgInviteRepository, tp trace.TracerProvider) *OrgInviteService {
	return &OrgInviteService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

type CreateInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (s *OrgInviteService) Create(ctx context.Context, orgID uuid.UUID, req CreateInviteRequest) (*model.OrgInvite, error) {
	ctx, span := s.tracer.Start(ctx, "OrgInviteService.Create")
	defer span.End()

	if req.Email == "" {
		return nil, fmt.Errorf("%w: email is required", model.ErrInvalidInput)
	}
	if req.Role == "owner" {
		return nil, fmt.Errorf("%w: owner role cannot be assigned via invite", model.ErrInvalidInput)
	}
	if !validInviteRoles[req.Role] {
		return nil, fmt.Errorf("%w: role must be one of admin, editor, viewer", model.ErrInvalidInput)
	}

	slog.InfoContext(ctx, "creating org invite",
		slog.String("org_id", orgID.String()),
		slog.String("email", req.Email),
		slog.String("role", req.Role),
	)

	token, err := generateToken()
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("generate invite token: %w", err)
	}

	now := time.Now()
	invite := &model.OrgInvite{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     req.Email,
		Role:      req.Role,
		Token:     token,
		ExpiresAt: now.Add(inviteTTL),
		CreatedAt: now,
	}

	created, err := s.repo.Insert(ctx, invite)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert org invite: %w", err)
	}
	return created, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
