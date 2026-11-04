package stripe

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCreateCheckoutSession_Request(t *testing.T) {
	var gotForm url.Values
	var gotAuth, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Write([]byte(`{"id":"cs_test_1","url":"https://checkout.stripe.com/c/pay/cs_test_1","payment_status":"unpaid","payment_intent":null,"amount_total":4700,"currency":"gbp"}`))
	}))
	defer server.Close()

	c := NewClient("rk_test_abc")
	c.BaseURL = server.URL
	expires := time.Unix(1800000000, 0)

	session, err := c.CreateCheckoutSession(CheckoutSessionParams{
		SuccessURL:        "https://example.org/payment-complete?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:         "https://example.org/payment-cancelled?s=1&t=x",
		ClientReferenceID: "6695",
		CustomerEmail:     "rider@example.org",
		Description:       "Training booking 6695",
		Metadata:          map[string]string{"submission_id": "6695"},
		ExpiresAt:         expires,
		LineItems: []LineItem{
			{Name: "Training: West Wilts", Description: "Horse: Ben", AmountPence: 2100},
			{Name: "Training: Widbrook", AmountPence: 2600},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "cs_test_1" || session.URL == "" {
		t.Fatalf("unexpected session %+v", session)
	}

	if gotAuth != "Bearer rk_test_abc" {
		t.Errorf("authorization header %q", gotAuth)
	}
	if gotPath != "/checkout/sessions" {
		t.Errorf("path %q", gotPath)
	}
	want := map[string]string{
		"mode":                    "payment",
		"success_url":             "https://example.org/payment-complete?session_id={CHECKOUT_SESSION_ID}",
		"cancel_url":              "https://example.org/payment-cancelled?s=1&t=x",
		"client_reference_id":     "6695",
		"customer_email":          "rider@example.org",
		"expires_at":              "1800000000",
		"metadata[submission_id]": "6695",
		"payment_intent_data[metadata][submission_id]":         "6695",
		"payment_intent_data[description]":                     "Training booking 6695",
		"line_items[0][quantity]":                              "1",
		"line_items[0][price_data][currency]":                  "gbp",
		"line_items[0][price_data][unit_amount]":               "2100",
		"line_items[0][price_data][product_data][name]":        "Training: West Wilts",
		"line_items[0][price_data][product_data][description]": "Horse: Ben",
		"line_items[1][price_data][unit_amount]":               "2600",
	}
	for key, value := range want {
		if got := gotForm.Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
	if _, ok := gotForm["line_items[1][price_data][product_data][description]"]; ok {
		t.Errorf("empty description should not be sent")
	}
}

func TestCreateCheckoutSession_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"type":"invalid_request_error","code":"secret_key_required","message":"This API call cannot be made with a restricted key"}}`))
	}))
	defer server.Close()

	c := NewClient("rk_test_abc")
	c.BaseURL = server.URL
	_, err := c.CreateCheckoutSession(CheckoutSessionParams{LineItems: []LineItem{{Name: "x", AmountPence: 100}}})
	if err == nil || !strings.Contains(err.Error(), "restricted key") || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected the Stripe error message, got %v", err)
	}
}

func TestCreateCheckoutSession_Validation(t *testing.T) {
	c := NewClient("rk_test_abc")
	if _, err := c.CreateCheckoutSession(CheckoutSessionParams{}); err == nil {
		t.Errorf("expected an error with no line items")
	}
	if _, err := c.CreateCheckoutSession(CheckoutSessionParams{LineItems: []LineItem{{Name: "x"}}}); err == nil {
		t.Errorf("expected an error with a zero amount")
	}
	if _, err := NewClient("").CreateCheckoutSession(CheckoutSessionParams{LineItems: []LineItem{{Name: "x", AmountPence: 1}}}); err == nil {
		t.Errorf("expected an error with no key")
	}
}

func TestCheckoutSession_PaymentIntentForms(t *testing.T) {
	for in, want := range map[string]string{
		`{"id":"cs_1","payment_intent":"pi_123"}`:        "pi_123",
		`{"id":"cs_1","payment_intent":{"id":"pi_456"}}`: "pi_456",
		`{"id":"cs_1","payment_intent":null}`:            "",
		`{"id":"cs_1"}`:                                  "",
	} {
		var s CheckoutSession
		if err := json.Unmarshal([]byte(in), &s); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if s.PaymentIntentID != want || s.ID != "cs_1" {
			t.Errorf("%s: got %q", in, s.PaymentIntentID)
		}
	}
}

const sampleEvent = `{"id":"evt_1","type":"checkout.session.completed","livemode":false,"data":{"object":{"id":"cs_test_1","payment_status":"paid","client_reference_id":"6695","amount_total":2100,"currency":"gbp","payment_intent":"pi_1","metadata":{"submission_id":"6695"}}}}`

func TestVerifyEvent(t *testing.T) {
	secret := "whsec_test_secret"
	now := time.Unix(1800000000, 0)
	payload := []byte(sampleEvent)
	header := SignatureHeader(payload, secret, now)

	event, err := VerifyEvent(payload, header, secret, now.Add(time.Minute), DefaultTolerance)
	if err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if event.Type != "checkout.session.completed" {
		t.Fatalf("type %q", event.Type)
	}
	session, err := event.CheckoutSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if session.PaymentStatus != "paid" || session.PaymentIntentID != "pi_1" || session.Metadata["submission_id"] != "6695" || session.AmountTotal != 2100 {
		t.Fatalf("unexpected session %+v", session)
	}

	// Extra v0 and an unrelated v1 alongside the right one are fine
	if _, err := VerifyEvent(payload, header+",v1=00ff,v0=abc", secret, now, DefaultTolerance); err != nil {
		t.Errorf("rejected with extra signatures: %v", err)
	}
}

func TestVerifyEvent_Rejections(t *testing.T) {
	secret := "whsec_test_secret"
	now := time.Unix(1800000000, 0)
	payload := []byte(sampleEvent)
	header := SignatureHeader(payload, secret, now)

	tests := map[string]func() error{
		"altered body": func() error {
			_, err := VerifyEvent([]byte(strings.Replace(sampleEvent, "2100", "1", 1)), header, secret, now, DefaultTolerance)
			return err
		},
		"wrong secret": func() error {
			_, err := VerifyEvent(payload, header, "whsec_other", now, DefaultTolerance)
			return err
		},
		"too old": func() error {
			_, err := VerifyEvent(payload, header, secret, now.Add(10*time.Minute), DefaultTolerance)
			return err
		},
		"no secret configured": func() error {
			_, err := VerifyEvent(payload, header, "", now, DefaultTolerance)
			return err
		},
		"missing header": func() error {
			_, err := VerifyEvent(payload, "", secret, now, DefaultTolerance)
			return err
		},
		"garbage header": func() error {
			_, err := VerifyEvent(payload, "t=abc,v1=zz", secret, now, DefaultTolerance)
			return err
		},
	}
	for name, run := range tests {
		if err := run(); !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("%s: expected ErrInvalidSignature, got %v", name, err)
		}
	}
}
