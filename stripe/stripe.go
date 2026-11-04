// Package stripe is a small client for the parts of the Stripe API used for training
// payments: creating Checkout Sessions, and verifying and parsing webhook events.
//
// It calls Stripe's HTTP API directly and only reads fields that are stable across API
// versions (ids, urls, statuses, amounts and metadata).
package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.stripe.com/v1"

// Client calls the Stripe API with a secret or restricted key.
type Client struct {
	key        string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client for the given secret or restricted API key.
func NewClient(key string) *Client {
	return &Client{
		key:        key,
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// LineItem is one priced line on a Checkout page, in GBP.
type LineItem struct {
	Name        string
	Description string
	AmountPence int64
}

// CheckoutSessionParams describes a one-off payment.
type CheckoutSessionParams struct {
	SuccessURL        string
	CancelURL         string
	ClientReferenceID string
	CustomerEmail     string
	Description       string
	Metadata          map[string]string
	LineItems         []LineItem
	// Stripe requires 30 minutes to 24 hours from now
	ExpiresAt time.Time
}

// CheckoutSession holds the fields of a Checkout Session that this application uses.
type CheckoutSession struct {
	ID                string            `json:"id"`
	URL               string            `json:"url"`
	PaymentStatus     string            `json:"payment_status"`
	ClientReferenceID string            `json:"client_reference_id"`
	AmountTotal       int64             `json:"amount_total"`
	Currency          string            `json:"currency"`
	Metadata          map[string]string `json:"metadata"`
	// PaymentIntentID is the payment's id (pi_...), when Stripe has created one
	PaymentIntentID string `json:"-"`
}

func (s *CheckoutSession) UnmarshalJSON(b []byte) error {
	type alias CheckoutSession
	aux := struct {
		*alias
		PaymentIntent json.RawMessage `json:"payment_intent"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	// payment_intent is an id string, null, or an expanded object
	s.PaymentIntentID = ""
	if len(aux.PaymentIntent) > 0 && string(aux.PaymentIntent) != "null" {
		var id string
		if err := json.Unmarshal(aux.PaymentIntent, &id); err == nil {
			s.PaymentIntentID = id
		} else {
			var obj struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(aux.PaymentIntent, &obj); err == nil {
				s.PaymentIntentID = obj.ID
			}
		}
	}
	return nil
}

// form encodes the parameters as Stripe expects them.
func (p CheckoutSessionParams) form() (url.Values, error) {
	if len(p.LineItems) == 0 {
		return nil, errors.New("a checkout session needs at least one line item")
	}
	v := url.Values{}
	v.Set("mode", "payment")
	v.Set("success_url", p.SuccessURL)
	v.Set("cancel_url", p.CancelURL)
	if p.ClientReferenceID != "" {
		v.Set("client_reference_id", p.ClientReferenceID)
	}
	if p.CustomerEmail != "" {
		v.Set("customer_email", p.CustomerEmail)
	}
	if !p.ExpiresAt.IsZero() {
		v.Set("expires_at", strconv.FormatInt(p.ExpiresAt.Unix(), 10))
	}
	if p.Description != "" {
		v.Set("payment_intent_data[description]", p.Description)
	}
	for key, value := range p.Metadata {
		// on the session, and on the payment so it is visible in the Dashboard and on refunds
		v.Set("metadata["+key+"]", value)
		v.Set("payment_intent_data[metadata]["+key+"]", value)
	}
	for i, item := range p.LineItems {
		if item.AmountPence <= 0 {
			return nil, fmt.Errorf("line item %q has no amount", item.Name)
		}
		prefix := fmt.Sprintf("line_items[%d]", i)
		v.Set(prefix+"[quantity]", "1")
		v.Set(prefix+"[price_data][currency]", "gbp")
		v.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(item.AmountPence, 10))
		v.Set(prefix+"[price_data][product_data][name]", item.Name)
		if item.Description != "" {
			v.Set(prefix+"[price_data][product_data][description]", item.Description)
		}
	}
	return v, nil
}

// CreateCheckoutSession creates a Stripe-hosted payment page.
func (c *Client) CreateCheckoutSession(p CheckoutSessionParams) (*CheckoutSession, error) {
	if c.key == "" {
		return nil, errors.New("no Stripe API key configured")
	}
	form, err := p.form()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed reading stripe response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, apiError(resp.StatusCode, body)
	}

	var session CheckoutSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed parsing stripe response: %w", err)
	}
	if session.URL == "" {
		return nil, fmt.Errorf("stripe returned checkout session %q without a url", session.ID)
	}
	return &session, nil
}

func apiError(status int, body []byte) error {
	var e struct {
		Error struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return fmt.Errorf("stripe error (HTTP %d, %s %s): %s", status, e.Error.Type, e.Error.Code, e.Error.Message)
	}
	return fmt.Errorf("stripe error (HTTP %d)", status)
}

// Event is a webhook event.
type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
	Livemode bool `json:"livemode"`
}

// CheckoutSession returns the event's object as a Checkout Session.
func (e *Event) CheckoutSession() (*CheckoutSession, error) {
	var s CheckoutSession
	if err := json.Unmarshal(e.Data.Object, &s); err != nil {
		return nil, fmt.Errorf("event %s: failed parsing checkout session: %w", e.ID, err)
	}
	return &s, nil
}

// DefaultTolerance is how old a webhook signature may be, as recommended by Stripe.
const DefaultTolerance = 5 * time.Minute

// ErrInvalidSignature means the webhook did not come from Stripe, was altered, or is too old.
var ErrInvalidSignature = errors.New("invalid stripe webhook signature")

// VerifyEvent checks the Stripe-Signature header against the raw request body and returns
// the parsed event. The body must be exactly as received.
func VerifyEvent(payload []byte, signatureHeader, secret string, now time.Time, tolerance time.Duration) (*Event, error) {
	if secret == "" {
		return nil, fmt.Errorf("%w: no webhook secret configured", ErrInvalidSignature)
	}

	var timestamp int64
	var signatures []string
	for _, part := range strings.Split(signatureHeader, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			t, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: bad timestamp", ErrInvalidSignature)
			}
			timestamp = t
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == 0 || len(signatures) == 0 {
		return nil, fmt.Errorf("%w: missing timestamp or v1 signature", ErrInvalidSignature)
	}

	expected := computeSignature(timestamp, payload, secret)
	valid := false
	for _, sig := range signatures {
		decoded, err := hex.DecodeString(sig)
		if err == nil && hmac.Equal(decoded, expected) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("%w: no matching signature", ErrInvalidSignature)
	}

	age := now.Sub(time.Unix(timestamp, 0))
	if tolerance > 0 && (age > tolerance || age < -tolerance) {
		return nil, fmt.Errorf("%w: timestamp outside the tolerance", ErrInvalidSignature)
	}

	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed parsing event: %w", err)
	}
	return &event, nil
}

func computeSignature(timestamp int64, payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return mac.Sum(nil)
}

// SignatureHeader builds a Stripe-Signature header, for tests.
func SignatureHeader(payload []byte, secret string, timestamp time.Time) string {
	sig := computeSignature(timestamp.Unix(), payload, secret)
	return fmt.Sprintf("t=%d,v1=%s", timestamp.Unix(), hex.EncodeToString(sig))
}
