---
description: Go backend patterns, anti-patterns, and code templates for api-dashboard and api-sdk services
globs: "**/*.go"
---

# Go Backend Rules

## Service Structure

```
services/api-dashboard/
├── cmd/server/main.go           # Entry point: config, DI, server start
├── internal/
│   ├── config/config.go         # Env-based config (envconfig)
│   ├── router/router.go         # chi router setup (extracted from main)
│   ├── handler/*.go             # HTTP handlers (request/response only)
│   ├── service/*.go             # Business logic + OTel + slog
│   ├── repository/*.go          # DB access (wraps sqlc-generated code)
│   ├── middleware/{auth,rbac}.go # JWT validation + role checks
│   └── model/*.go               # Domain types + sentinel errors
├── go.mod
└── go.sum
```

Router MUST be in `internal/router/router.go` as a standalone `NewRouter()` function — not inline in main.go.

## Import Order

stdlib, blank line, third-party, blank line, internal:

```go
import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)
```

## Good Code Examples

### Handler — decode, delegate, respond

```go
func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.OrgIDFromContext(ctx)

	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode request")
		return
	}

	campaign, err := h.service.Create(ctx, orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, campaign)
}
```

### Service — DI + OTel tracing + slog

```go
type CampaignService struct {
	repo   CampaignRepository
	tracer trace.Tracer
}

func NewCampaignService(repo CampaignRepository, tp trace.TracerProvider) *CampaignService {
	return &CampaignService{
		repo:   repo,
		tracer: tp.Tracer("brandmoment/api-dashboard"),
	}
}

func (s *CampaignService) Create(ctx context.Context, orgID uuid.UUID, req CreateCampaignRequest) (*Campaign, error) {
	ctx, span := s.tracer.Start(ctx, "CampaignService.Create")
	defer span.End()

	slog.InfoContext(ctx, "creating campaign",
		slog.String("name", req.Name),
		slog.String("org_id", orgID.String()),
	)

	campaign := &Campaign{ID: uuid.New(), OrgID: orgID, Name: req.Name}
	if err := s.repo.Insert(ctx, campaign); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert campaign: %w", err)
	}
	return campaign, nil
}
```

### Repository — wraps sqlc, never raw SQL

```go
type CampaignRepository interface {
	Insert(ctx context.Context, c *Campaign) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*Campaign, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]Campaign, error)
}

type campaignRepo struct {
	q *db.Queries // sqlc-generated
}

func NewCampaignRepository(pool *pgxpool.Pool) CampaignRepository {
	return &campaignRepo{q: db.New(pool)}
}

func (r *campaignRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*Campaign, error) {
	row, err := r.q.GetCampaignByID(ctx, db.GetCampaignByIDParams{OrgID: orgID, ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return toCampaign(row), nil
}
```

### Response helpers — shared across all handlers AND middleware

```go
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: &ErrorBody{Code: code, Message: message}})
}
```

Middleware MUST use these shared response helpers. Do NOT create duplicate error response functions.

### Router — extracted, with RBAC

```go
func NewRouter(h *Handlers, auth *middleware.Auth) http.Handler {
	r := chi.NewRouter()
	r.Use(otelchi.Middleware("api-dashboard"))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", h.Health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.ValidateJWT)
		r.Route("/campaigns", func(r chi.Router) {
			r.Use(auth.RequireRole("editor", "admin", "owner"))
			r.Post("/", h.Campaign.Create)
			r.Get("/", h.Campaign.List)
			r.Get("/{id}", h.Campaign.GetByID)
		})
	})
	return r
}
```

## Anti-Patterns (FORBIDDEN)

1. **NO global state or init()** — pass dependencies via constructors
2. **NO panics in business logic** — return errors, let handler decide HTTP status
3. **NO raw SQL in handlers or services** — all SQL lives in sqlc query files
4. **NO raw SQL in repository implementations** — repository wraps sqlc-generated `Queries`, never `pool.Query`/`pool.Exec` with SQL strings
5. **NO fmt.Println / log.Println** — use `slog.InfoContext` / `slog.ErrorContext` with typed attributes
6. **NO custom JWT parsing** — use `golang-jwt/jwt/v5` library, validate against BetterAuth JWKS
7. **NO duplicate response helpers** — middleware imports from handler package, one source of truth
8. **NO guessing dependency versions** — run `go mod tidy` after creating go.mod, verify with `go build`

## Tests

Every service method MUST have a table-driven test:

```go
func TestCampaignService_Create(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCampaignRequest
		wantErr bool
	}{
		{name: "valid campaign", req: CreateCampaignRequest{Name: "test", ...}},
		{name: "empty name", req: CreateCampaignRequest{Name: ""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ...
		})
	}
}
```

After generating go.mod — always run `go mod tidy` to resolve dependencies and generate go.sum.