package email

import (
	"benjitucker/bathrc-accounts/db"
)

type ConfirmData struct {
	FirstName, Venue, TrainingDate string
}

func (eh *EmailHandler) SendConfirm(member *db.MemberRecord, submission *db.TrainingSubmission) {
	eh.SendEmailPretty([]string{member.Email}, "confirm", &ConfirmData{
		FirstName:    member.FirstName,
		Venue:        submission.Venue,
		TrainingDate: formatCustomDate(submission.Date),
	})
}
