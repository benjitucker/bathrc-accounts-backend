package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"

	"github.com/go-kit/log/level"
)

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

	emailHandler.SendEmail(testEmail, "jotform webhook: Training Admin",
		fmt.Sprintf("Uploaded member table. Currently holding %d members\n", len(records)))

	return nil
}
