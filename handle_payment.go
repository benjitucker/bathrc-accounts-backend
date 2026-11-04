package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/stripe"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	// A link to pay for a booking: creates a fresh Stripe payment page. Query: s (submission
	// id), t (token from payLinkToken), and cancelled=1 when returning from Stripe unpaid.
	payPath = "/pay"
	// Where Stripe sends the member after paying.
	paymentCompletePath = "/payment-complete"
	// Where Stripe sends payment events.
	stripeWebhookPath = "/stripe-webhook"
)

// baseURL is this function URL's address, taken from the request.
func baseURL(req events.LambdaFunctionURLRequest) string {
	return "https://" + req.RequestContext.DomainName
}

func redirectResponse(location string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusSeeOther,
		Headers: map[string]string{
			"Location":      location,
			"Cache-Control": "no-store",
		},
	}
}

// handlePayLink handles a member following a payment link.
func handlePayLink(req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	submissionId := req.QueryStringParameters["s"]
	token := req.QueryStringParameters["t"]
	if !validPayLinkToken(payLinkSecret, submissionId, token) {
		return htmlResponse(http.StatusBadRequest, pageInvalidPayLink, clubEmail), nil
	}

	link := payLinkURL(baseURL(req), submissionId)
	if req.QueryStringParameters["cancelled"] != "" {
		return htmlResponse(http.StatusOK, paymentCancelledPage(link), clubEmail), nil
	}

	bookings, err := readBookings(submissionId)
	if err != nil {
		return paymentErrorResponse(submissionId, err), nil
	}

	switch paymentStatusOf(bookings) {
	case paymentAlreadyMade:
		return htmlResponse(http.StatusOK, pageAlreadyPaid, clubEmail), nil
	case paymentNotPossible:
		return htmlResponse(http.StatusOK, pagePaymentNotPossible, clubEmail), nil
	}

	if outcome := decideOutcome(bookings); outcome != outcomeReadyToPay {
		return htmlResponse(http.StatusOK, outcomePage(outcome), clubEmail), nil
	}

	if paymentWindowClosed(bookings, time.Now()) {
		return htmlResponse(http.StatusOK, pagePaymentWindowClosed, clubEmail), nil
	}

	checkoutURL, err := startCheckout(submissionId, bookings, baseURL(req))
	if err != nil {
		return paymentErrorResponse(submissionId, err), nil
	}
	return redirectResponse(checkoutURL), nil
}

// paymentErrorResponse reports a failure to the club and shows the member the general page.
func paymentErrorResponse(submissionId string, err error) events.LambdaFunctionURLResponse {
	fmt.Printf("ERROR: payment for submission %s: %v\n", submissionId, err)
	emailHandler.SendEmail(testEmail, "Training: PAYMENT PAGE FAILED",
		fmt.Sprintf("Could not start a payment for submission %s: %v", submissionId, err))
	return htmlResponse(http.StatusOK, outcomePage(outcomeError), clubEmail)
}

// handlePaymentComplete shows the page Stripe returns the member to after paying. The booking
// is confirmed by the webhook, which also sends the confirmation email.
func handlePaymentComplete(req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return htmlResponse(http.StatusOK, pagePaymentComplete, clubEmail), nil
}

// handleStripeWebhook records payments reported by Stripe.
func handleStripeWebhook(req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	payload := []byte(req.Body)
	if req.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return plainResponse(http.StatusBadRequest, "bad body"), nil
		}
		payload = decoded
	}

	event, err := stripe.VerifyEvent(payload, req.Headers["stripe-signature"], stripeWebhookSecret, time.Now(), stripe.DefaultTolerance)
	if err != nil {
		fmt.Printf("ERROR: stripe webhook rejected: %v\n", err)
		if errors.Is(err, stripe.ErrInvalidSignature) {
			return plainResponse(http.StatusBadRequest, "invalid signature"), nil
		}
		return plainResponse(http.StatusBadRequest, "bad event"), nil
	}
	fmt.Printf("Stripe event %s: %s\n", event.ID, event.Type)

	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		session, err := event.CheckoutSession()
		if err != nil {
			return plainResponse(http.StatusBadRequest, "bad event"), nil
		}
		if session.PaymentStatus != "paid" {
			// a delayed payment method: wait for async_payment_succeeded
			fmt.Printf("Checkout session %s is %s, waiting for payment\n", session.ID, session.PaymentStatus)
			return plainResponse(http.StatusOK, "ok"), nil
		}
		if err := recordStripePayment(session); err != nil {
			// Stripe retries on an error response
			fmt.Printf("ERROR: recording payment for checkout session %s: %v\n", session.ID, err)
			return plainResponse(http.StatusInternalServerError, "error"), nil
		}

	case "checkout.session.async_payment_failed":
		session, err := event.CheckoutSession()
		if err == nil {
			emailHandler.SendEmail(testEmail, "Training: PAYMENT FAILED",
				fmt.Sprintf("A delayed payment failed for submission %s (checkout session %s).",
					submissionIdOf(session), session.ID))
		}
	}

	return plainResponse(http.StatusOK, "ok"), nil
}

func submissionIdOf(session *stripe.CheckoutSession) string {
	if id := session.Metadata["submission_id"]; id != "" {
		return id
	}
	return session.ClientReferenceID
}

// recordStripePayment marks a submission's bookings as paid and sends the confirmation email.
// An error means it should be retried.
func recordStripePayment(session *stripe.CheckoutSession) error {
	submissionId := submissionIdOf(session)
	paymentId := session.PaymentIntentID
	if paymentId == "" {
		paymentId = session.ID
	}

	first, err := trainTable.Get(makeId(submissionId, 0))
	if err != nil {
		return err
	}
	if first == nil {
		emailHandler.SendEmail(testEmail, "Training: PAYMENT NEEDS ATTENTION",
			fmt.Sprintf("Payment %s (£%d.%02d) is for submission %q, which has no bookings. It may need refunding.",
				paymentId, session.AmountTotal/100, session.AmountTotal%100, submissionId))
		return nil
	}

	members, bookings, err := findSubmissionSet(first.LinkedSubmissionIds, nil)
	if err != nil {
		return err
	}

	applied := applyPayment(bookings, paymentId, session.AmountTotal)
	if applied.alreadyRecorded {
		fmt.Printf("Payment %s for submission %s was already recorded\n", paymentId, submissionId)
		return nil
	}
	if strings.ToLower(session.Currency) != "gbp" {
		applied.problems = append(applied.problems, fmt.Sprintf("Payment %s was in %q, not GBP.", paymentId, session.Currency))
	}

	if len(applied.changed) > 0 {
		if err := trainTable.PutAll(applied.changed); err != nil {
			return err
		}
	}
	fmt.Printf("Recorded payment %s for submission %s\n", paymentId, submissionId)

	if len(applied.problems) > 0 {
		emailHandler.SendEmail(testEmail, "Training: PAYMENT NEEDS ATTENTION",
			fmt.Sprintf("Submission %s:\n\n%s", submissionId, strings.Join(applied.problems, "\n")))
	}

	paid := false
	for _, booking := range applied.changed {
		if booking.SubmissionState == db.PaidSubmissionState {
			paid = true
		}
	}
	if paid {
		emailHandler.SendReceivedPayment(members, bookings, nil)
	}
	return nil
}

func plainResponse(statusCode int, body string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body:       body,
	}
}
