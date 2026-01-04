package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

type PayReminderData struct {
	FirstName, Venue, TrainingDate, AccountNumber, SortCode, Reference, Amount string
}

type PayReminder2Data struct {
	FirstName, Venue, TrainingDate, Venue2, TrainingDate2, AccountNumber, SortCode, Reference, Amount string
}

func (eh *EmailHandler) SendPayReminder(members []*db.MemberRecord, submissions []*db.TrainingSubmission) {
	if len(members) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

	if len(submissions) == 1 {
		member := members[0]
		submission := submissions[0]
		eh.SendEmailPretty([]string{member.Email}, "pay-reminder", &PayReminderData{
			FirstName:     member.FirstName,
			Venue:         submission.Venue,
			TrainingDate:  formatCustomDateTime(submission.TrainingDate),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submission.PaymentReference,
			Amount:        formatAmount(submission.AmountPence),
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

		eh.SendEmailPretty(recipients, "pay-reminder2", &PayReminder2Data{
			FirstName:     firstNames,
			Venue:         submissions[0].Venue,
			TrainingDate:  formatCustomDateTime(submissions[0].TrainingDate),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDateTime(submissions[1].TrainingDate),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submissions[0].PaymentReference,
			Amount:        formatAmount(submissions[0].AmountPence + submissions[1].AmountPence),
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
