package jotform_webhook

import (
	"fmt"
	"strings"
	"time"
)

func parseInLocation(dateStr string, loc *time.Location) (time.Time, error) {
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

func ParseSessionDate(dateStr string) (time.Time, error) {
	londonLoc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.Time{}, fmt.Errorf("load London location failed: %w", err)
	}

	return parseInLocation(dateStr, londonLoc)
}

func ParseSessionDateNew(dateStr, tz string) (time.Time, error) {
	londonLoc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.Time{}, fmt.Errorf("load London location failed: %w", err)
	}

	zone := tz
	if i := strings.Index(zone, " "); i > 0 {
		zone = zone[:i]
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load location failed: %w", err)
	}

	var t time.Time
	if t, err = parseInLocation(dateStr, loc); err != nil {
		return time.Time{}, fmt.Errorf("parse date failed: %w", err)
	}
	return t.In(londonLoc), nil
}
