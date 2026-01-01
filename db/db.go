package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"runtime/debug"
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
}

type dbItemIf interface {
	GetID() string
	SetID(string)
}

type DBItem struct {
	id string `dynamodbav:"ID"`
}

func (i DBItem) GetID() string {
	if i.id == "" {
		fmt.Printf("ERROR: dbItem at %p has no ID when GetID called\n%s\n", &i,
			debug.Stack()[:256])
	}
	return i.id
}

func (i *DBItem) SetID(id string) {
	i.id = id
}

func mapToString(item map[string]types.AttributeValue) string {
	result, _ := json.Marshal(item)
	return string(result)
}

func putItem[T dbItemIf](t *dbTable, record T) error {

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return err
	}

	item["ID"] = &types.AttributeValueMemberS{Value: record.GetID()}

	_, err = t.ddb.PutItem(t.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(t.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to PutItem: table %s; Item %s: %w", t.tableName, mapToString(item), err)
	}
	return nil
}

// getItem retrieves an item by id and unmarshals it into the generic type T.
// T must be a struct or pointer to a struct compatible with attributevalue.UnmarshalMap.
func getItem[T dbItemIf](t *dbTable, id string) (T, error) {

	// Create a zero value of T
	var out T

	// Marshal the key for the GetItem request
	key, err := attributevalue.MarshalMap(map[string]string{
		"ID": id,
	})
	if err != nil {
		return out, nil
	}

	// Fetch the item
	res, err := t.ddb.GetItem(t.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(t.tableName),
		Key:       key,
	})
	if err != nil {
		return out, err
	}

	// No item found
	if res.Item == nil {
		return out, nil
	}

	// Unmarshal DynamoDB item into T
	err = attributevalue.UnmarshalMap(res.Item, &out)
	if err != nil {
		return out, err
	}
	out.SetID(id)

	return out, nil
}

func queryItems[T dbItemIf](t *dbTable, query *dynamodb.QueryInput) ([]T, error) {
	var result []T
	paginator := dynamodb.NewQueryPaginator(t.ddb, query)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(t.ctx)
		if err != nil {
			return result, err
		}

		var pageItems []T
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageItems); err != nil {
			return nil, err
		}

		// Put the IDs in
		for i := range pageItems {
			idVal := page.Items[i]["ID"].(*types.AttributeValueMemberS).Value
			pageItems[i].SetID(idVal)
		}

		result = append(result, pageItems...)
	}

	return result, nil
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

		// Put the IDs in
		for i := range pageItems {
			idVal := page.Items[i]["ID"].(*types.AttributeValueMemberS).Value
			pageItems[i].SetID(idVal)
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
