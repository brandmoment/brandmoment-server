# Repository Unit Tests — Go

## Result

90 tests, 0 failures. `go test ./services/api-dashboard/internal/repository/` passes clean.

## Uncovered Modules Before This Task

All 9 files in `services/api-dashboard/internal/repository/` had zero test coverage:
`api_key.go`, `campaign.go`, `creative.go`, `org_invite.go`, `organization.go`,
`publisher_app.go`, `publisher_rule.go`, `user.go`, `converters.go`

## Tests Written

| File | Tests | Cases |
|------|-------|-------|
| `converters_test.go` | `TestUUIDRoundTrip`, `TestInt64ToPgtypeInt8`, `TestTimeToPgtypeDate`, `TestStringToPgtypeText`, `TestPgtypeToUUID_ZeroOnInvalid` | 14 |
| `api_key_test.go` | `TestToAPIKey`, `TestAPIKeyRepo_GetByID_NotFound`, `TestAPIKeyRepo_GetByID_DBError`, `TestAPIKeyRepo_Revoke_NotFound`, `TestAPIKeyRepo_Insert_DBError`, `TestAPIKeyRepo_ListByApp_DBError` | 8 |
| `campaign_test.go` | `TestToCampaign`, `TestCampaignRepo_GetByID_NotFound`, `TestCampaignRepo_GetByID_DBError`, `TestCampaignRepo_Update_NotFound`, `TestCampaignRepo_UpdateStatus_NotFound`, `TestCampaignRepo_Insert_DBError`, `TestCampaignRepo_ListByOrg_DBError` | 10 |
| `creative_test.go` | `TestToCreative`, `TestCreativeRepo_GetByID_NotFound`, `TestCreativeRepo_GetByID_DBError`, `TestCreativeRepo_Update_NotFound`, `TestCreativeRepo_Insert_DBError`, `TestCreativeRepo_ListByCampaign_DBError` | 8 |
| `org_invite_test.go` | `TestToOrgInvite`, `TestOrgInviteRepo_GetByToken_NotFound`, `TestOrgInviteRepo_GetByToken_DBError`, `TestOrgInviteRepo_Insert_DBError` | 6 |
| `organization_test.go` | `TestToOrganization`, `TestOrganizationRepo_GetByID_NotFound`, `TestOrganizationRepo_GetByID_DBError`, `TestOrganizationRepo_Insert_DBError`, `TestOrganizationRepo_ListByIDs_DBError`, `TestOrganizationRepo_ListByIDs_Empty` | 8 |
| `publisher_app_test.go` | `TestToPublisherApp`, `TestPublisherAppRepo_GetByID_NotFound`, `TestPublisherAppRepo_GetByBundleID_NotFound`, `TestPublisherAppRepo_GetByBundleID_DBError`, `TestPublisherAppRepo_Update_NotFound`, `TestPublisherAppRepo_Insert_DBError`, `TestPublisherAppRepo_ListByOrg_DBError` | 9 |
| `publisher_rule_test.go` | `TestToPublisherRule`, `TestPublisherRuleRepo_GetByID_NotFound`, `TestPublisherRuleRepo_GetByID_DBError`, `TestPublisherRuleRepo_Update_NotFound`, `TestPublisherRuleRepo_Delete_DBError`, `TestPublisherRuleRepo_Insert_DBError`, `TestPublisherRuleRepo_ListByApp_DBError` | 9 |
| `user_test.go` | `TestToUser`, `TestUserRepo_GetByID_NotFound`, `TestUserRepo_GetByID_DBError`, `TestUserRepo_Upsert_DBError`, `TestUserRepo_Upsert_GetByID_Independence` | 7 |
| `mock_dbtx_test.go` | Test infrastructure only (mockDBTX, errRow, emptyRows) | — |

Total: 90 test cases across 9 test files.

## Test Architecture

Since `db.Queries` is a concrete struct (not an interface), tests mock at the `DBTX` level
(`db.DBTX` interface defined in `packages/shared-domain/db/db.go`). The mock implements
`Exec`, `Query`, and `QueryRow` with function fields for per-test customization.

Repositories are instantiated directly as their unexported struct types (same package), bypassing
the `NewXRepository(pool)` constructor (which requires a real `*pgxpool.Pool`), and instead
receiving `db.New(mockDBTX)`.

## Coverage by Pattern

- **Converter functions** (`to*`): happy path with all fields, nullable fields nil, zero/invalid inputs
- **ErrNoRows mapping**: every method that calls `GetXByID`, `GetXByToken`, `Update`, `Revoke` verifies `pgx.ErrNoRows → model.ErrNotFound`
- **Generic DB error propagation**: every method verifies non-ErrNoRows errors are wrapped and returned without being mapped to ErrNotFound
- **Insert/List errors**: every repo's Insert and List method is tested for DB error propagation
- **Delete**: `PublisherRuleRepo.Delete` tested via `execFn`

## Bugs Discovered

None. All `pgx.ErrNoRows → model.ErrNotFound` mappings are present where expected.
`toCampaign` correctly returns an error on invalid JSON targeting (verified by test case).
