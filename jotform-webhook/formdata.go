package jotform_webhook

import (
	"encoding/json"
	"fmt"
	"time"
)

type FormData struct {
	Action       string
	WebhookURL   string
	Username     string
	FormID       string
	FormTitle    string
	SubmissionID string
	Pretty       string
	IP           string

	RawRequestStr string
	RawRequest    RawRequestPayload
}

func (f *FormData) String() string {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Sprintf("FormData marshal error: %v", err)
	}
	return string(b)
}

func (f *FormData) DebugString() string {
	header := fmt.Sprintf(
		"JotForm Submission: "+
			"FormTitle: %s; "+
			"SubmissionID: %s; "+
			"Username: %s; "+
			"IP: %s; ",
		f.FormTitle,
		f.SubmissionID,
		f.Username,
		f.IP,
	)

	switch rr := f.RawRequest.(type) {

	case TrainingRawRequest:
		out := header + fmt.Sprintf("Total: %s; PaymentRef: %s; ", rr.TotalAmount, rr.PaymentReference)
		for i, e := range rr.Entries {
			out += fmt.Sprintf(
				"Entry %d: %s %s %s %s %s; ",
				i+1,
				e.MembershipNumber,
				e.HorseName,
				e.Venue,
				e.SelectSession.StartLocal.Format(time.RFC1123),
				e.Amount,
			)
		}
		return out

	case TrainingAdminRawRequest:
		return header + fmt.Sprintf(
			"Training Administration: "+
				"Send Emails: %s; "+
				"Uploads: %v; "+
				"Submitted: %s",
			rr.SendEmailsNow,
			rr.UploadURLs,
			rr.SubmitDate.Time().Format(time.RFC1123),
		)

	default:
		return header + "Unknown rawRequest schema\n"
	}
}
