# TypeScript Implementation: Phase 2 Publisher Pages

Agent: ts-builder
Stage: Implement
Date: 2026-04-18

## Status

COMPLETE — `pnpm exec tsc --noEmit` exits with 0 errors.

## Files Created / Modified

### API Types (extended stub)
- `apps/dashboard/lib/api-types.gen.ts` — added 7 new paths and 8 new schemas for publisher-apps, api-keys, and publisher-rules endpoints

### Domain Types (new)
- `apps/dashboard/types/publisher-app.ts` — PublisherApp, CreatePublisherAppRequest, UpdatePublisherAppRequest, PublisherAppListResponse
- `apps/dashboard/types/api-key.ts` — APIKey, CreateAPIKeyRequest, CreateAPIKeyResponse, APIKeyListResponse
- `apps/dashboard/types/publisher-rule.ts` — RuleType, BlocklistAllowlistConfig, FrequencyCapConfig, GeoFilterConfig, PlatformFilterConfig, PublisherRule, CreatePublisherRuleRequest, UpdatePublisherRuleRequest, PublisherRuleListResponse

### Hooks (new — all use openapi-fetch via OrgContext apiClient)
- `apps/dashboard/hooks/usePublisherApps.ts`
- `apps/dashboard/hooks/usePublisherApp.ts`
- `apps/dashboard/hooks/useCreatePublisherApp.ts`
- `apps/dashboard/hooks/useUpdatePublisherApp.ts`
- `apps/dashboard/hooks/useApiKeys.ts`
- `apps/dashboard/hooks/useCreateApiKey.ts`
- `apps/dashboard/hooks/useRevokeApiKey.ts`
- `apps/dashboard/hooks/usePublisherRules.ts`
- `apps/dashboard/hooks/useCreateRule.ts`
- `apps/dashboard/hooks/useUpdateRule.ts`
- `apps/dashboard/hooks/useDeleteRule.ts`

### UI Primitives (new shadcn-style components)
- `apps/dashboard/components/ui/dialog.tsx` — uses @radix-ui/react-dialog (already in package.json)
- `apps/dashboard/components/ui/select.tsx` — uses @radix-ui/react-select (already in package.json)
- `apps/dashboard/components/ui/tabs.tsx` — plain React state (react-tabs not installed)
- `apps/dashboard/components/ui/badge.tsx` — cva-based, variants: default/secondary/destructive/outline/success/warning
- `apps/dashboard/components/ui/skeleton.tsx` — animate-pulse div
- `apps/dashboard/components/ui/textarea.tsx` — forwarded ref textarea

### Publisher Components (new)
- `apps/dashboard/components/publisher/AppsList.tsx` — paginated table, page size selector (20/50/100), click-to-navigate, empty state
- `apps/dashboard/components/publisher/CreateAppDialog.tsx` — react-hook-form + zod, redirects to /apps/:id on success
- `apps/dashboard/components/publisher/AppDetail.tsx` — tabs (Overview/API Keys/Rules), loading/error states
- `apps/dashboard/components/publisher/AppOverviewTab.tsx` — inline edit form, active/inactive toggle
- `apps/dashboard/components/publisher/APIKeysList.tsx` — key list with prefix display, create/revoke flow
- `apps/dashboard/components/publisher/ApiKeyRevealModal.tsx` — shows plaintext key once, requires checkbox confirmation, blocks dismiss
- `apps/dashboard/components/publisher/RulesList.tsx` — paginated list, activate/deactivate toggle, edit/delete per row
- `apps/dashboard/components/publisher/RuleEditorDialog.tsx` — type selector drives dynamic form fields (blocklist/allowlist/frequency_cap/geo_filter/platform_filter), create + edit modes

### Pages (new)
- `apps/dashboard/app/(dashboard)/apps/page.tsx` — server component, renders AppsList client component
- `apps/dashboard/app/(dashboard)/apps/loading.tsx` — skeleton loading state
- `apps/dashboard/app/(dashboard)/apps/[id]/page.tsx` — async server component, passes id to AppDetail
- `apps/dashboard/app/(dashboard)/apps/[id]/loading.tsx` — skeleton loading state

### Sidebar
- No changes needed — Sidebar.tsx already has `{ label: "Apps", href: "/apps", icon: LayoutGrid }` in PUBLISHER_NAV

## Key Design Decisions

1. **API types**: Extended the stub `api-types.gen.ts` directly since `pnpm codegen` requires the OpenAPI spec to be updated first (done in parallel by sql-builder / go-builder agents). The types match the spec in `01-spec.md` exactly.

2. **Tabs component**: `@radix-ui/react-tabs` is not in package.json. Implemented as a custom React context-based component with identical API surface to shadcn Tabs — no external dep needed.

3. **ApiKeyRevealModal**: Blocks both pointer-down-outside and escape key dismiss. The "Done" button is disabled until the user checks "I have copied this key." This satisfies AC-9 (key shown exactly once).

4. **Mutation query key invalidation**: All mutations invalidate the parent list query key to ensure fresh data after create/update/delete operations.

5. **Error surfacing**: All API errors are extracted from `error.error?.message` and surfaced via sonner toasts. This matches the spec's requirement (Section 9).

## Acceptance Criteria Coverage

- AC-8: AppsList renders a table with Name/Platform/Bundle ID/Status/Created columns; pagination controls present
- AC-9: ApiKeyRevealModal shows full key once with required copy confirmation; subsequent views show key_prefix only (list fetched fresh after modal closes)
- AC-10: RuleEditorDialog renders distinct fields per type — geo_filter shows country code multi-select grid; frequency_cap shows numeric inputs

## TypeScript Verification

```
pnpm exec tsc --noEmit
TypeScript: No errors found
```
