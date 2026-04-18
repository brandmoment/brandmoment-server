package model

import (
	"time"

	"github.com/google/uuid"
)

type PublisherApp struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	BundleID  string    `json:"bundle_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
