package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestUUIDRoundTrip verifies uuidToPgtype and pgtypeToUUID are inverses.
func TestUUIDRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "random UUID",
			id:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name: "nil UUID",
			id:   uuid.UUID{},
		},
		{
			name: "max UUID",
			id:   uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := uuidToPgtype(tt.id)
			if !pg.Valid {
				t.Error("uuidToPgtype should set Valid=true")
			}
			got := pgtypeToUUID(pg)
			if got != tt.id {
				t.Errorf("round-trip UUID mismatch: got %v, want %v", got, tt.id)
			}
		})
	}
}

// TestInt64ToPgtypeInt8 verifies nil and non-nil pointer conversions.
func TestInt64ToPgtypeInt8(t *testing.T) {
	tests := []struct {
		name      string
		input     *int64
		wantValid bool
		wantVal   int64
	}{
		{
			name:      "nil pointer yields invalid pgtype.Int8",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "zero value",
			input:     int64Ptr(0),
			wantValid: true,
			wantVal:   0,
		},
		{
			name:      "positive value",
			input:     int64Ptr(100_000),
			wantValid: true,
			wantVal:   100_000,
		},
		{
			name:      "negative value",
			input:     int64Ptr(-1),
			wantValid: true,
			wantVal:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int64ToPgtypeInt8(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && got.Int64 != tt.wantVal {
				t.Errorf("Int64 = %v, want %v", got.Int64, tt.wantVal)
			}
		})
	}
}

// TestTimeToPgtypeDate verifies nil and non-nil pointer conversions.
func TestTimeToPgtypeDate(t *testing.T) {
	ref := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     *time.Time
		wantValid bool
	}{
		{
			name:      "nil pointer yields invalid pgtype.Date",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "non-nil time",
			input:     &ref,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeToPgtypeDate(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && !got.Time.Equal(ref) {
				t.Errorf("Time = %v, want %v", got.Time, ref)
			}
		})
	}
}

// TestStringToPgtypeText verifies nil and non-nil pointer conversions.
func TestStringToPgtypeText(t *testing.T) {
	tests := []struct {
		name      string
		input     *string
		wantValid bool
		wantStr   string
	}{
		{
			name:      "nil pointer yields invalid pgtype.Text",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "empty string",
			input:     strPtr(""),
			wantValid: true,
			wantStr:   "",
		},
		{
			name:      "non-empty string",
			input:     strPtr("hello"),
			wantValid: true,
			wantStr:   "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringToPgtypeText(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if tt.wantValid && got.String != tt.wantStr {
				t.Errorf("String = %q, want %q", got.String, tt.wantStr)
			}
		})
	}
}

// TestPgtypeToUUID verifies invalid pgtype.UUID returns zero UUID.
func TestPgtypeToUUID_ZeroOnInvalid(t *testing.T) {
	invalid := pgtype.UUID{Valid: false}
	got := pgtypeToUUID(invalid)
	if got != (uuid.UUID{}) {
		t.Errorf("expected zero UUID for invalid pgtype.UUID, got %v", got)
	}
}

// helpers

func int64Ptr(v int64) *int64 { return &v }
func strPtr(v string) *string { return &v }
