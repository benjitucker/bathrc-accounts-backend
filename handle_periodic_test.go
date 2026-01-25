package main

import (
	"benjitucker/bathrc-accounts/db"
	"strings"
	"testing"
	"time"
)

func getMember(membershipNumber string) (*db.MemberRecord, error) {
	return &db.MemberRecord{
		MemberNumber: membershipNumber,
		FirstName:    "John",
		LastName:     "Doe",
		Email:        membershipNumber + "@example.com",
	}, nil
}

func TestWriteEmails_MultipleDatesAndTimes(t *testing.T) {
	now := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)

	// Helper to create times relative to now
	at := func(hours, minutes int) time.Time {
		return now.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)
	}

	// Same day, different times
	sameDayMorning := at(4, 0)
	sameDayEvening := at(10, 0)

	// Next day, still in 36h
	nextDayMorning := at(30, 0)

	// Outside 36h -> excluded
	outsideRange := at(40, 0)

	submissions := []*db.TrainingSubmission{
		{
			TrainingDate:      sameDayMorning,
			Venue:             "Arena1",
			MembershipNumber:  "M001",
			HorseName:         "Lightning",
			FoundMemberRecord: true,
			PaymentRecordId:   "",
		},
		{
			TrainingDate:       sameDayEvening,
			Venue:              "Arena1",
			MembershipNumber:   "M002",
			HorseName:          "Thunder",
			FoundMemberRecord:  true,
			PaymentRecordId:    "P123",
			PaymentDiscrepancy: true,
		},
		{
			TrainingDate:      nextDayMorning,
			Venue:             "Arena2",
			MembershipNumber:  "M003",
			HorseName:         "Storm",
			FoundMemberRecord: true,
			PaymentRecordId:   "",
		},
		{
			TrainingDate:      outsideRange,
			Venue:             "Arena3",
			MembershipNumber:  "M004",
			HorseName:         "Breeze",
			FoundMemberRecord: true,
			PaymentRecordId:   "P999",
		},
	}

	var emails []struct {
		subject string
		body    string
	}

	mockEmailer := func(subject, body string) {
		emails = append(emails, struct {
			subject string
			body    string
		}{subject, body})
	}

	err := writeEmails(now.Add(time.Hour*36), submissions, getMember, mockEmailer)
	if err != nil {
		t.Fatalf("writeEmails returned error: %v", err)
	}

	// Expect 2 emails (Arena1, Arena2)
	if len(emails) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(emails))
	}

	var arena1Email, arena2Email string
	for _, e := range emails {
		if strings.Contains(e.subject, "Arena1") {
			arena1Email = e.body
		}
		if strings.Contains(e.subject, "Arena2") {
			arena2Email = e.body
		}
	}

	if arena1Email == "" {
		t.Fatalf("Arena1 email missing")
	}
	if arena2Email == "" {
		t.Fatalf("Arena2 email missing")
	}

	// Arena1 riders
	if !strings.Contains(arena1Email, "Lightning") {
		t.Errorf("Arena1 email missing Lightning: %s", arena1Email)
	}
	if !strings.Contains(arena1Email, "Thunder") {
		t.Errorf("Arena1 email missing Thunder: %s", arena1Email)
	}

	// Arena1 should include both rider emails
	if !strings.Contains(arena1Email, "M001@example.com") {
		t.Errorf("Arena1 email missing email for M001: %s", arena1Email)
	}
	if !strings.Contains(arena1Email, "M002@example.com") {
		t.Errorf("Arena1 email missing email for M002: %s", arena1Email)
	}

	// Arena2 rider
	if !strings.Contains(arena2Email, "Storm") {
		t.Errorf("Arena2 email missing Storm: %s", arena2Email)
	}
	if !strings.Contains(arena2Email, "M003@example.com") {
		t.Errorf("Arena2 email missing email for M003: %s", arena2Email)
	}

	// Excluded session should not appear
	for _, e := range emails {
		if strings.Contains(e.body, "Breeze") || strings.Contains(e.subject, "Arena3") {
			t.Errorf("Email unexpectedly contains Breeze / Arena3 content: %#v", e)
		}
	}

	// Time formatting must appear in emails
	formattedMorning := formatTime(sameDayMorning)
	formattedEvening := formatTime(sameDayEvening)
	formattedNextDay := formatTime(nextDayMorning)

	if !strings.Contains(arena1Email, formattedMorning) &&
		!strings.Contains(arena1Email, formattedEvening) {
		t.Errorf("Arena1 email missing expected session times (%s or %s): %s",
			formattedMorning, formattedEvening, arena1Email)
	}

	if !strings.Contains(arena2Email, formattedNextDay) {
		t.Errorf("Arena2 email missing expected session time %s: %s",
			formattedNextDay, arena2Email)
	}
}

func TestWriteEmails_FutureSubmissionsSection(t *testing.T) {
	now := time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC)

	at := func(hours, minutes int) time.Time {
		return now.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)
	}

	currentSession := at(4, 0)
	futureSession := at(30, 0)
	differentVenueFuture := at(30, 0)

	submissions := []*db.TrainingSubmission{
		{
			TrainingDate:      currentSession,
			Venue:             "Arena1",
			MembershipNumber:  "M010",
			HorseName:         "Comet",
			FoundMemberRecord: true,
			PaymentRecordId:   "P010",
		},
		{
			TrainingDate:      futureSession,
			Venue:             "Arena1",
			MembershipNumber:  "M010",
			HorseName:         "Comet",
			FoundMemberRecord: true,
			PaymentRecordId:   "P011",
		},
		{
			TrainingDate:      differentVenueFuture,
			Venue:             "Arena2",
			MembershipNumber:  "M010",
			HorseName:         "Comet",
			FoundMemberRecord: true,
			PaymentRecordId:   "P012",
		},
	}

	var emails []struct {
		subject string
		body    string
	}

	mockEmailer := func(subject, body string) {
		emails = append(emails, struct {
			subject string
			body    string
		}{subject, body})
	}

	err := writeEmails(now.Add(time.Hour*36), submissions, getMember, mockEmailer)
	if err != nil {
		t.Fatalf("writeEmails returned error: %v", err)
	}

	if len(emails) != 1 {
		t.Fatalf("expected 1 email, got %d", len(emails))
	}

	body := emails[0].body
	if !strings.Contains(body, "Members with future training submissions at this venue:") {
		t.Fatalf("missing future submissions section: %s", body)
	}

	if !strings.Contains(body, "John Doe:") {
		t.Fatalf("missing member name in future submissions section: %s", body)
	}

	expectedSession := formatCustomDate(futureSession) + " " + formatTime(futureSession) + " riding Comet"
	if !strings.Contains(body, expectedSession) {
		t.Fatalf("missing future session details (%s): %s", expectedSession, body)
	}

	if strings.Contains(body, formatCustomDate(differentVenueFuture)) {
		t.Fatalf("unexpected future session from different venue included: %s", body)
	}
}
