package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"time"

	"github.com/go-kit/log/level"
)

func handleTransactions(records []*db.TransactionRecord) error {

	err := transactionTable.PutAll(records)
	if err != nil {
		return err
	}

	_ = level.Debug(logger).Log("msg", "Handle Request", "added/updated transactions", len(records))

	// TODO:
	// Match received payments for training sessions and email members confirmation of received payment
	// if they have not already been sent

	// TODO:
	// Check the number of paid entries per session and reject the latest ones if the numbers are too
	// high

	// TODO : remove this test code
	records, err = transactionTable.GetAllOfTypeRecent("CR", time.Now().Add(time.Hour*-200))
	if err != nil {
		return err
	}

	for _, record := range records {
		_ = level.Debug(logger).Log("msg", "Handle Request", "record in the last 72 hours from db", record)
	}

	emailHandler.SendEmail(testEmail, "jotform webhook: Training Admin",
		fmt.Sprintf("Found %d CR transactions in the last 72 hours\n", len(records)))

	return nil
}
