# E2E Smoke Update — Phase 0 + Phase 1 + Phase 2 + Phase 3

Agent: e2e-test-writer
Stage: Update Smoke (Mode B)
Date: 2026-04-18

---

## 1. Scenarios Generated

10 new smoke scenarios written, all in `tests/smoke/scenarios.md` (file created — did not exist before).

| Scenario | File | Tags |
|----------|------|------|
| login-page-renders | tests/smoke/login-page-renders.spec.ts | auth, rendering |
| signup-page-renders | tests/smoke/signup-page-renders.spec.ts | auth, rendering |
| unauthenticated-redirect | tests/smoke/unauthenticated-redirect.spec.ts | auth, middleware |
| protected-apps-route-redirect | tests/smoke/protected-apps-route-redirect.spec.ts | auth, middleware |
| accept-invite-stub | tests/smoke/accept-invite-stub.spec.ts | auth, rendering |
| onboarding-page-renders | tests/smoke/onboarding-page-renders.spec.ts | auth, rendering, wizard |
| login-form-validation | tests/smoke/login-form-validation.spec.ts | auth, form-validation |
| signup-form-validation | tests/smoke/signup-form-validation.spec.ts | auth, form-validation |
| login-to-signup-navigation | tests/smoke/login-to-signup-navigation.spec.ts | auth, navigation |
| signup-to-login-navigation | tests/smoke/signup-to-login-navigation.spec.ts | auth, navigation |

Playwright config: `tests/smoke/playwright.config.ts` (baseURL: http://localhost:3000, timeout: 30s, chromium).

---

## 2. Test Execution Results

Playwright run: `apps/dashboard/node_modules/.bin/playwright test --config=tests/smoke/playwright.config.ts`

| Scenario | Status | Duration | Failed Step |
|----------|--------|----------|-------------|
| login-page-renders | FAIL | 5.1s | Step 2: getByRole heading "Sign in" — not found |
| signup-page-renders | FAIL | 5.1s | Step 2: getByRole heading "Create account" — not found |
| unauthenticated-redirect | FAIL | 5.2s | Step 3/4: heading "Sign in" — not found (login page itself 500s) |
| protected-apps-route-redirect | PASS | 0.2s | — |
| accept-invite-stub | FAIL | 6.9s | Step 3: heading "Invite Acceptance" — not found |
| onboarding-page-renders | FAIL | 5.2s | Step 3: heading "Sign in" — not found (login page itself 500s) |
| login-form-validation | FAIL | 30.0s | Step 3: validation errors never appear (page 500s) |
| signup-form-validation | FAIL | 30.0s | Step 3: validation errors never appear (page 500s) |
| login-to-signup-navigation | FAIL | 30.0s | Step 4: heading "Create account" — not found |
| signup-to-login-navigation | FAIL | 30.0s (est) | Step 4: heading "Sign in" — not found |

Summary: 1 PASS / 9 FAIL

The single passing test (`protected-apps-route-redirect`) works because it only checks that `/apps` redirects to `/login` — the redirect itself happens via Next.js middleware before the login page needs to render, so the 500 error on the login page is not reached.

---

## 3. Root Cause Analysis

All 9 failures share the same underlying cause: the Next.js app returns HTTP 500 on every page request except the middleware-level redirect. Two distinct bugs confirmed from dev server output:

### Bug A — `ssr: false` in Server Component (CRITICAL)

**File:** `apps/dashboard/app/(auth)/login/page.tsx`

```
`ssr: false` is not allowed with `next/dynamic` in Server Components.
Please move it into a client component.
```

`login/page.tsx` is a Server Component (no `"use client"` directive) that calls `dynamic(() => import("./LoginForm"), { ssr: false })`. Next.js 15 / Turbopack rejects this combination at compile time, causing a `ModuleBuildError` that results in 500 for the `/login` route.

This was intended to be the SSR hydration fix from commit `4078fed`, but the fix was applied incorrectly: `ssr: false` was added to the page-level Server Component instead of wrapping it in a Client Component.

**Fix location:** `apps/dashboard/app/(auth)/login/page.tsx` — needs `"use client"` added to make it a Client Component, OR the `dynamic` wrapper must be moved into a separate Client Component file.

The same pattern likely applies to `/signup` if it also uses `dynamic + ssr: false`. Verified: `signup/page.tsx` uses `dynamic(() => import("./SignupForm"), { ssr: false })` (same pattern).

### Bug B — `localStorage.getItem is not a function` (CRITICAL)

**File:** `apps/dashboard/lib/auth-client.ts`

```
[TypeError: localStorage.getItem is not a function]
```

`auth-client.ts` uses a lazy `require("better-auth/client/plugins")` inside `getAuthClient()` to defer BetterAuth initialization. However, the `Proxy` wrapping `authClient` triggers `Reflect.get(getAuthClient(), prop)` on any property access. Because `organizationClient()` from `better-auth/client/plugins` accesses `localStorage` during plugin initialization, and this code path is reached during the root layout SSR render, it throws on the Node.js server where `localStorage` does not exist.

This error affects ALL routes that use the root layout (including `/accept-invite/[token]`), explaining why even the pure Server Component accept-invite page returns 500.

**Fix location:** `apps/dashboard/lib/auth-client.ts` — `getAuthClient()` must guard against server-side execution (`if (typeof window === "undefined") return`), or the `Proxy` must be replaced with a `"use client"`-only module so it never executes on the server.

---

## 4. Affected Components

| Component/File | Bug | Impact |
|----------------|-----|--------|
| `apps/dashboard/app/(auth)/login/page.tsx` | Bug A: ssr: false in Server Component | /login → 500 |
| `apps/dashboard/app/(auth)/signup/page.tsx` | Bug A: same pattern | /signup → 500 |
| `apps/dashboard/lib/auth-client.ts` | Bug B: localStorage in SSR path | All routes → 500 |

---

## 5. What Is Working

- Next.js middleware runs correctly (no 500 before page render): `/apps` → `/login?redirect=%2Fapps` redirect works (PASS)
- `middleware.ts` PUBLIC_PREFIXES check for `/accept-invite/` is correct at the routing level
- Route group separation `(auth)` vs `(dashboard)` is correctly structured
- Middleware cookie check for `better-auth.session_token` functions correctly

---

## 6. Smoke Test File Paths

```
tests/smoke/
├── scenarios.md
├── playwright.config.ts
├── login-page-renders.spec.ts
├── signup-page-renders.spec.ts
├── unauthenticated-redirect.spec.ts
├── protected-apps-route-redirect.spec.ts
├── accept-invite-stub.spec.ts
├── onboarding-page-renders.spec.ts
├── login-form-validation.spec.ts
├── signup-form-validation.spec.ts
├── login-to-signup-navigation.spec.ts
└── signup-to-login-navigation.spec.ts
```

Screenshots from the one completed failure: `test-results/login-page-renders-Scenari-41d69-der-all-login-form-elements-chromium/test-failed-1.png` — shows "Internal Server Error" text.
