package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	RuleTypeBlocklist       = "blocklist"
	RuleTypeAllowlist       = "allowlist"
	RuleTypeFrequencyCap    = "frequency_cap"
	RuleTypeGeoFilter       = "geo_filter"
	RuleTypePlatformFilter  = "platform_filter"
)

type PublisherRule struct {
	ID        uuid.UUID       `json:"id"`
	OrgID     uuid.UUID       `json:"org_id"`
	AppID     uuid.UUID       `json:"app_id"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Config structs for validation per rule type.

type BlocklistConfig struct {
	Domains   []string `json:"domains"`
	BundleIDs []string `json:"bundle_ids"`
}

type AllowlistConfig struct {
	Domains   []string `json:"domains"`
	BundleIDs []string `json:"bundle_ids"`
}

type FrequencyCapConfig struct {
	MaxImpressions int `json:"max_impressions"`
	WindowSeconds  int `json:"window_seconds"`
}

type GeoFilterConfig struct {
	Mode         string   `json:"mode"`
	CountryCodes []string `json:"country_codes"`
}

type PlatformFilterConfig struct {
	Mode      string   `json:"mode"`
	Platforms []string `json:"platforms"`
}
