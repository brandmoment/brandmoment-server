package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrUnauthorized is returned when the caller lacks permission to perform the operation.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidInput is returned when the caller supplies malformed or logically invalid data.
	ErrInvalidInput = errors.New("invalid input")
)

// Organization represents a tenant entity of type admin, publisher, or brand.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
