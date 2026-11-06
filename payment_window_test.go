package main

import (
	"benjitucker/bathrc-accounts/db"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParsePaymentWindow(t *testing.T) {
	for in, want := range map[string]time.Duration{"": 2 * time.Hour, "2h": 2 * time.Hour, "15m": 15 * time.Minute, " 90m ": 90 * time.Minute} {
		got, err := parsePaymentWindow(in)
		if err != nil || got != want {
			t.Errorf("parsePaymentWindow(%q) = %s, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"soon", "30s", "-1h", "0"} {
		if _, err := parsePaymentWindow(bad); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}

func TestStartPaymentWindow(t *testing.T) {
	paymentWindow = 2 * time.Hour
	now := time.Date(2026, 11, 5, 13, 45, 0, 0, time.UTC)
	earlier := now.Add(-time.Hour)

	fresh := pricedBooking(nil)
	started := pricedBooking(func(b *db.TrainingSubmission) { b.PaymentDueAt = earlier })
	startPaymentWindow([]*db.TrainingSubmission{fresh, started}, now)

	if !fresh.PaymentDueAt.Equal(now.Add(2 * time.Hour)) {
		t.Errorf("new deadline %s", fresh.PaymentDueAt)
	}
	if !started.PaymentDueAt.Equal(earlier) {
		t.Errorf("an existing deadline must not be extended, got %s", started.PaymentDueAt)
	}
}

func TestPaymentWindowClosed(t *testing.T) {
	due := time.Date(2026, 11, 5, 15, 45, 0, 0, time.UTC)
	b := []*db.TrainingSubmission{pricedBooking(func(b *db.TrainingSubmission) { b.PaymentDueAt = due })}

	if paymentWindowClosed(b, due.Add(-time.Minute)) {
		t.Errorf("closed before the deadline")
	}
	if !paymentWindowClosed(b, due.Add(time.Minute)) {
		t.Errorf("open after the deadline")
	}
	if paymentWindowClosed([]*db.TrainingSubmission{pricedBooking(nil)}, due) {
		t.Errorf("a booking with no deadline must stay open")
	}
}

func TestFormatDeadline_UKTime(t *testing.T) {
	// November is GMT, July is BST (UTC+1)
	if got := formatDeadline(time.Date(2026, 11, 5, 15, 45, 0, 0, time.UTC)); got != "3:45 PM on Thursday 5th November" {
		t.Errorf("winter: %q", got)
	}
	if got := formatDeadline(time.Date(2026, 7, 15, 14, 45, 0, 0, time.UTC)); got != "3:45 PM on Wednesday 15th July" {
		t.Errorf("summer: %q", got)
	}
	if formatDeadline(time.Time{}) != "" {
		t.Errorf("no deadline should be empty")
	}
}

func dueBooking(id string, due time.Time, mutate func(b *db.TrainingSubmission)) *db.TrainingSubmission {
	b := pricedBooking(func(b *db.TrainingSubmission) { b.PaymentDueAt = due })
	b.SetID(id)
	if mutate != nil {
		mutate(b)
	}
	return b
}

func TestExpiredUnpaidSubmissions(t *testing.T) {
	now := time.Date(2026, 11, 5, 18, 0, 0, 0, time.UTC)
	longAgo := now.Add(-paymentReleaseGrace - time.Minute) // past the deadline and the grace
	justPast := now.Add(-time.Minute)                      // past the deadline, within the grace

	bookings := []*db.TrainingSubmission{
		dueBooking("100-0", longAgo, nil),  // expired
		dueBooking("101-0", justPast, nil), // within the grace: kept, so a payment page opened in time can finish
		dueBooking("102-0", longAgo, func(b *db.TrainingSubmission) { b.PaymentRecordId = "pi_1" }), // paid
		dueBooking("103-0", time.Time{}, nil),                                                       // no deadline yet (held for a membership check)
		dueBooking("104-0", longAgo, nil),                                                           // two sessions, both expired
		dueBooking("104-1", longAgo, nil),
		dueBooking("105-0", longAgo, nil), // two sessions, one already paid: keep
		dueBooking("105-1", longAgo, func(b *db.TrainingSubmission) { b.SubmissionState = db.PaidSubmissionState }),
		dueBooking("106-0", longAgo, func(b *db.TrainingSubmission) { b.SubmissionState = db.DroppedSubmissionState }), // already dropped
	}

	got := expiredUnpaidSubmissions(bookings, now)
	want := []string{"100", "104"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReleaseGraceOutlastsPaymentPage(t *testing.T) {
	// A payment page opened just before the deadline must expire before the booking is released
	if paymentReleaseGrace <= checkoutSessionLifetime {
		t.Fatalf("release grace %s must be longer than a payment page's %s", paymentReleaseGrace, checkoutSessionLifetime)
	}
	if checkoutSessionLifetime < 30*time.Minute {
		t.Fatalf("Stripe requires payment pages to last at least 30 minutes")
	}
}

func TestPaymentWindowClosedPage(t *testing.T) {
	body := htmlResponse(http.StatusOK, pagePaymentWindowClosed, "club@example.org").Body
	if !strings.Contains(body, "The time to pay has passed") || !strings.Contains(body, "book again") {
		t.Errorf("page text missing")
	}
}
