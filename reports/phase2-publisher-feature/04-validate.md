# Validate — Phase 2 Publisher Feature
Agent: test-runner
Date: 2026-04-18

## Results Table

| Check | Status | Details |
|-------|--------|---------|
| go build ./... | PASS | No compilation errors |
| go vet ./... | PASS | No static analysis issues |
| go test ./... | PASS | 223 tests passed across 9 packages |
| sqlc generate | PASS | All queries compiled against schema |
| SQL migrations | PASS | 7 migrations, all have .up/.down, sequential |
| pnpm install | PASS | Dependencies resolved cleanly |
| pnpm exec tsc --noEmit | PASS | No type errors |
| pnpm exec next lint | FAIL | 2 files, 5 errors |

---

## Failures

### Failure 1 — `react/no-unescaped-entities` in APIKeysList.tsx

**Check**: `pnpm exec next lint`

**File**: `/Users/glavatskikh.denis/brandmomentai/brandmoment-server/apps/dashboard/components/publisher/APIKeysList.tsx`

**Error** (line 68):
```
68:57  Error: `"` can be escaped with `&quot;`, `&ldquo;`, `&#34;`, `&rdquo;`.  react/no-unescaped-entities
68:68  Error: `"` can be escaped with `&quot;`, `&ldquo;`, `&#34;`, `&rdquo;`.  react/no-unescaped-entities
68:71  Error: `"` can be escaped with `&quot;`, `&ldquo;`, `&#34;`, `&rdquo;`.  react/no-unescaped-entities
68:76  Error: `"` can be escaped with `&quot;`, `&ldquo;`, `&#34;`, `&rdquo;`.  react/no-unescaped-entities
```

**Current code (line 68)**:
```tsx
Give your API key a descriptive name (e.g., "Production", "Beta").
```

**Root Cause**: Raw `"` characters inside JSX text content. React ESLint rule `react/no-unescaped-entities` disallows literal `"` in JSX text — they must be replaced with HTML entities or moved into a JS string expression.

**Suggested Fix**: Replace the `<DialogDescription>` content with escaped entities:
```tsx
Give your API key a descriptive name (e.g., &ldquo;Production&rdquo;, &ldquo;Beta&rdquo;).
```
Or use a JS string expression:
```tsx
{'Give your API key a descriptive name (e.g., "Production", "Beta").'}
```

---

### Failure 2 — `@typescript-eslint/no-empty-object-type` in textarea.tsx

**Check**: `pnpm exec next lint`

**File**: `/Users/glavatskikh.denis/brandmomentai/brandmoment-server/apps/dashboard/components/ui/textarea.tsx`

**Error** (line 4):
```
4:18  Error: An interface declaring no members is equivalent to its supertype.  @typescript-eslint/no-empty-object-type
```

**Current code (line 4)**:
```ts
export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}
```

**Root Cause**: Empty interface that only extends a supertype. The `@typescript-eslint/no-empty-object-type` rule flags this — it is semantically equivalent to just using `React.TextareaHTMLAttributes<HTMLTextAreaElement>` directly.

**Suggested Fix**: Replace the empty interface with a type alias:
```ts
export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;
```
This is the shadcn/ui idiomatic pattern for component props that add no new fields.

---

## SQL Migration Audit

All 7 migrations (000001–000007) are present with matching `.up.sql` and `.down.sql` files. Numbering is sequential with no gaps. Phase 2 migrations (000005, 000006, 000007) cover `publisher_apps`, `api_keys`, and `publisher_rules` respectively.

## Summary

Go backend is fully green: build, vet, and all 223 tests pass. SQL codegen and migration structure are correct. TypeScript type checking passes. Two lint violations in the TypeScript layer must be resolved before merge — both are in non-logic UI code and have straightforward fixes:

1. `components/publisher/APIKeysList.tsx:68` — escape `"` characters in JSX text
2. `components/ui/textarea.tsx:4` — replace empty interface with type alias

**Recommended next action**: Return to Implement stage to fix the two lint violations, then re-run validation.
