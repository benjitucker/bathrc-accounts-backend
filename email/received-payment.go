package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

type ReceivedPaymentData struct {
	FirstName, Venue, TrainingDate, ExtraText1, ExtraText2, ExtraText3, ExtraText4 string
}

type ReceivedPayment2Data struct {
	FirstName, Venue, TrainingDate, Venue2, TrainingDate2, ExtraText1, ExtraText2, ExtraText3, ExtraText4 string
}

func (eh *EmailHandler) SendReceivedPayment(members []*db.MemberRecord, submissions []*db.TrainingSubmission, problemTexts []string) {
	if len(members) == 0 {
		fmt.Printf("Cannot send email, no valid membership numbers to send them too")
		return
	}

	var extraText1, extraText2, extraText3, extraText4 string
	if len(problemTexts) > 0 {
		extraText1 = "Please note:"
		for i, problemText := range problemTexts {
			problemText = " - " + problemText
			switch i {
			case 0:
				extraText2 = problemText
			case 1:
				extraText3 = problemText
			case 3:
				extraText4 = problemText
			}
		}
	}

	if len(submissions) == 1 {
		member := members[0]
		submission := submissions[0]
		eh.SendEmailPretty([]string{member.Email}, "received-payment", &ReceivedPaymentData{
			FirstName:    member.FirstName,
			Venue:        submission.Venue,
			TrainingDate: formatCustomDateTime(submission.TrainingDate),
			ExtraText1:   extraText1,
			ExtraText2:   extraText2,
			ExtraText3:   extraText3,
			ExtraText4:   extraText4,
		})
	} else if len(submissions) == 2 {
		// Assume entry 2 submission
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

		eh.SendEmailPretty(recipients, "received-payment2", &ReceivedPayment2Data{
			FirstName:     firstNames,
			Venue:         submissions[0].Venue,
			TrainingDate:  formatCustomDateTime(submissions[0].TrainingDate),
			Venue2:        submissions[1].Venue,
			TrainingDate2: formatCustomDateTime(submissions[1].TrainingDate),
			ExtraText1:    extraText1,
			ExtraText2:    extraText2,
			ExtraText3:    extraText3,
			ExtraText4:    extraText4,
		})
	} else {
		// TODO - more that 2 entry submission
	}
}
