package main

import (
	"benjitucker/bathrc-accounts/db"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	transactionRecordTTL = time.Hour * 24 * 365 * 2 // keep for 2 years
)

// parsePence converts a numeric currency string (e.g., "20.00", "-10.01") into an int64 representing the total value in pence.
func parsePence(s string) (int64, error) {
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 3)

	pounds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var pence int64
	if len(parts) == 2 {
		switch len(parts[1]) {
		case 1:
			pence, err = strconv.ParseInt(parts[1]+"0", 10, 64)
		case 2:
			pence, err = strconv.ParseInt(parts[1], 10, 64)
		default:
			return 0, fmt.Errorf("invalid pence value: %q", s)
		}
		if err != nil {
			return 0, err
		}
	}

	total := pounds*100 + pence
	if negative {
		total = -total
	}
	return total, nil
}

// splitDescription extracts first/last name and returns remaining text
func splitDescription(s string) (first, last, rest string) {
	fields := strings.Fields(s)

	switch len(fields) {
	case 0:
		return "", "", ""
	case 1:
		return fields[0], "", ""
	default:
		return fields[0], fields[1], strings.Join(fields[2:], " ")
	}
}

// parseTransactionsCSV parses a CSV byte slice containing transaction records into a slice of TransactionRecord structs.
func parseTransactionsCSV(csvData []byte) ([]*db.TransactionRecord, error) {
	r := csv.NewReader(bytes.NewReader(csvData))
	r.TrimLeadingSpace = true

	// Read header
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	var transactions []*db.TransactionRecord

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		date, err := time.Parse("02 Jan 2006", record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid date %q: %w", record[0], err)
		}

		amount, err := parsePence(record[3])
		if err != nil {
			return nil, fmt.Errorf("invalid amount %q: %w", record[3], err)
		}

		balance, err := parsePence(record[4])
		if err != nil {
			return nil, fmt.Errorf("invalid balance %q: %w", record[4], err)
		}

		first, last, remainder := splitDescription(record[2])

		transactions = append(transactions, &db.TransactionRecord{
			Date:         date,
			ExpireAt:     date.Add(transactionRecordTTL).Unix(),
			Type:         record[1],
			Description:  remainder,
			FirstName:    first,
			LastName:     last,
			AmountPence:  amount,
			BalancePence: balance,
		})
	}

	return transactions, nil
}
