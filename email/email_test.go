package email

import (
	"testing"
	"time"
)

func TestFormatCustomDate_Summertime(t *testing.T) {
	// BST (British Summer Time) usually starts on the last Sunday of March
	// 2026-03-29 01:00:00 UTC -> 2026-03-29 02:00:00 BST
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("Failed to load location Europe/London: %v", err)
	}

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "Before BST transition",
			time:     time.Date(2026, 3, 28, 12, 0, 0, 0, loc),
			expected: "Saturday 28th March",
		},
		{
			name:     "After BST transition",
			time:     time.Date(2026, 3, 29, 12, 0, 0, 0, loc),
			expected: "Sunday 29th March",
		},
		{
			name:     "In July (fully BST)",
			time:     time.Date(2026, 7, 15, 12, 0, 0, 0, loc),
			expected: "Wednesday 15th July",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCustomDate(tt.time)
			if got != tt.expected {
				t.Errorf("formatCustomDate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatCustomDateTime_Summertime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("Failed to load location Europe/London: %v", err)
	}

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "1st April at 10:30 AM (GMT)",
			time:     time.Date(2026, 4, 1, 10, 30, 0, 0, loc),
			expected: "Wednesday 1st April at 10:30 AM",
		},
		{
			name:     "1st May at 10:30 AM (BST)",
			time:     time.Date(2026, 5, 1, 10, 30, 0, 0, loc),
			expected: "Friday 1st May at 10:30 AM",
		},
		{
			name:     "1st May at 10:30 PM (BST)",
			time:     time.Date(2026, 5, 1, 22, 30, 0, 0, loc),
			expected: "Friday 1st May at 10:30 PM",
		},
		{
			name:     "Midnight transition",
			time:     time.Date(2026, 5, 1, 0, 5, 0, 0, loc),
			expected: "Friday 1st May at 12:05 AM",
		},
		{
			name:     "Noon",
			time:     time.Date(2026, 5, 1, 12, 0, 0, 0, loc),
			expected: "Friday 1st May at 12:00 PM",
		},
		{
			name: "Fall transition (BST ending)",
			// BST ends last Sunday of Oct. 2026-10-25 02:00 BST -> 01:00 GMT
			time:     time.Date(2026, 10, 25, 12, 0, 0, 0, loc),
			expected: "Sunday 25th October at 12:00 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCustomDateTime(tt.time)
			if got != tt.expected {
				t.Errorf("formatCustomDateTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
