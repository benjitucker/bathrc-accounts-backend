package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"errors"
	"fmt"
)

// handleTrainingAdmin processes administrative uploads from Jotform, such as transaction or member CSV files.
func handleTrainingAdmin(form *jotform_webhook.FormData, request jotform_webhook.TrainingAdminRawRequest) error {

	var errs []error
	var err error

	// process just the first uploaded file, there should only be one
	if len(request.UploadURLs) == 0 {
		err = fmt.Errorf("no uploaded files for form %v", form.DebugString())
		return err
	}

	uploadUrl := request.UploadURLs[0]

	uploadedCSVData, err := jotformClient.GetSubmissionFile(uploadUrl)
	if err != nil {
		return fmt.Errorf("failed getting submission file: %w", err)
	}

	// for test
	if request.ExtraCSV != nil {
		uploadedCSVData = append(uploadedCSVData, []byte(*request.ExtraCSV)...)
	}

	var transactions []*db.TransactionRecord
	var members []*db.MemberRecord

	transactions, err = parseTransactionsCSV(uploadedCSVData)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed transaction parsing: %w", err))
		members, err = parseMembersCSV(uploadedCSVData)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed members parsing: %w", err))
			// return errors only if all parsings fail
			return errors.Join(errs...)
		}
	}

	if members != nil {
		err = handleMembers(members)
		if err != nil {
			return err
		}
	}

	if transactions != nil {
		err = handleTransactions(transactions)
		if err != nil {
			return err
		}
	}

	if request.SendEmailsNow != "OFF" {
		if members == nil {
			return errors.New("no members data to send emails for")
		}
		err = handleSendTrainingAppIntroEmails(members)
		if err != nil {
			return err
		}
	}

	return nil
}
