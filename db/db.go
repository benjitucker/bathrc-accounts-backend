package db

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dbTable struct {
	ctx       context.Context
	ddb       *dynamodb.Client
	tableName string
}

type DBItem struct {
	ID string `dynamodbav:"id"`
}

func ensureTable(t *dbTable) error {
	_, err := t.ddb.DescribeTable(t.ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(t.tableName),
	})
	if err == nil {
		return nil // Table already exists
	}

	_, err = t.ddb.CreateTable(t.ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(t.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		return err
	}

	waiter := dynamodb.NewTableExistsWaiter(t.ddb)
	return waiter.Wait(t.ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(t.tableName),
	}, 5*time.Minute)
}

func putItem(t *dbTable, record any) error {
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return err
	}

	_, err = t.ddb.PutItem(t.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(t.tableName),
		Item:      item,
	})
	return err
}

// getItem retrieves an item by ID and unmarshals it into the generic type T.
// T must be a struct or pointer to a struct compatible with attributevalue.UnmarshalMap.
func getItem[T any](t *dbTable, id string) (*T, error) {

	// Marshal the key for the GetItem request
	key, err := attributevalue.MarshalMap(map[string]string{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	// Fetch the item
	res, err := t.ddb.GetItem(t.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(t.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	// No item found
	if res.Item == nil {
		return nil, nil
	}

	// Create a zero value of T
	var out T

	// Unmarshal DynamoDB item into T
	err = attributevalue.UnmarshalMap(res.Item, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}
