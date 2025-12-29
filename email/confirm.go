package email

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"time"
)

type ConfirmData struct {
	FirstName, Venue, TrainingDate string
}

func (eh *EmailHandler) SendConfirm(member *db.MemberRecord, submission *db.TrainingSubmission) {
	eh.SendEmailPretty(member.Email, "confirm", &ConfirmData{
		FirstName:    member.FirstName,
		Venue:        submission.Venue,
		TrainingDate: formatCustomDate(submission.Date),
	})
}

func formatCustomDate(t time.Time) string {
	hour := t.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	minute := t.Minute()
	ampm := t.Format("PM")

	return fmt.Sprintf("%s %s %s at %d:%d%d %s",
		t.Format("Monday"),
		dayWithSuffix(t.Day()),
		t.Format("January"),
		hour, minute/10, minute%10,
		ampm,
	)
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
