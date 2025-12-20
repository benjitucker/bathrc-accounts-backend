package main

import (
	"benjitucker/bathrc-accounts/jotform"
)

func handleTrainingAdmin(formData *jotform.FormData, request jotform.TrainingAdminRawRequest) error {

	/* TODO
	err := trainTable.Put(&db.TrainingSubmission{
		DBItem: db.DBItem{
			ID: formData.SubmissionID,
		},
		Date:             request.SelectSession.Date,
		MembershipNumber: request.MembershipNumber,
	})
	if err != nil {
		return err
	}

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
