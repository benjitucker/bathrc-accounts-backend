package db

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type TrainingSubmission struct {
	DBItem
	Date             time.Time `dynamodbav:"date"`
	MembershipNumber string    `dynamodbav:"brcMembership"`
}

type TrainingSubmissionTable struct {
	t *dbTable
}

func (t *TrainingSubmissionTable) Open(ctx context.Context, ddb *dynamodb.Client) error {
	t.t.ctx = ctx
	t.t.ddb = ddb
	t.t.tableName = "TrainingSubmissions"
	err := ensureTable(t.t)
	if err != nil {
		return err
	}
	return nil
}

func (t *TrainingSubmissionTable) Put(record *TrainingSubmission) error {
	err := putItem(t.t, record)
	if err != nil {
		return err
	}
	return nil
}

func (t *TrainingSubmissionTable) Get(id string) (*TrainingSubmission, error) {
	record, err := getItem[TrainingSubmission](t.t, id)
	if err != nil {
		return nil, err
	}
	return record, nil
}
