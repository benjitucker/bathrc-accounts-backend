package main

import (
	"benjitucker/bathrc-accounts/db"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

func parseMembersCSV(data []byte) ([]*db.MemberRecord, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow variable columns

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	headerIndex := make(map[string]int)
	for i, h := range headers {
		headerIndex[strings.Trim(h, `"`)] = i
	}

	var records []*db.MemberRecord

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		get := func(name string) string {
			if idx, ok := headerIndex[name]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		record := db.MemberRecord{
			FirstName:            get("First Name"),
			LastName:             get("Last Name"),
			SexAtBirth:           get("Sex at Birth"),
			Email:                get("Email Address"),
			MemberNumber:         get("Individual Membership Member No."),
			ClubMembershipStatus: get("BATH RIDING CLUB Membership Status"),
			MembershipType:       get("BATH RIDING CLUB Membership Membership Type"),
		}

		if dob := parseDate(get("Date of Birth")); dob != nil {
			record.DateOfBirth = dob
		}
		if from := parseDate(get("BATH RIDING CLUB Membership Valid From")); from != nil {
			record.MembershipValidFrom = from
		}
		if to := parseDate(get("BATH RIDING CLUB Membership Valid To")); to != nil {
			record.MembershipValidTo = to
		}

		// Check we have all the fields
		if record.FirstName == "" ||
			record.LastName == "" ||
			record.SexAtBirth == "" ||
			record.Email == "" ||
			record.MemberNumber == "" ||
			record.ClubMembershipStatus == "" ||
			record.MembershipType == "" ||
			record.MembershipValidTo == nil {
			return nil, fmt.Errorf("missing information in membership record: %s", record.String())
		}

		records = append(records, &record)
	}

	if records == nil {
		return nil, fmt.Errorf("no valid membership data processed")
	}

	return records, nil
}

const dateLayout = "2006-01-02"

func parseDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil
	}
	return &t
}
