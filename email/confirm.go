package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
)

type ConfirmData struct {
	FirstName, Venue, TrainingDate string
}

func (eh *EmailHandler) SendConfirm(member *db.MemberRecord, submission *db.TrainingSubmission) {
	t := submission.Date
	formatted := fmt.Sprintf("%s %s %s at %d%d%dPM",
		t.Format("Monday"),
		dayWithSuffix(t.Day()),
		t.Format("January"),
		t.Hour()%12,
		t.Minute()/10,
		t.Minute()%10)

	eh.SendEmailPretty(member.Email, "confirm", &ConfirmData{
		FirstName:    member.FirstName,
		Venue:        submission.Venue,
		TrainingDate: formatted,
	})
}

func dayWithSuffix(day int) string {
	if day >= 11 && day <= 13 {
		return fmt.Sprintf("%dth", day)
	}
	switch day % 10 {
	case 1:
		return fmt.Sprintf("%dst", day)
	case 2:
		return fmt.Sprintf("%dnd", day)
	case 3:
		return fmt.Sprintf("%drd", day)
	default:
		return fmt.Sprintf("%dth", day)
	}
}
