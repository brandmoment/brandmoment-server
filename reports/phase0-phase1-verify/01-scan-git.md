# Git Scan — Verification: Phase 0 + Phase 1 + Phase 2 + Phase 3

Date: 2026-04-18
Branch: feature/test-agents
Base: main
Author of all feature commits: Glavatskikh Denis <glavatskikh@brandmoment.ai>

---

## 1. Timeline — Feature Commits (chronological)

| Commit   | Date       | Message                                                                 |
|----------|------------|-------------------------------------------------------------------------|
| 87c6b55  | 2026-04-18 | feat: implement Phase 0 (Dashboard Foundation) + Phase 1 (Backend Identity) |
| 6f33a3d  | 2026-04-18 | feat: implement Phase 2 Publisher Domain (apps, API keys, rules)       |
| 8f5a708  | 2026-04-18 | feat: implement Phase 3 Brand/Campaign Domain (campaigns, creatives)   |
| 4078fed  | 2026-04-18 | fix: resolve SSR hydration errors in dashboard auth pages              |

Support/tooling commits on branch (not feature code): f69f66c, 5a8aea0, 2c7fb2f, 38195ed, d0448fe, e71e25f.

---

## 2. Scale of Change

234 files changed: 31 253 insertions, 888 deletions.
All changes are on the current branch — nothing has landed to main yet.

---

## 3. SQL Stack — New Migrations

8 new migration pairs (up + down) added in `infra/migrations/`:

| #      | Table                | Key columns                                       |
|--------|----------------------|---------------------------------------------------|
| 000002 | users                | id UUID, email, name                              |
| 000003 | org_memberships      | user_id, org_id, role                             |
| 000004 | org_invites          | org_id, email, token, accepted_at                 |
| 000005 | publisher_apps       | org_id FK, name, bundle_id, platform              |
| 000006 | api_keys             | org_id FK, publisher_app_id FK, key_hash, is_revoked |
| 000007 | publisher_rules      | org_id FK, publisher_app_id FK, type, conditions JSONB |
| 000008 | campaigns            | org_id FK, name, status, budget, targeting JSONB  |
| 000009 | creatives            | org_id FK, campaign_id FK, type, asset_url        |

sqlc-generated Go code updated: `packages/shared-domain/db/` — 8 new `.sql.go` files.
sqlc query files added: `packages/shared-domain/queries/` — 8 new `.sql` files.

---

## 4. Go Backend Stack — New Endpoints

Router: `services/api-dashboard/internal/router/router.go` (modified).
Auth: `services/api-dashboard/internal/middleware/auth.go` (modified — JWKS replacing HMAC).

### Full Endpoint Map (registered routes)

```
GET  /healthz                                             (public)
GET  /v1/me                                               (auth required)

GET  /v1/organizations                                    (viewer+)
POST /v1/organizations                                    (viewer+)
GET  /v1/organizations/{id}                               (viewer+)

POST /v1/orgs/{id}/invites                                (admin+)

GET  /v1/publisher-apps                                   (viewer+)
POST /v1/publisher-apps                                   (editor+)
GET  /v1/publisher-apps/{id}                              (viewer+)
PUT  /v1/publisher-apps/{id}                              (editor+)

GET  /v1/publisher-apps/{id}/api-keys                     (viewer+)
POST /v1/publisher-apps/{id}/api-keys                     (editor+)
DELETE /v1/publisher-apps/{id}/api-keys/{keyId}           (admin+)

GET  /v1/publisher-apps/{id}/rules                        (viewer+)
POST /v1/publisher-apps/{id}/rules                        (editor+)
GET  /v1/publisher-apps/{id}/rules/{ruleId}               (viewer+)
PUT  /v1/publisher-apps/{id}/rules/{ruleId}               (editor+)
DELETE /v1/publisher-apps/{id}/rules/{ruleId}             (admin+)

GET  /v1/campaigns                                        (viewer+)
POST /v1/campaigns                                        (editor+)
GET  /v1/campaigns/{id}                                   (viewer+)
PUT  /v1/campaigns/{id}                                   (editor+)
PATCH /v1/campaigns/{id}/status                           (editor+)

GET  /v1/campaigns/{id}/creatives                         (viewer+)
POST /v1/campaigns/{id}/creatives                         (editor+)
```

### New Go files added

Handlers (8 new): `api_key.go`, `campaign.go`, `creative.go`, `org_invite.go`, `publisher_app.go`, `publisher_rule.go`, `user.go` + tests for all.

Services (8 new): `api_key.go`, `campaign.go`, `creative.go`, `org_invite.go`, `publisher_app.go`, `publisher_rule.go`, `user.go` + tests for all.

Repositories (7 new): `api_key.go`, `campaign.go`, `creative.go`, `org_invite.go`, `publisher_app.go`, `publisher_rule.go`, `user.go`.

Models (7 new): `api_key.go`, `campaign.go`, `creative.go`, `org_invite.go`, `publisher_app.go`, `publisher_rule.go`, `user.go`.

Auth middleware: `auth.go` modified (JWKS), `auth_test.go` + `testing.go` added.

Note: route prefix changed from `/api/v1` to `/v1` (commit 87c6b55 message explicitly calls this out).

---

## 5. TypeScript Frontend Stack — New UI Pages and Components

Entire `apps/dashboard/` is new (did not exist on main).

### Auth Routes (route group: `(auth)`)

| Route                          | File                                              | Notes                          |
|--------------------------------|---------------------------------------------------|--------------------------------|
| `/login`                       | `app/(auth)/login/page.tsx` + `LoginForm.tsx`     | Email/password, BetterAuth     |
| `/signup`                      | `app/(auth)/signup/page.tsx` + `SignupForm.tsx`   | Registration form              |
| `/onboarding`                  | `app/(auth)/onboarding/page.tsx`                  | Multi-step wizard (org setup)  |
| `/accept-invite/[token]`       | `app/(auth)/accept-invite/[token]/page.tsx`       | Invite acceptance stub         |

Fixed in commit 4078fed: SSR hydration errors on auth pages.

### Dashboard Routes (route group: `(dashboard)`, protected by `middleware.ts`)

| Route                          | File                                              | Key components                              |
|--------------------------------|---------------------------------------------------|---------------------------------------------|
| `/` (home/overview)            | `app/(dashboard)/page.tsx`                        | Dashboard overview page                     |
| `/apps`                        | `app/(dashboard)/apps/page.tsx`                   | AppsList, CreateAppDialog                   |
| `/apps/[id]`                   | `app/(dashboard)/apps/[id]/page.tsx`              | AppDetail, AppOverviewTab, RulesList        |
| `/campaigns`                   | `app/(dashboard)/campaigns/page.tsx`              | CampaignsList, CreateCampaignDialog         |
| `/campaigns/new`               | `app/(dashboard)/campaigns/new/page.tsx`          | New campaign form                           |
| `/campaigns/[id]`              | `app/(dashboard)/campaigns/[id]/page.tsx`         | CampaignDetail, CampaignOverviewTab         |

### Key components per domain

Publisher domain: `AppsList`, `CreateAppDialog`, `AppDetail`, `AppOverviewTab`, `APIKeysList`, `ApiKeyRevealModal`, `RulesList`, `RuleEditorDialog`.

Campaign domain: `CampaignsList`, `CreateCampaignDialog`, `CampaignDetail`, `CampaignOverviewTab`, `CampaignStatusBadge`, `CreativesList`, `CreativeUploadDialog`, `CreativePreview`, `TargetingEditor`.

Shell: `Sidebar`, `Topbar`, `OrgSwitcher`.

### Hooks (all new, 18 total)

`useActiveOrg`, `useApiKeys`, `useCampaign`, `useCampaigns`, `useCreateApiKey`, `useCreateCampaign`, `useCreateCreative`, `useCreatePublisherApp`, `useCreateRule`, `useCreatives`, `useDeleteRule`, `usePublisherApp`, `usePublisherApps`, `usePublisherRules`, `useRevokeApiKey`, `useUpdateCampaign`, `useUpdateCampaignStatus`, `useUpdatePublisherApp`, `useUpdateRule`.

### Auth infrastructure

- `apps/dashboard/lib/auth.ts` — BetterAuth server config
- `apps/dashboard/lib/auth-client.ts` — BetterAuth client
- `apps/dashboard/middleware.ts` — Next.js middleware: protects all `(dashboard)` routes, redirects unauthenticated to `/login`
- `apps/dashboard/app/api/auth/[...all]/route.ts` — BetterAuth catch-all API route

---

## 6. New Features Requiring E2E Smoke Testing

Prioritized by user-facing criticality:

### Auth flows (blocker — gate to everything else)
1. Login with valid credentials → redirect to `/`
2. Signup → account created → redirect to onboarding
3. Onboarding wizard → org created → redirect to dashboard
4. Accept invite via token → user joins org
5. Unauthenticated access to `/apps` → redirect to `/login`

### Publisher App management
6. Create publisher app → appears in list at `/apps`
7. View app detail at `/apps/[id]`
8. Create API key for app → appears in keys list; reveal modal shows key
9. Revoke API key → key marked revoked
10. Create targeting rule for app → appears in rules list
11. Edit targeting rule
12. Delete targeting rule

### Campaign management
13. Create campaign → appears in list at `/campaigns`
14. View campaign detail at `/campaigns/[id]`
15. Update campaign status (draft → active → paused)
16. Upload creative to campaign → appears in creatives list

### Org switching
17. Switch active org via OrgSwitcher → data reloads for new org

---

## 7. Notable Observations

- Route prefix change: `/api/v1` → `/v1` (breaking if any client still uses old prefix)
- Auth middleware replaced HMAC with JWKS validation — any environment without JWKS endpoint reachable will break all authenticated endpoints
- `apps/dashboard/` is entirely new — no prior smoke scenarios exist for any of these routes
- `middleware.ts` protects the `(dashboard)` group; auth pages are intentionally public — verify the route group separation holds
- `pnpm-lock.yaml` added (5 463 lines) — first time frontend dependencies are locked; verify no conflicts on CI
- SSR hydration fix (4078fed) specifically on auth pages — worth a targeted smoke run on login/signup/onboarding after deploy

