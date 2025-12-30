package db

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type TrainingSubmission struct {
	DBItem
	Date                      time.Time `dynamodbav:"trainingDate"`
	PayByDate                 time.Time `dynamodbav:"payByDate"`
	DateUnix                  int64     `dynamodbav:"trainingDateUnix"`
	MembershipNumber          string    `dynamodbav:"brcMembership"`
	Venue                     string    `dynamodbav:"trainingVenue"`
	AmountPence               int64     `dynamodbav:"amountPence"`
	HorseName                 string    `dynamodbav:"horseName"`
	RequestDate               time.Time `dynamodbav:"requestDate"`
	RequestDateUnix           int64     `dynamodbav:"requestDateUnix"`
	PaymentReference          string    `dynamodbav:"paymentReference"`
	RequestCurrMem            bool      `dynamodbav:"requestCurrMem"`
	ActualCurrMem             bool      `dynamodbav:"actualCurrMem"`
	FoundMemberRecord         bool      `dynamodbav:"foundMemberRecord"`
	AlreadyBooked             bool      `dynamodbav:"alreadyBooked"`
	AlreadyBookedSubmissionId bool      `dynamodbav:"alreadyBookedSubmissionId"`
	LinkedSubmissionIds       []string  `dynamodbav:"linkedSubmissionIds"`
}

type TrainingSubmissionTable struct {
	t *dbTable
}

func (t *TrainingSubmissionTable) Open(ctx context.Context, ddb *dynamodb.Client) error {
	t.t = new(dbTable)
	t.t.ctx = ctx
	t.t.ddb = ddb
	t.t.tableName = "TrainingSubmissions"
	return nil
}

func (t *TrainingSubmissionTable) Put(record *TrainingSubmission, id string) error {
	record.SetID(id)
	return putItem[*TrainingSubmission](t.t, record)
}

func (t *TrainingSubmissionTable) Get(id string) (*TrainingSubmission, error) {
	return getItem[*TrainingSubmission](t.t, id)
}

func (t *TrainingSubmissionTable) GetAll() ([]*TrainingSubmission, error) {
	return scanAllItems[*TrainingSubmission](t.t)
}
