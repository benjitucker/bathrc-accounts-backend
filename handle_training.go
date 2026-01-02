package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	payBeforeSessionDuration = time.Hour * -36
)

func makeId(formData *jotform_webhook.FormData, entryIndex int) string {
	return fmt.Sprintf("%s-%d", formData.SubmissionID, entryIndex)
}

func handleTrainingRequest(formData *jotform_webhook.FormData, request jotform_webhook.TrainingRawRequest) error {

	var submissions []*db.TrainingSubmission

	for _, entry := range request.Entries {

		amount, err := strconv.ParseFloat(entry.Amount, 64)
		if err != nil {
			return fmt.Errorf("amount float number (%s): %w", entry.Amount, err)
		}
		amountPence := math.Floor(amount * 100)
		requestDate := time.Time(request.SubmitDate)
		currentMembership := len(entry.CurrentMembershipSelection) > 0 &&
			len(entry.CurrentMembershipSelection[0]) > 0

		submissions = append(submissions, &db.TrainingSubmission{
			SubmissionState:          db.ReceivedSubmissionState,
			TrainingDate:             entry.SelectSession.StartLocal,
			DateUnix:                 entry.SelectSession.StartLocal.Unix(),
			PayByDate:                entry.SelectSession.StartLocal.Add(payBeforeSessionDuration),
			PaidFee:                  false,
			MembershipNumber:         strings.Trim(entry.MembershipNumber, " "),
			RequestCurrMem:           currentMembership,
			Venue:                    entry.Venue,
			AmountPence:              int64(amountPence),
			HorseName:                entry.HorseName,
			RequestDate:              requestDate,
			RequestDateUnix:          requestDate.Unix(),
			PaymentReference:         request.PaymentReference,
			FoundMemberRecord:        true,
			AlreadyBooked:            false,
			ReceivedRequestEmailSent: true, // Assume that it will be
		})
	}

	memberRecords := make([]*db.MemberRecord, 2)
	sendReceivedRequestEmail := true

	for entryIndex, submission := range submissions {
		// fill the cross-references
		for i := range submissions {
			submission.LinkedSubmissionIds =
				append(submission.LinkedSubmissionIds, makeId(formData, i))
		}

		err := trainTable.Put(submission, makeId(formData, entryIndex))
		if err != nil {
			return err
		}

		// Check membership number
		memberRecord, err := memberTable.Get(submission.MembershipNumber)
		if memberRecord == nil || err != nil {
			// if not all members are found, dont send an email at this time at all
			sendReceivedRequestEmail = false

			// email me on invalid membership number incase it's a new member
			emailHandler.SendEmail(testEmail, "Training: REFRESH MEMBERSHIP",
				fmt.Sprintf("no membership record (%s)", submission.MembershipNumber))

			submission.FoundMemberRecord = false
			// update
			err = trainTable.Put(submission, submission.GetID())
			if err != nil {
				return err
			}
			continue
		}

		// Use test email addresses if in test mode
		if testMode {
			if entryIndex == 0 {
				memberRecord.Email = testEmail
			} else {
				memberRecord.Email = testEmail2
			}
		}

		memberRecords[entryIndex] = memberRecord

		// check that the membership is current, send me a message if not. Inconsistency with the form data
		// will be flagged with member when update membership data is received
		submission.ActualCurrMem = membershipDateCheck(
			memberRecord.MembershipValidFrom, memberRecord.MembershipValidTo, &submission.TrainingDate)

		if submission.ActualCurrMem != submission.RequestCurrMem {

			if submission.RequestCurrMem == true {
				// if any of the requests claim membership but don't have it, don't send emails at this
				// time as the membership may have just been renewed
				sendReceivedRequestEmail = false

				// email me on expired membership incase it's just been renewed
				emailHandler.SendEmail(testEmail, "Training: REFRESH MEMBERSHIP",
					fmt.Sprintf("membership check for %s %s (%s) failed",
						memberRecord.FirstName, memberRecord.LastName, memberRecord.MemberNumber))
			}

			// update
			err = trainTable.Put(submission, submission.GetID())
			if err != nil {
				return err
			}

			continue
		}

		// TODO:
		// check that a training entry for the same date/time has not already been received
		// including the two submissions
		// Check for submissions by the same member on the same date and include that information in
		// the email

		// TODO:
		// check that number of requested (and paid) entries per session and reject the entry if
		// the numbers are two high

		// TODO:
		// When numbers are filled by paid submissions, email anyone that has not paid and explain
		// they are too late.

		// TODO:
		// Send payment reminder email to people who have not paid

		// TODO:
		// Send the list of training submissions (including who has paid) to bathrc@hotmail.co.uk
		// mid day the day before

		/* TODO remove:
		records, err := trainTable.GetAll()
		if err != nil {
			return err
		}

		_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
		for _, record := range records {
			_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
		}
		*/
	}

	if sendReceivedRequestEmail {
		emailHandler.SendReceivedRequest(memberRecords, submissions, "")
	} else {
		for _, submission := range submissions {
			submission.ReceivedRequestEmailSent = false
			err := trainTable.Put(submission, submission.GetID())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func membershipDateCheck(start, end, target *time.Time) bool {
	if start == nil {
		return false
	}
	if end == nil {
		return false
	}
	if target == nil {
		return false
	}
	return (target.Equal(*start) || target.After(*start)) &&
		(target.Equal(*end) || target.Before(*end))
}
