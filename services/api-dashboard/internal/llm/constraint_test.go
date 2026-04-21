package llm

import (
	"context"
	"testing"
)

func TestCheckConstraint_ValidRules(t *testing.T) {
	tests := []struct {
		name       string
		rawJSON    string
		wantStatus ConfidenceStatus
	}{
		{
			name: "valid blocklist with domains",
			rawJSON: `[{"type":"blocklist","config":{"domains":["evil.com"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid blocklist with bundle_ids",
			rawJSON: `[{"type":"blocklist","config":{"bundle_ids":["com.evil.app"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid allowlist",
			rawJSON: `[{"type":"allowlist","config":{"domains":["good.com"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid frequency_cap",
			rawJSON: `[{"type":"frequency_cap","config":{"max_impressions":3,"window_seconds":86400}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid geo_filter include",
			rawJSON: `[{"type":"geo_filter","config":{"mode":"include","country_codes":["US","GB"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid geo_filter exclude",
			rawJSON: `[{"type":"geo_filter","config":{"mode":"exclude","country_codes":["RU"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid platform_filter ios android",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"include","platforms":["ios","android"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "valid platform_filter web ctv",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"exclude","platforms":["web","ctv"]}}]`,
			wantStatus: ConfidenceStatusOK,
		},
		{
			name: "multiple valid rules",
			rawJSON: `[
				{"type":"blocklist","config":{"domains":["bad.com"]}},
				{"type":"frequency_cap","config":{"max_impressions":5,"window_seconds":3600}}
			]`,
			wantStatus: ConfidenceStatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q; errors: %v", result.Status, tt.wantStatus, result.Errors)
			}
		})
	}
}

func TestCheckConstraint_InvalidJSON(t *testing.T) {
	tests := []struct {
		name       string
		rawJSON    string
		wantStatus ConfidenceStatus
	}{
		{
			name:       "malformed JSON",
			rawJSON:    `{not valid json`,
			wantStatus: ConfidenceStatusFail,
		},
		{
			name:       "empty string",
			rawJSON:    ``,
			wantStatus: ConfidenceStatusFail,
		},
		{
			name:       "object instead of array",
			rawJSON:    `{"type":"blocklist","config":{}}`,
			wantStatus: ConfidenceStatusFail,
		},
		{
			name:       "empty array",
			rawJSON:    `[]`,
			wantStatus: ConfidenceStatusFail,
		},
		{
			name:       "null",
			rawJSON:    `null`,
			wantStatus: ConfidenceStatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != ConfidenceStatusFail {
				t.Errorf("Status = %q, want %q", result.Status, ConfidenceStatusFail)
			}
		})
	}
}

func TestCheckConstraint_UnknownRuleType(t *testing.T) {
	rawJSON := `[{"type":"unknown_type","config":{"foo":"bar"}}]`
	result := CheckConstraint(context.Background(), rawJSON)
	if result.Status != ConfidenceStatusFail {
		t.Errorf("Status = %q, want FAIL for unknown rule type", result.Status)
	}
	if len(result.Errors) == 0 {
		t.Error("expected Errors to be non-empty for unknown rule type")
	}
}

func TestCheckConstraint_BlocklistInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		rawJSON string
	}{
		{
			name:    "malformed config JSON",
			rawJSON: `[{"type":"blocklist","config":not_json}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != ConfidenceStatusFail {
				t.Errorf("Status = %q, want FAIL", result.Status)
			}
		})
	}
}

func TestCheckConstraint_FrequencyCapInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		rawJSON string
	}{
		{
			name:    "zero max_impressions",
			rawJSON: `[{"type":"frequency_cap","config":{"max_impressions":0,"window_seconds":3600}}]`,
		},
		{
			name:    "negative window_seconds",
			rawJSON: `[{"type":"frequency_cap","config":{"max_impressions":3,"window_seconds":-1}}]`,
		},
		{
			name:    "both zero",
			rawJSON: `[{"type":"frequency_cap","config":{"max_impressions":0,"window_seconds":0}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != ConfidenceStatusFail {
				t.Errorf("Status = %q, want FAIL", result.Status)
			}
			if len(result.Errors) == 0 {
				t.Error("expected Errors to be non-empty")
			}
		})
	}
}

func TestCheckConstraint_GeoFilterInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		rawJSON string
	}{
		{
			name:    "invalid mode",
			rawJSON: `[{"type":"geo_filter","config":{"mode":"both","country_codes":["US"]}}]`,
		},
		{
			name:    "empty country_codes",
			rawJSON: `[{"type":"geo_filter","config":{"mode":"include","country_codes":[]}}]`,
		},
		{
			name:    "missing country_codes",
			rawJSON: `[{"type":"geo_filter","config":{"mode":"include"}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != ConfidenceStatusFail {
				t.Errorf("Status = %q, want FAIL", result.Status)
			}
		})
	}
}

func TestCheckConstraint_PlatformFilterInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		rawJSON string
	}{
		{
			name:    "invalid mode",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"only","platforms":["ios"]}}]`,
		},
		{
			name:    "empty platforms",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"include","platforms":[]}}]`,
		},
		{
			name:    "unknown platform value",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"include","platforms":["windows"]}}]`,
		},
		{
			name:    "missing platforms",
			rawJSON: `[{"type":"platform_filter","config":{"mode":"exclude"}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckConstraint(context.Background(), tt.rawJSON)
			if result.Status != ConfidenceStatusFail {
				t.Errorf("Status = %q, want FAIL", result.Status)
			}
		})
	}
}

func TestCheckConstraint_Latency(t *testing.T) {
	rawJSON := `[{"type":"blocklist","config":{"domains":["example.com"]}}]`
	result := CheckConstraint(context.Background(), rawJSON)
	if result.Latency <= 0 {
		t.Error("expected Latency > 0")
	}
}
