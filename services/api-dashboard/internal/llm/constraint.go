package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/model"
)

// ConstraintResult is the outcome of the constraint-based confidence check.
type ConstraintResult struct {
	Status  ConfidenceStatus `json:"status"`
	Errors  []string         `json:"errors,omitempty"`
	Latency time.Duration    `json:"-"`
}

// CheckConstraint validates parsed rules against the structural constraints of
// the rule engine without making any LLM calls. It calls json.Unmarshal and the
// same validateRuleConfig logic used by PublisherRuleService.Create.
//
// The function accepts the raw JSON string that the LLM produced (must be an
// array of {type, config} objects).
func CheckConstraint(_ context.Context, rawJSON string) ConstraintResult {
	start := time.Now()

	type rawRule struct {
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}

	var rules []rawRule
	if err := json.Unmarshal([]byte(rawJSON), &rules); err != nil {
		return ConstraintResult{
			Status:  ConfidenceStatusFail,
			Errors:  []string{fmt.Sprintf("json parse: %s", err)},
			Latency: time.Since(start),
		}
	}
	if len(rules) == 0 {
		return ConstraintResult{
			Status:  ConfidenceStatusFail,
			Errors:  []string{"no rules produced"},
			Latency: time.Since(start),
		}
	}

	var errs []string
	for i, r := range rules {
		if err := validateRuleType(r.Type); err != nil {
			errs = append(errs, fmt.Sprintf("rule[%d].type: %s", i, err))
			continue
		}
		if err := validateRuleConfig(r.Type, r.Config); err != nil {
			errs = append(errs, fmt.Sprintf("rule[%d].config: %s", i, err))
		}
	}

	if len(errs) > 0 {
		return ConstraintResult{
			Status:  ConfidenceStatusFail,
			Errors:  errs,
			Latency: time.Since(start),
		}
	}
	return ConstraintResult{Status: ConfidenceStatusOK, Latency: time.Since(start)}
}

// validateRuleType mirrors the service-layer check without importing the service package.
func validateRuleType(ruleType string) error {
	switch ruleType {
	case model.RuleTypeBlocklist,
		model.RuleTypeAllowlist,
		model.RuleTypeFrequencyCap,
		model.RuleTypeGeoFilter,
		model.RuleTypePlatformFilter:
		return nil
	}
	return fmt.Errorf("unknown rule type %q", ruleType)
}

// validateRuleConfig mirrors service.validateRuleConfig without importing service.
func validateRuleConfig(ruleType string, config json.RawMessage) error {
	if len(config) == 0 {
		return fmt.Errorf("config is required")
	}
	switch ruleType {
	case model.RuleTypeBlocklist:
		var cfg model.BlocklistConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return err
		}
		if len(cfg.Domains) == 0 && len(cfg.BundleIDs) == 0 {
			// categories field is not in model.BlocklistConfig; tolerate extension
		}
	case model.RuleTypeAllowlist:
		var cfg model.AllowlistConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return err
		}
	case model.RuleTypeFrequencyCap:
		var cfg model.FrequencyCapConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return err
		}
		if cfg.MaxImpressions <= 0 {
			return fmt.Errorf("max_impressions must be > 0")
		}
		if cfg.WindowSeconds <= 0 {
			return fmt.Errorf("window_seconds must be > 0")
		}
	case model.RuleTypeGeoFilter:
		var cfg model.GeoFilterConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return err
		}
		if cfg.Mode != "include" && cfg.Mode != "exclude" {
			return fmt.Errorf("mode must be include or exclude")
		}
		if len(cfg.CountryCodes) == 0 {
			return fmt.Errorf("at least one country_code required")
		}
	case model.RuleTypePlatformFilter:
		var cfg model.PlatformFilterConfig
		if err := json.Unmarshal(config, &cfg); err != nil {
			return err
		}
		if cfg.Mode != "include" && cfg.Mode != "exclude" {
			return fmt.Errorf("mode must be include or exclude")
		}
		if len(cfg.Platforms) == 0 {
			return fmt.Errorf("at least one platform required")
		}
		for _, p := range cfg.Platforms {
			switch strings.ToLower(p) {
			case "ios", "android", "web", "ctv":
			default:
				return fmt.Errorf("unknown platform %q", p)
			}
		}
	}
	return nil
}
