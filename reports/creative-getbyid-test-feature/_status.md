# Task: Add CreativeHandler.GetByID test coverage
Profile: Feature (Test)  |  Stage: Spec  |  Next: go-test-writer
Created: 2026-04-19  |  Updated: 2026-04-19

## Context
Issue #7: creative_test.go only tests List and Create. GetByID has no coverage.
Need to add test cases covering success, not-found, and org_id mismatch scenarios.

## Handoff
next: go-test-writer | reason: test-only issue, skip to Test stage | input: services/api-dashboard/internal/handler/creative.go, existing tests
