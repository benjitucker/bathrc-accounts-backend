package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/email"
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

	// publicBaseURL is this Lambda's function URL, for links in emails. It comes from the SSM
	// parameter named by PUBLIC_BASE_URL_PARAM, or failing that from the first request received.
	publicBaseURL string
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
	if name := os.Getenv("PUBLIC_BASE_URL_PARAM"); name != "" {
		publicBaseURL = strings.TrimSuffix(getSecret(name), "/")
	}
}

// rememberBaseURL records this function URL's address from a request, if it isn't configured.
func rememberBaseURL(domainName string) {
	if publicBaseURL == "" && domainName != "" {
		publicBaseURL = "https://" + domainName
	}
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
			// keep real members' addresses out of Stripe test data
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

// payLinkDetailsFor is the payment link and amount for a submission's bookings.
func payLinkDetailsFor(bookings []*db.TrainingSubmission) (email.PayLinkDetails, error) {
	if len(bookings) == 0 {
		return email.PayLinkDetails{}, fmt.Errorf("no bookings")
	}
	if publicBaseURL == "" || payLinkSecret == "" {
		return email.PayLinkDetails{}, fmt.Errorf("payment links are not configured")
	}
	submissionId, _, err := parseId(bookings[0].GetID())
	if err != nil {
		return email.PayLinkDetails{}, err
	}
	var amount int64
	for _, booking := range bookings {
		if !booking.PricesFound {
			return email.PayLinkDetails{}, fmt.Errorf("booking %s has no prices", booking.GetID())
		}
		amount += chargeAmountPence(booking)
	}
	return email.PayLinkDetails{URL: payLinkURL(publicBaseURL, submissionId), AmountPence: amount}, nil
}

// The booking emails. In Stripe mode they carry a payment link; otherwise the bank transfer
// details, as before. (In test mode the email layer sends every email only to the test address.)

func emailReceivedRequest(members []*db.MemberRecord, bookings []*db.TrainingSubmission, extraText string) {
	if paymentMode != paymentModeStripe {
		emailHandler.SendReceivedRequest(members, bookings, extraText)
		return
	}
	pay, err := payLinkDetailsFor(bookings)
	if err != nil {
		reportPayLinkProblem(bookings, err)
		return
	}
	emailHandler.SendReceivedRequestPayLink(members, bookings, extraText, pay)
}

func emailPayReminder(members []*db.MemberRecord, bookings []*db.TrainingSubmission) {
	if paymentMode != paymentModeStripe {
		emailHandler.SendPayReminder(members, bookings)
		return
	}
	pay, err := payLinkDetailsFor(bookings)
	if err != nil {
		reportPayLinkProblem(bookings, err)
		return
	}
	emailHandler.SendPayReminderPayLink(members, bookings, pay)
}

// emailDuplicateRequest tells the member their request repeats an earlier one. In Stripe
// mode the payment link is for the earlier booking.
func emailDuplicateRequest(members []*db.MemberRecord, bookings, earlierBookings []*db.TrainingSubmission) {
	if paymentMode != paymentModeStripe {
		emailHandler.SendReceivedDuplicateRequest(members, bookings, earlierBookings)
		return
	}
	pay, err := payLinkDetailsFor(earlierBookings)
	if err != nil {
		// the earlier booking may be paid already, or not payable online; send without a link
		pay = email.PayLinkDetails{}
	}
	emailHandler.SendReceivedDuplicateRequestPayLink(members, bookings, earlierBookings, pay)
}

func reportPayLinkProblem(bookings []*db.TrainingSubmission, err error) {
	id := ""
	if len(bookings) > 0 {
		id = bookings[0].GetID()
	}
	fmt.Printf("ERROR: no payment link for booking %s: %v\n", id, err)
	emailHandler.SendEmail(testEmail, "Training: PAYMENT LINK FAILED",
		fmt.Sprintf("Could not email a payment link for booking %s, so the member has not been emailed: %v", id, err))
}

// releaseHeldBookings is called in Stripe mode when a membership refresh may release a held
// submission set. Each booking's membership is checked again against the new data, its price
// follows the result (member price if valid on the session date, otherwise non-member price),
// and the member is emailed a payment link.
func releaseHeldBookings(members []*db.MemberRecord, bookings []*db.TrainingSubmission, records []*db.MemberRecord) error {
	if len(bookings) != len(members) {
		// a rider in the set is still unknown; that rider's own check drops the set
		return nil
	}

	lapsed, nowMember := false, false
	for _, booking := range bookings {
		member := findMemberInRecords(booking.MembershipNumber, records)
		if member == nil {
			found, err := memberTable.Get(booking.MembershipNumber)
			if err != nil {
				return err
			}
			member = found
		}
		if member == nil {
			return nil
		}
		booking.FoundMemberRecord = true
		booking.ActualCurrMem = membershipDateCheck(member, &booking.TrainingDate)
		booking.LapsedMembership = false
		if booking.RequestCurrMem && !booking.ActualCurrMem {
			lapsed = true
		}
		if !booking.RequestCurrMem && booking.ActualCurrMem {
			nowMember = true
		}
	}

	emailReceivedRequest(members, bookings, refreshedPriceText(lapsed, nowMember))
	for _, booking := range bookings {
		booking.ReceivedRequestEmailSent = true
	}
	return trainTable.PutAll(bookings)
}

// refreshedPriceText explains a price that differs from the one the member asked for.
func refreshedPriceText(lapsed, nowMember bool) string {
	switch {
	case lapsed:
		return "Your membership doesn’t cover the date of the training session, so the non-member price applies. " +
			"If you have renewed your membership with Sport80 since booking, please contact the club before paying."
	case nowMember:
		return "Good news: your membership is valid on the date of the session, so the member price applies."
	}
	return ""
}
