package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"errors"
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
		currentMembership := len(entry.CurrentMembershipSelection[0]) > 0

		submissions = append(submissions, &db.TrainingSubmission{
			Date:                     entry.SelectSession.StartLocal,
			DateUnix:                 entry.SelectSession.StartLocal.Unix(),
			PayByDate:                entry.SelectSession.StartLocal.Add(payBeforeSessionDuration),
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
			err2 := fmt.Errorf("no membership record for %s: %w", submission.MembershipNumber, err)
			emailHandler.SendEmail(testEmail, "jotform webhook: FAIL", err2.Error())

			submission.FoundMemberRecord = false
			// update
			err = trainTable.Put(submission, submission.GetID())
			if err != nil {
				err2 = errors.Join(err2, err)
				return err2
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

		// check that the membership is current, and flag inconsistency with the form data with member
		submission.ActualCurrMem = membershipDateCheck(
			memberRecord.MembershipValidFrom, memberRecord.MembershipValidTo, &submission.Date)

		if submission.ActualCurrMem != submission.RequestCurrMem {
			// email me on invalid membership incase it's a new membership
			err2 := fmt.Errorf("membership check for %s %s failed", memberRecord.FirstName, memberRecord.LastName)

			// update
			err = trainTable.Put(submission, submission.GetID())
			if err != nil {
				err2 = errors.Join(err2, err)
			}
			// TODO
			// Email membership inconsistency
			//emailHandler.SendEmail(memberRecord.Email,

			return err2
		}

		// TODO:
		// check that a training entry for the same date/time has not already been received
		// including the two submissions
		// Check for submissions by the same member on the same date and include that information in
		// the email

		// TODO:
		// check that number of requested (and paid) entries per session and reject the entry if
		// the numbers are two high

		// email member to confirm that their training entry has been received, pending payment
		// TODO pending payment

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
		emailHandler.SendReceivedRequest(memberRecords, submissions)
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
