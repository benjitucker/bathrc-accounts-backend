package db

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type MemberRecord struct {
	DBItem
	FirstName            string     `dynamodbav:"firstName"`
	LastName             string     `dynamodbav:"lastName"`
	DateOfBirth          *time.Time `dynamodbav:"dateOfBirth"`
	SexAtBirth           string     `dynamodbav:"sexAtBirth"`
	Email                string     `dynamodbav:"email"`
	MemberNumber         string     `dynamodbav:"memberNumber"`
	ClubMembershipStatus string     `dynamodbav:"clubMembershipStatus"`
	MembershipValidFrom  *time.Time `dynamodbav:"membershipValidFrom"`
	MembershipValidTo    *time.Time `dynamodbav:"membershipValidTo"`
	MembershipType       string     `dynamodbav:"membershipType"`
}

func (m MemberRecord) String() string {
	formatDate := func(t *time.Time) string {
		if t == nil {
			return "<nil>"
		}
		return t.Format("2006-01-02")
	}

	return fmt.Sprintf(
		"MemberRecord{FirstName=%q, LastName=%q, DOB=%s, SexAtBirth=%q, Email=%q, MemberNumber=%q, Status=%q, ValidFrom=%s, ValidTo=%s, MembershipType=%q}",
		m.FirstName,
		m.LastName,
		formatDate(m.DateOfBirth),
		m.SexAtBirth,
		m.Email,
		m.MemberNumber,
		m.ClubMembershipStatus,
		formatDate(m.MembershipValidFrom),
		formatDate(m.MembershipValidTo),
		m.MembershipType,
	)
}

type MemberTable struct {
	t *dbTable
}

func (t *MemberTable) Open(ctx context.Context, ddb *dynamodb.Client) error {
	t.t = new(dbTable)
	t.t.ctx = ctx
	t.t.ddb = ddb
	t.t.tableName = "Members"
	t.t.pkValue = "ID"
	/* Handled by Terraform
	err := ensureTable(t.t)
	if err != nil {
		return err
	}
	*/
	return nil
}

func (t *MemberTable) Put(record *MemberRecord) error {
	return putItem(t.t, record)
}

func (t *MemberTable) Get(id string) (*MemberRecord, error) {
	return getItem[MemberRecord](t.t, id)
}

func (t *MemberTable) GetAll() ([]MemberRecord, error) {
	return scanAllItems[MemberRecord](t.t)
}

func (t *MemberTable) PutCSV(csv []byte) error {
	records, err := parseMembersCSV(csv)
	if err != nil {
		return err
	}

	const maxParallel = 20 // Limit parallelism
	semaphore := make(chan struct{}, maxParallel)

	var wg sync.WaitGroup

	for _, record := range records {
		wg.Add(1)
		// the id is the membership number
		record.ID = record.MemberNumber
		go func(rec *MemberRecord) {
			defer wg.Done()

			// Block if there are max number of threads running
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := updateItem(t.t, rec, rec.MemberNumber); err != nil {
				log.Printf("Failed to update record %v: %v", rec.ID, err)
			} else {
				log.Printf("Updated record %v successfully", rec.ID)
			}
		}(&record)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return nil
}

func parseMembersCSV(data []byte) ([]MemberRecord, error) {
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

	var records []MemberRecord

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

		record := MemberRecord{
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
