# TypeScript Implementation — Phase 3 Campaign Pages

Agent: ts-builder
Stage: Implement
Date: 2026-04-18

## Result: COMPLETE

All TypeScript/React artifacts for Phase 3 campaign domain are implemented and pass `pnpm exec tsc --noEmit` (0 errors) and `pnpm exec next lint` (0 warnings, 0 errors).

---

## Files Created / Modified

### Types
- `apps/dashboard/types/campaign.ts` — Campaign, Creative, CampaignStatus, CreativeType, CampaignTargeting, CreateCampaignRequest, UpdateCampaignRequest, UpdateCampaignStatusRequest, CreateCreativeRequest, CampaignListResponse, CreativeListResponse, VALID_TRANSITIONS constant

### API Types stub
- `apps/dashboard/lib/api-types.gen.ts` — Added paths: `/v1/campaigns`, `/v1/campaigns/{id}`, `/v1/campaigns/{id}/status` (PATCH), `/v1/campaigns/{id}/creatives`; added schemas: Campaign, Creative, CreateCampaignRequest, UpdateCampaignRequest, UpdateCampaignStatusRequest, CreateCreativeRequest

### Hooks (7 files)
- `apps/dashboard/hooks/useCampaigns.ts` — list with limit/offset/status filter
- `apps/dashboard/hooks/useCampaign.ts` — single by ID
- `apps/dashboard/hooks/useCreateCampaign.ts` — POST mutation, invalidates `["campaigns"]`
- `apps/dashboard/hooks/useUpdateCampaign.ts` — PUT mutation, invalidates `["campaign", id]` + `["campaigns"]`
- `apps/dashboard/hooks/useUpdateCampaignStatus.ts` — PATCH mutation, invalidates both campaign + campaigns keys
- `apps/dashboard/hooks/useCreatives.ts` — list by campaignId
- `apps/dashboard/hooks/useCreateCreative.ts` — POST mutation, invalidates `["creatives", campaignId]`

### Components (8 files in `components/campaign/`)
- `CampaignStatusBadge.tsx` — maps draft→outline, active→success, paused→warning, completed→secondary
- `CampaignsList.tsx` — table with status filter dropdown, pagination, empty state, row click navigation
- `CreateCampaignDialog.tsx` — react-hook-form + zod, name/budget/currency/dates, navigates to new campaign on success
- `TargetingEditor.tsx` — TagInput for geo/platforms/interests, age range min/max inputs, read-only chip view in non-edit mode
- `CampaignOverviewTab.tsx` — inline edit with save/cancel, status transition buttons (only valid next states), read-only targeting chip display
- `CampaignDetail.tsx` — back link, name + status badge header, Tabs: Overview / Creatives
- `CreativesList.tsx` — table with type badge, file size, active badge, preview button, upload button
- `CreativeUploadDialog.tsx` — name/type/file_url/file_size form, Phase 4 notice banner
- `CreativePreview.tsx` — Dialog with `<iframe sandbox="allow-scripts allow-same-origin">`, dimension selector dropdown (320×50, 300×250, 728×90, 160×600), null preview_url placeholder

### Pages (5 files)
- `app/(dashboard)/campaigns/page.tsx` — server component, renders `<CampaignsList />`
- `app/(dashboard)/campaigns/loading.tsx` — skeleton loading state
- `app/(dashboard)/campaigns/new/page.tsx` — client redirect to `/campaigns` (dialog launched from list)
- `app/(dashboard)/campaigns/[id]/page.tsx` — server component with async params, renders `<CampaignDetail campaignId={id} />`
- `app/(dashboard)/campaigns/[id]/loading.tsx` — skeleton loading state

### Sidebar
- `apps/dashboard/components/Sidebar.tsx` — already had Campaigns in BRAND_NAV with Megaphone icon. No changes needed.

---

## Key Implementation Decisions

### No zod `.transform()` on numeric fields
react-hook-form's `SubmitHandler` generic conflicts with zod schemas that transform string→number at the type level. All numeric conversions (budget_cents, file_size_bytes) are done manually inside `onSubmit` using `parseFloat`/`parseInt`, keeping the zod schema as string-typed to match the form's internal state.

### Status transition display
`VALID_TRANSITIONS` is exported from `types/campaign.ts` and consumed in `CampaignOverviewTab`. Only valid next states render as buttons. A terminal `completed` campaign shows no transition buttons.

### `preview_url` in Phase 3
`CreativeUploadDialog` sets `preview_url = file_url` on submit, matching the spec assumption (A-2). `CreativePreview` renders a null-safe fallback when `preview_url` is null.

### Targeting JSONB defaults
When a campaign is created without targeting fields, the API returns `{ geo: [], platforms: [], interests: [] }`. The `TargetingEditor` initialises from these defaults safely.

---

## Validation Results

```
pnpm exec tsc --noEmit   → TypeScript: No errors found
pnpm exec next lint      → ✔ No ESLint warnings or errors
```

Two intermediate TS errors were fixed during implementation:
1. `TS2322` — budget_cents default value typed as string (form internal state)
2. `TS2345` — zod transform removed from schema, manual conversion at submit

---

## Component Hierarchy

```
CampaignsPage (server)
  └── CampaignsList (client)
        ├── useCampaigns (hook)
        ├── CampaignStatusBadge
        └── CreateCampaignDialog (client)
              └── useCreateCampaign (hook)

CampaignDetailPage (server)
  └── CampaignDetail (client)
        ├── useCampaign (hook)
        ├── CampaignStatusBadge
        ├── CampaignOverviewTab (client)
        │     ├── useUpdateCampaignStatus (hook)
        │     ├── useUpdateCampaign (hook)
        │     ├── CampaignStatusBadge
        │     └── TargetingEditor (client)
        └── CreativesList (client)
              ├── useCreatives (hook)
              ├── CreativePreview (client)
              └── CreativeUploadDialog (client)
                    └── useCreateCreative (hook)
```
