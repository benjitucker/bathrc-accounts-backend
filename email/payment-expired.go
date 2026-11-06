package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"strings"
)

type PaymentExpiredData struct {
	FirstName, Sessions, Deadline string
}

// SendPaymentExpired tells members that a booking was cancelled because it wasn't paid for by
// its deadline, and that the place has been released.
func (eh *EmailHandler) SendPaymentExpired(members []*db.MemberRecord, submissions []*db.TrainingSubmission, deadline string) {
	if len(members) == 0 || len(submissions) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

	var recipients []string
	firstNames := members[0].FirstName
	if len(members) == 1 || members[0].GetID() == members[1].GetID() {
		recipients = append(recipients, members[0].Email)
	} else {
		for _, member := range members {
			recipients = append(recipients, member.Email)
		}
		firstNames = fmt.Sprintf("%s and %s", members[0].FirstName, members[1].FirstName)
	}

	var sessions []string
	for _, submission := range submissions {
		sessions = append(sessions, fmt.Sprintf("a training session at %s on %s",
			VenueDisplayName(submission.Venue), formatCustomDateTime(submission.TrainingDate)))
	}

	eh.SendEmailPretty(recipients, "payment-expired", &PaymentExpiredData{
		FirstName: firstNames,
		Sessions:  strings.Join(sessions, " and "),
		Deadline:  deadline,
	})
}
