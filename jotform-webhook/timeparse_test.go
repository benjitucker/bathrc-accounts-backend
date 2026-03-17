package jotform_webhook

import (
	"testing"
	"time"
)

func TestParseSessionDate_EDT(t *testing.T) {
	dateStr := "2026-03-19 15:00"
	tz := "America/New_York"

	result, err := ParseSessionDate(dateStr, tz)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.IsZero() {
		t.Error("expected non-zero time")
	}

	// 2026-03-19 15:00 America/New_York (EDT, UTC-4)
	// UTC: 2026-03-19 19:00
	// London: 2026-03-19 19:00 GMT (no BST yet)
	expectedYear := 2026
	expectedMonth := time.March
	expectedDay := 19
	expectedHour := 19
	expectedMinute := 0

	if result.Year() != expectedYear {
		t.Errorf("expected year %d, got %d", expectedYear, result.Year())
	}

	if result.Month() != expectedMonth {
		t.Errorf("expected month %v, got %v", expectedMonth, result.Month())
	}

	if result.Day() != expectedDay {
		t.Errorf("expected day %d, got %d", expectedDay, result.Day())
	}

	if result.Hour() != expectedHour {
		t.Errorf("expected hour %d, got %d", expectedHour, result.Hour())
	}

	if result.Minute() != expectedMinute {
		t.Errorf("expected minute %d, got %d", expectedMinute, result.Minute())
	}

	if result.Location().String() != "Europe/London" {
		t.Errorf("expected location %s, got %s", "Europe/London", result.Location().String())
	}
}

func TestParseSessionDate_Summertime(t *testing.T) {
	// London: GMT to BST transition
	// 2026-03-29 01:00:00 UTC -> 02:00:00 BST
	tz := "Europe/London"

	tests := []struct {
		name     string
		dateStr  string
		expected time.Time
	}{
		{
			name:    "Before transition (GMT)",
			dateStr: "2026-03-28 15:00",
			// GMT offset is 0
			expected: time.Date(2026, 3, 28, 15, 0, 0, 0, time.FixedZone("GMT", 0)),
		},
		{
			name:    "After transition (BST)",
			dateStr: "2026-03-30 15:00",
			// BST offset is +1h (3600s)
			expected: time.Date(2026, 3, 30, 15, 0, 0, 0, time.FixedZone("BST", 3600)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSessionDate(tt.dateStr, tz)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// We compare UTC to avoid issues with different Location pointers representing same offset
			if !result.Equal(tt.expected) {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, result)
			}

			// Also check the hour in local time is what we expect from dateStr
			if result.Hour() != 15 {
				t.Errorf("%s: expected hour 15, got %d", tt.name, result.Hour())
			}
		})
	}
}
