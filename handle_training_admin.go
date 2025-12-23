package main

import (
	"benjitucker/bathrc-accounts/db"
	"benjitucker/bathrc-accounts/jotform-webhook"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-kit/log/level"
)

func handleTrainingAdmin(_ *jotform_webhook.FormData, request jotform_webhook.TrainingAdminRawRequest) error {

	numberOfMembers := 0

	// process just the first uploaded file, there should only be one
	if len(request.UploadURLs) > 0 {
		uploadUrl := request.UploadURLs[0]

		uploadedCSVData, err := jotformClient.GetSubmissionFile(uploadUrl)
		if err != nil {
			return err
		}

		records, err := parseMembersCSV(uploadedCSVData)
		if err != nil {
			return err
		}

		err = memberTable.PutAll(records)
		if err != nil {
			return err
		}

		records, err = memberTable.GetAll()
		if err != nil {
			return err
		}
		numberOfMembers = len(records)

		_ = level.Debug(logger).Log("msg", "Handle Request", "number of records", len(records))
		/*
			for _, record := range records {
				_ = level.Debug(logger).Log("msg", "Handle Request", "record from db", record)
			}
		*/
	} else {
		_ = level.Error(logger).Log("msg", "Handle Request, no uploaded files")
	}

	sendEmail(ctx, "ben@churchfarmmonktonfarleigh.co.uk", "jotform webhook: Training Admin",
		fmt.Sprintf("Uploaded member table. Currently holding %d members\n", numberOfMembers))

	return nil
}

func parseMembersCSV(data []byte) ([]db.MemberRecord, error) {
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

	var records []db.MemberRecord

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

		records = append(records, record)
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
