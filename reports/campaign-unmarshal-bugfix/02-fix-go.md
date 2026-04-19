# Fix: campaign JSON unmarshal error lacks context

## File changed

`services/api-dashboard/internal/repository/campaign.go`

## Root cause

`toCampaign` wrapped the JSON unmarshal error as `"unmarshal targeting: %w"`. When this error surfaced in logs or traces it gave no indication of which campaign record caused the failure, making diagnosis in production needlessly slow.

## Change

In `toCampaign`, resolved the campaign UUID up front (re-using the existing `pgtypeToUUID` helper that is already called for `c.ID`) and threaded it into the error message:

```go
// before
return nil, fmt.Errorf("unmarshal targeting: %w", err)

// after
id := pgtypeToUUID(row.ID)
...
return nil, fmt.Errorf("unmarshal targeting for campaign %s: %w", id, err)
```

The pattern matches the project convention `"verb noun for <entity> <id>: %w"` used elsewhere in the repository layer.

## Validation

- `go build ./services/api-dashboard/...` — success
- `go test ./services/api-dashboard/...` — 344 tests passed across 9 packages
