# Bug Fix: Multi-tenancy violation in Organization GetByID
Date: 2026-04-18
Status: Fixed

## Problem
GET /api/v1/organizations/{id} allowed any authenticated user to read any organization's data by providing its UUID. The GetByID handler extracted the org ID from the URL parameter but never validated that the requested organization belonged to the user's JWT orgs[] claim.

## Reproduction
1. Authenticate as user A (member of org-1 only)
2. Send GET /api/v1/organizations/{org-2-uuid}
3. Response: 200 OK with org-2 data (should be 403 Forbidden)

## Root Cause
In services/api-dashboard/internal/handler/organization.go, the GetByID method (originally lines 41-56) parsed the UUID from chi.URLParam and passed it directly to service.GetByID without any membership check. The List handler in the same file correctly used middleware.OrgIDsFromContext() to scope results, but this pattern was not applied to GetByID. The omission was present since the initial implementation (commit 83defd7, 2026-04-15).

## Fix
Added JWT membership validation to GetByID handler (lines 50-54):
- After parsing the UUID, retrieve allowed org IDs via middleware.OrgIDsFromContext()
- Use slices.Contains to verify the requested ID is in the user's orgs
- Return 403 Forbidden with code "FORBIDDEN" if not a member

File modified: services/api-dashboard/internal/handler/organization.go (added import "slices", added 5-line membership check)

## Validation
| Check | Result |
|-------|--------|
| go build ./... (api-dashboard) | PASS |
| go vet ./... (api-dashboard) | PASS |
| go test ./... (api-dashboard) | PASS (12 tests) |
| go build (workspace) | PASS |

## Additional Security Findings (out of scope)
The security review identified additional issues in the same area:
- CRITICAL: viewer role can POST /organizations (RBAC too permissive)
- CRITICAL: HMAC symmetric secret instead of JWKS for JWT validation
- HIGH: Created orgs not linked to creator (orphaned)
- These should be addressed in separate tickets.
