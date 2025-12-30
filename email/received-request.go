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

func formatAmount(amountPence int64) string {
	return fmt.Sprintf("%d.%d", amountPence/100, amountPence%100)
}

func (eh *EmailHandler) SendReceivedRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission) {
	if len(submissions) == 1 {
		member := members[0]
		submission := submissions[0]
		eh.SendEmailPretty([]string{member.Email}, "received-request", &ReceivedRequestData{
			FirstName:     member.FirstName,
			Venue:         submission.Venue,
			TrainingDate:  formatCustomDate(submission.Date),
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

		eh.SendEmailPretty(recipients, "received-request2", &ReceivedRequest2Data{
			FirstName:     firstNames,
			Venue:         submissions[0].Venue,
			TrainingDate:  formatCustomDate(submissions[0].Date),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDate(submissions[1].Date),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submissions[0].PaymentReference,
			Amount:        formatAmount(submissions[0].AmountPence + submissions[1].AmountPence),
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
