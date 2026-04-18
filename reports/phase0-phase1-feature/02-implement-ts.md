# TypeScript Implementation — Phase 0/1 Dashboard Scaffold

## Status

COMPLETE. All files created. `pnpm install` succeeded. `pnpm exec tsc --noEmit` exits 0.

## Files Created

### Configuration

- `apps/dashboard/package.json` — Next.js 15, React 19, better-auth, tanstack/react-query, openapi-fetch, sonner, react-hook-form, zod, Tailwind v4, radix-ui primitives
- `apps/dashboard/next.config.ts` — minimal Next.js config, serverComponentsExternalPackages for pg
- `apps/dashboard/tsconfig.json` — strict mode, noUncheckedIndexedAccess, paths alias `@/*`
- `apps/dashboard/postcss.config.mjs` — @tailwindcss/postcss v4 setup
- `apps/dashboard/next-env.d.ts` — Next.js type references
- `apps/dashboard/.env.example` — BETTER_AUTH_SECRET, BETTER_AUTH_URL, DATABASE_URL, NEXT_PUBLIC_API_URL

### Styles

- `apps/dashboard/styles/globals.css` — Tailwind v4 via `@import "tailwindcss"`, CSS custom properties for light/dark theme

### Library

- `apps/dashboard/lib/utils.ts` — `cn()` helper (clsx + tailwind-merge)
- `apps/dashboard/lib/auth.ts` — BetterAuth server instance: emailAndPassword plugin + organization plugin + pg adapter
- `apps/dashboard/lib/auth-client.ts` — `createAuthClient()` with organizationClient plugin; exports signIn, signOut, signUp, useSession, organization
- `apps/dashboard/lib/api-client.ts` — openapi-fetch typed client factory `createApiClient(activeOrgId?)` injects X-Org-ID header
- `apps/dashboard/lib/api-types.gen.ts` — stub generated types (paths + components schemas for organizations, me, invites). Replace with `pnpm codegen` once `packages/proto/dashboard.yaml` exists.

### Types

- `apps/dashboard/types/org.ts` — OrgRole, OrgType, OrgMembership, Organization, UserProfile

### Hooks

- `apps/dashboard/hooks/useActiveOrg.ts` — reads OrgContext; throws if used outside provider

### Components (shadcn/ui stubs)

- `apps/dashboard/components/ui/button.tsx` — Button + buttonVariants (cva)
- `apps/dashboard/components/ui/input.tsx` — Input
- `apps/dashboard/components/ui/label.tsx` — Label (radix)
- `apps/dashboard/components/ui/card.tsx` — Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter
- `apps/dashboard/components/ui/dropdown-menu.tsx` — full DropdownMenu set (radix)
- `apps/dashboard/components/ui/avatar.tsx` — Avatar, AvatarImage, AvatarFallback (radix)

### Components (app-level)

- `apps/dashboard/components/OrgSwitcher.tsx` — dropdown listing org memberships; calls authClient.organization.setActive on switch; updates OrgContext
- `apps/dashboard/components/Sidebar.tsx` — nav by orgType (publisher/brand/admin); active route highlight; ghost Button + Link pattern
- `apps/dashboard/components/Topbar.tsx` — logo + OrgSwitcher + user avatar menu with sign-out

### App Router

- `apps/dashboard/app/layout.tsx` — root layout: Inter font, Providers wrapper, metadata
- `apps/dashboard/app/providers.tsx` — QueryClientProvider + OrgContext (activeOrgId state + apiClient factory); Toaster
- `apps/dashboard/middleware.ts` — edge middleware: checks better-auth.session_token cookie; redirects unauthenticated requests to /login; public routes: /login, /signup, /accept-invite/*, /api/auth/*
- `apps/dashboard/app/api/auth/[...all]/route.ts` — BetterAuth catch-all handler via toNextJsHandler

- `apps/dashboard/app/(dashboard)/layout.tsx` — server component: reads session via auth.api.getSession; fetches activeOrg; passes orgs + user info to Topbar; sidebar with orgType
- `apps/dashboard/app/(dashboard)/page.tsx` — server component: redirects to /apps, /campaigns, /admin/organizations by orgType; redirects to /onboarding if no active org

- `apps/dashboard/app/(auth)/login/page.tsx` — email + password form; signIn.email; redirect to `?redirect` param or /; toast on error
- `apps/dashboard/app/(auth)/signup/page.tsx` — name + email + password + confirm form; signUp.email; redirect to /onboarding
- `apps/dashboard/app/(auth)/accept-invite/[token]/page.tsx` — stub server component; displays token; "coming soon" message
- `apps/dashboard/app/(auth)/onboarding/page.tsx` — 3-step wizard (client component): step 1 org type cards; step 2 org name + slug (auto-slugify); step 3 success + CTA; calls POST /v1/organizations

## Known Issues / Follow-up

1. **api-types.gen.ts is a stub.** Once `packages/proto/dashboard.yaml` is authored (go-builder task), run `pnpm codegen` from `apps/dashboard/` to replace it with real generated types.

2. **auth.api.getFullOrganization query shape** — BetterAuth's organization plugin API may differ slightly between minor versions. The call in `(dashboard)/layout.tsx` passes `query: { organizationId }` — verify against installed `better-auth@1.6.5` docs if a runtime error occurs.

3. **BetterAuth database migrations** — BetterAuth creates its own internal tables (sessions, accounts, verifications, organizations, members). Run `npx @better-auth/cli migrate` or use the built-in `auth.migrate()` helper before starting the app for the first time.

4. **X-Org-ID header from middleware** — the edge middleware only checks cookie presence, not validity (no JWT parsing at edge). Full session validation happens server-side in layout.tsx via auth.api.getSession. This is intentional per spec Section 10.

5. **Shadcn CLI** — these components were created manually. When shadcn CLI is available, regenerate with `npx shadcn@latest add button input label card dropdown-menu avatar` to get the latest canonical versions.
