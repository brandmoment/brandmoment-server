package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type OrgType string

const (
	OrgTypeAdmin     OrgType = "admin"
	OrgTypePublisher OrgType = "publisher"
	OrgTypeBrand     OrgType = "brand"
)

func (t OrgType) Valid() bool {
	switch t {
	case OrgTypeAdmin, OrgTypePublisher, OrgTypeBrand:
		return true
	}
	return false
}

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Type      OrgType   `json:"type"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
