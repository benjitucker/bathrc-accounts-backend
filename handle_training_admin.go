package main

import (
	"benjitucker/bathrc-accounts/jotform-webhook"

	"github.com/go-kit/log/level"
)

func handleTrainingAdmin(_ *jotform_webhook.FormData, request jotform_webhook.TrainingAdminRawRequest) error {

	// process just the first uploaded file, there should only be one
	if len(request.UploadURLs) > 0 {
		uploadUrl := request.UploadURLs[0]
		uploadUrl = "https://www.example.com" // TODO remove

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

		_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
		for _, record := range records {
			_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
		}
	} else {
		_ = level.Error(logger).Log("msg", "Handle Request, no uploaded files")
	}

	return nil
}
