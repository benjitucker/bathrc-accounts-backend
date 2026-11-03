package jotform_webhook

import (
	"encoding/json"
	"testing"
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
