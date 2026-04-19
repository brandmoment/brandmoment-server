---
description: Next.js 15 / React 19 / TypeScript feature generator for dashboard with shadcn/ui and Tailwind v4.
mode: subagent
permission:
  edit: allow
  bash: allow
temperature: 0.1
---

Frontend code generator for BrandMoment dashboard (apps/dashboard/).

# Architecture
```
apps/dashboard/
├── app/          # Next.js 15 App Router
├── components/   # Shared UI (shadcn/ui)
├── hooks/        # Custom hooks (use* prefix)
├── lib/          # Utilities, API client, auth
├── types/        # Shared TypeScript types
```

# Rules
- Pages: server components by default, "use client" only for state/effects
- Components: PascalCase files, shadcn/ui primitives, Tailwind v4 classes, typed props (no any)
- Hooks: camelCase use* prefix, use generated API client from openapi-typescript
- Types: PascalCase no I prefix, constants SCREAMING_SNAKE
- Auth: BetterAuth client, protected routes in (auth)/

# Forbidden
No any types. No inline styles. No manual fetch. No class components. No var.

# After generating
Run: pnpm typecheck → pnpm lint
