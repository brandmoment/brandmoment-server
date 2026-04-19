# Stage: Implement — model package godoc

Agent: go-builder
Date: 2026-04-19

## Summary

Added godoc comments to every exported identifier in
`services/api-dashboard/internal/model/`. No logic or signatures were changed.

## Files modified

| File | Identifiers documented |
|------|------------------------|
| `api_key.go` | `APIKey` |
| `campaign.go` | `CampaignStatus`, `StatusDraft`, `StatusActive`, `StatusPaused`, `StatusCompleted`, `AgeRange`, `CampaignTargeting`, `Campaign` |
| `creative.go` | `CreativeType`, `TypeHTML5`, `TypeImage`, `TypeVideo`, `Creative` |
| `org_invite.go` | `OrgInvite` |
| `organization.go` | `ErrNotFound`, `ErrUnauthorized`, `ErrInvalidInput`, `Organization` |
| `publisher_app.go` | `PublisherApp` |
| `publisher_rule.go` | `RuleTypeBlocklist`, `RuleTypeAllowlist`, `RuleTypeFrequencyCap`, `RuleTypeGeoFilter`, `RuleTypePlatformFilter`, `PublisherRule`, `BlocklistConfig`, `AllowlistConfig`, `FrequencyCapConfig`, `GeoFilterConfig`, `PlatformFilterConfig` |
| `user.go` | `User`, `OrgMembership` |

## Convention applied

Every comment starts with the identifier name followed by a single concise sentence,
per the standard Go godoc format (`// TypeName does/represents/holds …`).

## Validation

```
go build ./services/api-dashboard/...  →  success (no errors)
```
