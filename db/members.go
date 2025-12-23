package db

import (
	"context"
	"fmt"
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

func (t *MemberTable) PutAll(records []MemberRecord) error {

	// the record ID is the member number
	for _, record := range records {
		record.ID = record.MemberNumber
	}

	return updateAllItems[MemberRecord](t.t, records)
}
