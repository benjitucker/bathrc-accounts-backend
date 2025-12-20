package jotform

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
		"JotForm Submission\n"+
			"FormTitle: %s\n"+
			"SubmissionID: %s\n"+
			"Username: %s\n"+
			"IP: %s\n",
		f.FormTitle,
		f.SubmissionID,
		f.Username,
		f.IP,
	)

	switch rr := f.RawRequest.(type) {

	case TrainingRawRequest:
		return header + fmt.Sprintf(
			"Training Form\n"+
				"Member: %s\n"+
				"Horse: %s\n"+
				"Session: %s (%d mins)\n"+
				"Venue: %s\n"+
				"Amount: %s\n",
			rr.MembershipNumber,
			rr.HorseName,
			rr.SelectSession.Date.Format(time.RFC1123),
			rr.SelectSession.Duration,
			rr.SelectedVenue,
			rr.Amount,
		)

	case TrainingAdminRawRequest:
		return header + fmt.Sprintf(
			"Training Administration\n"+
				"Send Emails: %s\n"+
				"Uploads: %v\n"+
				"Submitted: %s\n",
			rr.SendEmailsNow,
			rr.UploadURLs,
			rr.SubmitDate.Time().Format(time.RFC1123),
		)

	default:
		return header + "Unknown rawRequest schema\n"
	}
}
