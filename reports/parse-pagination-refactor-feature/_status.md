# Task: Extract parsePagination to shared handler utils
Profile: Feature  |  Stage: Implement  |  Next: refactor-go
Created: 2026-04-19  |  Updated: 2026-04-19

## Context
Issue #8: parsePagination is defined in publisher_app.go but called from campaign.go and publisher_rule.go.
Move to handler/helpers.go so it has a proper shared location.

## Handoff
next: refactor-go | reason: refactoring task | input: services/api-dashboard/internal/handler/publisher_app.go:101-116
