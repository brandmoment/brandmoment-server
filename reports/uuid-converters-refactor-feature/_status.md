# Task: Consolidate UUID conversion helpers
Profile: Feature  |  Stage: Implement  |  Next: refactor-go
Created: 2026-04-19  |  Updated: 2026-04-19

## Context
Issue #10: uuidToPgtype and pgtypeToUUID defined in organization.go but used across repos.
Move to repository/converters.go.

## Handoff
next: refactor-go | reason: refactoring | input: services/api-dashboard/internal/repository/organization.go:73-79
