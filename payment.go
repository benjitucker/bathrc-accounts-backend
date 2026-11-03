package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/stripe"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"
)

// Stripe payments, used when PAYMENT_MODE is "stripe".
//
// The secrets are read from SSM parameters whose names are given by environment variables,
// so each environment uses its own: STRIPE_SECRET_KEY_PARAM (a restricted key that can write
// Checkout Sessions), STRIPE_WEBHOOK_SECRET_PARAM (the webhook endpoint's signing secret) and
// PAY_LINK_SECRET_PARAM (signs the payment links in emails and pages).

const (
	// How long a Stripe payment page stays usable. Stripe requires 30 minutes to 24 hours.
	checkoutSessionLifetime = time.Hour
)

var (
	stripeClient        *stripe.Client
	stripeWebhookSecret string
	payLinkSecret       string
)

// loadStripeConfig reads the Stripe secrets. Missing secrets are reported, not fatal: the
// pages that need them show an error and the club is emailed.
func loadStripeConfig() {
	readParam := func(envName string) string {
		name := os.Getenv(envName)
		if name == "" {
			fmt.Printf("ERROR: %s is not set\n", envName)
			return ""
		}
		return getSecret(name)
	}
	stripeClient = stripe.NewClient(readParam("STRIPE_SECRET_KEY_PARAM"))
	stripeWebhookSecret = readParam("STRIPE_WEBHOOK_SECRET_PARAM")
	payLinkSecret = readParam("PAY_LINK_SECRET_PARAM")
}

// payLinkToken signs a submission id, so payment links can't be made up for other bookings.
func payLinkToken(secret, submissionId string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("pay:" + submissionId))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func validPayLinkToken(secret, submissionId, token string) bool {
	if secret == "" || submissionId == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(payLinkToken(secret, submissionId)), []byte(token))
}

// payLinkURL is the link a member follows to pay for a booking. It creates a fresh Stripe
// payment page each time, so it never expires.
func payLinkURL(baseURL, submissionId string) string {
	return fmt.Sprintf("%s%s?s=%s&t=%s", baseURL, payPath, url.QueryEscape(submissionId),
		url.QueryEscape(payLinkToken(payLinkSecret, submissionId)))
}

// venueDisplayName turns a venue value such as "StockleyFarm" into "Stockley Farm".
func venueDisplayName(venue string) string {
	var b strings.Builder
	runes := []rune(venue)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) ||
			(i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1]))) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// checkoutLineItems describes each booking for the Stripe payment page, at the price to charge.
func checkoutLineItems(bookings []*db.TrainingSubmission) []stripe.LineItem {
	items := make([]stripe.LineItem, 0, len(bookings))
	for _, booking := range bookings {
		item := stripe.LineItem{
			Name: fmt.Sprintf("Training at %s, %s %s", venueDisplayName(booking.Venue),
				formatCustomDate(booking.TrainingDate), formatTime(booking.TrainingDate)),
			AmountPence: chargeAmountPence(booking),
		}
		if booking.HorseName != "" {
			item.Description = "Riding " + booking.HorseName
		}
		items = append(items, item)
	}
	return items
}

// paymentStatus summarises whether a submission's bookings can be paid for now.
type paymentStatus int

const (
	paymentDue paymentStatus = iota
	paymentAlreadyMade
	paymentNotPossible
)

func paymentStatusOf(bookings []*db.TrainingSubmission) paymentStatus {
	if len(bookings) == 0 {
		return paymentNotPossible
	}
	for _, booking := range bookings {
		if booking.PaymentRecordId != "" || booking.SubmissionState == db.PaidSubmissionState {
			return paymentAlreadyMade
		}
	}
	for _, booking := range bookings {
		if booking.SubmissionState != db.ReceivedSubmissionState {
			// dropped, or the session has passed
			return paymentNotPossible
		}
	}
	return paymentDue
}

// startCheckout creates a Stripe payment page for a submission's bookings and returns its URL.
// The bookings must already be ready to pay (see decideOutcome and paymentStatusOf).
func startCheckout(submissionId string, bookings []*db.TrainingSubmission, baseURL string) (string, error) {
	if stripeClient == nil {
		return "", fmt.Errorf("stripe is not configured")
	}

	customerEmail := ""
	if member, err := memberTable.Get(bookings[0].MembershipNumber); err == nil && member != nil {
		customerEmail = member.Email
		if testMode {
			customerEmail = testEmail
		}
	}

	session, err := stripeClient.CreateCheckoutSession(stripe.CheckoutSessionParams{
		SuccessURL:        baseURL + paymentCompletePath + "?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:         payLinkURL(baseURL, submissionId) + "&cancelled=1",
		ClientReferenceID: submissionId,
		CustomerEmail:     customerEmail,
		Description:       fmt.Sprintf("Training booking, ref %s", bookings[0].PaymentReference),
		Metadata: map[string]string{
			"submission_id":     submissionId,
			"payment_reference": bookings[0].PaymentReference,
		},
		LineItems: checkoutLineItems(bookings),
		ExpiresAt: time.Now().Add(checkoutSessionLifetime),
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("Created Stripe checkout session %s for submission %s\n", session.ID, submissionId)
	return session.URL, nil
}

// paymentApplied describes what recording a payment against a submission's bookings did.
type paymentApplied struct {
	// the bookings to save
	changed []*db.TrainingSubmission
	// problems for the club to look at, such as a payment that needs refunding
	problems []string
	// true when this payment was already recorded (a repeated webhook)
	alreadyRecorded bool
}

// applyPayment records a Stripe payment against a submission's bookings.
func applyPayment(bookings []*db.TrainingSubmission, paymentId string, amountPaidPence int64) paymentApplied {
	var result paymentApplied

	alreadyRecorded := len(bookings) > 0
	for _, booking := range bookings {
		if booking.PaymentRecordId != paymentId {
			alreadyRecorded = false
		}
	}
	if alreadyRecorded {
		result.alreadyRecorded = true
		return result
	}

	var expected int64
	for _, booking := range bookings {
		expected += chargeAmountPence(booking)

		if booking.PaymentRecordId != "" && booking.PaymentRecordId != paymentId {
			result.problems = append(result.problems, fmt.Sprintf(
				"Booking %s was already paid by %s, so payment %s is a duplicate and should be refunded.",
				booking.GetID(), booking.PaymentRecordId, paymentId))
			continue
		}

		booking.PaymentRecordId = paymentId
		switch booking.SubmissionState {
		case db.ReceivedSubmissionState:
			booking.SubmissionState = db.PaidSubmissionState
		case db.DroppedSubmissionState:
			result.problems = append(result.problems, fmt.Sprintf(
				"Booking %s was dropped but has been paid by %s, which should be refunded.", booking.GetID(), paymentId))
		case db.InPastSubmissionState:
			result.problems = append(result.problems, fmt.Sprintf(
				"Booking %s was paid by %s after the session date.", booking.GetID(), paymentId))
		}
		result.changed = append(result.changed, booking)
	}

	if amountPaidPence != expected {
		for _, booking := range result.changed {
			booking.PaymentDiscrepancy = true
		}
		result.problems = append(result.problems, fmt.Sprintf(
			"Payment %s was for £%d.%02d but the bookings cost £%d.%02d.",
			paymentId, amountPaidPence/100, amountPaidPence%100, expected/100, expected%100))
	}
	return result
}

// testModeRecipients replaces members' email addresses with the test addresses in test mode.
func testModeRecipients(members []*db.MemberRecord) []*db.MemberRecord {
	if !testMode {
		return members
	}
	result := make([]*db.MemberRecord, len(members))
	for i, member := range members {
		copied := *member
		if i == 0 {
			copied.Email = testEmail
		} else {
			copied.Email = testEmail2
		}
		result[i] = &copied
	}
	return result
}
