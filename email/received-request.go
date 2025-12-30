package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

type ReceivedRequestData struct {
	FirstName, Venue, TrainingDate, AccountNumber, SortCode, Reference, Amount string
}

type ReceivedRequest2Data struct {
	FirstName, Venue, TrainingDate, Venue2, TrainingDate2, AccountNumber, SortCode, Reference, Amount string
}

func (eh *EmailHandler) SendReceivedRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission) {
	if len(submissions) == 1 {
		member := members[0]
		submission := submissions[0]
		eh.SendEmailPretty([]string{member.Email}, "received-request", &ReceivedRequestData{
			FirstName:    member.FirstName,
			Venue:        submission.Venue,
			TrainingDate: formatCustomDate(submission.Date),
		})
	} else {
		// Assume 2 submissions
		var recipients []string
		var firstNames string
		if members[0] == members[1] {
			recipients = append(recipients, members[0].Email)
			firstNames = members[0].FirstName
		} else {
			for _, member := range members {
				recipients = append(recipients, member.Email)
			}
			firstNames = fmt.Sprint("%s and %s", members[0].FirstName, members[1].FirstName)
		}

		eh.SendEmailPretty(recipients, "received-request2", &ReceivedRequest2Data{
			FirstName:     firstNames,
			Venue:         submissions[0].Venue,
			TrainingDate:  formatCustomDate(submissions[0].Date),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDate(submissions[1].Date),
		})
	}
}
