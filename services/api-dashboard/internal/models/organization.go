package models

import (
	"github.com/google/uuid"
	"time"
)

type Organization struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	OrgType   string    `db:"org_type"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}