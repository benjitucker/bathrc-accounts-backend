package main

import (
	"benjitucker/bathrc-accounts/db"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestSubmissionIDFromRedirect(t *testing.T) {
	body := "formID=263062182377357&submission_id=6695255243214467187&q18_horseName18=Ben"

	tests := []struct {
		name    string
		body    string
		base64  bool
		want    string
		wantErr bool
	}{
		{name: "plain body", body: body, want: "6695255243214467187"},
		{name: "base64 body", body: base64.StdEncoding.EncodeToString([]byte(body)), base64: true, want: "6695255243214467187"},
		{name: "missing submission id", body: "formID=263062182377357", wantErr: true},
		{name: "non-numeric submission id", body: "submission_id=abc%3Bdrop", wantErr: true},
		{name: "invalid base64", body: "%%%", base64: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := submissionIDFromRedirect(tt.body, tt.base64)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func booking(mutate func(b *db.TrainingSubmission)) *db.TrainingSubmission {
	b := &db.TrainingSubmission{
		SubmissionState:   db.ReceivedSubmissionState,
		FoundMemberRecord: true,
		Processed:         true,
		PricesFound:       true,
	}
	if mutate != nil {
		mutate(b)
	}
	return b
}

func TestDecideOutcome(t *testing.T) {
	tests := []struct {
		name     string
		bookings []*db.TrainingSubmission
		want     redirectOutcome
	}{
		{name: "no bookings", bookings: nil, want: outcomeError},
		{name: "valid member", bookings: []*db.TrainingSubmission{booking(nil)}, want: outcomeReadyToPay},
		{
			name: "non-member price but actually a member is still ready to pay",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) {
				b.RequestCurrMem = false
				b.ActualCurrMem = true
			})},
			want: outcomeReadyToPay,
		},
		{
			name: "lapsed non-member price is ready to pay",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) {
				b.RequestCurrMem = false
				b.ActualCurrMem = false
			})},
			want: outcomeReadyToPay,
		},
		{
			name:     "member not found",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) { b.FoundMemberRecord = false })},
			want:     outcomeCheckingMembership,
		},
		{
			name:     "lapsed member claiming member price",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) { b.LapsedMembership = true })},
			want:     outcomeCheckingMembership,
		},
		{
			name: "one of two sessions held holds the whole submission",
			bookings: []*db.TrainingSubmission{
				booking(nil),
				booking(func(b *db.TrainingSubmission) { b.FoundMemberRecord = false }),
			},
			want: outcomeCheckingMembership,
		},
		{
			name:     "prices missing",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) { b.PricesFound = false })},
			want:     outcomeError,
		},
		{
			name: "membership hold takes priority over missing prices",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) {
				b.PricesFound = false
				b.LapsedMembership = true
			})},
			want: outcomeCheckingMembership,
		},
		{
			name:     "duplicate",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) { b.DuplicateOfId = "123-0" })},
			want:     outcomeAlreadyBooked,
		},
		{
			name: "dropped takes priority over membership",
			bookings: []*db.TrainingSubmission{booking(func(b *db.TrainingSubmission) {
				b.SubmissionState = db.DroppedSubmissionState
				b.FoundMemberRecord = false
			})},
			want: outcomeAlreadyBooked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideOutcome(tt.bookings); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAllProcessed(t *testing.T) {
	if allProcessed(nil) {
		t.Errorf("no bookings must not count as processed")
	}
	if !allProcessed([]*db.TrainingSubmission{booking(nil), booking(nil)}) {
		t.Errorf("expected processed")
	}
	if allProcessed([]*db.TrainingSubmission{booking(nil), booking(func(b *db.TrainingSubmission) { b.Processed = false })}) {
		t.Errorf("one unprocessed booking must make the set unprocessed")
	}
}

func TestOutcomePages(t *testing.T) {
	for _, outcome := range []redirectOutcome{outcomeReadyToPay, outcomeCheckingMembership, outcomeAlreadyBooked, outcomeError} {
		resp := htmlResponse(http.StatusOK, outcomePage(outcome), "club@example.org")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d", outcome, resp.StatusCode)
		}
		if !strings.HasPrefix(resp.Headers["Content-Type"], "text/html") {
			t.Errorf("%s: content type %q", outcome, resp.Headers["Content-Type"])
		}
		if !strings.Contains(resp.Body, outcomePage(outcome).Title) {
			t.Errorf("%s: title missing from page", outcome)
		}
	}

	checking := htmlResponse(http.StatusOK, outcomePage(outcomeCheckingMembership), "club@example.org").Body
	if !strings.Contains(checking, "checking your membership status") {
		t.Errorf("checking membership page is missing its explanation")
	}
	if !strings.Contains(checking, "mailto:club@example.org") {
		t.Errorf("checking membership page is missing the club's email address")
	}
}

func TestOutcomePage_EscapesClubEmail(t *testing.T) {
	body := htmlResponse(http.StatusOK, outcomePage(outcomeError), `<script>x</script>`).Body
	if strings.Contains(body, "<script>") {
		t.Fatalf("club email was not escaped")
	}
}

func TestRedirect_DisabledInBACSMode(t *testing.T) {
	paymentMode = paymentModeBACS

	// In BACS mode the endpoint must answer 404 without touching Jotform, the database or email
	resp, err := handleAPIRequest(events.LambdaFunctionURLRequest{
		RawPath: redirectPath,
		Body:    "submission_id=6695255243214467187",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 in BACS mode, got %d", resp.StatusCode)
	}
}

func TestParsePaymentMode(t *testing.T) {
	for in, want := range map[string]string{"": paymentModeBACS, "bacs": paymentModeBACS, "STRIPE": paymentModeStripe, " stripe ": paymentModeStripe} {
		got, err := parsePaymentMode(in)
		if err != nil || got != want {
			t.Errorf("parsePaymentMode(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := parsePaymentMode("card"); err == nil {
		t.Errorf("expected an error for an unknown payment mode")
	}
}

func TestParseTrainingFormID(t *testing.T) {
	got, err := parseTrainingFormID("")
	if err != nil || got != productionTrainingFormID {
		t.Errorf("empty value: got %d, %v; want the production form", got, err)
	}
	got, err = parseTrainingFormID("263062182377357")
	if err != nil || got != 263062182377357 {
		t.Errorf("dev form: got %d, %v", got, err)
	}
	for _, bad := range []string{"abc", "-1", "0"} {
		if _, err := parseTrainingFormID(bad); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}
