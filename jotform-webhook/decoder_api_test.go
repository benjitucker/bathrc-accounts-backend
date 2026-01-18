package jotform_webhook

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshal_Training_FromAPIJSON(t *testing.T) {

	// trimmed JSON: 2 submissions, one with 1 entry, one with 2 entries
	apiJSON := `
  [
    {
      "id": "6429329371501736170",
      "form_id": "252725624662359",
      "ip": "109.152.150.51",
      "created_at": "2025-12-30 14:42:18",
      "status": "ACTIVE",
      "answers": {
        "15": { "name": "brcMembership15", "answer": "99691212" },
        "18": { "name": "horseName18", "answer": "Luke" },
        "31": { "name": "amount", "answer": "20" },
        "34": { "name": "selectedVenue", "answer": "Widbrook" },
        "5":  { 
          "name": "selectWidbrookSession", 
          "answer": { "date": "2026-01-02 15:00", "duration": "60", "timezone": "Europe/London (GMT)" }
        },
        "12": { "name": "typeA", "answer": "CATJ" },
        "53": { "name": "paymentRef", "answer": "CATJ" },
        "58": { "name": "totalAmount", "answer": "20" }
      }
    },
    {
      "id": "6429028811507594617",
      "form_id": "252725624662359",
      "ip": "109.152.150.51",
      "created_at": "2025-12-30 06:21:21",
      "status": "ACTIVE",
      "answers": {
        "15": { "name": "brcMembership15", "answer": "11111111" },
        "18": { "name": "horseName18", "answer": "test1" },
        "31": { "name": "amount", "answer": "21" },
        "34": { "name": "selectedVenue", "answer": "WestWilts" },
        "5":  { 
          "name": "selectWestWiltsSession",
          "answer": { "date": "2026-01-01 13:00", "duration": "60", "timezone": "Europe/London (GMT+01:00)" }
        },

        "48": { "name": "brcMembership15-2", "answer": "22222222" },
        "50": { "name": "horseName18-2", "answer": "test2" },
        "59": { "name": "amount-2", "answer": "16" },
        "60": { "name": "selectedVenue-2", "answer": "Widbrook" },
        "51": { 
          "name": "selectWidbrookSession-2",
          "answer": { "date": "2026-01-02 13:00", "duration": "60", "timezone": "Europe/London (GMT+00:00)" }
        },

        "12": { "name": "typeA", "answer": "VSHE" },
        "53": { "name": "paymentRef", "answer": "VSHE" },
        "58": { "name": "totalAmount", "answer": "37" }
      }
    }
  ]`

	var content []TrainingRawRequestWithID

	if err := json.Unmarshal([]byte(apiJSON), &content); err != nil {
		t.Fatalf("failed to unmarshal api JSON: %v", err)
	}

	// ---------- Top-level assertions ----------

	if len(content) != 2 {
		t.Fatalf("expected 2 submissions, got %d", len(content))
	}

	first := content[0]
	second := content[1]

	// ---------- First submission checks ----------
	if first.SubmissionID != "6429329371501736170" {
		t.Fatalf("expected id 6429329371501736170, got %s", first.SubmissionID)
	}

	if len(first.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(first.Entries))
	}

	e := first.Entries[0]

	if e.MembershipNumber != "99691212" {
		t.Errorf("membership mismatch")
	}

	if e.HorseName != "Luke" {
		t.Errorf("horse mismatch")
	}

	if e.Venue != "Widbrook" {
		t.Errorf("venue mismatch")
	}

	if e.Amount != "20" {
		t.Errorf("amount mismatch")
	}

	if first.PaymentRef != "CATJ" {
		t.Errorf("paymentRef mismatch")
	}

	if first.TotalAmount != "20" {
		t.Errorf("totalAmount mismatch")
	}

	if e.SelectSession.StartLocal.IsZero() {
		t.Errorf("expected non-zero session time")
	}

	// ---------- Second submission checks ----------

	if len(second.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(second.Entries))
	}

	e1 := second.Entries[0]
	e2 := second.Entries[1]

	if e1.MembershipNumber != "11111111" || e2.MembershipNumber != "22222222" {
		t.Errorf("membership mismatch on multi-entry decode")
	}

	if e2.HorseName != "test2" {
		t.Errorf("horse mismatch on entry2")
	}

	if second.PaymentRef != "VSHE" {
		t.Errorf("paymentRef mismatch")
	}

	if second.TotalAmount != "37" {
		t.Errorf("totalAmount mismatch")
	}

	// ---------- Time validation ----------

	if time.Time(second.SubmitDate).IsZero() {
		t.Errorf("submit date not parsed")
	}

	// ---------- Decoder contract check ----------

	// These numeric keys still existed in JSON — but the decoder should NOT use them.
	// This assertion protects your name-based decoder logic conceptually.
	for _, k := range []string{"5", "18", "31"} {
		_ = k // just documenting intent; nothing to assert programmatically
	}
}
