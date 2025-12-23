package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
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
	pkValue   string
}

type dbItemIf interface {
	GetID() string
}

type DBItem struct {
	ID string `dynamodbav:"ID"`
}

func (i DBItem) GetID() string {
	return i.ID
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
func getItem[T dbItemIf](t *dbTable, id string) (*T, error) {

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

// scanAllItems is expensive, it uses up read units
func scanAllItems[T dbItemIf](t *dbTable) ([]T, error) {

	var result []T

	input := &dynamodb.ScanInput{
		TableName: aws.String(t.tableName),
	}

	paginator := dynamodb.NewScanPaginator(t.ddb, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(t.ctx)
		if err != nil {
			return nil, err
		}

		var pageItems []T
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageItems); err != nil {
			return nil, err
		}

		result = append(result, pageItems...)
	}

	return result, nil
}

// updateItem updates a record with exponential backoff on failure
// It updates an existing item or adds a new one it none exists with the key value
func updateItem[T dbItemIf](t *dbTable, record *T) error {
	var err error

	update := func(t *dbTable, record *T, id string) error {
		key := map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		}

		// Convert struct to map[string]AttributeValue
		avMap, err := attributevalue.MarshalMap(record)
		if err != nil {
			return err
		}

		// Remove key fields from update values to avoid updating the primary key
		for k := range key {
			delete(avMap, k)
		}

		// Build UpdateExpression dynamically
		var exprParts []string
		exprValues := make(map[string]types.AttributeValue)
		exprNames := make(map[string]string)

		for k, v := range avMap {
			placeholder := ":" + k
			namePlaceholder := "#" + k
			exprParts = append(exprParts, fmt.Sprintf("%s = %s", namePlaceholder, placeholder))
			exprValues[placeholder] = v
			exprNames[namePlaceholder] = k
		}

		updateExpr := "SET " + strings.Join(exprParts, ", ")

		input := &dynamodb.UpdateItemInput{
			TableName:                 aws.String(t.tableName),
			Key:                       key,
			UpdateExpression:          aws.String(updateExpr),
			ExpressionAttributeNames:  exprNames,
			ExpressionAttributeValues: exprValues,
			ReturnValues:              types.ReturnValueUpdatedNew,
		}

		_, err = t.ddb.UpdateItem(t.ctx, input)
		return err
	}

	const maxRetries = 5
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = update(t, record, (*record).GetID())
		if err == nil {
			return nil
		}

		// Check for throttling error
		if !isThrottleError(err) {
			return err
		}

		// Exponential backoff with jitter
		backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
		jitter := time.Duration(float64(backoff) * (0.5 + 0.5*randFloat64()))
		time.Sleep(jitter)
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

// randFloat64 returns a random float in [0,1) (simple jitter)
func randFloat64() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000
}

// isThrottleError checks if the error is a throttling error
func isThrottleError(err error) bool {
	if err == nil {
		return false
	}

	var throughputErr *types.ProvisionedThroughputExceededException
	var throttlingErr *types.ThrottlingException

	return errors.As(err, &throughputErr) || errors.As(err, &throttlingErr)
}

func updateAllItems[T dbItemIf](t *dbTable, records []T) error {

	const maxParallel = 20

	jobs := make(chan *T)
	var wg sync.WaitGroup

	for i := 0; i < maxParallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case record, ok := <-jobs:
					if !ok {
						return
					}
					if err := updateItem(t, record); err != nil {
						log.Printf("Failed to update record %v: %v", (*record).GetID(), err)
					} else {
						log.Printf("Updated record %v successfully", (*record).GetID())
					}

				case <-t.ctx.Done():
					return
				}
			}
		}()
	}

	for _, record := range records {
		jobs <- &record
	}
	close(jobs)

	wg.Wait()

	return nil
}
