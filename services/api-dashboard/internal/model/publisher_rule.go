package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	// RuleTypeBlocklist identifies a rule that blocks specific domains or bundle IDs.
	RuleTypeBlocklist = "blocklist"
	// RuleTypeAllowlist identifies a rule that restricts delivery to specific domains or bundle IDs.
	RuleTypeAllowlist = "allowlist"
	// RuleTypeFrequencyCap identifies a rule that limits impression frequency per user window.
	RuleTypeFrequencyCap = "frequency_cap"
	// RuleTypeGeoFilter identifies a rule that filters delivery by geographic region.
	RuleTypeGeoFilter = "geo_filter"
	// RuleTypePlatformFilter identifies a rule that filters delivery by device platform.
	RuleTypePlatformFilter = "platform_filter"
)

// PublisherRule represents a delivery rule applied to a publisher app.
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

// BlocklistConfig holds the parameters for a blocklist rule.
type BlocklistConfig struct {
	Domains   []string `json:"domains"`
	BundleIDs []string `json:"bundle_ids"`
}

// AllowlistConfig holds the parameters for an allowlist rule.
type AllowlistConfig struct {
	Domains   []string `json:"domains"`
	BundleIDs []string `json:"bundle_ids"`
}

// FrequencyCapConfig holds the parameters for a frequency-cap rule.
type FrequencyCapConfig struct {
	MaxImpressions int `json:"max_impressions"`
	WindowSeconds  int `json:"window_seconds"`
}

// GeoFilterConfig holds the parameters for a geo-filter rule.
type GeoFilterConfig struct {
	Mode         string   `json:"mode"`
	CountryCodes []string `json:"country_codes"`
}

// PlatformFilterConfig holds the parameters for a platform-filter rule.
type PlatformFilterConfig struct {
	Mode      string   `json:"mode"`
	Platforms []string `json:"platforms"`
}
