# BrandMoment Server

Multi-tenant ad network platform. Monorepo: Go backend + Next.js 15 frontend.

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go 1.23, chi router, pgx, sqlc |
| Frontend | Next.js 15 App Router, React 19, TypeScript |
| UI | shadcn/ui, Tailwind v4 |
| Auth | BetterAuth (self-hosted) |
| DB | Postgres 17 (OLTP), S3/R2 Parquet (analytics) |
| Analytics | Rill Developer (internal BI), Recharts (user-facing) |
| Observability | OpenTelemetry, Jaeger (dev), Grafana Cloud (prod) |
| Migrations | golang-migrate |
| API Contract | OpenAPI 3.1 → oapi-codegen (Go) + openapi-typescript (TS) |
| Monorepo | Turborepo, pnpm 9.15, go.work |

## Architecture

```
brandmoment-server/
├── services/
│   ├── api-dashboard/       # Go REST API (chi, CRUD, auth middleware)
│   └── api-sdk/             # Go hot-path API for mobile SDKs (v2)
├── apps/
│   └── dashboard/           # Next.js 15 UI (BetterAuth, Rill embed)
├── packages/
│   ├── shared-domain/       # Go shared models, DB queries (sqlc)
│   └── proto/               # OpenAPI spec (source of truth)
├── infra/
│   ├── docker/              # docker-compose (Postgres, MinIO, Rill, OTel, Jaeger)
│   ├── rill/                # Rill dashboards, models, sources
│   ├── seed/                # Go seed data generator
│   └── migrations/          # SQL migrations (golang-migrate)
└── docs/                    # Submodule: external docs repo
```

## Multi-Tenancy Model

3 org types: **admin**, **publisher**, **brand**. JWT carries org memberships with roles (owner|admin|editor|viewer). All DB queries filtered by `org_id` — row-level isolation.

## Naming Conventions

### Go
- Packages: lowercase, no underscores (`apidashboard`, `shareddomain`)
- Files: `snake_case.go`
- Types/Interfaces: PascalCase with suffix (`CampaignService`, `OrgRepository`, `AppHandler`)
- Constructors: `NewCampaignService(deps)`
- Errors: `ErrNotFound`, `ErrUnauthorized`, `ErrInvalidInput`
- Tests: `*_test.go`, functions `TestCreateCampaign`

### TypeScript / Next.js
- Components: PascalCase files and exports (`PublisherAppsList.tsx`)
- Hooks: camelCase, `use` prefix (`usePublisherApps.ts`)
- Utils: camelCase (`formatDate.ts`)
- Types: PascalCase, no `I` prefix (`Organization`, `Campaign`)
- Constants: `SCREAMING_SNAKE_CASE`

### Database
- Tables: `snake_case`, plural (`organizations`, `campaigns`, `api_keys`)
- PKs: `id UUID`
- FKs: `{entity}_id` (`org_id`, `campaign_id`)
- Timestamps: `created_at`, `updated_at`
- Booleans: `is_active`, `is_revoked`
- JSONB: `targeting`, `config`, `metadata`

### API
- Endpoints: `kebab-case` (`/api/v1/publisher-apps`)
- Versioned: `/api/v1/...`

## Go Service Structure (api-dashboard)

```
services/api-dashboard/
├── cmd/
│   └── server/
│       └── main.go          # Entry point: config, DI, server start
├── internal/
│   ├── config/
│   │   └── config.go        # Env-based config (envconfig)
│   ├── handler/
│   │   ├── campaign.go      # HTTP handlers (request/response)
│   │   └── health.go
│   ├── middleware/
│   │   ├── auth.go          # JWT validation + org extraction
│   │   └── rbac.go          # Role-based access control
│   ├── service/
│   │   └── campaign.go      # Business logic
│   ├── repository/
│   │   └── campaign.go      # DB queries (sqlc-generated)
│   └── model/
│       └── campaign.go      # Domain types
├── go.mod
└── go.sum
```

## Good Code Examples

### 1. Handler with proper error handling and context

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

### 2. Service with dependency injection

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

	campaign := &Campaign{
		ID:    uuid.New(),
		OrgID: orgID,
		Name:  req.Name,
	}

	if err := s.repo.Insert(ctx, campaign); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("insert campaign: %w", err)
	}

	return campaign, nil
}
```

### 3. Repository with sqlc pattern

```go
type CampaignRepository interface {
	Insert(ctx context.Context, c *Campaign) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*Campaign, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]Campaign, error)
}

type campaignRepo struct {
	q *sqlc.Queries
}

func NewCampaignRepository(db *pgxpool.Pool) CampaignRepository {
	return &campaignRepo{q: sqlc.New(db)}
}

func (r *campaignRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*Campaign, error) {
	row, err := r.q.GetCampaignByID(ctx, sqlc.GetCampaignByIDParams{
		OrgID: orgID,
		ID:    id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return toCampaign(row), nil
}
```

### 4. Standardized JSON response

```go
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Error: &ErrorBody{Code: code, Message: message},
	})
}
```

### 5. Router setup with middleware chain

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

### 1. NO global state or init()

```go
// BAD
var db *pgxpool.Pool

func init() {
	db, _ = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
}

// GOOD: pass dependencies via constructors
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
```

### 2. NO panics in business logic

```go
// BAD
campaign, err := repo.GetByID(ctx, id)
if err != nil {
	panic(err)
}

// GOOD: return errors, let handler decide HTTP status
campaign, err := repo.GetByID(ctx, id)
if err != nil {
	return nil, fmt.Errorf("get campaign: %w", err)
}
```

### 3. NO raw SQL strings in handlers or services

```go
// BAD: SQL outside repository layer
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.db.Query(r.Context(), "SELECT * FROM campaigns WHERE org_id = $1", orgID)
}

// GOOD: all SQL in repository (sqlc-generated)
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.service.ListByOrg(ctx, orgID, limit, offset)
}
```

### 4. NO queries without org_id filter

```go
// BAD: leaks data across tenants
func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	return r.q.GetCampaign(ctx, id) // WHERE id = $1 — NO org_id!
}

// GOOD: always filter by org_id
func (r *repo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*Campaign, error) {
	return r.q.GetCampaign(ctx, sqlc.GetCampaignParams{OrgID: orgID, ID: id})
}
```

### 5. NO fmt.Println / log.Println in production code

```go
// BAD
fmt.Println("creating campaign", req.Name)

// GOOD: structured logging with slog
slog.InfoContext(ctx, "creating campaign",
	slog.String("name", req.Name),
	slog.String("org_id", orgID.String()),
)
```

## Typical Go File Template

```go
package handler

import (
	// stdlib
	"context"
	"encoding/json"
	"net/http"

	// third-party
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	// internal
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/middleware"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
)

type CampaignHandler struct {
	service *service.CampaignService
}

func NewCampaignHandler(svc *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{service: svc}
}

func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := middleware.OrgIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "invalid campaign id")
		return
	}

	campaign, err := h.service.GetByID(ctx, orgID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, campaign)
}
```

Import order: stdlib, blank line, third-party, blank line, internal.

## API Response Format

Always wrap in standard envelope:

```json
// Success
{"data": { ... }, "error": null}

// Error
{"data": null, "error": {"code": "CAMPAIGN_NOT_FOUND", "message": "Campaign not found"}}
```

## Dev Commands

```bash
make infra-up       # Start Postgres, MinIO, Rill, OTel, Jaeger
make infra-down     # Stop docker-compose stack
make seed           # Generate 50k session events → Parquet → MinIO
make rill-ui        # Open Rill (http://localhost:9009)
make jaeger-ui      # Open Jaeger (http://localhost:16686)
make minio-ui       # Open MinIO Console (http://localhost:9001)
```
