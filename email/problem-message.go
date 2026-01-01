package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

type ProblemMessageData struct {
	FirstName, Venue, TrainingDate, Description string
}

func (eh *EmailHandler) SendProblemMessage(members []*db.MemberRecord, submission *db.TrainingSubmission, description string) {
	if len(members) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

	// Assume max entry 2 submission
	var recipients []string
	var firstNames string
	if len(members) == 1 || members[0].GetID() == members[1].GetID() {
		recipients = append(recipients, members[0].Email)
		firstNames = members[0].FirstName
	} else {
		for _, member := range members {
			recipients = append(recipients, member.Email)
		}
		firstNames = fmt.Sprintf("%s and %s", members[0].FirstName, members[1].FirstName)
	}

	eh.SendEmailPretty(recipients, "problem-message", &ProblemMessageData{
		FirstName:    firstNames,
		Venue:        submission.Venue,
		TrainingDate: formatCustomDateTime(submission.TrainingDate),
		Description:  description,
	})
}
