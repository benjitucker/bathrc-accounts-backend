package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type TrainingSubmission struct {
	DBItem
	SubmissionState           string    `dynamodbav:"submissionState"`
	TrainingDate              time.Time `dynamodbav:"trainingDate"`
	PayByDate                 time.Time `dynamodbav:"payByDate"`
	PaymentRecordId           string    `dynamodbav:"paymentRecordId"`
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
	LapsedMembership          bool      `dynamodbav:"lapsedMembership"`
	AlreadyBooked             bool      `dynamodbav:"alreadyBooked"`
	AlreadyBookedSubmissionId bool      `dynamodbav:"alreadyBookedSubmissionId"`
	LinkedSubmissionIds       []string  `dynamodbav:"linkedSubmissionIds"`
	ReceivedRequestEmailSent  bool      `dynamodbav:"receivedRequestEmailSent"`
	PaymentDiscrepancy        bool      `dynamodbav:"paymentDiscrepancy"`
}

// State Machine
const (
	ReceivedSubmissionState = "RECEIVED"
	PaidSubmissionState     = "PAID"
	InPastSubmissionState   = "IN_PAST"
	DroppedSubmissionState  = "DROPPED"
)

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

// PutAll relies on the ID of all the records to be in place
func (t *TrainingSubmissionTable) PutAll(records []*TrainingSubmission) error {
	return updateAllItems(t.t, records)
}

func (t *TrainingSubmissionTable) Get(id string) (*TrainingSubmission, error) {
	return getItem[*TrainingSubmission](t.t, id)
}

func (t *TrainingSubmissionTable) GetAll() ([]*TrainingSubmission, error) {
	return scanAllItems[*TrainingSubmission](t.t)
}

func (t *TrainingSubmissionTable) GetAllOfState(submissionState string) ([]*TrainingSubmission, error) {
	keyCond := expression.Key("submissionState").Equal(expression.Value(submissionState))

	expr, err := expression.NewBuilder().
		WithKeyCondition(keyCond).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %v", err)
	}

	return queryItems[*TrainingSubmission](t.t, &dynamodb.QueryInput{
		TableName:                 aws.String(t.t.tableName),
		IndexName:                 aws.String("StateDateIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
}

func (t *TrainingSubmissionTable) GetAllOfStateRecent(submissionState string, trainingDate time.Time) ([]*TrainingSubmission, error) {
	trainingDateStr := trainingDate.Format(time.RFC3339)

	keyCond := expression.Key("submissionState").Equal(expression.Value(submissionState)).
		And(expression.Key("trainingDate").GreaterThanEqual(expression.Value(trainingDateStr)))

	expr, err := expression.NewBuilder().
		WithKeyCondition(keyCond).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %v", err)
	}

	return queryItems[*TrainingSubmission](t.t, &dynamodb.QueryInput{
		TableName:                 aws.String(t.t.tableName),
		IndexName:                 aws.String("StateDateIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
}
