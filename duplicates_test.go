package main

import (
	"benjitucker/bathrc-accounts/db"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOpenForDuplicateCheck(t *testing.T) {
	now := time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	open := dueBooking("1-0", now.Add(time.Minute), nil)     // still has time to pay
	expired := dueBooking("2-0", now.Add(-time.Minute), nil) // time to pay has passed
	held := dueBooking("3-0", time.Time{}, nil)              // no deadline yet (held for a membership check)

	got := openForDuplicateCheck([]*db.TrainingSubmission{open, expired, held}, now)
	if len(got) != 2 || got[0] != open || got[1] != held {
		t.Fatalf("expected the open and held bookings only, got %d", len(got))
	}
}

func TestAlreadyBookedPage_NamesTheSessions(t *testing.T) {
	duplicate := pricedBooking(func(b *db.TrainingSubmission) {
		b.DuplicateOfId = "123-0"
		b.TrainingDate = time.Date(2026, 11, 8, 18, 0, 0, 0, time.UTC)
	})
	other := pricedBooking(func(b *db.TrainingSubmission) {
		b.TrainingDate = time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC)
	})

	body := htmlResponse(http.StatusOK, alreadyBookedPage([]*db.TrainingSubmission{other, duplicate}), "club@example.org").Body
	for _, want := range []string{
		"West Wilts on Sunday 8th November at 6:00 PM, so we haven’t booked it again",
		"none of the sessions in this request have been booked, including West Wilts on Thursday 12th November at 6:00 PM",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %q", want)
		}
	}

	single := htmlResponse(http.StatusOK, alreadyBookedPage([]*db.TrainingSubmission{duplicate}), "").Body
	if strings.Contains(single, "none of the sessions") {
		t.Errorf("a single-session request should not mention other sessions")
	}
}

func TestAlreadyBookedPage_DroppedWithoutDuplicate(t *testing.T) {
	dropped := pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.DroppedSubmissionState })
	body := htmlResponse(http.StatusOK, alreadyBookedPage([]*db.TrainingSubmission{dropped}), "").Body
	if !strings.Contains(body, "It looks like you’ve already sent a request") {
		t.Errorf("expected the general already-requested page")
	}
}
