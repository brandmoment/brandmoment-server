# Stage: Implement — repository godoc comments

## Summary

Added godoc comments to all exported interfaces, their methods, and constructors
in `services/api-dashboard/internal/repository/`. No logic or signatures were changed.

## Files modified

| File | Symbols documented |
|------|--------------------|
| `api_key.go` | `APIKeyRepository` (interface + 4 methods), `NewAPIKeyRepository` |
| `campaign.go` | `CampaignRepository` (interface + 5 methods), `NewCampaignRepository` |
| `creative.go` | `CreativeRepository` (interface + 4 methods), `NewCreativeRepository` |
| `org_invite.go` | `OrgInviteRepository` (interface + 2 methods), `NewOrgInviteRepository` |
| `organization.go` | `OrganizationRepository` (interface + 3 methods), `NewOrganizationRepository` |
| `publisher_app.go` | `PublisherAppRepository` (interface + 5 methods), `NewPublisherAppRepository` |
| `publisher_rule.go` | `PublisherRuleRepository` (interface + 5 methods), `NewPublisherRuleRepository` |
| `user.go` | `UserRepository` (interface + 2 methods), `NewUserRepository` |

`converters.go` — no exported symbols; no changes required.

## Convention applied

- Interface comment: `// <TypeName> defines persistence operations for <resource>.`
- Method comment: `// <MethodName> <verb phrase ending with period>`, starting with method name.
- Constructor comment: `// New<Type> constructs a <InterfaceType> backed by the given connection pool.`

## Verification

`go build ./services/api-dashboard/...` — success, zero errors.
