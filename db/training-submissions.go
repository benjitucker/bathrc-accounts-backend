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
	t.t = new(dbTable)
	t.t.ctx = ctx
	t.t.ddb = ddb
	t.t.tableName = "TrainingSubmissions"
	t.t.pkValue = "ID"
	/* Handled by Terraform
	err := ensureTable(t.t)
	if err != nil {
		return err
	}
	*/
	return nil
}

func (t *TrainingSubmissionTable) Put(record *TrainingSubmission) error {
	return putItem(t.t, record)
}

func (t *TrainingSubmissionTable) Get(id string) (*TrainingSubmission, error) {
	return getItem[TrainingSubmission](t.t, id)
}

func (t *TrainingSubmissionTable) GetAll() ([]TrainingSubmission, error) {
	return scanAllItems[TrainingSubmission](t.t)
}
