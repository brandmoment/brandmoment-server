# Bug Fix: Multi-tenancy violation in Organization GetByID
Date: 2026-04-18
Status: Fixed

## Problem
GET /api/v1/organizations/{id} allowed any authenticated user to read any organization's data by guessing the UUID. The GetByID handler extracted the org ID from the URL parameter but never validated that the requested organization belongs to the user's allowed orgs from their JWT claims.

## Reproduction
Reading the handler code at services/api-dashboard/internal/handler/organization.go confirmed the bug — GetByID parsed the URL param and called service.GetByID directly without any membership check. Compare with the List handler which correctly uses middleware.OrgIDsFromContext().

## Root Cause
The GetByID handler was missing a JWT membership validation check. It accepted any valid UUID from the URL without verifying the requesting user has access to that organization via their JWT orgs[] claim. This broke multi-tenancy isolation.

## Fix
File: services/api-dashboard/internal/handler/organization.go

Added a membership check before calling service.GetByID:
1. Extract user's allowed org IDs via middleware.OrgIDsFromContext()
2. Check if the requested org ID exists in the user's allowed orgs
3. Return 403 Forbidden if the user doesn't have access
4. Added containsOrgID() helper function for the membership check

## Validation
- go build ./... — PASS
- go vet ./... — PASS  
- go test ./... — PASS (12 tests across 9 packages)
