package model

import (
	"time"

	"github.com/google/uuid"
)

// CampaignStatus represents the lifecycle state of a campaign.
type CampaignStatus string

const (
	// StatusDraft indicates the campaign is being prepared and not yet running.
	StatusDraft CampaignStatus = "draft"
	// StatusActive indicates the campaign is currently running.
	StatusActive CampaignStatus = "active"
	// StatusPaused indicates the campaign has been temporarily halted.
	StatusPaused CampaignStatus = "paused"
	// StatusCompleted indicates the campaign has finished and cannot be restarted.
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

// AgeRange defines an inclusive minimum/maximum age bracket for campaign targeting.
type AgeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// CampaignTargeting holds the audience-selection criteria for a campaign.
type CampaignTargeting struct {
	Geo       []string  `json:"geo"`
	Platforms []string  `json:"platforms"`
	AgeRange  *AgeRange `json:"age_range,omitempty"`
	Interests []string  `json:"interests"`
}

// Campaign represents an advertising campaign owned by an organisation.
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
