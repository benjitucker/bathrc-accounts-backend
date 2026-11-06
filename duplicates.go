package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/email"
	"fmt"
	"strings"
	"time"
)

// openForDuplicateCheck leaves out bookings whose payment deadline has passed (Stripe mode).
// They are about to be released by the hourly job, so a member whose time to pay ran out can
// book the same session again straight away rather than being told it is a duplicate.
func openForDuplicateCheck(bookings []*db.TrainingSubmission, now time.Time) []*db.TrainingSubmission {
	var open []*db.TrainingSubmission
	for _, booking := range bookings {
		if !booking.PaymentDueAt.IsZero() && now.After(booking.PaymentDueAt) {
			continue
		}
		open = append(open, booking)
	}
	return open
}

// sessionDescription describes a booking's session, e.g. "West Wilts on Sunday 8th November at 6:00 PM".
func sessionDescription(booking *db.TrainingSubmission) string {
	return fmt.Sprintf("%s on %s at %s", email.VenueDisplayName(booking.Venue),
		formatCustomDate(booking.TrainingDate), formatTime(booking.TrainingDate))
}

// alreadyBookedPage tells the member which session they had already requested and, for a
// request with more than one session, that none of its sessions have been booked.
func alreadyBookedPage(bookings []*db.TrainingSubmission) page {
	var duplicates, others []string
	for _, booking := range bookings {
		if booking.DuplicateOfId != "" {
			duplicates = append(duplicates, sessionDescription(booking))
		} else {
			others = append(others, sessionDescription(booking))
		}
	}
	if len(duplicates) == 0 {
		return outcomePage(outcomeAlreadyBooked)
	}

	p := page{
		Title: "You’ve already requested this session",
		Paragraphs: []string{
			fmt.Sprintf("You’ve already sent a request for %s, so we haven’t booked it again.", strings.Join(duplicates, " and ")),
		},
		ContactClub: true,
	}
	if len(others) > 0 {
		p.Paragraphs = append(p.Paragraphs, fmt.Sprintf(
			"Because of this, none of the sessions in this request have been booked, including %s. "+
				"Please send a new request for any session you still want.", strings.Join(others, " and ")))
	}
	p.Paragraphs = append(p.Paragraphs,
		"Please check your email for the details of your earlier request. If you think this is a mistake, please contact the club.")
	return p
}
