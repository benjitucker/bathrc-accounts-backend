package jotform_webhook

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTrainingRawRequestWithID_FormID(t *testing.T) {
	js := `{"id":"6695255243214467187","form_id":"263062182377357","created_at":"2026-11-03 09:32:05","answers":{}}`

	var r TrainingRawRequestWithID
	if err := json.Unmarshal([]byte(js), &r); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if r.SubmissionID != "6695255243214467187" {
		t.Errorf("unexpected submission id %q", r.SubmissionID)
	}
	if r.FormID != "263062182377357" {
		t.Errorf("unexpected form id %q", r.FormID)
	}
	if len(r.Entries) != 0 {
		t.Errorf("expected no entries, got %d", len(r.Entries))
	}
}

func TestTrainingRawRequestWithID_CreatedAtIsUSEastern(t *testing.T) {
	// The API reports 08:51:47 US Eastern for a booking made at 13:51:47 UK time (GMT)
	js := `{"id":"1","form_id":"263062182377357","created_at":"2026-11-05 08:51:47","answers":{}}`

	var r TrainingRawRequestWithID
	if err := json.Unmarshal([]byte(js), &r); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	got := time.Time(r.SubmitDate).UTC()
	want := time.Date(2026, 11, 5, 13, 51, 47, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("submit date %s, want %s", got, want)
	}
	if loc := time.Time(r.SubmitDate).Location(); loc != time.UTC {
		t.Errorf("submit date should be stored in UTC, got %s", loc)
	}
}
