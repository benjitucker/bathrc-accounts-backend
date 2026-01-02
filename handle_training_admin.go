package main

import (
	"benjitucker/bathrc-accounts/jotform-webhook"
	"errors"
	"fmt"

	"github.com/go-kit/log/level"
)

func handleTrainingAdmin(form *jotform_webhook.FormData, request jotform_webhook.TrainingAdminRawRequest) error {

	var errs []error
	var err error

	// process just the first uploaded file, there should only be one
	if len(request.UploadURLs) == 0 {
		err = fmt.Errorf("no uploaded files for form %v", form.DebugString())
		_ = level.Warn(logger).Log("msg", "nothing to do", "err", err)
		return err
	}

	uploadUrl := request.UploadURLs[0]

	uploadedCSVData, err := jotformClient.GetSubmissionFile(uploadUrl)
	if err != nil {
		_ = level.Warn(logger).Log("msg", "failed getting submission file", "err", err)
		return err
	}

	// for test
	if request.ExtraCSV != nil {
		uploadedCSVData = append(uploadedCSVData, []byte(*request.ExtraCSV)...)
	}

	transactions, err := parseTransactionsCSV(uploadedCSVData)
	if err == nil {
		return handleTransactions(transactions)
	}
	_ = level.Warn(logger).Log("msg", "transactions parse", "err", err)
	errs = append(errs, err)

	members, err := parseMembersCSV(uploadedCSVData)
	if err == nil {
		return handleMembers(members)
	}
	errs = append(errs, err)

	err = errors.Join(errs...)
	_ = level.Warn(logger).Log("msg", "handle admin", "err", err)
	return err
}
