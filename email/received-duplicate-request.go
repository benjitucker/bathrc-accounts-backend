package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

// SendReceivedDuplicateRequest sends an acknowledgment email for one or more training requests where it was found to contain a duplicate
// of an existing training request (same member, venue, and training time)
func (eh *EmailHandler) SendReceivedDuplicateRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission, dupOfSubmissions []*db.TrainingSubmission) {
	if len(members) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

	if len(submissions) == 1 {
		member := members[0]
		submission := dupOfSubmissions[0]
		templateName := "received-duplicate-request"
		if submission.SubmissionState == db.PaidSubmissionState {
			templateName = "received-duplicate-paid-request"
		}
		eh.SendEmailPretty([]string{member.Email}, templateName, &ReceivedRequestData{
			FirstName:     member.FirstName,
			Venue:         submission.Venue,
			TrainingDate:  formatCustomDateTime(submission.TrainingDate),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submission.PaymentReference,
			Amount:        formatAmount(submission.AmountPence),
			PayDate:       formatCustomDate(submission.PayByDate),
		})
	} else if len(submissions) == 2 {
		// Assume entry 2 submission
		var recipients []string
		var firstNames string
		if members[0].GetID() == members[1].GetID() {
			recipients = append(recipients, members[0].Email)
			firstNames = members[0].FirstName
		} else {
			for _, member := range members {
				recipients = append(recipients, member.Email)
			}
			firstNames = fmt.Sprintf("%s and %s", members[0].FirstName, members[1].FirstName)
		}

		var unpaidDuplicate *db.TrainingSubmission
		for _, dupOfSub := range dupOfSubmissions {
			if dupOfSub.SubmissionState != db.PaidSubmissionState {
				unpaidDuplicate = dupOfSub
			}
		}

		templateName := "received-duplicate-paid-request2"
		if unpaidDuplicate != nil {
			templateName = "received-duplicate-request2"
		}

		eh.SendEmailPretty(recipients, templateName, &ReceivedRequest2Data{
			FirstName:     firstNames,
			Venue:         submissions[0].Venue,
			TrainingDate:  formatCustomDateTime(submissions[0].TrainingDate),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDateTime(submissions[1].TrainingDate),
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
