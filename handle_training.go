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

func handleTrainingRequest(formData *jotform_webhook.FormData, request jotform_webhook.TrainingRawRequest) error {

	amount, err := strconv.ParseFloat(request.Amount, 64)
	if err != nil {
		return fmt.Errorf("amount float number (%s): %w", request.Amount, err)
	}
	amountPence := math.Floor(amount * 100)
	requestDate := time.Time(request.SubmitDate)
	currentMembership := len(request.CurrentMembershipSelection[0]) > 0

	submission := db.TrainingSubmission{
		Date:              request.SelectSession.Date,
		DateUnix:          request.SelectSession.Date.Unix(),
		MembershipNumber:  strings.Trim(request.MembershipNumber, " "),
		RequestCurrMem:    currentMembership,
		ActualCurrMem:     currentMembership, // Initially assume its correct
		Venue:             request.SelectedVenue,
		AmountPence:       int64(amountPence),
		HorseName:         request.HorseName,
		RequestDate:       requestDate,
		RequestDateUnix:   requestDate.Unix(),
		PaymentReference:  request.PaymentReference,
		FoundMemberRecord: true,
		AlreadyBooked:     false,
	}
	err = trainTable.Put(&submission, formData.SubmissionID)
	if err != nil {
		return err
	}

	// Check membership number
	memberRecord, err := memberTable.Get(submission.MembershipNumber)
	if err != nil {
		// email me on invalid membership number incase it's a new member
		err2 := fmt.Errorf("membership record for number (%s): %w", submission.MembershipNumber, err)

		submission.FoundMemberRecord = false
		// update
		err = trainTable.Put(&submission, formData.SubmissionID)
		if err != nil {
			err2 = errors.Join(err2, err)
		}
		return err2
	}

	// check that the membership is current, and flag inconsistency with the form data with member
	submission.ActualCurrMem = membershipDateCheck(
		memberRecord.MembershipValidFrom, memberRecord.MembershipValidTo, &submission.Date)

	if submission.ActualCurrMem != submission.RequestCurrMem {
		// email me on invalid membership incase it's a new member
		err2 := fmt.Errorf("membership check for %s %s failed", memberRecord.FirstName, memberRecord.LastName)

		// update
		err = trainTable.Put(&submission, formData.SubmissionID)
		if err != nil {
			err2 = errors.Join(err2, err)
		}
		// TODO
		// Email membership inconsistency
		//emailHandler.SendEmail(memberRecord.Email,

		return err2
	}

	// TODO:
	// check that a training request for the same date/time has not already been received

	// TODO:
	// check that number of requested (and paid) entries per session and reject the request if
	// the numbers are two high

	// email member to confirm that their training request has been received, pending payment
	// TODO pending payment
	emailHandler.SendConfirm(memberRecord, &submission)

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
