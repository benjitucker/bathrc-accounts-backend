package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/stripe"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func TestPayLinkToken(t *testing.T) {
	token := payLinkToken("secret", "6695")
	if len(token) != 32 {
		t.Fatalf("token length %d", len(token))
	}
	if token != payLinkToken("secret", "6695") {
		t.Fatalf("token is not deterministic")
	}
	if !validPayLinkToken("secret", "6695", token) {
		t.Errorf("valid token rejected")
	}
	for name, ok := range map[string]bool{
		"other submission": validPayLinkToken("secret", "6696", token),
		"other secret":     validPayLinkToken("other", "6695", token),
		"no secret":        validPayLinkToken("", "6695", payLinkToken("", "6695")),
		"empty token":      validPayLinkToken("secret", "6695", ""),
		"altered token":    validPayLinkToken("secret", "6695", strings.Repeat("0", 32)),
	} {
		if ok {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestPayLinkURL(t *testing.T) {
	payLinkSecret = "secret"
	t.Cleanup(func() { payLinkSecret = "" })

	got := payLinkURL("https://example.org", "6695")
	want := "https://example.org/pay?s=6695&t=" + payLinkToken("secret", "6695")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestVenueDisplayName(t *testing.T) {
	for in, want := range map[string]string{
		"WestWilts":    "West Wilts",
		"StockleyFarm": "Stockley Farm",
		"Widbrook":     "Widbrook",
		"RiverMead":    "River Mead",
		"ABCFarm":      "ABC Farm",
		"BRC":          "BRC",
		"":             "",
	} {
		if got := venueDisplayName(in); got != want {
			t.Errorf("venueDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func pricedBooking(mutate func(b *db.TrainingSubmission)) *db.TrainingSubmission {
	return booking(func(b *db.TrainingSubmission) {
		b.Venue = "WestWilts"
		b.HorseName = "Ben"
		b.TrainingDate = time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
		b.MemberPricePence = 2100
		b.NonMemberPricePence = 2600
		b.ActualCurrMem = true
		b.RequestCurrMem = true
		if mutate != nil {
			mutate(b)
		}
	})
}

func TestCheckoutLineItems(t *testing.T) {
	items := checkoutLineItems([]*db.TrainingSubmission{
		pricedBooking(nil),
		pricedBooking(func(b *db.TrainingSubmission) { b.ActualCurrMem = false; b.HorseName = "" }),
	})
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Name != "Training at West Wilts, Thursday 5th November 7:00 PM" {
		t.Errorf("name %q", items[0].Name)
	}
	if items[0].AmountPence != 2100 || items[0].Description != "Riding Ben" {
		t.Errorf("member item %+v", items[0])
	}
	if items[1].AmountPence != 2600 || items[1].Description != "" {
		t.Errorf("non-member item %+v", items[1])
	}
}

func TestPaymentStatusOf(t *testing.T) {
	tests := []struct {
		name     string
		bookings []*db.TrainingSubmission
		want     paymentStatus
	}{
		{"none", nil, paymentNotPossible},
		{"due", []*db.TrainingSubmission{pricedBooking(nil)}, paymentDue},
		{"paid state", []*db.TrainingSubmission{pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.PaidSubmissionState })}, paymentAlreadyMade},
		{"payment recorded", []*db.TrainingSubmission{pricedBooking(func(b *db.TrainingSubmission) { b.PaymentRecordId = "pi_1" })}, paymentAlreadyMade},
		{"session passed", []*db.TrainingSubmission{pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.InPastSubmissionState })}, paymentNotPossible},
		{"dropped", []*db.TrainingSubmission{pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.DroppedSubmissionState })}, paymentNotPossible},
	}
	for _, tt := range tests {
		if got := paymentStatusOf(tt.bookings); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestApplyPayment_Normal(t *testing.T) {
	bookings := []*db.TrainingSubmission{
		pricedBooking(nil),
		pricedBooking(func(b *db.TrainingSubmission) { b.ActualCurrMem = false }),
	}
	result := applyPayment(bookings, "pi_1", 2100+2600)
	if result.alreadyRecorded || len(result.problems) != 0 || len(result.changed) != 2 {
		t.Fatalf("unexpected result %+v", result)
	}
	for _, b := range bookings {
		if b.SubmissionState != db.PaidSubmissionState || b.PaymentRecordId != "pi_1" || b.PaymentDiscrepancy {
			t.Errorf("booking not marked paid: %+v", b)
		}
	}

	// Stripe repeating the same event changes nothing
	again := applyPayment(bookings, "pi_1", 4700)
	if !again.alreadyRecorded || len(again.changed) != 0 {
		t.Errorf("repeat not recognised: %+v", again)
	}
}

func TestApplyPayment_Problems(t *testing.T) {
	t.Run("duplicate payment", func(t *testing.T) {
		b := pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.PaidSubmissionState; b.PaymentRecordId = "pi_1" })
		result := applyPayment([]*db.TrainingSubmission{b}, "pi_2", 2100)
		if b.PaymentRecordId != "pi_1" {
			t.Errorf("the original payment was overwritten")
		}
		if len(result.problems) != 1 || !strings.Contains(result.problems[0], "refunded") {
			t.Errorf("problems %v", result.problems)
		}
	})

	t.Run("dropped booking", func(t *testing.T) {
		b := pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.DroppedSubmissionState })
		result := applyPayment([]*db.TrainingSubmission{b}, "pi_3", 2100)
		if b.SubmissionState != db.DroppedSubmissionState || b.PaymentRecordId != "pi_3" {
			t.Errorf("booking %+v", b)
		}
		if len(result.problems) != 1 || !strings.Contains(result.problems[0], "dropped") {
			t.Errorf("problems %v", result.problems)
		}
	})

	t.Run("session passed", func(t *testing.T) {
		b := pricedBooking(func(b *db.TrainingSubmission) { b.SubmissionState = db.InPastSubmissionState })
		result := applyPayment([]*db.TrainingSubmission{b}, "pi_4", 2100)
		if b.SubmissionState != db.InPastSubmissionState {
			t.Errorf("state changed to %s", b.SubmissionState)
		}
		if len(result.problems) != 1 {
			t.Errorf("problems %v", result.problems)
		}
	})

	t.Run("wrong amount", func(t *testing.T) {
		b := pricedBooking(nil)
		result := applyPayment([]*db.TrainingSubmission{b}, "pi_5", 100)
		if !b.PaymentDiscrepancy || b.SubmissionState != db.PaidSubmissionState {
			t.Errorf("booking %+v", b)
		}
		if len(result.problems) != 1 || !strings.Contains(result.problems[0], "£1.00") || !strings.Contains(result.problems[0], "£21.00") {
			t.Errorf("problems %v", result.problems)
		}
	})
}

func functionURLRequest(method, path string, query map[string]string) events.LambdaFunctionURLRequest {
	req := events.LambdaFunctionURLRequest{RawPath: path, QueryStringParameters: query}
	req.RequestContext.HTTP.Method = method
	req.RequestContext.DomainName = "abc.lambda-url.eu-west-3.on.aws"
	return req
}

func withPaymentMode(t *testing.T, mode string) {
	paymentMode = mode
	t.Cleanup(func() { paymentMode = paymentModeBACS })
}

func TestPaymentEndpoints_DisabledInBACSMode(t *testing.T) {
	withPaymentMode(t, paymentModeBACS)
	for _, path := range []string{redirectPath, payPath, paymentCompletePath, stripeWebhookPath} {
		resp, err := handleAPIRequest(functionURLRequest("POST", path, nil))
		if err != nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: got %d, %v; want 404", path, resp.StatusCode, err)
		}
	}
}

func TestHandler_RoutesGETRequests(t *testing.T) {
	withPaymentMode(t, paymentModeBACS)

	// A GET has no body; it must still be treated as a function URL request
	raw, _ := json.Marshal(functionURLRequest("GET", payPath, map[string]string{"s": "1"}))
	out, err := handler(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp, ok := out.(events.LambdaFunctionURLResponse)
	if !ok || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %#v", out)
	}
}

func TestPayLink_InvalidAndCancelled(t *testing.T) {
	withPaymentMode(t, paymentModeStripe)
	payLinkSecret = "secret"
	t.Cleanup(func() { payLinkSecret = "" })

	// An invalid token is refused without touching the database
	resp, _ := handleAPIRequest(functionURLRequest("GET", payPath, map[string]string{"s": "6695", "t": "nope"}))
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(resp.Body, "isn’t valid") {
		t.Errorf("invalid link: %d %s", resp.StatusCode, resp.Body)
	}

	// Returning from Stripe unpaid shows a page with a link to try again
	token := payLinkToken("secret", "6695")
	resp, _ = handleAPIRequest(functionURLRequest("GET", payPath, map[string]string{"s": "6695", "t": token, "cancelled": "1"}))
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Body, "Payment not completed") {
		t.Fatalf("cancelled: %d %s", resp.StatusCode, resp.Body)
	}
	wantLink := "https://abc.lambda-url.eu-west-3.on.aws/pay?s=6695&amp;t=" + token
	if !strings.Contains(resp.Body, wantLink) {
		t.Errorf("cancelled page is missing the pay link %q", wantLink)
	}
}

func TestPaymentComplete(t *testing.T) {
	withPaymentMode(t, paymentModeStripe)
	resp, _ := handleAPIRequest(functionURLRequest("GET", paymentCompletePath, map[string]string{"session_id": "cs_1"}))
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Body, "Thank you for your payment") {
		t.Errorf("got %d %s", resp.StatusCode, resp.Body)
	}
}

func TestStripeWebhook_SignatureAndIgnoredEvents(t *testing.T) {
	withPaymentMode(t, paymentModeStripe)
	stripeWebhookSecret = "whsec_test"
	t.Cleanup(func() { stripeWebhookSecret = "" })

	payload := `{"id":"evt_1","type":"payment_intent.created","data":{"object":{}}}`

	// A bad signature is refused without touching the database
	req := functionURLRequest("POST", stripeWebhookPath, nil)
	req.Body = payload
	req.Headers = map[string]string{"stripe-signature": "t=1,v1=00"}
	resp, _ := handleAPIRequest(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad signature: got %d", resp.StatusCode)
	}

	// A correctly signed event of a type we don't use is acknowledged and ignored
	req.Headers = map[string]string{"stripe-signature": stripe.SignatureHeader([]byte(payload), "whsec_test", time.Now())}
	resp, _ = handleAPIRequest(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("ignored event: got %d", resp.StatusCode)
	}

	// A completed session that isn't paid yet (delayed payment) is acknowledged without recording anything
	unpaid := `{"id":"evt_2","type":"checkout.session.completed","data":{"object":{"id":"cs_1","payment_status":"unpaid","metadata":{"submission_id":"6695"}}}}`
	req.Body = unpaid
	req.Headers = map[string]string{"stripe-signature": stripe.SignatureHeader([]byte(unpaid), "whsec_test", time.Now())}
	resp, _ = handleAPIRequest(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unpaid session: got %d", resp.StatusCode)
	}
}
