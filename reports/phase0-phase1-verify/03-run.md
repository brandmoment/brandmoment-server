# Run — Full Check Suite
Agent: test-runner
Stage: Run
Date: 2026-04-18

---

## 1. Go Checks

### 1.1 `go build ./...`

Commands run (workspace modules separately, as go.work uses explicit module paths):
```
cd services/api-dashboard && go build ./...
cd packages/shared-domain && go build ./...
```

| Module | Exit Code | Result |
|--------|-----------|--------|
| `services/api-dashboard` | 0 | PASS |
| `packages/shared-domain` | 0 | PASS |

No errors.

---

### 1.2 `go vet ./...`

| Module | Exit Code | Result |
|--------|-----------|--------|
| `services/api-dashboard` | 0 | PASS |
| `packages/shared-domain` | 0 | PASS |

No warnings.

---

### 1.3 `go test ./... -v`

#### `services/api-dashboard` (exit 0 — PASS)

| Test | Result |
|------|--------|
| TestAPIKeyHandler_Create | PASS |
| TestAPIKeyHandler_ListByApp | PASS |
| TestAPIKeyHandler_Revoke | PASS |
| TestCampaignHandler_Create | PASS |
| TestCampaignHandler_GetByID | PASS |
| TestCampaignHandler_List | PASS |
| TestCampaignHandler_Update | PASS |
| TestCampaignHandler_UpdateStatus | PASS |
| TestCreativeHandler_Create | PASS |
| TestCreativeHandler_ListByCampaign | PASS |
| TestOrgInviteHandler_Create | PASS |
| TestOrganizationHandler_Create | PASS |
| TestOrganizationHandler_GetByID | PASS |
| TestOrganizationHandler_List | PASS |
| TestHandleServiceError | PASS |
| TestPublisherAppHandler_Create | PASS |
| TestPublisherAppHandler_GetByID | PASS |
| TestPublisherAppHandler_List | PASS |
| TestPublisherAppHandler_Update | PASS |
| TestPublisherRuleHandler_Create | PASS |
| TestPublisherRuleHandler_GetByID | PASS |
| TestPublisherRuleHandler_ListByApp | PASS |
| TestPublisherRuleHandler_Update | PASS |
| TestPublisherRuleHandler_Delete | PASS |
| TestUserHandler_GetMe | PASS |
| TestRespondJSON | PASS |
| TestRespondError | PASS |
| TestRespondJSON_DataFieldPresent | PASS |
| TestRespondError_NoDataField | PASS |
| TestAuth_ValidateJWT_HeaderChecks | PASS |
| TestAuth_RequireRole | PASS |
| TestContextHelpers_ZeroValues | PASS |
| TestContextHelpers_SetValues | PASS |
| TestAPIKeyService_Provision | PASS |
| TestAPIKeyService_ListByApp | PASS |
| TestAPIKeyService_Revoke | PASS |
| TestCampaignService_Create | PASS |
| TestCampaignService_UpdateStatus | PASS |
| TestCampaignService_List | PASS |
| TestCampaignService_Update | PASS |
| TestCreativeService_Create | PASS |
| TestCreativeService_ListByCampaign | PASS |
| TestOrgInviteService_Create | PASS |
| TestOrganizationService_Create | PASS |
| TestOrganizationService_GetByID | PASS |
| TestOrganizationService_ListByIDs | PASS |
| TestPublisherAppService_Create | PASS |
| TestPublisherAppService_GetByID | PASS |
| TestPublisherAppService_List | PASS |
| TestPublisherAppService_Update | PASS |
| TestPublisherRuleService_Create | PASS |
| TestPublisherRuleService_List | PASS |
| TestPublisherRuleService_Update | PASS |
| TestPublisherRuleService_Delete | PASS |
| TestUserService_GetMe | PASS |
| TestUserService_UpsertUser | PASS |

**Packages passed:** `internal/handler`, `internal/httputil`, `internal/middleware`, `internal/service`

#### `packages/shared-domain` (exit 0 — PASS)

No test files (sqlc-generated code, no unit tests). Result: `[no test files]`.

**Total Go:** 56 tests, 56 passed, 0 failed.

---

## 2. SQL Check

### 2.1 `sqlc generate` (packages/shared-domain)

Exit code: 0 — PASS

No errors. All queries compile against schema.

### 2.2 Migration file audit

Location: `infra/migrations/`

| Migration | Up | Down | Paired |
|-----------|----|------|--------|
| 000001_create_organizations | yes | yes | OK |
| 000002_create_users | yes | yes | OK |
| 000003_create_org_memberships | yes | yes | OK |
| 000004_create_org_invites | yes | yes | OK |
| 000005_create_publisher_apps | yes | yes | OK |
| 000006_create_api_keys | yes | yes | OK |
| 000007_create_publisher_rules | yes | yes | OK |
| 000008_create_campaigns | yes | yes | OK |
| 000009_create_creatives | yes | yes | OK |

Numbering: sequential 000001–000009. No gaps. All 9 migrations have matching `.up.sql` and `.down.sql`.

---

## 3. TypeScript Checks

### 3.1 `pnpm typecheck` (apps/dashboard)

Command: `tsc --noEmit`
Exit code: 2 — FAIL

**4 errors:**

| Error | File | Line | Rule |
|-------|------|------|------|
| TS2304: Cannot find name 'signIn' | `app/(auth)/login/LoginForm.tsx` | 48:28 | undefined variable |
| TS2304: Cannot find name 'signUp' | `app/(auth)/signup/SignupForm.tsx` | 54:28 | undefined variable |
| TS2339: Property 'organization' does not exist on type | `components/OrgSwitcher.tsx` | 34:16 | BetterAuth client API mismatch |
| TS2305: Module '@/lib/auth-client' has no exported member 'signOut' | `components/Topbar.tsx` | 4:10 | missing export |

**Root causes:**

1. `LoginForm.tsx:48` and `SignupForm.tsx:54` — Both forms use `signIn.email(...)` and `signUp.email(...)` as bare global calls. These are NOT exported from `@/lib/auth-client`. The auth-client module exports `authClient` (the proxy object). The correct usage is `authClient.signIn.email(...)` and `authClient.signUp.email(...)`.

2. `OrgSwitcher.tsx:34` — Uses `authClient.organization` which does not exist on the inferred type. The `organizationClient()` plugin from BetterAuth augments the client, but the TypeScript type is not being picked up — likely because `auth-client.ts` uses dynamic `require()` inside `getAuthClient()` which defeats type inference.

3. `Topbar.tsx:4` — Imports `signOut` as a named export from `@/lib/auth-client`. The module only has a default `authClient` proxy export. The correct call is `authClient.signOut()`.

**Suggested problem location:** `apps/dashboard/lib/auth-client.ts` — the module's export shape does not expose individual methods as named exports, and the Proxy-based lazy init prevents TypeScript from inferring the BetterAuth plugin-augmented type.

---

### 3.2 `pnpm lint` (apps/dashboard)

Note: The default `pnpm lint` run hits `.next/` compiled bundle files (Turbopack cache), producing 22,172 noise errors from minified bundles. This is a misconfigured ESLint setup — `eslint.config.mjs` does not exclude `.next/**`.

**Source-file-only lint** (running with `--ignore-pattern ".next/**"`):

Exit code: 1 — FAIL (source violations)

**2 errors in source files:**

| File | Line | Rule | Message |
|------|------|------|---------|
| `app/(auth)/login/LoginForm.tsx` | 10:10 | `@typescript-eslint/no-unused-vars` | `'authClient'` is defined but never used |
| `app/(auth)/signup/SignupForm.tsx` | 10:10 | `@typescript-eslint/no-unused-vars` | `'authClient'` is defined but never used |

**Note on suppressed violations in `lib/auth-client.ts`:**
Two `@typescript-eslint/no-require-imports` violations are suppressed inline with `// eslint-disable-next-line` at lines 13 and 15. These are intentional suppressions for the dynamic `require()` pattern in the lazy auth-client init.

**Secondary issue:** ESLint config does not exclude `.next/` directory. When running plain `pnpm lint`, the linter processes Turbopack/Next.js compiled artifacts (~22k errors). The `.eslintignore` file does not exist and `eslint.config.mjs` has no ignore patterns. This causes CI lint runs to be unusable.

---

## 4. Playwright E2E

Not re-run in this stage. Results from `02-update-smoke.md` (smoke test run performed during Update Smoke stage):

- 1 PASS / 9 FAIL
- All 9 failures caused by HTTP 500 on page load
- Root causes: Bug A (`ssr: false` in Server Component) + Bug B (`localStorage` access during SSR)
- See `02-update-smoke.md` section 3 for full root cause analysis

---

## 5. Summary Table

| Check | Status | Details |
|-------|--------|---------|
| go build (api-dashboard) | PASS | Clean compile |
| go build (shared-domain) | PASS | Clean compile |
| go vet (api-dashboard) | PASS | No warnings |
| go vet (shared-domain) | PASS | No warnings |
| go test (api-dashboard) | PASS | 56/56 tests pass |
| go test (shared-domain) | PASS | No test files (sqlc-generated) |
| sqlc generate | PASS | All queries compile |
| SQL migration pairs | PASS | 9/9 paired, sequential numbering |
| pnpm typecheck | FAIL | 4 errors — auth-client API mismatch |
| pnpm lint (source) | FAIL | 2 errors — unused `authClient` import in LoginForm + SignupForm |
| pnpm lint (config) | ISSUE | ESLint config missing `.next/` ignore → 22k noise errors on CI |
| playwright e2e | FAIL | 9/10 tests fail — HTTP 500 on all auth pages (from 02-update-smoke.md) |

---

## 6. Failures Detail

### Failure 1: TypeScript — signIn/signUp not in scope

- **Check:** pnpm typecheck
- **Error:** `TS2304: Cannot find name 'signIn'` / `Cannot find name 'signUp'`
- **Files:** `app/(auth)/login/LoginForm.tsx:48`, `app/(auth)/signup/SignupForm.tsx:54`
- **Root cause:** Forms call `signIn.email(...)` and `signUp.email(...)` as globals. The auth client does not export these as standalone names. The proxy object from `@/lib/auth-client` must be destructured or accessed via `authClient.signIn.email(...)`.
- **Suggested fix:** Either export `signIn` and `signUp` from `auth-client.ts`, or update LoginForm/SignupForm to call `authClient.signIn.email(...)` / `authClient.signUp.email(...)`.

### Failure 2: TypeScript — organization property not on type

- **Check:** pnpm typecheck
- **Error:** `TS2339: Property 'organization' does not exist on type`
- **File:** `components/OrgSwitcher.tsx:34`
- **Root cause:** `auth-client.ts` uses dynamic `require()` in a Proxy to lazy-load BetterAuth plugins. TypeScript cannot infer the augmented type (including the `organization` plugin) from a dynamic require inside a Proxy handler.
- **Suggested fix:** Restructure `auth-client.ts` to use static ESM imports with proper SSR guard, so TypeScript can resolve the BetterAuth plugin types.

### Failure 3: TypeScript — signOut not exported

- **Check:** pnpm typecheck
- **Error:** `TS2305: Module '@/lib/auth-client' has no exported member 'signOut'`
- **File:** `components/Topbar.tsx:4`
- **Root cause:** `auth-client.ts` does not export `signOut` as a named member. It only exports the `authClient` proxy.
- **Suggested fix:** Export `signOut` from `auth-client.ts` as `export const signOut = authClient.signOut`, or update `Topbar.tsx` to import `authClient` and call `authClient.signOut()`.

### Failure 4: Lint — unused authClient import

- **Check:** pnpm lint (source files)
- **Error:** `'authClient' is defined but never used`
- **Files:** `app/(auth)/login/LoginForm.tsx:10`, `app/(auth)/signup/SignupForm.tsx:10`
- **Root cause:** Both form files import `authClient` but never use it (they instead incorrectly call `signIn`/`signUp` as globals). This is a consequence of Failure 1 — fixing the TypeScript error will resolve this lint violation.
- **Suggested fix:** Same as Failure 1. Once forms use `authClient.signIn.email(...)`, the import will be used.

### Failure 5: Lint config — missing .next/ ignore

- **Check:** pnpm lint (configuration issue)
- **Error:** 22,172 ESLint errors in compiled bundle files
- **File:** `eslint.config.mjs` — no ignore patterns set
- **Root cause:** `eslint.config.mjs` uses `compat.extends(...)` only, no `ignores` key. Next.js flat config needs explicit `ignores: [".next/**"]`.
- **Suggested fix:** Add `{ ignores: [".next/**", "node_modules/**"] }` to the exported array in `eslint.config.mjs`.

### Failure 6: Playwright E2E — HTTP 500 on auth pages

- **Check:** playwright e2e (smoke tests)
- **Scenario results:** 9 FAIL, 1 PASS
- **Root cause:** Two critical bugs in `apps/dashboard/` (see 02-update-smoke.md section 3):
  - Bug A: `ssr: false` used inside Server Component in `login/page.tsx` and `signup/page.tsx`
  - Bug B: `localStorage.getItem` called during SSR in `lib/auth-client.ts`
- **Suggested fix location:** `apps/dashboard/app/(auth)/login/page.tsx`, `apps/dashboard/app/(auth)/signup/page.tsx`, `apps/dashboard/lib/auth-client.ts`
