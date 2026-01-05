package main

import (
	"benjitucker/bathrc-accounts/db"
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"
)

var txnTypes = [...]string{"CR", "BP", "VIS", "CHG", "DD"}
var venueCodes = map[string]string{
	"West Wilts": "WWEC",
	"Widbrook":   "WID",
}

func handleTransactionsSummary() error {

	var err error
	twoMonthsAgo := time.Now().AddDate(0, -2, 0)

	// Get the last 2 months of training submissions
	// Get all training submissions here as they are used an a few places
	receivedSubmissions, err := trainTable.GetAllOfStateRecent(db.ReceivedSubmissionState, twoMonthsAgo)
	if err != nil {
		return err
	}

	paidSubmissions, err := trainTable.GetAllOfStateRecent(db.PaidSubmissionState, twoMonthsAgo)
	if err != nil {
		return err
	}

	inPastSubmissions, err := trainTable.GetAllOfStateRecent(db.InPastSubmissionState, twoMonthsAgo)
	if err != nil {
		return err
	}

	droppedSubmissions, err := trainTable.GetAllOfStateRecent(db.DroppedSubmissionState, twoMonthsAgo)
	if err != nil {
		return err
	}

	allSubmissions := append(inPastSubmissions, append(paidSubmissions, append(receivedSubmissions, droppedSubmissions...)...)...)

	// Get the last 2 months of transactions, sorted by date
	var transactions []*db.TransactionRecord
	for _, txnType := range txnTypes {
		t, err := transactionTable.GetAllOfTypeRecent(txnType, twoMonthsAgo)
		if err != nil {
			return err
		}
		transactions = append(transactions, t...)
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	writeEmail(transactions, allSubmissions, twoMonthsAgo, func(subject, body string) {
		emailHandler.SendEmail(testEmail, subject, body)
	})
	return nil
}

func writeEmail(transactions []*db.TransactionRecord, allSubmissions []*db.TrainingSubmission,
	twoMonthsAgo time.Time, emailer func(subject, body string)) {

	builder := new(strings.Builder)
	_, _ = fmt.Fprintf(builder, "Transactions CSV since %s\n\n", formatCustomDate(twoMonthsAgo))

	for _, transaction := range transactions {

		// Find all submissions linked to the transaction
		var submissions []*db.TrainingSubmission
		notes := ""
		foundPayment := false
		for _, submission := range allSubmissions {
			if submission.PaymentRecordId == transaction.GetID() {
				foundPayment = true
				submissions = append(submissions, submission)
				if submission.PaymentDiscrepancy {
					notes = notes + " Payment Discrepancy, Member " + submission.MembershipNumber
				}
			}
		}
		if !foundPayment {
			notes = notes + " No Payment Found"
		}

		_, _ = fmt.Fprintf(builder, "%s\n", toCSVLine(transaction, submissions, notes))

	}

	emailer(fmt.Sprintf("Transactions summary since %s", formatCustomDate(twoMonthsAgo)), builder.String())
}

// helper to format int64 pence to "123.45" or "-10.01"
func formatPounds(pence int64) string {
	sign := ""
	if pence < 0 {
		sign = "-"
		pence = -pence
	}
	pounds := pence / 100
	penceRemainder := pence % 100
	return fmt.Sprintf("%s%d.%02d", sign, pounds, penceRemainder)
}

// reconstruct the *original* description
func fullDescription(t *db.TransactionRecord) string {
	name := strings.TrimSpace(strings.Join([]string{t.FirstName, t.LastName}, " "))
	if t.Description == "" {
		return name
	}
	if name == "" {
		return t.Description
	}
	return name + " " + t.Description
}

// ToCSVLine returns a CSV line matching the original format
func toCSVLine(t *db.TransactionRecord, submissions []*db.TrainingSubmission, note string) string {

	// Get all the venues
	venuesSet := ""
	reference := ""
	for i, submission := range submissions {
		reference = submission.PaymentReference

		venueCode := ""
		ok := false
		if venueCode, ok = venueCodes[submission.Venue]; !ok {
			venueCode = submission.Venue
		}

		venuesSet = venuesSet + venueCode
		if i > 0 {
			venuesSet += " "
		}
	}

	record := []string{
		t.Date.Format("02 Jan 2006"),
		t.Type,
		fullDescription(t),
		formatPounds(t.AmountPence),
		venuesSet,
		formatPounds(t.BalancePence),
		note,
		reference,
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(record)
	w.Flush()
	return strings.TrimRight(buf.String(), "\n")
}
