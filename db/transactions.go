package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactionRecord struct {
	DBItem
	Date         time.Time `dynamodbav:"txnDate"`
	DateUnix     int64     `dynamodbav:"txnDateUnix"`
	Type         string    `dynamodbav:"txnType"`
	Description  string    `dynamodbav:"txnDescription"`
	FirstName    string    `dynamodbav:"txnFirstName"`
	LastName     string    `dynamodbav:"txnLastName"`
	AmountPence  int64     `dynamodbav:"txnAmount"`
	BalancePence int64     `dynamodbav:"txnBalance"`
}

func (t TransactionRecord) String() string {
	sign := ""
	amountPounds := float64(t.AmountPence) / 100.0
	balancePounds := float64(t.BalancePence) / 100.0

	if t.AmountPence < 0 {
		sign = "-"
		amountPounds = -amountPounds
	}

	return fmt.Sprintf(
		"%s | %s | %s %s | %s | Amount: %s£%.2f | Balance: £%.2f",
		t.Date.Format("2006-01-02"),
		t.Type,
		t.FirstName,
		t.LastName,
		t.Description,
		sign,
		amountPounds,
		balancePounds,
	)
}

// Hash generates a SHA-256 hash of the transaction
func (t TransactionRecord) Hash() string {
	data := fmt.Sprintf(
		"%d|%s|%s|%s|%s|%d|%d",
		t.DateUnix,
		t.Type,
		t.Description,
		t.FirstName,
		t.LastName,
		t.AmountPence,
		t.BalancePence,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type TransactionTable struct {
	t *dbTable
}

func (t *TransactionTable) Open(ctx context.Context, ddb *dynamodb.Client) error {
	t.t = new(dbTable)
	t.t.ctx = ctx
	t.t.ddb = ddb
	t.t.tableName = "Transactions"
	t.t.pkValue = "ID"
	/* Handled by Terraform
	err := ensureTable(t.t)
	if err != nil {
		return err
	}
	*/
	return nil
}

func (t *TransactionTable) Put(record *TransactionRecord) error {
	record.SetID(record.Hash())
	return putItem(t.t, record)
}

func (t *TransactionTable) Get(id string) (*TransactionRecord, error) {
	return getItem[*TransactionRecord](t.t, id)
}

func (t *TransactionTable) GetAllOfTypeRecent(txnType string, startDate time.Time) ([]*TransactionRecord, error) {
	startDateStr := startDate.Format(time.RFC3339)

	query := &dynamodb.QueryInput{
		TableName: aws.String(t.t.tableName),
		IndexName: aws.String("TypeDateIndex"),

		KeyConditionExpression: aws.String(
			"txnType = :txnType AND txnDate >= :startDate",
		),

		ExpressionAttributeValues: map[string]types.AttributeValue{
			":txnType": &types.AttributeValueMemberS{
				Value: txnType,
			},
			":startDate": &types.AttributeValueMemberS{
				Value: startDateStr,
			},
		},

		// sort ascending
		//ScanIndexForward: aws.Bool(true),
	}

	fmt.Printf("Query: %#v", query)

	return queryItems[*TransactionRecord](t.t, query)
}

func (t *TransactionTable) GetAll() ([]*TransactionRecord, error) {
	return scanAllItems[*TransactionRecord](t.t)
}

func (t *TransactionTable) PutAll(records []*TransactionRecord) error {

	// the record id is its hash
	for _, record := range records {
		record.SetID(record.Hash())
	}

	return updateAllItems[*TransactionRecord](t.t, records)
}
