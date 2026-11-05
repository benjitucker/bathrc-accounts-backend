package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/email"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

// The post-submission redirect.
//
// The Training form's Thank You page redirects the member's browser here, with the
// submission sent as an HTTP POST. Only the submission ID is taken from that POST: the
// browser could have altered anything else, so the submission itself is fetched from the
// Jotform API. It is then processed (unless the webhook has already done so) and the member
// is shown what happens next.
//
// The endpoint is only enabled when PAYMENT_MODE is "stripe". A member whose booking is ready
// to pay is sent straight to a Stripe payment page.

const (
	redirectPath = "/redirect"

	// How long to wait for another invocation (normally the webhook) to finish processing
	// the same submission.
	processedWaitTimeout = 10 * time.Second
	processedPollPeriod  = 250 * time.Millisecond
)

type redirectOutcome int

const (
	outcomeReadyToPay redirectOutcome = iota
	outcomeCheckingMembership
	outcomeAlreadyBooked
	outcomeError
)

func (o redirectOutcome) String() string {
	switch o {
	case outcomeReadyToPay:
		return "ready to pay"
	case outcomeCheckingMembership:
		return "checking membership"
	case outcomeAlreadyBooked:
		return "already booked"
	default:
		return "error"
	}
}

// handleRedirectRequest handles a member's browser arriving after submitting the Training form.
func handleRedirectRequest(req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if paymentMode != paymentModeStripe {
		return htmlResponse(http.StatusNotFound, pageNotFound, clubEmail), nil
	}

	submissionId, err := submissionIDFromRedirect(req.Body, req.IsBase64Encoded)
	if err != nil {
		fmt.Printf("ERROR: redirect: %v\n", err)
		return htmlResponse(http.StatusOK, outcomePage(outcomeError), clubEmail), nil
	}

	outcome, bookings, err := redirectOutcomeFor(submissionId)
	if err != nil {
		fmt.Printf("ERROR: redirect for submission %s: %v\n", submissionId, err)
		emailHandler.SendEmail(testEmail, "jotform redirect: FAIL",
			fmt.Sprintf("submission %s: %v", submissionId, err))
		outcome = outcomeError
	}

	fmt.Printf("Redirect for submission %s: %s\n", submissionId, outcome)

	if outcome == outcomeReadyToPay && paymentStatusOf(bookings) == paymentDue {
		checkoutURL, err := startCheckout(submissionId, bookings, baseURL(req))
		if err != nil {
			return paymentErrorResponse(submissionId, err), nil
		}
		return redirectResponse(checkoutURL), nil
	}

	return htmlResponse(http.StatusOK, outcomePage(outcome), clubEmail), nil
}

// submissionIDFromRedirect extracts the Jotform submission ID from the form-encoded POST
// body that Jotform's "Redirect with HTTP POST" sends.
func submissionIDFromRedirect(body string, isBase64Encoded bool) (string, error) {
	if isBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return "", fmt.Errorf("failed to decode body: %w", err)
		}
		body = string(decoded)
	}

	values, err := url.ParseQuery(body)
	if err != nil {
		return "", fmt.Errorf("failed to parse body: %w", err)
	}

	submissionId := values.Get("submission_id")
	if submissionId == "" {
		return "", fmt.Errorf("no submission_id in redirect")
	}
	if _, err := strconv.ParseInt(submissionId, 10, 64); err != nil {
		return "", fmt.Errorf("invalid submission_id %q", submissionId)
	}
	return submissionId, nil
}

// redirectOutcomeFor processes the submission, if that hasn't happened already, and decides
// what the member sees next. It also returns the submission's bookings.
func redirectOutcomeFor(submissionId string) (redirectOutcome, []*db.TrainingSubmission, error) {
	sid, err := strconv.ParseInt(submissionId, 10, 64)
	if err != nil {
		return outcomeError, nil, err
	}

	// If we already have this submission (the webhook got there first, or the member has
	// reloaded the page), decide from our own records: the Jotform submission may since have
	// been deleted, for example as a duplicate.
	existing, err := readBookings(submissionId)
	if err != nil {
		return outcomeError, nil, err
	}

	if len(existing) == 0 {
		data, err := jotformClient.GetSubmission(sid)
		if err != nil {
			return outcomeError, nil, fmt.Errorf("failed to get submission from Jotform: %w", err)
		}

		var submission jotform_webhook.TrainingRawRequestWithID
		if err := json.Unmarshal(data, &submission); err != nil {
			return outcomeError, nil, fmt.Errorf("failed to parse submission from Jotform: %w", err)
		}

		if submission.FormID != strconv.FormatInt(trainingFormID, 10) {
			return outcomeError, nil, fmt.Errorf("submission belongs to form %q, not the training form %d",
				submission.FormID, trainingFormID)
		}

		if _, err := processTrainingRequest(submissionId, &submission); err != nil {
			return outcomeError, nil, err
		}
	}

	bookings, err := waitForProcessedBookings(submissionId)
	if err != nil {
		return outcomeError, nil, err
	}

	return decideOutcome(bookings), bookings, nil
}

// waitForProcessedBookings reads the booking records for a submission, waiting for another
// invocation to finish processing them if necessary.
func waitForProcessedBookings(submissionId string) ([]*db.TrainingSubmission, error) {
	deadline := time.Now().Add(processedWaitTimeout)
	for {
		bookings, err := readBookings(submissionId)
		if err != nil {
			return nil, err
		}
		if allProcessed(bookings) {
			return bookings, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for submission %s to be processed", submissionId)
		}
		time.Sleep(processedPollPeriod)
	}
}

// readBookings returns the booking records for a submission, or none if it has not been
// stored yet.
func readBookings(submissionId string) ([]*db.TrainingSubmission, error) {
	first, err := trainTable.Get(makeId(submissionId, 0))
	if err != nil {
		return nil, err
	}
	if first == nil {
		return nil, nil
	}

	var bookings []*db.TrainingSubmission
	for _, id := range first.LinkedSubmissionIds {
		booking, err := trainTable.Get(id)
		if err != nil {
			return nil, err
		}
		if booking == nil {
			// Not stored yet
			return nil, nil
		}
		bookings = append(bookings, booking)
	}
	return bookings, nil
}

func allProcessed(bookings []*db.TrainingSubmission) bool {
	if len(bookings) == 0 {
		return false
	}
	for _, booking := range bookings {
		if !booking.Processed {
			return false
		}
	}
	return true
}

// decideOutcome applies the booking rules to a processed submission set. All sessions in a
// submission are treated together.
func decideOutcome(bookings []*db.TrainingSubmission) redirectOutcome {
	if len(bookings) == 0 {
		return outcomeError
	}
	for _, booking := range bookings {
		if booking.SubmissionState == db.DroppedSubmissionState || booking.DuplicateOfId != "" {
			return outcomeAlreadyBooked
		}
	}
	for _, booking := range bookings {
		// Payment is held until the membership data has been refreshed: the member could
		// not be found, or claimed the member price but their membership has lapsed.
		if !booking.FoundMemberRecord || booking.LapsedMembership {
			return outcomeCheckingMembership
		}
	}
	for _, booking := range bookings {
		// Never ask for a payment without a price read from the form. The club has been
		// emailed; the member sees the general "we'll email you" page.
		if !booking.PricesFound {
			return outcomeError
		}
	}
	return outcomeReadyToPay
}

type page struct {
	Title      string
	Paragraphs []string
	// ContactClub adds a line with the club's email address, when it is known
	ContactClub bool
	// An optional button
	LinkText, LinkURL string
}

var pageNotFound = page{
	Title:      "Page not found",
	Paragraphs: []string{"This page isn’t available."},
}

var pagePaymentComplete = page{
	Title: "Thank you for your payment",
	Paragraphs: []string{
		"Your payment has been received. We’ll confirm your training booking by email shortly.",
	},
}

var pageAlreadyPaid = page{
	Title:      "Already paid",
	Paragraphs: []string{"This training booking has already been paid for. Thank you!"},
}

var pagePaymentNotPossible = page{
	Title: "This booking can’t be paid online",
	Paragraphs: []string{
		"This training booking is no longer open for payment, for example because the session has passed.",
		"Please contact the club if you think this is a mistake.",
	},
	ContactClub: true,
}

var pageInvalidPayLink = page{
	Title: "This payment link isn’t valid",
	Paragraphs: []string{
		"Please check that you’ve used the whole link from your email, or contact the club.",
	},
	ContactClub: true,
}

func paymentCancelledPage(payLink string) page {
	return page{
		Title: "Payment not completed",
		Paragraphs: []string{
			"Your training request has been received, but your place isn’t confirmed until it’s paid for.",
		},
		LinkText: "Pay now",
		LinkURL:  payLink,
	}
}

func outcomePage(outcome redirectOutcome) page {
	switch outcome {
	case outcomeReadyToPay:
		return page{
			Title: "Training request received",
			Paragraphs: []string{
				"Thank you, we’ve received your training request.",
				"Online payment isn’t available yet. You’ll receive an email with details of how to pay.",
			},
		}
	case outcomeCheckingMembership:
		return page{
			Title: "We’re checking your membership",
			Paragraphs: []string{
				"Thank you, we’ve received your training request.",
				"We’re checking your membership status and will email you when we’ve done that.",
				"If you haven’t heard from us within a few days, please contact the club.",
			},
			ContactClub: true,
		}
	case outcomeAlreadyBooked:
		return page{
			Title: "You’ve already requested this session",
			Paragraphs: []string{
				"It looks like you’ve already sent a request for this training session, so we haven’t booked it again.",
				"Please check your email for the details of your earlier request. If you think this is a mistake, please contact the club.",
			},
			ContactClub: true,
		}
	default:
		return page{
			Title: "Training request received",
			Paragraphs: []string{
				"Thank you, we’ve received your training request. We’ll email you shortly with the next steps.",
				"If you haven’t heard from us within a day, please contact the club.",
			},
			ContactClub: true,
		}
	}
}

// The pages match the club's emails: the logo on a white card over the club's deep red.
const (
	clubColour   = "#8B0707"
	accentColour = "#6e0303" // buttons
)

var pageTemplate = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Page.Title}} – Bath Riding Club</title>
<style>
  body { margin: 0; padding: 30px 12px; background: ` + clubColour + `; color: #333333;
         font-family: Arial, Helvetica, sans-serif; font-size: 16px; line-height: 1.6; }
  main { max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 8px; overflow: hidden;
         box-shadow: 0 4px 12px rgba(0,0,0,0.1); padding: 30px; box-sizing: border-box; }
  .logo { display: block; margin: 0 auto 24px; width: 100px; height: 100px; }
  h1 { font-size: 18px; font-weight: bold; margin: 0 0 20px; }
  p { margin: 0 0 20px; }
  a { color: ` + clubColour + `; }
  a.button { display: inline-block; padding: 12px 28px; border-radius: 6px; background: ` + accentColour + `;
             color: #ffffff; text-decoration: none; font-weight: bold; }
  a.button:hover, a.button:focus { background: ` + clubColour + `; }
  .club { margin: 30px 0 0; font-weight: bold; color: ` + clubColour + `; }
</style>
</head>
<body>
<main>
{{if .Logo}}<img class="logo" src="{{.Logo}}" alt="Bath Riding Club" width="100" height="100">
{{end}}<h1>{{.Page.Title}}</h1>
{{range .Page.Paragraphs}}<p>{{.}}</p>
{{end}}{{if .Page.LinkURL}}<p><a class="button" href="{{.Page.LinkURL}}">{{.Page.LinkText}}</a></p>
{{end}}{{if and .Page.ContactClub .ClubEmail}}<p>You can contact the club at <a href="mailto:{{.ClubEmail}}">{{.ClubEmail}}</a>.</p>
{{end}}<p class="club">Bath Riding Club</p>
</main>
</body>
</html>
`))

// logoDataURI is the club logo as a data URI, so the pages need no other requests.
var logoDataURI = sync.OnceValue(func() template.URL {
	logo, err := email.LogoPNG()
	if err != nil {
		fmt.Printf("ERROR: loading logo: %v\n", err)
		return ""
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(logo))
})

// htmlResponse renders a page as a function URL response.
func htmlResponse(statusCode int, p page, clubEmailAddress string) events.LambdaFunctionURLResponse {
	var buf bytes.Buffer
	err := pageTemplate.Execute(&buf, struct {
		Page      page
		ClubEmail string
		Logo      template.URL
	}{p, clubEmailAddress, logoDataURI()})
	if err != nil {
		fmt.Printf("ERROR: rendering page: %v\n", err)
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       http.StatusText(http.StatusInternalServerError),
		}
	}

	return events.LambdaFunctionURLResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":  "text/html; charset=utf-8",
			"Cache-Control": "no-store",
		},
		Body: buf.String(),
	}
}
