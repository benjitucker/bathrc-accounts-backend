package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"

	"github.com/go-kit/log/level"
)

func handleTrainingRequest(formData *jotform_webhook.FormData, request jotform_webhook.TrainingRawRequest) error {

	submission := db.TrainingSubmission{
		Date:             request.SelectSession.Date,
		DateUnix:         request.SelectSession.Date.Unix(),
		MembershipNumber: request.MembershipNumber,
	}

	err := trainTable.Put(&submission, formData.SubmissionID)
	if err != nil {
		return err
	}

	// TODO:
	// Check membership number
	// email me on invalid membership number incase its a new member
	// check that the membership is current, and flag inconsistency with the form data with member
	// check that a training request for the same date/time has not already been received
	// email member to confirm that their training request has been received, pending payment

	records, err := trainTable.GetAll()
	if err != nil {
		return err
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
	for _, record := range records {
		_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
	}

	return nil
}
