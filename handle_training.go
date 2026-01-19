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
	payBeforeSessionDuration    = time.Hour * -36
	trainingSubmissionRecordTTL = time.Hour * 24 * 365 * 2 // Keep for 2 years
)

func makeId(submissionId string, entryIndex int) string {
	return fmt.Sprintf("%s-%d", submissionId, entryIndex)
}

// handleTrainingRequest processes a single training request submission from Jotform, creating multiple database entries if needed.
func handleTrainingRequest(submissionId string, request jotform_webhook.TrainingRequest) error {

	var submissions []*db.TrainingSubmission
	rawRequest := request.GetRawRequest()

	for _, entry := range rawRequest.Entries {

		amount, err := strconv.ParseFloat(entry.Amount, 64)
		if err != nil {
			return fmt.Errorf("amount float number (%s): %w", entry.Amount, err)
		}
		amountPence := math.Floor(amount * 100)
		requestDate := time.Time(rawRequest.SubmitDate)
		currentMembership := len(entry.CurrentMembershipSelection) > 0 &&
			len(entry.CurrentMembershipSelection[0]) > 0

		submissions = append(submissions, &db.TrainingSubmission{
			SubmissionState:  db.ReceivedSubmissionState,
			TrainingDate:     entry.SelectSession.StartLocal,
			PayByDate:        entry.SelectSession.StartLocal.Add(payBeforeSessionDuration),
			MembershipNumber: strings.Trim(entry.MembershipNumber, " "),
			RequestCurrMem:   currentMembership,
			Venue:            entry.Venue,
			AmountPence:      int64(amountPence),
			HorseName:        entry.HorseName,
			RequestDate:      requestDate,
			ExpireAt:         requestDate.Add(trainingSubmissionRecordTTL).Unix(),
			PaymentReference: rawRequest.PaymentReference,
			// Assume everything will be ok to start with
			FoundMemberRecord:        true,
			LapsedMembership:         false,
			AlreadyBooked:            false,
			ReceivedRequestEmailSent: true,
		})
	}

	memberRecords := make([]*db.MemberRecord, 2)
	sendReceivedRequestEmail := true

	for entryIndex, submission := range submissions {
		// fill the cross-references
		for i := range submissions {
			submission.LinkedSubmissionIds =
				append(submission.LinkedSubmissionIds, makeId(submissionId, i))
		}

		err := trainTable.Put(submission, makeId(submissionId, entryIndex))
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
		submission.ActualCurrMem = membershipDateCheck(memberRecord, &submission.TrainingDate)

		if submission.ActualCurrMem != submission.RequestCurrMem {

			if submission.RequestCurrMem == true {
				submission.LapsedMembership = true

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

// membershipDateCheck verifies if a member's membership is valid on a specific target date.
func membershipDateCheck(member *db.MemberRecord, target *time.Time) bool {
	start := member.MembershipValidFrom
	end := member.MembershipValidTo
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
