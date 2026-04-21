# Validation Report: confidence-check-rule-parser

Date: 2026-04-21
Branch: feature/confidence-check-rule-parser

---

## Results Table

| Check | Status | Details |
|-------|--------|---------|
| go build ./... | PASS | All packages build cleanly |
| go vet ./... | PASS | No vet issues |
| go test ./... | PASS | All test packages pass (see breakdown below) |
| pnpm typecheck | FAIL | 4 type errors across 3 files |
| pnpm lint | SKIPPED | Stopped on typecheck failure |

---

## Go Test Breakdown

| Package | Result | Duration |
|---------|--------|----------|
| services/api-dashboard/cmd/server | no test files | — |
| services/api-dashboard/finetune | ok | 96.411s |
| services/api-dashboard/internal/config | no test files | — |
| services/api-dashboard/internal/handler | ok | 1.738s |
| services/api-dashboard/internal/httputil | ok | 0.487s |
| services/api-dashboard/internal/llm | ok | 0.697s |
| services/api-dashboard/internal/middleware | ok | 1.510s |
| services/api-dashboard/internal/model | no test files | — |
| services/api-dashboard/internal/repository | ok | 0.977s |
| services/api-dashboard/internal/router | no test files | — |
| services/api-dashboard/internal/service | ok | 1.956s |
| packages/shared-domain/db | no test files | — |

---

## Failures

### TypeScript: pnpm typecheck — FAIL

**Check**: TypeScript compilation (`tsc --noEmit`)
**Exit code**: 1

#### Error 1
- **File**: `app/(auth)/login/LoginForm.tsx:48:28`
- **Error**: `TS2304: Cannot find name 'signIn'`
- **Root cause**: `signIn` is not imported or exported from `@/lib/auth-client`. The auth client module is missing this export.

#### Error 2
- **File**: `app/(auth)/signup/SignupForm.tsx:54:28`
- **Error**: `TS2304: Cannot find name 'signUp'`
- **Root cause**: `signUp` is not imported or exported from `@/lib/auth-client`. Same module is missing this export.

#### Error 3
- **File**: `components/OrgSwitcher.tsx:34:16`
- **Error**: `TS2339: Property 'organization' does not exist on type '{ signIn: ... }'`
- **Root cause**: The auth client type returned by `useSession` or equivalent hook does not include an `organization` property. The type definition is incomplete or the property was renamed/removed.

#### Error 4
- **File**: `components/Topbar.tsx:4:10`
- **Error**: `TS2305: Module '"@/lib/auth-client"' has no exported member 'signOut'`
- **Root cause**: `signOut` is not exported from `@/lib/auth-client`. The auth client module is missing this export.

**Classification**: Compilation errors (TypeScript)

**Suggested fix**: The `@/lib/auth-client` module (likely `apps/dashboard/lib/auth-client.ts`) is not exporting `signIn`, `signUp`, and `signOut`. These are standard BetterAuth client exports. The fix is to ensure the auth client is initialized with `createAuthClient()` from `better-auth/react` and that named exports for these functions are re-exported from `lib/auth-client.ts`. The `organization` property on the session type likely requires the organization plugin to be registered in the BetterAuth client config.

**Note**: These type errors appear pre-existing (unrelated to confidence-check-rule-parser Go/TS changes) — the spec and implement stages made no changes to `LoginForm.tsx`, `SignupForm.tsx`, `OrgSwitcher.tsx`, or `Topbar.tsx`. Verify whether these errors existed on main before attributing them to this feature branch.

---

## Summary

Go stack: fully green — build, vet, and all tests pass.
TypeScript stack: blocked at typecheck with 4 compilation errors in auth-related UI components. Lint was not run.

Per validation failure routing rules: TypeScript compilation errors route back to the Implement stage (source broken).
