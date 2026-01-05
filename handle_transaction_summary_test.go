package main

import (
	"benjitucker/bathrc-accounts/db"
	"strconv"
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

func TestWriteEmail_FullCoverage(t *testing.T) {
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
		{
			Date:         time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC),
			Type:         "CASH",
			FirstName:    "Bob",
			LastName:     "Builder",
			Description:  "Training",
			AmountPence:  2500,
			BalancePence: 10000,
		},
	}

	for i, transaction := range transactions {
		transaction.SetID("tx" + strconv.Itoa(i+1))
	}

	submissions := []*db.TrainingSubmission{
		{
			PaymentRecordId:    "tx1",
			PaymentReference:   "REF123",
			Venue:              "West Wilts",
			MembershipNumber:   "M100",
			PaymentDiscrepancy: true,
		},
		{
			PaymentRecordId:    "tx1",
			PaymentReference:   "REF456",
			Venue:              "Widbrook",
			MembershipNumber:   "M101",
			PaymentDiscrepancy: false,
		},
		{
			PaymentRecordId:    "tx3",
			PaymentReference:   "REF789",
			Venue:              "UnknownVenue",
			MembershipNumber:   "M102",
			PaymentDiscrepancy: false,
		},
	}

	var gotSubject string
	var gotBody string

	emailer := func(subject, body string) {
		gotSubject = subject
		gotBody = body
	}

	writeEmail(transactions, submissions, twoMonthsAgo, emailer)

	// ---------- SUBJECT ----------
	expectedSubjectPrefix := "Transactions summary since "
	if !strings.HasPrefix(gotSubject, expectedSubjectPrefix) {
		t.Fatalf("expected subject to start with %q, got %q", expectedSubjectPrefix, gotSubject)
	}

	// ---------- BODY ----------
	lines := strings.Split(strings.TrimSpace(gotBody), "\n")

	// First line should be the CSV header message
	if !strings.HasPrefix(lines[0], "Transactions CSV since ") {
		t.Fatalf("expected body first line to start with CSV header, got %q", lines[0])
	}

	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines (header + 3 CSV rows), got %d", len(lines))
	}

	// --- TX1: multiple submissions, one discrepancy ---
	tx1Line := lines[2]
	if !strings.Contains(tx1Line, "Payment Discrepancy") {
		t.Errorf("expected tx1 line to contain Payment Discrepancy note, got %q", tx1Line)
	}
	if !strings.Contains(tx1Line, "WID") || !strings.Contains(tx1Line, "WWEC") {
		t.Errorf("expected tx1 line to contain both venue codes WWEC and WID, got %q", tx1Line)
	}
	if !strings.HasSuffix(tx1Line, "REF456") {
		t.Errorf("expected tx1 line to use last submission reference REF456, got %q", tx1Line)
	}

	// --- TX2: no submissions ---
	tx2Line := lines[3]
	if !strings.Contains(tx2Line, "No Payment Found") {
		t.Errorf("expected tx2 line to contain 'No Payment Found', got %q", tx2Line)
	}

	// --- TX3: unknown venue (should use original venue name) ---
	tx3Line := lines[4]
	if !strings.Contains(tx3Line, "UnknownVenue") {
		t.Errorf("expected tx3 line to contain original venue name 'UnknownVenue', got %q", tx3Line)
	}
	if !strings.HasSuffix(tx3Line, "REF789") {
		t.Errorf("expected tx3 line to end with reference REF789, got %q", tx3Line)
	}
}
