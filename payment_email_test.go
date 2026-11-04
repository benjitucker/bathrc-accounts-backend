package main

import (
	"benjitucker/bathrc-accounts/db"
	"net/http"
	"strings"
	"testing"
)

func withPayLinkConfig(t *testing.T, base, secret string) {
	publicBaseURL, payLinkSecret = base, secret
	t.Cleanup(func() { publicBaseURL, payLinkSecret = "", "" })
}

func TestPayLinkDetailsFor(t *testing.T) {
	withPayLinkConfig(t, "https://abc.lambda-url.eu-west-3.on.aws", "secret")

	member := pricedBooking(nil)
	member.SetID("6696-0")
	lapsed := pricedBooking(func(b *db.TrainingSubmission) { b.ActualCurrMem = false })
	lapsed.SetID("6696-1")

	pay, err := payLinkDetailsFor([]*db.TrainingSubmission{member, lapsed})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pay.AmountPence != 2100+2600 {
		t.Errorf("amount %d", pay.AmountPence)
	}
	want := "https://abc.lambda-url.eu-west-3.on.aws/pay?s=6696&t=" + payLinkToken("secret", "6696")
	if pay.URL != want {
		t.Errorf("url %q, want %q", pay.URL, want)
	}
}

func TestPayLinkDetailsFor_Errors(t *testing.T) {
	b := pricedBooking(nil)
	b.SetID("6696-0")

	withPayLinkConfig(t, "", "secret")
	if _, err := payLinkDetailsFor([]*db.TrainingSubmission{b}); err == nil {
		t.Errorf("expected an error with no base URL")
	}

	withPayLinkConfig(t, "https://abc", "")
	if _, err := payLinkDetailsFor([]*db.TrainingSubmission{b}); err == nil {
		t.Errorf("expected an error with no link secret")
	}

	withPayLinkConfig(t, "https://abc", "secret")
	noPrices := pricedBooking(func(b *db.TrainingSubmission) { b.PricesFound = false })
	noPrices.SetID("6697-0")
	if _, err := payLinkDetailsFor([]*db.TrainingSubmission{noPrices}); err == nil || !strings.Contains(err.Error(), "no prices") {
		t.Errorf("expected a missing prices error, got %v", err)
	}
	if _, err := payLinkDetailsFor(nil); err == nil {
		t.Errorf("expected an error with no bookings")
	}
}

func TestRefreshedPriceText(t *testing.T) {
	if got := refreshedPriceText(false, false); got != "" {
		t.Errorf("no change should have no text, got %q", got)
	}
	if got := refreshedPriceText(true, false); !strings.Contains(got, "non-member price applies") {
		t.Errorf("lapsed: %q", got)
	}
	if got := refreshedPriceText(false, true); !strings.Contains(got, "member price applies") || strings.Contains(got, "non-member") {
		t.Errorf("now a member: %q", got)
	}
}

func TestTestModeRecipients_SkipsMissingMembers(t *testing.T) {
	testMode, testEmail, testEmail2 = true, "test1@example.org", "test2@example.org"
	t.Cleanup(func() { testMode, testEmail, testEmail2 = false, "", "" })

	// the booking code passes a two-slot list even for a single booking
	got := testModeRecipients([]*db.MemberRecord{{Email: "a@example.org"}, nil})
	if got[0].Email != "test1@example.org" || got[1] != nil {
		t.Errorf("got %+v", got)
	}
}

func TestRememberBaseURL(t *testing.T) {
	t.Cleanup(func() { publicBaseURL = "" })

	publicBaseURL = ""
	rememberBaseURL("abc.lambda-url.eu-west-3.on.aws")
	if publicBaseURL != "https://abc.lambda-url.eu-west-3.on.aws" {
		t.Errorf("got %q", publicBaseURL)
	}

	// a configured address is not replaced
	publicBaseURL = "https://configured.example"
	rememberBaseURL("other.example")
	if publicBaseURL != "https://configured.example" {
		t.Errorf("configured address replaced: %q", publicBaseURL)
	}
}

func TestNonPostToWebhookIsNotFound(t *testing.T) {
	withPaymentMode(t, paymentModeBACS)
	t.Cleanup(func() { publicBaseURL = "" })

	// e.g. a browser asking for /favicon.ico; must not reach the Jotform webhook decoder
	for _, path := range []string{"/favicon.ico", "/"} {
		resp, err := handleAPIRequest(functionURLRequest("GET", path, nil))
		if err != nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: got %d, %v; want 404", path, resp.StatusCode, err)
		}
	}
}
