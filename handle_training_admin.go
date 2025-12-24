package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"errors"
	"fmt"
	"time"

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

func handleMembers(records []*db.MemberRecord) error {
	err := memberTable.PutAll(records)
	if err != nil {
		return err
	}

	records, err = memberTable.GetAll()
	if err != nil {
		return err
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "total members now", len(records))
	/*
		for _, record := range records {
			_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
		}
	*/

	// TODO:
	// Work out which member training confirmations email have not been sent and send them

	sendEmail(ctx, "ben@churchfarmmonktonfarleigh.co.uk", "jotform webhook: Training Admin",
		fmt.Sprintf("Uploaded member table. Currently holding %d members\n", len(records)))

	return nil
}

func handleTransactions(records []*db.TransactionRecord) error {

	err := transactionTable.PutAll(records)
	if err != nil {
		return err
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "added/updated transactions", len(records))

	// TODO:
	// Match received payments for training sessions and email members confirmation of received payment
	// if they have not already been sent

	// TODO : remove this test code
	records, err = transactionTable.GetAllOfTypeRecent("CR", time.Now().Add(time.Hour*-72))
	if err != nil {
		return err
	}

	for _, record := range records {
		_ = level.Debug(logger).Log("msg", "Handle Request", "record in the last 72 hours from db", record)
	}

	sendEmail(ctx, "ben@churchfarmmonktonfarleigh.co.uk", "jotform webhook: Training Admin",
		fmt.Sprintf("Found %d CR transactions in the last 72 hours\n", len(records)))

	return nil
}
