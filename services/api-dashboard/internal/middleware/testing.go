package middleware

import (
	"context"

	"github.com/google/uuid"
)

// InjectTestContext sets JWT-derived context values directly, bypassing JWKS validation.
// Intended only for use in tests. Do not call in production code paths.
func InjectTestContext(ctx context.Context, userID, orgID uuid.UUID, role string, orgIDs []uuid.UUID) context.Context {
	claims := make([]OrgClaim, 0, len(orgIDs))
	for _, id := range orgIDs {
		claims = append(claims, OrgClaim{OrgID: id.String(), Role: role})
	}
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxOrgID, orgID)
	ctx = context.WithValue(ctx, ctxRole, role)
	ctx = context.WithValue(ctx, ctxOrgIDs, orgIDs)
	ctx = context.WithValue(ctx, ctxOrgClaims, claims)
	return ctx
}
