package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"strings"
	"time"
)

// releaseExpiredBookings releases bookings that weren't paid for in time (Stripe mode): their
// state becomes Dropped, the Jotform submission is deleted so the place can be booked again,
// and the member is told. It returns the received bookings that are still open.
func releaseExpiredBookings(received []*db.TrainingSubmission, now time.Time) ([]*db.TrainingSubmission, error) {
	var problems []string
	for _, submissionId := range expiredUnpaidSubmissions(received, now) {
		first, err := trainTable.Get(makeId(submissionId, 0))
		if err != nil {
			return nil, err
		}
		if first == nil {
			continue
		}
		members, bookings, err := findSubmissionSet(first.LinkedSubmissionIds, received)
		if err != nil {
			return nil, err
		}
		// Check the whole set, in case part of it wasn't in the list given
		if !setExpiredUnpaid(bookings, now) {
			continue
		}

		deadline := paymentDeadline(bookings)
		for _, booking := range bookings {
			booking.SubmissionState = db.DroppedSubmissionState
		}
		if err := trainTable.PutAll(bookings); err != nil {
			return nil, err
		}
		fmt.Printf("Released unpaid submission %s (payment was due by %s)\n", submissionId, deadline.Format(time.RFC3339))

		// Free the place in Jotform
		if err := dropJotformTrainingSubmission(submissionId); err != nil {
			problems = append(problems, fmt.Sprintf("Submission %s: %v", submissionId, err))
		}

		emailHandler.SendPaymentExpired(members, bookings, formatDeadline(deadline))
	}

	if len(problems) > 0 {
		emailHandler.SendEmail(testEmail, "Training: UNPAID BOOKING NOT REMOVED FROM JOTFORM",
			"These unpaid bookings were released, but their Jotform submissions couldn't be deleted, so the places "+
				"may still look taken in the form:\n\n"+strings.Join(problems, "\n"))
	}

	var stillOpen []*db.TrainingSubmission
	for _, booking := range received {
		if booking.SubmissionState == db.ReceivedSubmissionState {
			stillOpen = append(stillOpen, booking)
		}
	}
	return stillOpen, nil
}
