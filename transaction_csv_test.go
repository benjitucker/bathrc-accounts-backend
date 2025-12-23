package main

import (
	"benjitucker/bathrc-accounts/db"
	"reflect"
	"testing"
	"time"
)

func TestParsePence(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"20.00", 2000},
		{"0.99", 99},
		{"10.01", 1001},
		{"-5.50", -550},
		{"-0.01", -1},
	}

	for _, tt := range tests {
		got, err := parsePence(tt.input)
		if err != nil {
			t.Errorf("parsePence(%q) returned error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parsePence(%q) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}

func TestParseCSV(t *testing.T) {
	csvData := []byte(`Date,Type,Description,Amount,Balance
22 Dec 2025,CR,FOG BA BOB FOG,20.00,13141.19
22 Dec 2025,CR,Spot Payments,-10.01,13121.19`)

	transactions, err := parseTransactionsCSV(csvData)
	if err != nil {
		t.Fatalf("parseCSV returned error: %v", err)
	}

	expected := []*db.TransactionRecord{
		{
			Date:         mustParseDate("22 Dec 2025"),
			Type:         "CR",
			Description:  "FOG BA BOB FOG",
			AmountPence:  2000,
			BalancePence: 1314119,
		},
		{
			Date:         mustParseDate("22 Dec 2025"),
			Type:         "CR",
			Description:  "Spot Payments",
			AmountPence:  -1001,
			BalancePence: 1312119,
		},
	}

	if !reflect.DeepEqual(transactions, expected) {
		t.Errorf("parseCSV() = %+v; want %+v", transactions, expected)
	}
}

// helper to avoid repetitive error handling in tests
func mustParseDate(s string) time.Time {
	d, err := time.Parse("02 Jan 2006", s)
	if err != nil {
		panic(err)
	}
	return d
}
