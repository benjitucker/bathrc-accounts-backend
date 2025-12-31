package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"time"
)

type ReceivedRequestData struct {
	FirstName, Venue, TrainingDate, AccountNumber, SortCode, Reference, Amount, PayDate, ExtraText string
}

type ReceivedRequest2Data struct {
	FirstName, Venue, TrainingDate, Venue2, TrainingDate2, AccountNumber, SortCode, Reference, Amount, PayDate, ExtraText string
}

func formatAmount(amountPence int64) string {
	return fmt.Sprintf("%d.%02d", amountPence/100, amountPence%100)
}

func earliestDate(dates ...time.Time) time.Time {
	if len(dates) == 0 {
		return time.Time{}
	}

	earliest := dates[0]
	for _, d := range dates[1:] {
		if d.Before(earliest) {
			earliest = d
		}
	}
	return earliest
}

func (eh *EmailHandler) SendReceivedRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission, extraText string) {
	if len(submissions) == 1 {
		member := members[0]
		submission := submissions[0]
		eh.SendEmailPretty([]string{member.Email}, "received-request", &ReceivedRequestData{
			FirstName:     member.FirstName,
			Venue:         submission.Venue,
			TrainingDate:  formatCustomDateTime(submission.TrainingDate),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submission.PaymentReference,
			Amount:        formatAmount(submission.AmountPence),
			PayDate:       formatCustomDate(submission.PayByDate),
			ExtraText:     extraText,
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
			TrainingDate:  formatCustomDateTime(submissions[0].TrainingDate),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDateTime(submissions[1].TrainingDate),
			AccountNumber: eh.params.AccountNumber,
			SortCode:      eh.params.SortCode,
			Reference:     submissions[0].PaymentReference,
			Amount:        formatAmount(submissions[0].AmountPence + submissions[1].AmountPence),
			PayDate:       formatCustomDate(earliestDate(submissions[0].PayByDate, submissions[1].PayByDate)),
			ExtraText:     extraText,
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
