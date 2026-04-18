package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/httputil"
)

type contextKey string

const (
	ctxOrgID     contextKey = "org_id"
	ctxRole      contextKey = "role"
	ctxOrgIDs    contextKey = "org_ids"
	ctxUserID    contextKey = "user_id"
	ctxOrgClaims contextKey = "org_claims"
)

type OrgClaim struct {
	OrgID string `json:"org_id"`
	Role  string `json:"role"`
}

type Claims struct {
	jwt.RegisteredClaims
	Orgs []OrgClaim `json:"orgs"`
}

type Auth struct {
	jwks keyfunc.Keyfunc
}

func NewAuth(jwksURL string) (*Auth, error) {
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}
	return &Auth{jwks: k}, nil
}

func (a *Auth) ValidateJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, a.jwks.Keyfunc)
		if err != nil || !token.Valid {
			httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}

		// Parse user ID from sub claim
		subStr, err := claims.GetSubject()
		if err != nil || subStr == "" {
			httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing sub claim")
			return
		}
		userID, err := uuid.Parse(subStr)
		if err != nil {
			httputil.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sub claim is not a valid UUID")
			return
		}

		xOrgID := r.Header.Get("X-Org-ID")
		if xOrgID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "MISSING_ORG_ID", "X-Org-ID header is required")
			return
		}

		orgID, err := uuid.Parse(xOrgID)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "INVALID_ORG_ID", "X-Org-ID must be a valid UUID")
			return
		}

		var role string
		var orgIDs []uuid.UUID
		for _, oc := range claims.Orgs {
			parsed, err := uuid.Parse(oc.OrgID)
			if err != nil {
				continue
			}
			orgIDs = append(orgIDs, parsed)
			if parsed == orgID {
				role = oc.Role
			}
		}

		if role == "" {
			httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", "you are not a member of this organization")
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxUserID, userID)
		ctx = context.WithValue(ctx, ctxOrgID, orgID)
		ctx = context.WithValue(ctx, ctxRole, role)
		ctx = context.WithValue(ctx, ctxOrgIDs, orgIDs)
		ctx = context.WithValue(ctx, ctxOrgClaims, claims.Orgs)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			httputil.RespondError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		})
	}
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxUserID).(uuid.UUID)
	return id
}

func OrgClaimsFromContext(ctx context.Context) []OrgClaim {
	claims, _ := ctx.Value(ctxOrgClaims).([]OrgClaim)
	return claims
}

func OrgIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxOrgID).(uuid.UUID)
	return id
}

func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(ctxRole).(string)
	return role
}

func OrgIDsFromContext(ctx context.Context) []uuid.UUID {
	ids, _ := ctx.Value(ctxOrgIDs).([]uuid.UUID)
	return ids
}
