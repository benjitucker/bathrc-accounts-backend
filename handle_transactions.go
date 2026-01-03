package main

import (
	"benjitucker/bathrc-accounts/db"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gogf/gf/v2/text/gstr"
)

const (
	recentTransactionsDuration = time.Hour * 24 * 30 // 30 days
	latePaymentsDuration       = time.Hour * 24 * 90 // 90 days
)

func formatAmount(amountPence int64) string {
	return fmt.Sprintf("£%d.%02d", amountPence/100, amountPence%100)
}

func handleTransactions(records []*db.TransactionRecord) error {

	emailHandler.SendEmail(testEmail, "jotform webhook: Training Admin",
		fmt.Sprintf("Handle %d transactions\n", len(records)))

	err := transactionTable.PutAll(records)
	if err != nil {
		return err
	}

	fmt.Printf("added/updated %d transactions", len(records))

	// TODO:
	// Check the number of paid entries per session and reject the latest ones if the numbers are too
	// high

	// Get all the relevant transactions in one request
	records, err = transactionTable.GetAllOfTypeRecent("CR", time.Now().Add(-recentTransactionsDuration))
	if err != nil {
		return err
	}

	// Get all unpaid training submissions
	receivedSubmissions, err := trainTable.GetAllOfState(db.ReceivedSubmissionState)
	if err != nil {
		return err
	}

	// Get all paid training submissions for sessions in the last 90 days
	paidSubmissions, err := trainTable.GetAllOfStateRecent(db.PaidSubmissionState, time.Now().Add(-latePaymentsDuration))
	if err != nil {
		return err
	}

	log.Printf("Got %d received submission records successfully", len(receivedSubmissions))

	// Update submissions that are in the past
	receivedSubmissions, err = updateInPastSubmissions(receivedSubmissions)
	if err != nil {
		return fmt.Errorf("failed to update in-past submissions: %w", err)
	}
	paidSubmissions, err = updateInPastSubmissions(paidSubmissions)
	if err != nil {
		return fmt.Errorf("failed to update in-past submissions: %w", err)
	}

	for _, submission := range receivedSubmissions {
		if submission.PaymentRecordId != "" {
			// Could have been updated already as it is linked
			continue
		}

		// Note, continue to process submissions for past events so that we can receive payments
		// after the event

		var matchedRecord *db.TransactionRecord
		bestDistance := 1000
		// Find the closest matching payment
		for _, record := range records {
			// not before the submission was made, note the transaction date does not include the time
			if record.Date.Before(submission.RequestDate) {
				continue
			}
			// not already attached to a submission
			alreadyAttached := false
			// Check receivedSubmissions as well because there may be one we have just attached
			for _, paidSubmission := range append(paidSubmissions, receivedSubmissions...) {
				if paidSubmission.PaymentRecordId == record.GetID() {
					alreadyAttached = true
					break
				}
			}
			if alreadyAttached == true {
				continue
			}
			// Find the best match
			distance := calcDistance(submission, record)
			if distance < bestDistance {
				bestDistance = distance
				matchedRecord = record
			}
			log.Printf("Received Submission ID %s (ref:%s), payment %s, distance %d",
				submission.GetID(), submission.PaymentReference, record.String(), distance)
		}

		// If not exact match, allow just one character of difference
		if bestDistance > 2 || matchedRecord == nil {
			// No match
			continue
		}

		linkedMemberRecords, linkedSubmissions, err := findSubmissionSet(submission.LinkedSubmissionIds, receivedSubmissions)
		if err != nil {
			return err
		}

		var problemTexts []string
		var totalAmount int64
		lapsedMembership := false
		for _, linkedSubmission := range linkedSubmissions {
			// update state
			linkedSubmission.PaymentRecordId = matchedRecord.GetID()

			// It could be a past submission so in that case don't update to paid state
			if linkedSubmission.SubmissionState == db.ReceivedSubmissionState {
				linkedSubmission.SubmissionState = db.PaidSubmissionState
			}

			// calc total for check
			totalAmount = totalAmount + linkedSubmission.AmountPence

			// check lapsed
			if linkedSubmission.LapsedMembership {
				lapsedMembership = true
			}
		}
		if lapsedMembership == true {
			problemTexts = append(problemTexts, fmt.Sprintf(
				`Your membership runs out before the training session. Please renew your memebrship with Sport80.`))
		}

		if totalAmount != matchedRecord.AmountPence {
			for _, sub := range linkedSubmissions {
				sub.PaymentDiscrepancy = true
			}

			problemTexts = append(problemTexts, fmt.Sprintf(
				`The payment amount is incorrect. The requested session[s] total price is %s, payment received %s.`,
				formatAmount(totalAmount), formatAmount(matchedRecord.AmountPence)))
		}

		// send received payment emails
		emailHandler.SendReceivedPayment(linkedMemberRecords, linkedSubmissions, problemTexts)

		// update linked submissions
		err = trainTable.PutAll(linkedSubmissions)
		if err != nil {
			return err
		}

		for _, sub := range linkedSubmissions {
			if sub.FoundMemberRecord == false {
				// email me on payment received when the membership is invalid
				emailHandler.SendEmail(testEmail, "Training: Paid but bad member",
					fmt.Sprintf("Payment ref %s, total amount %s bad member number %s",
						sub.PaymentReference, formatAmount(matchedRecord.AmountPence), sub.MembershipNumber))
			}
		}
	}

	return nil
}

func calcDistance(submission *db.TrainingSubmission, transactionRecord *db.TransactionRecord) int {
	lookForRef := strings.ToUpper(submission.PaymentReference)
	trnDesc := strings.ToUpper(transactionRecord.Description)
	if strings.Contains(trnDesc, lookForRef) {
		return 0
	}
	lowest := 1000
	descLen := len(trnDesc)
	if descLen < 6 {
		lowest = gstr.Levenshtein(lookForRef, trnDesc, 0, 1, 10)
	} else {
		// Check a 5 character window in every position
		i := 0
		for {
			if i >= descLen-4 {
				break
			}
			ld := gstr.Levenshtein(lookForRef, trnDesc[i:i+5], 0, 1, 10)
			if ld < lowest {
				lowest = ld
			}
			i = i + 1
		}
	}
	return lowest + 1 // plus 1 because it's not a straight match
}
