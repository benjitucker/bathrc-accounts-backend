package main

import (
	"benjitucker/bathrc-accounts/db"
	"log"
	"time"
)

func handleSendTrainingAppIntroEmails(records []*db.MemberRecord) error {
	now := time.Now()
	twelveMonthsAgo := now.AddDate(-1, 0, 0)

	for _, member := range records {
		if member.MembershipValidTo == nil {
			continue
		}

		// Check if they had a valid membership in the last 12 months.
		// This means their MembershipValidTo must be after twelveMonthsAgo.
		if member.MembershipValidTo.After(twelveMonthsAgo) {
			log.Printf("Sending app intro email to %s (%s)", member.FirstName+" "+member.LastName, member.Email)
			emailHandler.SendAppIntro(member)
		}
	}

	return nil
}
