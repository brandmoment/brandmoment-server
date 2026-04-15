package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const orgIDKey contextKey = "org_id"

type Auth struct {
	jwtSecret []byte
}

func NewAuth(jwtSecret string) *Auth {
	return &Auth{jwtSecret: []byte(jwtSecret)}
}

func OrgIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(orgIDKey).(uuid.UUID)
	return id
}

type jwtClaims struct {
	OrgID string `json:"org_id"`
}

func (a *Auth) ValidateJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "MISSING_TOKEN", "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeAuthError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid authorization header format")
			return
		}
		token := parts[1]

		claims, err := a.parseAndVerify(token)
		if err != nil {
			slog.WarnContext(r.Context(), "jwt validation failed", slog.String("error", err.Error()))
			writeAuthError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")
			return
		}

		orgID, err := uuid.Parse(claims.OrgID)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid org_id in token")
			return
		}

		ctx := context.WithValue(r.Context(), orgIDKey, orgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) parseAndVerify(token string) (*jwtClaims, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, errInvalidToken
	}

	// Verify HMAC-SHA256 signature
	signingInput := segments[0] + "." + segments[1]
	mac := hmac.New(sha256.New, a.jwtSecret)
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(segments[2]), []byte(expectedSig)) {
		return nil, errInvalidToken
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, errInvalidToken
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errInvalidToken
	}

	return &claims, nil
}

var errInvalidToken = &tokenError{msg: "invalid token"}

type tokenError struct {
	msg string
}

func (e *tokenError) Error() string { return e.msg }

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"data": nil,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
