# Spec: Creative Update Endpoint

## What to Build
PUT /api/v1/campaigns/{id}/creatives/{creativeId} — update an existing creative.

## Layers

### 1. SQL (packages/shared-domain/queries/creatives.sql)
Add `UpdateCreative` query: update name, type, file_url, file_size_bytes, preview_url, is_active.
Must filter by `id`, `org_id`, `campaign_id`. RETURNING *.

### 2. Repository (services/api-dashboard/internal/repository/creative.go)
Add `Update(ctx, orgID, campaignID, id, params)` method wrapping sqlc `UpdateCreative`.

### 3. Service (services/api-dashboard/internal/service/creative.go)
Add `Update(ctx, orgID, campaignID, id, req)`:
- Validate name (1-200 chars), type (html5|image|video) — same validation as Create
- Call repo.Update
- Return updated creative or ErrNotFound

### 4. Handler (services/api-dashboard/internal/handler/creative.go)
Add `Update(w, r)`:
- Parse campaignID from {id}, creativeID from {creativeId}
- Decode JSON body
- Call service.Update
- Respond 200 with updated creative

### 5. Router (services/api-dashboard/internal/router/router.go)
Add `r.Put("/{creativeId}", h.Creative.Update)` in creatives group with editor+ RBAC.

## Acceptance Criteria
- PUT returns 200 with updated creative
- 404 if creative not found or wrong org/campaign
- 400 if invalid input
- RBAC: editor+ only
- All existing tests still pass
