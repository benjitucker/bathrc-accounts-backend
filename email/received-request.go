package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"time"
)

type ReceivedRequestData struct {
	FirstName, Venue, TrainingDate, AccountNumber, SortCode, Reference, Amount, PayDate, ExtraText string
	// PayLink, when set, replaces the bank transfer details with a payment link
	PayLink string
}

type ReceivedRequest2Data struct {
	FirstName, Venue, TrainingDate, Venue2, TrainingDate2, AccountNumber, SortCode, Reference, Amount, PayDate, ExtraText string
	// PayLink, when set, replaces the bank transfer details with a payment link
	PayLink string
}

// PayLinkDetails is an online payment link and the amount it asks for.
type PayLinkDetails struct {
	URL         string
	AmountPence int64
}

// amountText is the amount to show: the payment link's amount, or the bookings' total.
func amountText(pay *PayLinkDetails, submissions ...*db.TrainingSubmission) string {
	if pay != nil {
		return formatAmount(pay.AmountPence)
	}
	var total int64
	for _, submission := range submissions {
		total += submission.AmountPence
	}
	return formatAmount(total)
}

func payLinkURL(pay *PayLinkDetails) string {
	if pay == nil {
		return ""
	}
	return pay.URL
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

// SendReceivedRequest sends an acknowledgment email for one or more training requests, including payment instructions.
func (eh *EmailHandler) SendReceivedRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission, extraText string) {
	eh.sendReceivedRequest(members, submissions, extraText, nil)
}

// SendReceivedRequestPayLink sends the acknowledgment email with an online payment link instead of bank details.
func (eh *EmailHandler) SendReceivedRequestPayLink(members []*db.MemberRecord, submissions []*db.TrainingSubmission, extraText string, pay PayLinkDetails) {
	eh.sendReceivedRequest(members, submissions, extraText, &pay)
}

func (eh *EmailHandler) sendReceivedRequest(members []*db.MemberRecord, submissions []*db.TrainingSubmission, extraText string, pay *PayLinkDetails) {
	if len(members) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

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
			Amount:        amountText(pay, submission),
			PayDate:       formatCustomDate(submission.PayByDate),
			ExtraText:     extraText,
			PayLink:       payLinkURL(pay),
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
			Amount:        amountText(pay, submissions[0], submissions[1]),
			PayDate:       formatCustomDate(earliestDate(submissions[0].PayByDate, submissions[1].PayByDate)),
			ExtraText:     extraText,
			PayLink:       payLinkURL(pay),
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
