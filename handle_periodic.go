package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// handleHourly performs periodic tasks such as checking for missing submissions, sending payment reminders, and generating summaries.
func handleHourly(testMode bool) error {

	now := time.Now()
	// Get all training submissions here as they are used an a few places
	receivedSubmissions, err := trainTable.GetAllOfStateRecent(db.ReceivedSubmissionState, now)
	if err != nil {
		return err
	}
	receivedSubmissions, err = updateInPastSubmissions(receivedSubmissions)
	if err != nil {
		return fmt.Errorf("failed to update in-past submissions: %w", err)
	}

	paidSubmissions, err := trainTable.GetAllOfStateRecent(db.PaidSubmissionState, now)
	if err != nil {
		return err
	}
	paidSubmissions, err = updateInPastSubmissions(paidSubmissions)
	if err != nil {
		return fmt.Errorf("failed to update in-past submissions: %w", err)
	}

	submissions := append(receivedSubmissions, paidSubmissions...)
	log.Printf("Got %d received and %d paid submissions for future sessions",
		len(receivedSubmissions), len(paidSubmissions))

	err = handleSubmissionsCheck(submissions)
	if err != nil {
		return err
	}

	err = handlePayReminder(receivedSubmissions)
	if err != nil {
		return err
	}

	// Generate Training Summaries for today and the next two days
	aDay := time.Hour * 24
	until := dateOnly(now).Add(3 * aDay)
	if testMode == true {
		// Extend the summary period out to a month in test mode
		until = now.Add(time.Hour * 24 * 31)
	}

	// Email a summary of training submissions to the club email lunchtime on the day before
	if now.Hour() == 12 || testMode == true {
		err := handleTrainingSummary(submissions, until)
		if err != nil {
			return err
		}
	}

	// Email a summary of transactions to me on the 5th of the month at 10AM
	if (now.Day() == 5 && now.Hour() == 10) || testMode == true {
		err := handleTransactionsSummary()
		if err != nil {
			return err
		}
	}
	return nil
}

// handlePayReminder identifies submissions that haven't been paid for and sends reminder emails to the respective members.
func handlePayReminder(receivedSubmissions []*db.TrainingSubmission) error {
	for _, submission := range receivedSubmissions {
		if submission.ReceivedRequestEmailSent == false {
			// if the received request have not yet been sent there is no point sending a reminder
			continue
		}

		if submission.PayReminderEmailSent == true {
			continue
		}

		if submission.PaymentRecordId != "" {
			// Shouldn't ever get here
			continue
		}

		now := time.Now()

		// Don't send a reminder less than an hour after the request was submitted
		if submission.RequestDate.After(now.Add(-time.Hour)) {
			continue
		}

		linkedMemberRecords, linkedSubmissions, err := findSubmissionSet(submission.LinkedSubmissionIds, receivedSubmissions)
		if err != nil {
			return err
		}

		// Find the earliest of the linked submissions
		var earliestSubmission = linkedSubmissions[0]
		for _, linkedSubmission := range linkedSubmissions {
			if linkedSubmission.PayByDate.Before(earliestSubmission.PayByDate) {
				earliestSubmission = linkedSubmission
			}
		}

		if earliestSubmission.PayByDate.After(time.Now()) || earliestSubmission.PayReminderEmailSent == true {
			continue
		}

		// only send if the linked set all have valid members
		if len(linkedSubmissions) == len(linkedMemberRecords) {
			fmt.Printf("sending a reminder for payment of submission id %s and linked\n", earliestSubmission.GetID())

			emailHandler.SendPayReminder(linkedMemberRecords, linkedSubmissions)

			// update linked submissions
			for _, sub := range linkedSubmissions {
				sub.PayReminderEmailSent = true
			}
			err = trainTable.PutAll(linkedSubmissions)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func formatTime(t time.Time) string {
	hour := t.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	minute := t.Minute()
	ampm := t.Format("PM")

	return fmt.Sprintf("%d:%d%d %s", hour, minute/10, minute%10, ampm)
}

// TODO common custom formatting code
func formatCustomDate(t time.Time) string {
	return fmt.Sprintf("%s %s %s",
		t.Format("Monday"),
		dayWithSuffix(t.Day()),
		t.Format("January"),
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

func handleTrainingSummary(submissions []*db.TrainingSubmission, until time.Time) error {
	return writeEmails(until, submissions, memberTable.Get,
		func(subject, body string) {
			email := clubEmail
			if testMode == true {
				email = testEmail
			}
			emailHandler.SendEmail(email, subject, body)
		})
}

func dateOnly(timeDate time.Time) time.Time {
	y, m, d := timeDate.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, timeDate.Location())
}

// writeEmails generates and sends summary emails for upcoming training sessions, grouped by venue and time.
func writeEmails(until time.Time, submissions []*db.TrainingSubmission,
	getMember func(id string) (*db.MemberRecord, error),
	emailer func(subject, body string)) error {
	var err error
	today := dateOnly(time.Now())
	yesterday := today.AddDate(0, 0, -1)

	// Filter into a map of training dates
	sessionSubmissions := make(map[time.Time]map[time.Time][]*db.TrainingSubmission)

	for _, submission := range submissions {
		// Check for submissions before 36 hours from now
		tDate := submission.TrainingDate
		if tDate.Before(until) {
			dayKey := dateOnly(tDate)
			if sessionSubmissions[dayKey] == nil {
				sessionSubmissions[dayKey] = make(map[time.Time][]*db.TrainingSubmission)
			}
			sessionSubmissions[dayKey][tDate] =
				append(sessionSubmissions[dayKey][tDate], submission)
		}
	}

	type venueSummary struct {
		members      map[string]*db.MemberRecord
		messageLines []string
	}

	for tDate, daySubs := range sessionSubmissions {
		summaries := make(map[string]*venueSummary)
		for tTime, submissions := range daySubs {

			// Sort submissions: Paid first, then Incorrect Payment, then NOT PAID
			// Within each group, sort by request date
			sort.SliceStable(submissions, func(i, j int) bool {
				sI := submissions[i]
				sJ := submissions[j]

				rank := func(s *db.TrainingSubmission) int {
					if s.PaymentRecordId != "" {
						if !s.PaymentDiscrepancy {
							return 1 // Paid
						}
						return 2 // Incorrect Payment
					}
					return 3 // NOT PAID
				}

				rI := rank(sI)
				rJ := rank(sJ)
				if rI != rJ {
					return rI < rJ
				}

				return sI.RequestDate.Before(sJ.RequestDate)
			})

			for _, submission := range submissions {
				if submission.FoundMemberRecord == false {
					continue
				}

				if summaries[submission.Venue] == nil {
					summaries[submission.Venue] = &venueSummary{
						members:      map[string]*db.MemberRecord{},
						messageLines: []string{formatTime(tTime), ""},
					}
				}

				member := summaries[submission.Venue].members[submission.MembershipNumber]
				if member == nil {
					member, err = getMember(submission.MembershipNumber)
					if err != nil {
						return err
					}
					summaries[submission.Venue].members[submission.MembershipNumber] = member
				}

				notPaidString := ""
				if submission.PaymentRecordId == "" {
					notPaidString = " *NOT PAID*"
				} else if submission.PaymentDiscrepancy == true {
					notPaidString = " *Incorrect Payment*"
				}
				requestedLabel := ""
				requestDay := dateOnly(submission.RequestDate)
				if requestDay.Equal(today) {
					requestedLabel = " *Requested Today*"
				} else if requestDay.Equal(yesterday) {
					requestedLabel = " *Requested Yesterday*"
				}
				summaries[submission.Venue].messageLines = append(summaries[submission.Venue].messageLines,
					fmt.Sprintf(" %s %s riding %s%s",
						member.FirstName, member.LastName, submission.HorseName, notPaidString+requestedLabel))
			}
		}

		// Send an email with the summary for each training date/venue
		for venue, summary := range summaries {
			var builder strings.Builder
			_, _ = fmt.Fprintf(&builder, "Training requests summary for %s sessions\n\n", formatCustomDate(tDate))

			for _, line := range summary.messageLines {
				_, _ = fmt.Fprintf(&builder, "%s\n", line)
			}

			type futureEntry struct {
				member *db.MemberRecord
				lines  []string
			}
			futureByMember := make(map[string]*futureEntry)
			for _, submission := range submissions {
				if submission.Venue != venue {
					continue
				}
				if dateOnly(submission.TrainingDate).After(tDate) == false {
					continue
				}
				member := summary.members[submission.MembershipNumber]
				if member == nil {
					continue
				}
				entry := futureByMember[submission.MembershipNumber]
				if entry == nil {
					entry = &futureEntry{member: member}
					futureByMember[submission.MembershipNumber] = entry
				}
				entry.lines = append(entry.lines, fmt.Sprintf("%s %s riding %s",
					formatCustomDate(submission.TrainingDate),
					formatTime(submission.TrainingDate),
					submission.HorseName))
			}

			if len(futureByMember) > 0 {
				_, _ = fmt.Fprintf(&builder, "\nMembers with future training submissions at this venue:\n")
				for _, entry := range futureByMember {
					_, _ = fmt.Fprintf(&builder, " %s %s: %s\n",
						entry.member.FirstName,
						entry.member.LastName,
						strings.Join(entry.lines, "; "))
				}
			}

			_, _ = fmt.Fprintf(&builder, "\n\nMember email addresses\n")

			for _, member := range summary.members {
				_, _ = fmt.Fprintf(&builder, "%s; ", member.Email)
			}

			emailer(fmt.Sprintf("%s Training Request Summary", venue), builder.String())
		}
	}
	return nil
}
