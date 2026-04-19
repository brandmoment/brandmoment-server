package model

import (
	"time"

	"github.com/google/uuid"
)

type CreativeType string

const (
	TypeHTML5 CreativeType = "html5"
	TypeImage CreativeType = "image"
	TypeVideo CreativeType = "video"
)

// Creative represents a creative asset in campaigns.
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
