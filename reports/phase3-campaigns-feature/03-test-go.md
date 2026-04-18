# Handler Tests — Phase 3 Campaign/Creative

Agent: go-test-writer
Stage: Test
Date: 2026-04-18

## Result

All new handler tests pass. Full suite is green.

```
go test ./services/api-dashboard/internal/handler/...  → 167 passed
go test ./services/api-dashboard/...                   → 317 passed (264 pre-existing + 53 new)
```

---

## Files Created

### `services/api-dashboard/internal/handler/campaign_test.go`

Mock: `mockCampaignRepoForHandler` — implements `repository.CampaignRepository` with function fields.
Compile-time interface check: `var _ repository.CampaignRepository = (*mockCampaignRepoForHandler)(nil)`.
Constructor: `newCampaignHandler(repo) *CampaignHandler` — wires repo → `NewCampaignService` (noop OTel) → `NewCampaignHandler`.
Stub: `stubCampaign(orgID)` — minimal valid `model.Campaign` with status=draft.

| Test function | Cases |
|---|---|
| `TestCampaignHandler_Create` | valid 201, invalid JSON 400, empty name 400, name >200 chars 400, negative budget 400, end before start 400, repo error 500 |
| `TestCampaignHandler_GetByID` | found 200, invalid UUID 400, not found 404, cross-org (ErrNotFound) 404 |
| `TestCampaignHandler_List` | all campaigns (no filter), status=active filter passed through, invalid status 400, empty list, repo error 500 |
| `TestCampaignHandler_Update` | name update 200, budget+currency update 200, invalid UUID 400, invalid JSON 400, not found 404, empty name 400, end before start 400 |
| `TestCampaignHandler_UpdateStatus` | draft→active 200, active→paused 200, active→completed 200, **invalid: draft→paused 400**, **invalid: completed→active 400**, invalid UUID 400, invalid JSON 400, not found 404, repo error 500 |

Invalid transition cases verify that `handleServiceError` maps `model.ErrInvalidInput` → HTTP 400 with code `INVALID_INPUT`, confirming the state machine is enforced end-to-end through the handler layer.

### `services/api-dashboard/internal/handler/creative_test.go`

Mock: `mockCreativeRepoForHandler` — implements `repository.CreativeRepository` with function fields.
Compile-time interface check: `var _ repository.CreativeRepository = (*mockCreativeRepoForHandler)(nil)`.
Constructor: `newCreativeHandler(campaignRepo, creativeRepo) *CreativeHandler` — both repos needed because `CreativeService` uses `campaignRepo.GetByID` for ownership verification.
Stub: `stubCreative(orgID, campaignID)` — minimal valid `model.Creative` with type=html5, is_active=true.

| Test function | Cases |
|---|---|
| `TestCreativeHandler_Create` | valid html5 201, campaign not found 404, cross-org campaign 404, invalid campaign UUID 400, invalid JSON 400, empty name 400, invalid type "gif" 400, negative file_size_bytes 400, repo insert error 500 |
| `TestCreativeHandler_ListByCampaign` | returns creatives with total, empty list, campaign not found 404, invalid campaign UUID 400, repo list error 500 |

---

## Patterns Used (consistent with Phase 2)

- `injectAuthContext` — sets JWT context via `middleware.InjectTestContext`
- `withChiID` — injects `{id}` into chi route context
- `marshalBody` / `assertStatus` / `assertErrorCode` / `decodeRespBody` — shared helpers defined in `organization_test.go` and `publisher_app_test.go`
- All tests are table-driven with anonymous struct slices
- No testify or external mocking libraries — function fields on struct mocks only

---

## Layer Rules Verified

- Tests only reach the handler boundary — no direct repo calls in test assertions
- `org_id` comes from `injectAuthContext` (JWT context), never from request body
- Campaign ownership verification tested via `campaignNotFoundRepo` stubs in creative tests — cross-org creative injection is blocked at the service layer (`campaignRepo.GetByID` returns `ErrNotFound`)

---

## Next Steps

- Validate (test-runner): run full suite + `go build ./...` + `go vet ./...`
- Report (report-writer): compile final phase 3 report
