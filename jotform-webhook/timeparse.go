package jotform_webhook

import (
	"fmt"
	"strings"
	"time"
)

// TODO - could be optimised by assuming UK timezone always
func ParseSessionDate(dateStr, tz string) (time.Time, error) {
	zone := tz
	if i := strings.Index(zone, " "); i > 0 {
		zone = zone[:i]
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load location failed: %w", err)
	}

	layouts := []string{
		"2006-01-02 15:04",
		time.RFC3339,
	}

	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, dateStr, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %s", dateStr)
}
