# Fix: Add defer r.Body.Close() to handlers that decode JSON bodies

## Summary

All handler methods that call `json.NewDecoder(r.Body).Decode(...)` were missing
`defer r.Body.Close()`, causing HTTP request bodies to remain open until GC.

## Affected methods (11 total)

| File | Method |
|------|--------|
| `handler/api_key.go` | `APIKeyHandler.Create` |
| `handler/campaign.go` | `CampaignHandler.Create` |
| `handler/campaign.go` | `CampaignHandler.Update` |
| `handler/campaign.go` | `CampaignHandler.UpdateStatus` |
| `handler/creative.go` | `CreativeHandler.Create` |
| `handler/creative.go` | `CreativeHandler.Update` |
| `handler/org_invite.go` | `OrgInviteHandler.Create` |
| `handler/organization.go` | `OrganizationHandler.Create` |
| `handler/publisher_app.go` | `PublisherAppHandler.Create` |
| `handler/publisher_app.go` | `PublisherAppHandler.Update` |
| `handler/publisher_rule.go` | `PublisherRuleHandler.Create` |
| `handler/publisher_rule.go` | `PublisherRuleHandler.Update` |

## Methods skipped (no body read)

- `APIKeyHandler.List`, `APIKeyHandler.Revoke` — no body decode
- `CampaignHandler.List`, `CampaignHandler.GetByID` — no body decode
- `CreativeHandler.GetByID`, `CreativeHandler.List` — no body decode
- `OrganizationHandler.GetByID`, `OrganizationHandler.List` — no body decode
- `PublisherAppHandler.List`, `PublisherAppHandler.GetByID` — no body decode
- `PublisherRuleHandler.List`, `PublisherRuleHandler.GetByID`, `PublisherRuleHandler.Delete` — no body decode
- `UserHandler.GetMe` — no body decode

## Change pattern

`defer r.Body.Close()` added immediately before the `var req ...` declaration in
each affected method, so it runs on all return paths including early error returns.

## Validation

```
go build ./services/api-dashboard/...  OK
go vet ./services/api-dashboard/...    OK
```
