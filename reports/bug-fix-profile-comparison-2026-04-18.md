# Bug Fix Profile: Comparison of Two Runs
Date: 2026-04-18

## Context
Same bug (multi-tenancy violation in Organization GetByID) was fixed twice:
- **Run 1** — before Agent Launch Policy enforcement
- **Run 2** — after adding MANDATORY agent launch rules to CLAUDE.md

## Comparison

| Aspect                  | Run 1 (without policy)                    | Run 2 (with policy)                              |
|-------------------------|-------------------------------------------|--------------------------------------------------|
| Agents launched         | 2 (test-runner, report-writer)            | 5 (go-diagnostics, git-investigator, security-reviewer, go-builder, test-runner, report-writer) |
| Diagnose stage          | Skipped — main read code itself           | go-diagnostics + git-investigator + security-reviewer in parallel |
| Fix stage               | Main wrote fix directly                   | go-builder wrote fix                             |
| Reproduction steps      | Generic ("reading the handler code")      | Concrete scenario (user A reads org-2, gets 200) |
| Root cause detail       | "Missing validation check"                | Traced to specific commit (83defd7), compared with List handler pattern |
| Fix approach            | Custom containsOrgID() helper             | slices.Contains (stdlib, cleaner)                |
| Validation detail       | 3 checks, counts                          | 4 checks in table format, workspace build added  |
| Security side-findings  | None                                      | 3 additional issues found (RBAC, HMAC, orphaned orgs) |
| Report quality          | Functional but shallow                    | Detailed, actionable, with follow-up items       |

## Key Differences

### 1. Agent launch changed the fix quality
Run 1 used a custom `containsOrgID()` helper. Run 2 used `slices.Contains` from stdlib — simpler, no extra code.

### 2. Security reviewer found 3 additional issues
Without the mandatory security-reviewer agent, Run 1 missed:
- Viewer role can POST /organizations (RBAC gap)
- HMAC symmetric secret instead of JWKS
- Created orgs not linked to creator

These are real issues that would have gone unnoticed.

### 3. git-investigator traced the origin
Run 2 identified the exact commit (83defd7) where the bug was introduced. Run 1 had no historical context.

### 4. Reproduction was concrete vs abstract
Run 1: "reading the handler code confirmed the bug." Run 2: step-by-step scenario with expected vs actual response codes.

## Conclusion
Mandatory agent launch significantly improves:
- **Fix quality** — agents choose better approaches (stdlib over custom helpers)
- **Security coverage** — side-findings catch related issues
- **Report depth** — concrete reproduction, historical context, follow-up items
- **Audit trail** — every stage has agent output as evidence

The cost is ~20-30 seconds more execution time per run. The value is substantially higher confidence in the fix and visibility into related problems.
