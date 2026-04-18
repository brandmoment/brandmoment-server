# Phase 3: Brand/Campaign Domain + Dashboard Brand Pages

## 1. Context & Problem

### Current system context
- Phase 1+2 are complete: JWT auth, identity (users, org_memberships, org_invites), publisher_apps, api_keys, publisher_rules
- 7 migrations exist (000001–000007); next numbers are 000008 and 000009
- 223 Go tests pass; service layer follows table-driven patterns with mock interfaces and noop OTel
- Dashboard has: auth pages, layout shell (sidebar/topbar/org-switcher), publisher pages (/apps, /apps/:id)
- Router is at `/v1` prefix (not `/api/v1`); all authenticated routes use `auth.ValidateJWT` + per-route `auth.RequireRole`
- Shared response helpers live in `services/api-dashboard/internal/httputil/response.go` (`RespondJSON` / `RespondError`)
- Pagination: `limit/offset` query params, default limit=20, max=100, enforced in service layer

### Problem / user need
Brand organisations have no way to create campaigns, manage their lifecycle, or attach creatives. The dashboard has no brand-facing pages.

### Business goals
- Enable brands to self-serve: create campaigns, set targeting, upload creatives, manage status
- Provide the data foundation that Phase 4 analytics will query

---

## 2. Goals & Non-Goals

### Goals
- `campaigns` and `creatives` tables with migrations
- Full Campaign CRUD (list, create, get, update name/targeting/budget)
- Campaign status management via a dedicated PATCH endpoint with enforced state machine
- Creative metadata registration (file_url stored as placeholder; no actual S3 upload in Phase 3)
- Creative list endpoint per campaign
- Preview URL surfaced from stored `preview_url` field (no signed-URL generation in Phase 3)
- Dashboard: `/campaigns` list page with status filter, `/campaigns/:id` detail page, creative upload dialog, sandboxed preview iframe

### Non-Goals
- Actual S3/R2 file upload (Phase 4)
- Signed preview URL generation (Phase 4)
- Campaign deletion (soft-delete deferred; admin use case)
- Analytics queries against campaigns (Phase 4)
- Creative validation / HTML5 bundle parsing (Phase 4)
- Video or static image creative types (Phase 4+)

---

## 3. User Stories

- As a brand editor I want to create a campaign with name, targeting, budget, and dates so that I can start planning ad delivery.
- As a brand editor I want to transition a campaign from draft to active so that it becomes eligible for matching.
- As a brand editor I want to pause an active campaign so that I can temporarily stop delivery without losing configuration.
- As a brand editor I want to upload a creative to a campaign so that the system has a record of the asset.
- As a brand viewer I want to list my org's campaigns filtered by status so that I can see what is running.
- As a brand viewer I want to view campaign details including targeting and attached creatives so that I can review configuration.
- As a brand viewer I want to preview a creative in a sandboxed iframe so that I can verify the content before activation.

---

## 4. Scope

### In Scope
- SQL migrations 000008 (campaigns) and 000009 (creatives)
- sqlc query files for both entities
- Go model, repository, service, handler, and tests for both entities
- Router registration under `/v1`
- Dashboard pages: `/campaigns`, `/campaigns/[id]`
- Dashboard components: `CampaignsList`, `CampaignDetail`, `CampaignStatusBadge`, `CreateCampaignDialog`, `CreativeUploadDialog`, `CreativePreview`
- Dashboard hooks: `useCampaigns`, `useCampaign`, `useCreateCampaign`, `useUpdateCampaignStatus`, `useCreateCreative`, `useCreatives`
- TypeScript types for Campaign and Creative

### Out of Scope
- Actual binary file upload to S3/R2
- Signed preview URL generation
- Campaign delete endpoint
- Admin cross-org campaign view (Phase 5)
- E2E / Playwright tests (separate agent)

---

## 5. Functional Requirements

- **FR-1**: A campaign belongs to exactly one org (`org_id`). All campaign queries MUST filter by `org_id` extracted from JWT context.
- **FR-2**: Campaign status transitions follow a strict state machine. Any invalid transition returns `400 INVALID_TRANSITION`.
- **FR-3**: `budget_cents` must be a non-negative integer. `currency` defaults to `"USD"`. Both are nullable (brand may not set budget in Phase 3).
- **FR-4**: `targeting` is a JSONB column with a defined schema; invalid targeting structure returns `400 INVALID_INPUT`.
- **FR-5**: `start_date` and `end_date` are optional. If both are set, `end_date` must be after `start_date`.
- **FR-6**: A creative belongs to one campaign (`campaign_id`) and one org (`org_id`). All creative queries filter by both.
- **FR-7**: `file_url` on creative is stored as a string placeholder. The handler accepts it in the request body; no upload processing occurs in Phase 3.
- **FR-8**: `preview_url` on creative is stored as a string placeholder equal to `file_url` in Phase 3.
- **FR-9**: Creative `type` must be one of `html5`, `image`, `video` (CHECK constraint; only `html5` surfaced in UI in Phase 3, but the column is not restricted at the API layer).
- **FR-10**: `file_size_bytes` must be a positive integer when provided.
- **FR-11**: List campaigns supports `status` query param for filtering (single value; e.g., `?status=active`).
- **FR-12**: Pagination on list endpoints: `limit` (default 20, max 100), `offset` (default 0). Same as Phase 2.
- **FR-13**: Viewer role can only call GET endpoints. Editor+ can POST creatives and create/update campaigns. Admin+ can change campaign status to `completed` (final state).

Assumption: any role (editor, admin, owner) can transition status, including to `completed`. This matches the product expectation that a brand owner can manually complete a campaign.

---

## 6. API Changes

### Existing Endpoints Affected

- `internal/router/router.go` — add `Campaign` and `Creative` handler fields to `Handlers` struct; register new routes
- `cmd/server/main.go` — wire new repositories, services, and handlers in DI block

### New Endpoints

#### Campaign Endpoints

**POST /v1/campaigns**

Create a campaign.

Request:
```json
{
  "name": "Summer 2026",
  "targeting": {
    "geo": ["US", "CA"],
    "platforms": ["ios", "android"],
    "age_range": {"min": 18, "max": 35},
    "interests": ["sports", "music"]
  },
  "budget_cents": 500000,
  "currency": "USD",
  "start_date": "2026-06-01",
  "end_date": "2026-08-31"
}
```

Response `201 Created`:
```json
{
  "data": {
    "id": "uuid",
    "org_id": "uuid",
    "name": "Summer 2026",
    "status": "draft",
    "targeting": { ... },
    "budget_cents": 500000,
    "currency": "USD",
    "start_date": "2026-06-01",
    "end_date": "2026-08-31",
    "created_at": "2026-04-18T00:00:00Z",
    "updated_at": "2026-04-18T00:00:00Z"
  }
}
```

Errors:
- `400 INVALID_BODY` — malformed JSON
- `400 INVALID_INPUT` — name empty, name > 200 chars, dates invalid, budget negative

---

**GET /v1/campaigns**

List campaigns for the active org. Supports `status` and pagination query params.

Query params: `status` (optional, one of draft/active/paused/completed), `limit` (int, default 20, max 100), `offset` (int, default 0)

Response `200 OK`:
```json
{
  "data": {
    "items": [ { /* Campaign */ } ],
    "total": 42,
    "limit": 20,
    "offset": 0
  }
}
```

---

**GET /v1/campaigns/{id}**

Get a single campaign by ID within the active org.

Response `200 OK`: `{"data": { /* Campaign */ }}`

Errors:
- `400 INVALID_ID` — non-UUID path param
- `404 NOT_FOUND` — campaign not found or belongs to different org

---

**PUT /v1/campaigns/{id}**

Update campaign name, targeting, budget, currency, start_date, end_date. Does NOT change status.

Request: same shape as POST (all fields optional, use pointer types in Go).

Response `200 OK`: `{"data": { /* Campaign */ }}`

Errors: same as POST plus `404 NOT_FOUND`

---

**PATCH /v1/campaigns/{id}/status**

Transition campaign status.

Request:
```json
{ "status": "active" }
```

Response `200 OK`: `{"data": { /* Campaign */ }}`

Errors:
- `400 INVALID_BODY`
- `400 INVALID_TRANSITION` — transition not allowed by state machine
- `404 NOT_FOUND`

---

#### Creative Endpoints

**POST /v1/campaigns/{id}/creatives**

Register a creative record (no file upload in Phase 3).

Request:
```json
{
  "name": "Banner 320x50",
  "type": "html5",
  "file_url": "s3://brandmoment-creatives/org-uuid/campaign-uuid/banner.zip",
  "file_size_bytes": 204800,
  "preview_url": "s3://brandmoment-creatives/org-uuid/campaign-uuid/banner.zip"
}
```

Response `201 Created`:
```json
{
  "data": {
    "id": "uuid",
    "org_id": "uuid",
    "campaign_id": "uuid",
    "name": "Banner 320x50",
    "type": "html5",
    "file_url": "...",
    "file_size_bytes": 204800,
    "preview_url": "...",
    "is_active": true,
    "created_at": "2026-04-18T00:00:00Z",
    "updated_at": "2026-04-18T00:00:00Z"
  }
}
```

Errors:
- `400 INVALID_BODY`
- `400 INVALID_INPUT` — name empty, type invalid, file_size_bytes <= 0
- `404 NOT_FOUND` — campaign not found or not in org

---

**GET /v1/campaigns/{id}/creatives**

List creatives for a campaign.

Response `200 OK`:
```json
{
  "data": {
    "items": [ { /* Creative */ } ],
    "total": 3
  }
}
```

No pagination required in Phase 3 (assumption: creative count per campaign is small; max 50). Service enforces a hard cap of 50 results.

---

### Error Code Reference (additions)

| Trigger | HTTP | Code |
|---|---|---|
| Invalid status transition | 400 | `INVALID_TRANSITION` |
| Unknown `status` filter value | 400 | `INVALID_INPUT` |

---

## 7. Data Model

### campaigns table (migration 000008)

```sql
-- 000008_create_campaigns.up.sql
CREATE TABLE IF NOT EXISTS campaigns (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    status       TEXT        NOT NULL DEFAULT 'draft'
                             CHECK (status IN ('draft', 'active', 'paused', 'completed')),
    targeting    JSONB       NOT NULL DEFAULT '{}',
    budget_cents BIGINT      CHECK (budget_cents >= 0),
    currency     TEXT        NOT NULL DEFAULT 'USD',
    start_date   DATE,
    end_date     DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_campaigns_org_id          ON campaigns (org_id);
CREATE INDEX idx_campaigns_org_id_status   ON campaigns (org_id, status);
CREATE INDEX idx_campaigns_org_id_created  ON campaigns (org_id, created_at DESC);
```

```sql
-- 000008_create_campaigns.down.sql
DROP TABLE IF EXISTS campaigns;
```

### creatives table (migration 000009)

```sql
-- 000009_create_creatives.up.sql
CREATE TABLE IF NOT EXISTS creatives (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    campaign_id     UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    type            TEXT        NOT NULL CHECK (type IN ('html5', 'image', 'video')),
    file_url        TEXT        NOT NULL,
    file_size_bytes BIGINT      CHECK (file_size_bytes > 0),
    preview_url     TEXT,
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_creatives_campaign_id          ON creatives (campaign_id);
CREATE INDEX idx_creatives_org_id              ON creatives (org_id);
CREATE INDEX idx_creatives_campaign_id_active  ON creatives (campaign_id, is_active);
```

```sql
-- 000009_create_creatives.down.sql
DROP TABLE IF EXISTS creatives;
```

### Migration plan

Migration numbering continues from 000007:
- `000008_create_campaigns` — no external dependencies beyond `organizations`
- `000009_create_creatives` — depends on `campaigns` (FK); must run after 000008

No existing tables are altered.

### Multi-tenancy

- `campaigns.org_id` — ALWAYS in `WHERE org_id = $1` for all campaign queries
- `creatives.org_id` — ALWAYS in `WHERE org_id = $1 AND campaign_id = $2` for all creative queries
- `campaign_id` from URL path must be re-validated against `org_id` before operating on creatives (handler fetches campaign first, or repository JOIN enforces it)

---

## 8. sqlc Queries

File: `packages/shared-domain/queries/campaign.sql`

```sql
-- name: GetCampaignByID :one
SELECT id, org_id, name, status, targeting, budget_cents, currency,
       start_date, end_date, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id AND id = @id;

-- name: ListCampaignsByOrg :many
SELECT id, org_id, name, status, targeting, budget_cents, currency,
       start_date, end_date, created_at, updated_at
FROM campaigns
WHERE org_id = @org_id
  AND (@status_filter::TEXT IS NULL OR status = @status_filter)
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: CountCampaignsByOrg :one
SELECT COUNT(*) FROM campaigns
WHERE org_id = @org_id
  AND (@status_filter::TEXT IS NULL OR status = @status_filter);

-- name: InsertCampaign :one
INSERT INTO campaigns (id, org_id, name, status, targeting, budget_cents, currency,
                       start_date, end_date, created_at, updated_at)
VALUES (@id, @org_id, @name, @status, @targeting, @budget_cents, @currency,
        @start_date, @end_date, @created_at, @updated_at)
RETURNING *;

-- name: UpdateCampaign :one
UPDATE campaigns
SET name = @name, targeting = @targeting, budget_cents = @budget_cents,
    currency = @currency, start_date = @start_date, end_date = @end_date,
    updated_at = @updated_at
WHERE org_id = @org_id AND id = @id
RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE campaigns
SET status = @status, updated_at = @updated_at
WHERE org_id = @org_id AND id = @id
RETURNING *;
```

File: `packages/shared-domain/queries/creative.sql`

```sql
-- name: GetCreativeByID :one
SELECT id, org_id, campaign_id, name, type, file_url, file_size_bytes,
       preview_url, is_active, created_at, updated_at
FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id AND id = @id;

-- name: ListCreativesByCampaign :many
SELECT id, org_id, campaign_id, name, type, file_url, file_size_bytes,
       preview_url, is_active, created_at, updated_at
FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id
ORDER BY created_at ASC
LIMIT 50;

-- name: CountCreativesByCampaign :one
SELECT COUNT(*) FROM creatives
WHERE org_id = @org_id AND campaign_id = @campaign_id;

-- name: InsertCreative :one
INSERT INTO creatives (id, org_id, campaign_id, name, type, file_url,
                       file_size_bytes, preview_url, is_active, created_at, updated_at)
VALUES (@id, @org_id, @campaign_id, @name, @type, @file_url,
        @file_size_bytes, @preview_url, @is_active, @created_at, @updated_at)
RETURNING *;
```

After writing query files: run `sqlc generate` in `packages/shared-domain/`.

---

## 9. Go Layer — Files to Create / Modify

### New files (follow the new-entity checklist order)

#### Models

`services/api-dashboard/internal/model/campaign.go`
- `CampaignStatus` type (string alias)
- Constants: `StatusDraft`, `StatusActive`, `StatusPaused`, `StatusCompleted`
- `CampaignTargeting` struct: `Geo []string`, `Platforms []string`, `AgeRange *AgeRange`, `Interests []string`
- `AgeRange` struct: `Min int`, `Max int`
- `Campaign` struct: `ID`, `OrgID`, `Name`, `Status CampaignStatus`, `Targeting CampaignTargeting`, `BudgetCents *int64`, `Currency string`, `StartDate *time.Time`, `EndDate *time.Time`, `CreatedAt`, `UpdatedAt`
- `ValidTransitions` map — defines allowed next states per current state (used by service)

`services/api-dashboard/internal/model/creative.go`
- `CreativeType` string alias; constants `TypeHTML5`, `TypeImage`, `TypeVideo`
- `Creative` struct: `ID`, `OrgID`, `CampaignID`, `Name`, `Type CreativeType`, `FileURL`, `FileSizeBytes *int64`, `PreviewURL *string`, `IsActive bool`, `CreatedAt`, `UpdatedAt`

#### Repositories

`services/api-dashboard/internal/repository/campaign.go`
```
CampaignRepository interface:
  Insert(ctx, *model.Campaign) (*model.Campaign, error)
  GetByID(ctx, orgID, id uuid.UUID) (*model.Campaign, error)
  ListByOrg(ctx, orgID uuid.UUID, statusFilter *string, limit, offset int32) ([]model.Campaign, int64, error)
  Update(ctx, *model.Campaign) (*model.Campaign, error)
  UpdateStatus(ctx, orgID, id uuid.UUID, status model.CampaignStatus, updatedAt time.Time) (*model.Campaign, error)
```

`services/api-dashboard/internal/repository/creative.go`
```
CreativeRepository interface:
  Insert(ctx, *model.Creative) (*model.Creative, error)
  ListByCampaign(ctx, orgID, campaignID uuid.UUID) ([]model.Creative, int64, error)
```

#### Services

`services/api-dashboard/internal/service/campaign.go`
- `CampaignService` struct with `repo CampaignRepository`, `tracer trace.Tracer`
- Request types: `CreateCampaignRequest`, `UpdateCampaignRequest`, `UpdateCampaignStatusRequest`
- `CampaignListResult` struct: `Items []model.Campaign`, `Total int64`, `Limit int32`, `Offset int32`
- Methods: `Create`, `GetByID`, `List`, `Update`, `UpdateStatus`
- Status machine enforcement in `UpdateStatus`: lookup `model.ValidTransitions[current]`, reject if target not in slice; return `model.ErrInvalidInput` with message including `INVALID_TRANSITION` code hint

`services/api-dashboard/internal/service/creative.go`
- `CreativeService` struct with `campaignRepo CampaignRepository`, `creativeRepo CreativeRepository`, `tracer trace.Tracer`
- Request types: `CreateCreativeRequest`
- `CreativeListResult` struct: `Items []model.Creative`, `Total int64`
- Methods: `Create` (validates campaign exists in org first), `ListByCampaign`

#### Handlers

`services/api-dashboard/internal/handler/campaign.go`
- `CampaignHandler` with methods: `Create`, `List`, `GetByID`, `Update`, `UpdateStatus`
- `List` handler reads `status` query param; passes as `*string` to service (nil = no filter, non-nil = filter)
- `UpdateStatus` handler reads `{"status": "..."}` body

`services/api-dashboard/internal/handler/creative.go`
- `CreativeHandler` with methods: `Create`, `List`
- Both methods extract `campaignID` from chi URL param `id`

#### Tests

`services/api-dashboard/internal/service/campaign_test.go`
Table-driven tests for:
- `TestCampaignService_Create` — valid, empty name, name too long, end before start, negative budget
- `TestCampaignService_UpdateStatus` — each valid transition, each invalid transition, not found
- `TestCampaignService_List` — no filter, status filter, pagination clamping
- `TestCampaignService_Update` — partial update (only name), all fields

`services/api-dashboard/internal/service/creative_test.go`
Table-driven tests for:
- `TestCreativeService_Create` — valid, campaign not found, empty name, invalid type, negative file_size
- `TestCreativeService_ListByCampaign` — found, campaign not found

#### Router modification

`services/api-dashboard/internal/router/router.go`

Add to `Handlers` struct:
```go
Campaign  *handler.CampaignHandler
Creative  *handler.CreativeHandler
```

Register routes:
```go
r.Route("/campaigns", func(r chi.Router) {
    r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.Campaign.List)
    r.With(auth.RequireRole("editor", "admin", "owner")).Post("/", h.Campaign.Create)

    r.Route("/{id}", func(r chi.Router) {
        r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.Campaign.GetByID)
        r.With(auth.RequireRole("editor", "admin", "owner")).Put("/", h.Campaign.Update)
        r.With(auth.RequireRole("editor", "admin", "owner")).Patch("/status", h.Campaign.UpdateStatus)

        r.Route("/creatives", func(r chi.Router) {
            r.With(auth.RequireRole("viewer", "editor", "admin", "owner")).Get("/", h.Creative.List)
            r.With(auth.RequireRole("editor", "admin", "owner")).Post("/", h.Creative.Create)
        })
    })
})
```

#### DI wiring

`services/api-dashboard/cmd/server/main.go` — add:
```go
campaignRepo    := repository.NewCampaignRepository(pool)
campaignSvc     := service.NewCampaignService(campaignRepo, tp)
campaignHandler := handler.NewCampaignHandler(campaignSvc)

creativeRepo    := repository.NewCreativeRepository(pool)
creativeSvc     := service.NewCreativeService(campaignRepo, creativeRepo, tp)
creativeHandler := handler.NewCreativeHandler(creativeSvc)
```

---

## 10. Campaign Status State Machine

```
         ┌─────────────────────────────────────┐
         │                                     │
    [draft] ──→ [active] ──→ [paused] ──→ [completed]
                    │                ↑
                    └────────────────┘
                    (active → paused → active allowed)
```

Valid transitions (enforced in `CampaignService.UpdateStatus`):

| Current status | Allowed next statuses |
|---|---|
| `draft` | `active` |
| `active` | `paused`, `completed` |
| `paused` | `active`, `completed` |
| `completed` | (none — terminal) |

Invalid transition returns `fmt.Errorf("%w: cannot transition from %s to %s (INVALID_TRANSITION)", model.ErrInvalidInput, current, target)`.

The handler maps `ErrInvalidInput` to HTTP 400 with code `INVALID_INPUT`. The error message string carries `INVALID_TRANSITION` as a readable indicator; no separate error code type is introduced (consistent with existing pattern).

---

## 11. TypeScript / Dashboard Layer — Files to Create / Modify

### Types

`apps/dashboard/types/campaign.ts`
```ts
export type CampaignStatus = "draft" | "active" | "paused" | "completed";
export type CreativeType = "html5" | "image" | "video";

export interface CampaignTargeting {
  geo: string[];
  platforms: string[];
  age_range?: { min: number; max: number };
  interests: string[];
}

export interface Campaign {
  id: string;
  org_id: string;
  name: string;
  status: CampaignStatus;
  targeting: CampaignTargeting;
  budget_cents: number | null;
  currency: string;
  start_date: string | null;
  end_date: string | null;
  created_at: string;
  updated_at: string;
}

export interface Creative {
  id: string;
  org_id: string;
  campaign_id: string;
  name: string;
  type: CreativeType;
  file_url: string;
  file_size_bytes: number | null;
  preview_url: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
```

### Hooks

`apps/dashboard/hooks/useCampaigns.ts`
- TanStack Query, calls `GET /v1/campaigns` with `limit`, `offset`, optional `status`
- Returns `{ data: { items: Campaign[], total, limit, offset }, isLoading, isError }`

`apps/dashboard/hooks/useCampaign.ts`
- `GET /v1/campaigns/:id`

`apps/dashboard/hooks/useCreateCampaign.ts`
- `useMutation` wrapping `POST /v1/campaigns`; invalidates `["campaigns"]` on success

`apps/dashboard/hooks/useUpdateCampaignStatus.ts`
- `useMutation` wrapping `PATCH /v1/campaigns/:id/status`; invalidates `["campaign", id]` and `["campaigns"]`

`apps/dashboard/hooks/useUpdateCampaign.ts`
- `useMutation` wrapping `PUT /v1/campaigns/:id`; invalidates `["campaign", id]`

`apps/dashboard/hooks/useCreatives.ts`
- `GET /v1/campaigns/:id/creatives`

`apps/dashboard/hooks/useCreateCreative.ts`
- `useMutation` wrapping `POST /v1/campaigns/:id/creatives`; invalidates `["creatives", campaignId]`

### Components

`apps/dashboard/components/campaign/CampaignsList.tsx`
- Client component mirroring `AppsList.tsx` structure
- TanStack Table with columns: Name, Status (badge), Budget, Start Date, End Date, Created
- Toolbar: status filter dropdown (`Select` from shadcn/ui), "New Campaign" button
- Row click navigates to `/campaigns/:id`
- Pagination: same pattern as `AppsList` (page size selector, prev/next)

`apps/dashboard/components/campaign/CampaignStatusBadge.tsx`
- Renders status as `Badge` with variant:
  - `draft` → `outline`
  - `active` → `success`
  - `paused` → `warning`
  - `completed` → `secondary`

`apps/dashboard/components/campaign/CreateCampaignDialog.tsx`
- Dialog with React Hook Form + zod
- Fields: name (required), targeting (geo, platforms, age range, interests — multi-select tags), budget_cents (optional number), currency (default USD), start_date, end_date
- On submit: calls `useCreateCampaign`, closes dialog, navigates to new campaign detail

`apps/dashboard/components/campaign/CampaignDetail.tsx`
- Client component for `/campaigns/:id` page
- Sections:
  1. Header: campaign name, status badge, "Edit" button
  2. Status management: buttons for valid next transitions only (based on current status)
  3. Targeting summary: geo, platforms, age range, interests as read-only chips
  4. Budget + dates row
  5. Creatives section (embedded `CreativesList` + "Upload Creative" button)

`apps/dashboard/components/campaign/CreativesList.tsx`
- Table columns: Name, Type, File Size, Active, Created
- Preview button per row opens `CreativePreview`

`apps/dashboard/components/campaign/CreativeUploadDialog.tsx`
- Dialog with fields: name (text), type (select: html5/image/video), file_url (text input — placeholder in Phase 3; drag-and-drop shell with disabled file input), file_size_bytes (number, optional)
- Calls `useCreateCreative`; invalidates and closes on success
- Phase 3: a notice banner explains "Actual file upload is coming soon. Enter the file path manually."

`apps/dashboard/components/campaign/CreativePreview.tsx`
- `Dialog` containing `<iframe>` with `sandbox="allow-scripts allow-same-origin"` attribute
- `src` set to `creative.preview_url`
- If `preview_url` is null: displays placeholder message "No preview available"
- Dimension selector: dropdown with common sizes (320x50, 300x250, 728x90) to resize iframe

### Pages

`apps/dashboard/app/(dashboard)/campaigns/page.tsx`
```tsx
import { CampaignsList } from "@/components/campaign/CampaignsList";
export default function CampaignsPage() {
  return <CampaignsList />;
}
```

`apps/dashboard/app/(dashboard)/campaigns/[id]/page.tsx`
```tsx
import { CampaignDetail } from "@/components/campaign/CampaignDetail";
export default function CampaignDetailPage({ params }: { params: { id: string } }) {
  return <CampaignDetail campaignId={params.id} />;
}
```

### Sidebar update

`apps/dashboard/components/layout/Sidebar.tsx` (or equivalent sidebar nav file) — add "Campaigns" nav item for brand org type, pointing to `/campaigns`. Icon: `Megaphone` from lucide-react.

---

## 12. State & Flows

### Happy path: create and activate campaign

1. Brand editor opens `/campaigns`, clicks "New Campaign"
2. `CreateCampaignDialog` submits to `POST /v1/campaigns` → campaign created with status `draft`
3. Dialog closes; user navigated to `/campaigns/:id`
4. User clicks "Activate" → `PATCH /v1/campaigns/:id/status` with `{"status": "active"}`
5. Campaign status badge updates to "Active"; "Activate" button disappears, "Pause" and "Complete" appear

### Happy path: upload creative

1. User on `/campaigns/:id` clicks "Upload Creative"
2. `CreativeUploadDialog` opens; user fills name, selects type, enters file_url placeholder
3. Submits `POST /v1/campaigns/:id/creatives` → creative record created
4. Dialog closes; `CreativesList` refetches and shows new row
5. User clicks "Preview" → `CreativePreview` dialog opens with sandboxed iframe

### Edge cases

- Attempt to activate a `completed` campaign: API returns `400 INVALID_INPUT` with message containing `INVALID_TRANSITION`; dashboard shows toast error "Cannot transition from completed to active"
- Creative on a campaign belonging to a different org: `GetCampaignByID(orgID, campaignID)` returns `ErrNotFound` → 404
- List with invalid `status` filter value: service validates against known values, returns `ErrInvalidInput` → 400
- `end_date` before `start_date`: service returns `ErrInvalidInput` → 400

### Error handling

- API errors → sonner toast with `error.message` from response body
- Network errors → generic toast "Something went wrong. Please try again."
- 404 on detail page → redirect to `/campaigns` with toast "Campaign not found"

---

## 13. Non-Functional Requirements

### Performance
- `GET /v1/campaigns` with status filter: index `idx_campaigns_org_id_status` covers the query; target p99 < 50ms for orgs with up to 10,000 campaigns
- `GET /v1/campaigns/:id/creatives`: index `idx_creatives_campaign_id` covers lookup; hard cap of 50 rows; target p99 < 20ms

### Security
- All campaign and creative queries filter by `org_id` from JWT context — never from request body or URL param alone
- `org_id` in creative creation is taken from JWT context, not from request body
- `campaign_id` ownership is verified by calling `campaignRepo.GetByID(ctx, orgID, campaignID)` before inserting creative — prevents cross-org creative injection
- Creative `preview_url` rendered in `<iframe sandbox="allow-scripts allow-same-origin">` — disables top navigation, forms, popups
- RBAC: viewer role cannot mutate; editor+ can create/update; all roles can GET

### Observability
- Each service method starts a child OTel span: `"CampaignService.Create"`, `"CampaignService.UpdateStatus"`, etc.
- `span.RecordError(err)` on all non-nil errors
- `slog.InfoContext` at entry of each service method with `org_id`, `id` (where applicable)
- Status transitions logged at INFO with `from_status` and `to_status` attributes

---

## 14. Dependencies

### Services affected
- `services/api-dashboard` — new handlers, services, repositories
- `packages/shared-domain` — new migrations + sqlc query files; regenerate with `sqlc generate`
- `apps/dashboard` — new pages, components, hooks, types

### External APIs
- None in Phase 3 (file upload deferred to Phase 4)

### Infrastructure changes
- None — Postgres, MinIO, OTel, and Jaeger are already running via docker-compose

---

## 15. Testing Strategy

### Unit tests (Go)

All service tests are table-driven with mock repositories using func fields (no testify/gomock). `noop.NewTracerProvider()` for OTel.

`TestCampaignService_Create`:
| Case | Input | Expected |
|---|---|---|
| valid | name + targeting | campaign with status=draft |
| empty name | "" | ErrInvalidInput |
| name too long | 201 chars | ErrInvalidInput |
| end before start | end=yesterday, start=today | ErrInvalidInput |
| negative budget | budget_cents=-1 | ErrInvalidInput |

`TestCampaignService_UpdateStatus`:
| Case | Current | Target | Expected |
|---|---|---|---|
| draft → active | draft | active | updated campaign |
| active → paused | active | paused | updated campaign |
| paused → active | paused | active | updated campaign |
| active → completed | active | completed | updated campaign |
| paused → completed | paused | completed | updated campaign |
| completed → active | completed | active | ErrInvalidInput |
| draft → paused | draft | paused | ErrInvalidInput |
| not found | — | — | ErrNotFound |

`TestCreativeService_Create`:
| Case | Input | Expected |
|---|---|---|
| valid html5 | full request | creative with is_active=true |
| campaign not found | bad campaign_id | ErrNotFound |
| empty name | "" | ErrInvalidInput |
| invalid type | "gif" | ErrInvalidInput |
| negative file_size | -1 | ErrInvalidInput |

### Integration tests
- None defined for Phase 3 (no test infra for DB integration tests established yet)

### Manual verification steps
1. `go build ./...` — must compile cleanly
2. `go vet ./...` — no issues
3. `go test ./...` — 223 existing tests + new tests all pass
4. `sqlc generate` — succeeds without errors
5. Curl smoke tests:
   - `POST /v1/campaigns` → 201
   - `GET /v1/campaigns` → 200 with items array
   - `PATCH /v1/campaigns/:id/status` with `{"status":"active"}` → 200
   - `PATCH /v1/campaigns/:id/status` with `{"status":"draft"}` (from active) → 400
   - `POST /v1/campaigns/:id/creatives` → 201
   - `GET /v1/campaigns/:id/creatives` → 200

---

## 16. Acceptance Criteria

- **AC-1**: `POST /v1/campaigns` creates a campaign with status `draft`; response includes all fields; org_id is from JWT context, not request body.
- **AC-2**: `GET /v1/campaigns?status=active` returns only active campaigns for the org; omitting `status` returns all.
- **AC-3**: `GET /v1/campaigns?limit=5&offset=0` returns at most 5 items; `total` reflects the full count.
- **AC-4**: `PATCH /v1/campaigns/:id/status {"status":"active"}` on a draft campaign returns 200 with status=active.
- **AC-5**: `PATCH /v1/campaigns/:id/status {"status":"draft"}` on an active campaign returns 400.
- **AC-6**: `PATCH /v1/campaigns/:id/status` on a completed campaign with any target returns 400 for all targets.
- **AC-7**: `POST /v1/campaigns/:id/creatives` on a campaign belonging to a different org returns 404.
- **AC-8**: `GET /v1/campaigns/:id/creatives` returns all creatives for the campaign, max 50 items.
- **AC-9**: Dashboard `/campaigns` page renders the campaign list with a status filter dropdown; selecting "Active" re-fetches with `?status=active`.
- **AC-10**: Dashboard `/campaigns/:id` page shows campaign details and a "Creatives" section; status transition buttons reflect only valid next states.
- **AC-11**: `CreativePreview` renders an `<iframe>` with `sandbox="allow-scripts allow-same-origin"` and no other sandbox values; `src` equals `creative.preview_url`.
- **AC-12**: All new Go service methods produce an OTel child span; errors are recorded with `span.RecordError`.
- **AC-13**: A viewer-role JWT cannot call `POST /v1/campaigns`; the response is 403.
- **AC-14**: `go build ./...` and `go test ./...` pass with no errors after implementation.

---

## 17. Implementation Order

```
1. SQL (sql-builder)
   a. 000008_create_campaigns.up.sql + .down.sql
   b. 000009_create_creatives.up.sql + .down.sql
   c. packages/shared-domain/queries/campaign.sql
   d. packages/shared-domain/queries/creative.sql
   e. Run: sqlc generate

2. Go backend (go-builder) — after sql-builder completes
   a. model/campaign.go, model/creative.go
   b. repository/campaign.go, repository/creative.go
   c. service/campaign.go, service/creative.go
   d. service/campaign_test.go, service/creative_test.go
   e. handler/campaign.go, handler/creative.go
   f. router/router.go (add Campaign + Creative to Handlers, register routes)
   g. cmd/server/main.go (DI wiring)

3. TypeScript (ts-builder) — parallel with go-builder after sql-builder completes
   a. types/campaign.ts
   b. hooks/ (7 hook files)
   c. components/campaign/ (6 component files)
   d. app/(dashboard)/campaigns/page.tsx
   e. app/(dashboard)/campaigns/[id]/page.tsx
   f. Sidebar: add Campaigns nav item for brand org type

4. Validate (test-runner)
   a. go build ./...
   b. go vet ./...
   c. go test ./...
   d. pnpm typecheck (apps/dashboard)
   e. pnpm lint (apps/dashboard)
```

---

## 18. Risks & Open Questions

### Risks

- **sqlc nullable date columns**: `start_date DATE` and `end_date DATE` are nullable. sqlc with pgx/v5 generates `pgtype.Date` for nullable DATE columns. The repository must convert `pgtype.Date` ↔ `*time.Time` correctly. Verify generated types before writing repository implementation.
- **Targeting JSONB scan**: sqlc generates `[]byte` for JSONB columns. The repository must `json.Unmarshal` into `model.CampaignTargeting` on read and `json.Marshal` on write. Add helper functions `toCampaign(row)` and `fromCampaign(c)` to the repository file.
- **Status filter as nullable param in sqlc**: the `@status_filter::TEXT IS NULL OR status = @status_filter` pattern requires careful pgx parameter passing. Test with `sqlc generate` to confirm correct nullable TEXT handling; may need `pgtype.Text` as the param type.
- **Badge variant `success` and `warning`**: the publisher AppsList uses `variant="success"` on Badge. Verify this variant exists in the project's shadcn/ui configuration before using it in `CampaignStatusBadge`; if not, add it or use a className override.

### Open Questions

None — all design decisions were made per the spec brief. The following are recorded as explicit assumptions:
- A-1: File upload integration is Phase 4; Phase 3 stores `file_url` as a plain string from the request body.
- A-2: Preview URL in Phase 3 equals `file_url`; no signed URL generation.
- A-3: All roles (editor, admin, owner) can transition campaign status including to `completed`. Viewer cannot.
- A-4: Creative list has no pagination; hard cap of 50 rows in Phase 3.
- A-5: The `status` filter in list accepts exactly one value (not a multi-value filter).
