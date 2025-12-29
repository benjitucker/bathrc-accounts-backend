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
			"IP: %s, ",
		f.FormTitle,
		f.SubmissionID,
		f.Username,
		f.IP,
	)

	switch rr := f.RawRequest.(type) {

	case TrainingRawRequest:
		return header + fmt.Sprintf(
			"Training Form: "+
				"Member: %s; "+
				"Horse: %s; "+
				"Session: %s (%d mins); "+
				"Venue: %s; "+
				"Amount: %s",
			rr.MembershipNumber,
			rr.HorseName,
			rr.SelectSession.Date.Format(time.RFC1123),
			rr.SelectSession.Duration,
			rr.SelectedVenue,
			rr.Amount,
		)

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
