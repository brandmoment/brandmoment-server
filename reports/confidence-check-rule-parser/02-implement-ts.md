# Implementation: Rule Parser UI

**Stage**: Implement (ts-builder)
**Date**: 2026-04-21

---

## Summary

Built the Rule Parser dashboard page at `app/(dashboard)/publisher-rules/parse/`. The implementation consists of 5 files: 2 types, 1 hook, 1 component, and 2 route files.

---

## File Tree

```
apps/dashboard/
├── types/
│   └── rule-parser.ts                          # NEW — ParseRuleRequest/Response, ConfidenceReport, approach types
├── hooks/
│   └── useParseRule.ts                         # NEW — useMutation wrapping POST /v1/publisher-rules/parse
├── components/publisher/
│   └── RuleParserPage.tsx                      # NEW — full page component
└── app/(dashboard)/publisher-rules/parse/
    ├── page.tsx                                # NEW — server wrapper, exports metadata
    └── loading.tsx                             # NEW — skeleton fallback
```

---

## Component Hierarchy

```
PublisherRulesParsePage (server, page.tsx)
└── RuleParserPage (client, components/publisher/RuleParserPage.tsx)
    ├── Input Card
    │   ├── Textarea (phrase input)
    │   ├── Provider radio buttons (openai | gemini)
    │   └── Approach checkboxes (constraint | scoring | self_check | redundancy)
    ├── LoadingSkeleton (shown while isPending)
    ├── Error panel (shown on error)
    └── ResultsPanel (shown on success)
        ├── Parse Result Card (overall badge + rule JSON cards)
        └── Confidence Breakdown Card (table: approach | status | latency | details)
            └── ApproachRow × N
```

---

## Design Decisions

### API call pattern
The `POST /v1/publisher-rules/parse` endpoint is not yet in `api-types.gen.ts` (it will be added after Go implementation). Used direct `fetch` in the hook instead of the typed `openapi-fetch` client. Followed the same base URL env var (`NEXT_PUBLIC_API_URL`) and `X-Org-ID` header injection pattern used by `createApiClient`.

### Radio/Checkbox
`@radix-ui/react-radio-group` and `@radix-ui/react-checkbox` are not installed. Used native `<input type="radio">` and `<input type="checkbox">` with `accent-primary` Tailwind class — consistent with how the project handles missing primitives.

### At-least-one-approach guard
The toggle function prevents deselecting the last selected approach, keeping the UI in a valid state without needing a toast or error.

### 501 handling
Explicit check for HTTP 501 status with a user-friendly message about missing API key, per the spec requirement.

---

## Typecheck Results

`pnpm typecheck` reports 4 pre-existing errors in auth files (`LoginForm.tsx`, `SignupForm.tsx`, `OrgSwitcher.tsx`, `Topbar.tsx`). Zero errors introduced by this implementation.

---

## Next Steps

1. `pnpm lint` — run ESLint
2. Go implementation of `POST /v1/publisher-rules/parse` (go-builder stage)
3. Once Go handler is live, run `pnpm codegen` to add the endpoint to `api-types.gen.ts` and migrate the hook to use the typed client
4. Add sidebar nav item for "Rule Parser" under `PUBLISHER_NAV` in `Sidebar.tsx` (requires modifying shared layout — needs user approval per architecture rules)
