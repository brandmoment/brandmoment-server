package model

import (
	"time"

	"github.com/google/uuid"
)

type CampaignStatus string

const (
	StatusDraft     CampaignStatus = "draft"
	StatusActive    CampaignStatus = "active"
	StatusPaused    CampaignStatus = "paused"
	StatusCompleted CampaignStatus = "completed"
)

// ValidTransitions defines the allowed next states for each campaign status.
// A terminal state (completed) has no allowed transitions.
var ValidTransitions = map[CampaignStatus][]CampaignStatus{
	StatusDraft:     {StatusActive},
	StatusActive:    {StatusPaused, StatusCompleted},
	StatusPaused:    {StatusActive, StatusCompleted},
	StatusCompleted: {},
}

type AgeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type CampaignTargeting struct {
	Geo       []string  `json:"geo"`
	Platforms []string  `json:"platforms"`
	AgeRange  *AgeRange `json:"age_range,omitempty"`
	Interests []string  `json:"interests"`
}

// Campaign represents a marketing campaign.
  type Campaign struct {
	ID          uuid.UUID         `json:"id"`
	OrgID       uuid.UUID         `json:"org_id"`
	Name        string            `json:"name"`
	Status      CampaignStatus    `json:"status"`
	Targeting   CampaignTargeting `json:"targeting"`
	BudgetCents *int64            `json:"budget_cents"`
	Currency    string            `json:"currency"`
	StartDate   *time.Time        `json:"start_date"`
	EndDate     *time.Time        `json:"end_date"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
