# Go Implementation — Phase 3 Campaign/Creative

Agent: go-builder
Stage: Implement (Go backend)
Date: 2026-04-18

## Result

All files created and verified. Build and tests pass cleanly.

```
go build ./services/api-dashboard/...  → OK
go vet ./services/api-dashboard/...    → OK
go test ./services/api-dashboard/...   → 264 passed (223 pre-existing + 41 new)
```

---

## Files Created

### Models

`services/api-dashboard/internal/model/campaign.go`
- `CampaignStatus` string alias; constants `StatusDraft`, `StatusActive`, `StatusPaused`, `StatusCompleted`
- `AgeRange` struct, `CampaignTargeting` struct
- `Campaign` struct with all fields (BudgetCents *int64, StartDate/EndDate *time.Time)
- `ValidTransitions` map — state machine definition

`services/api-dashboard/internal/model/creative.go`
- `CreativeType` string alias; constants `TypeHTML5`, `TypeImage`, `TypeVideo`
- `Creative` struct with FileSizeBytes *int64, PreviewURL *string

### Repositories

`services/api-dashboard/internal/repository/campaign.go`
- `CampaignRepository` interface: Insert, GetByID, ListByOrg, Update, UpdateStatus
- `campaignRepo` struct wrapping `db.Queries`
- `toCampaign(row db.Campaign) (*model.Campaign, error)` — JSON unmarshal of targeting, pgtype.Date→*time.Time, pgtype.Int8→*int64
- `int64ToPgtypeInt8(*int64) pgtype.Int8` and `timeToPgtypeDate(*time.Time) pgtype.Date` helpers

`services/api-dashboard/internal/repository/creative.go`
- `CreativeRepository` interface: Insert, GetByID, ListByCampaign
- `creativeRepo` struct
- `toCreative(row db.Creative) *model.Creative` — pgtype.Int8→*int64, pgtype.Text→*string
- `stringToPgtypeText(*string) pgtype.Text` helper

### Services

`services/api-dashboard/internal/service/campaign.go`
- Request types: `CreateCampaignRequest`, `UpdateCampaignRequest`, `UpdateCampaignStatusRequest`
- `CampaignListResult` struct
- `Create` — validates name (empty, >200), budget (negative), dates (format, order); defaults currency to USD; sets status=draft
- `GetByID` — delegates to repo with org_id isolation
- `List` — validates statusFilter against known values; clamps limit (default 20, max 100)
- `Update` — fetch existing, apply partial update, re-validate dates
- `UpdateStatus` — fetches current, looks up `model.ValidTransitions`, rejects invalid; logs from_status/to_status

`services/api-dashboard/internal/service/creative.go`
- `CreateCreativeRequest`, `CreativeListResult`
- `Create` — validates name, type (html5/image/video), file_size_bytes (>0); calls `campaignRepo.GetByID` first to verify campaign ownership
- `GetByID`, `ListByCampaign` — both verify campaign ownership via campaignRepo

### Handlers

`services/api-dashboard/internal/handler/campaign.go`
- `CampaignHandler`: Create, List (reads `status` query param as `*string`), GetByID, Update, UpdateStatus (PATCH /status)
- All use `httputil.RespondJSON` / `httputil.RespondError`; `handleServiceError` from organization.go

`services/api-dashboard/internal/handler/creative.go`
- `CreativeHandler`: Create, List — both extract `campaignID` from chi URLParam `"id"`

### Tests

`services/api-dashboard/internal/service/campaign_test.go`
- `mockCampaignRepo` with function fields; compile-time interface check
- `TestCampaignService_Create` — 5 cases: valid, empty name, name too long, end before start, negative budget
- `TestCampaignService_UpdateStatus` — 8 cases: all 5 valid transitions, 2 invalid transitions, not found
- `TestCampaignService_List` — 5 cases: no filter, status filter, invalid filter, default limit, clamped limit
- `TestCampaignService_Update` — 7 cases: name only, all fields, empty name, long name, negative budget, end before start, not found

`services/api-dashboard/internal/service/creative_test.go`
- `mockCreativeRepo` with function fields; compile-time interface check
- `TestCreativeService_Create` — 7 cases: valid html5, campaign not found, empty name, name too long, invalid type, negative file_size, zero file_size
- `TestCreativeService_ListByCampaign` — 3 cases: returns creatives, empty list, campaign not found

### Router (modified)

`services/api-dashboard/internal/router/router.go`
- Added `Campaign *handler.CampaignHandler` and `Creative *handler.CreativeHandler` to `Handlers` struct
- Registered `/campaigns` route group:
  - `GET /` (viewer+), `POST /` (editor+)
  - `GET /{id}` (viewer+), `PUT /{id}` (editor+), `PATCH /{id}/status` (editor+)
  - `GET /{id}/creatives` (viewer+), `POST /{id}/creatives` (editor+)

### Main (modified)

`services/api-dashboard/cmd/server/main.go`
- Added DI blocks for campaign and creative repos, services, handlers
- Creative service receives `campaignRepo` as first argument (for ownership checks)
- Wired `Campaign` and `Creative` into `router.Handlers`

---

## pgtype Conversions Implemented

| Field | DB type | Domain type | Conversion |
|---|---|---|---|
| targeting | `[]byte` | `model.CampaignTargeting` | json.Marshal/Unmarshal in toCampaign |
| budget_cents | `pgtype.Int8` | `*int64` | `int64ToPgtypeInt8` / `Valid` check |
| start_date, end_date | `pgtype.Date` | `*time.Time` | `timeToPgtypeDate` / `Valid` check |
| file_size_bytes | `pgtype.Int8` | `*int64` | same as budget_cents |
| preview_url | `pgtype.Text` | `*string` | `stringToPgtypeText` / `Valid` check |
| status_filter | `pgtype.Text` | `*string` | `pgtype.Text{Valid: false}` = no filter |

---

## Layer Rules Verified

- Handler → Service only (never repository, never SQL)
- Service → Repository only (never handler)
- Repository → `db.Queries` only (no raw SQL)
- `org_id` always from JWT context (`middleware.OrgIDFromContext`) — never from request body
- Every campaign query includes `WHERE org_id = $1`
- Every creative query includes `WHERE org_id = $1 AND campaign_id = $2`
- Creative creation verifies campaign ownership via `campaignRepo.GetByID` before inserting

---

## Multi-Tenancy Verification

- `CampaignService.Create` — org_id from parameter (set by handler from JWT context)
- `CampaignService.UpdateStatus` — fetches with org_id; UpdateStatus also scoped by org_id
- `CreativeService.Create` — org_id from parameter; campaignRepo.GetByID(orgID, campaignID) prevents cross-org injection
- `CreativeService.ListByCampaign` — campaign ownership verified before listing

---

## Status State Machine

Implemented in `CampaignService.UpdateStatus`:
1. Fetch current campaign (scoped by org_id)
2. Look up `model.ValidTransitions[current.Status]`
3. If target not in allowed list → `fmt.Errorf("%w: cannot transition from %s to %s (INVALID_TRANSITION)", model.ErrInvalidInput, ...)`
4. Handler maps `ErrInvalidInput` → HTTP 400 `INVALID_INPUT`

Transitions tested: draft→active, active→paused, paused→active, active→completed, paused→completed (all valid), completed→active, draft→paused (invalid).

---

## Next Steps

- TypeScript (ts-builder): implement types, hooks, components, pages
- Validate (test-runner): re-run full suite + `pnpm typecheck` + `pnpm lint`
