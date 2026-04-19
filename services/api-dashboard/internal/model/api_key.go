package model

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents a user's API key.
  type APIKey struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     uuid.UUID  `json:"org_id"`
	AppID     uuid.UUID  `json:"app_id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	KeyPrefix string     `json:"key_prefix"`
	IsRevoked bool       `json:"is_revoked"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}
