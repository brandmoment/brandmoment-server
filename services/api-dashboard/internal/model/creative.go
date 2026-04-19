package model

import (
	"time"

	"github.com/google/uuid"
)

// CreativeType enumerates the supported ad creative formats.
type CreativeType string

const (
	// TypeHTML5 represents an interactive HTML5 ad unit.
	TypeHTML5 CreativeType = "html5"
	// TypeImage represents a static image ad.
	TypeImage CreativeType = "image"
	// TypeVideo represents a video ad.
	TypeVideo CreativeType = "video"
)

// Creative represents an ad creative asset belonging to a campaign.
type Creative struct {
	ID            uuid.UUID    `json:"id"`
	OrgID         uuid.UUID    `json:"org_id"`
	CampaignID    uuid.UUID    `json:"campaign_id"`
	Name          string       `json:"name"`
	Type          CreativeType `json:"type"`
	FileURL       string       `json:"file_url"`
	FileSizeBytes *int64       `json:"file_size_bytes"`
	PreviewURL    *string      `json:"preview_url"`
	IsActive      bool         `json:"is_active"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
