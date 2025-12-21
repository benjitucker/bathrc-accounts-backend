package main

import (
	"benjitucker/bathrc-accounts/jotform-webhook"
	"fmt"

	"github.com/go-kit/log/level"
)

func handleTrainingAdmin(_ *jotform_webhook.FormData, request jotform_webhook.TrainingAdminRawRequest) error {

	numberOfMembers := 0

	// process just the first uploaded file, there should only be one
	if len(request.UploadURLs) > 0 {
		uploadUrl := request.UploadURLs[0]

		uploadedCSVData, err := jotformClient.GetSubmissionFile(uploadUrl)
		if err != nil {
			return err
		}

		err = memberTable.PutCSV(uploadedCSVData)
		if err != nil {
			return err
		}

		records, err := memberTable.GetAll()
		if err != nil {
			return err
		}
		numberOfMembers = len(records)

		_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
		/*
			for _, record := range records {
				_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
			}
		*/
	} else {
		_ = level.Error(logger).Log("msg", "Handle Request, no uploaded files")
	}

	sendEmail(ctx, "ben@churchfarmmonktonfarleigh.co.uk", "jotform webhook: Training Admin",
		fmt.Sprintf("Uploaded member table. Currently holding %d members\n", numberOfMembers))

	return nil
}
