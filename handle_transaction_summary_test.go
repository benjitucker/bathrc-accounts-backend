package main

import (
	"benjitucker/bathrc-accounts/db"
	"strings"
	"testing"
	"time"
)

func TestWriteEmail_BuildsSubjectAndCSVBody(t *testing.T) {
	twoMonthsAgo := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)

	transactions := []*db.TransactionRecord{
		{
			Date:         time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			Type:         "CARD",
			FirstName:    "John",
			LastName:     "Doe",
			Description:  "Membership",
			AmountPence:  12345,
			BalancePence: 54321,
		},
		{
			Date:         time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC),
			Type:         "CARD",
			FirstName:    "Jane",
			LastName:     "Smith",
			Description:  "",
			AmountPence:  5000,
			BalancePence: 60000,
		},
	}
	transactions[0].SetID("tx1")
	transactions[1].SetID("tx2")

	submissions := []*db.TrainingSubmission{
		{
			PaymentRecordId:    "tx1",
			PaymentReference:   "REF123",
			Venue:              "West Wilts",
			MembershipNumber:   "M100",
			PaymentDiscrepancy: true,
		},
	}

	var gotSubject string
	var gotBody string

	emailer := func(subject, body string) {
		gotSubject = subject
		gotBody = body
	}

	writeEmail(transactions, submissions, twoMonthsAgo, emailer)

	// -------- SUBJECT ASSERTION --------
	expectedSubjectPrefix := "Transactions summary since "
	if !strings.HasPrefix(gotSubject, expectedSubjectPrefix) {
		t.Fatalf("expected subject to start with %q, got %q", expectedSubjectPrefix, gotSubject)
	}

	// -------- BODY ASSERTIONS --------
	lines := strings.Split(strings.TrimSpace(gotBody), "\n")

	// First line should be CSV header line message
	if !strings.HasPrefix(lines[0], "Transactions CSV since ") {
		t.Fatalf("expected body first line to start with CSV header, got %q", lines[0])
	}

	// We expect 2 CSV lines (one per transaction)
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines (header + 2 CSV rows), got %d", len(lines))
	}

	tx1Line := lines[2]
	tx2Line := lines[3]

	// -------- TX1 ASSERTIONS (has matching submission + discrepancy) --------
	if !strings.Contains(tx1Line, "Payment Discrepancy") {
		t.Errorf("expected tx1 line to contain Payment Discrepancy note, got %q", tx1Line)
	}
	if !strings.Contains(tx1Line, "REF123") {
		t.Errorf("expected tx1 line to contain reference REF123, got %q", tx1Line)
	}
	if !strings.Contains(tx1Line, "WWEC") {
		t.Errorf("expected tx1 line to contain venue code WWEC, got %q", tx1Line)
	}

	// -------- TX2 ASSERTIONS (no submissions → No Payment Found) --------
	if !strings.Contains(tx2Line, "No Payment Found") {
		t.Errorf("expected tx2 line to contain 'No Payment Found', got %q", tx2Line)
	}
}
