# Task: Fix unchecked json.Encode error in httputil
Profile: Bug Fix  |  Stage: Fix  |  Next: go-builder
Created: 2026-04-19  |  Updated: 2026-04-19

## Context
Issue #1: json.NewEncoder(w).Encode() return value not checked in RespondJSON and RespondError.
Bug confirmed at httputil/response.go:24 and :31.

## Handoff
next: go-builder | reason: straightforward fix | input: services/api-dashboard/internal/httputil/response.go lines 24, 31
