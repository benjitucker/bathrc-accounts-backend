package db

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TestItem struct {
	DBItem
	Name string `dynamodbav:"Name"`
	Age  int    `dynamodbav:"Age"`
}

const testTable = "UnitTestTable"

func localClient(t *testing.T) *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("us-west-2"),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: "http://localhost:8000"}, nil
				},
			),
		),
	)

	if err != nil {
		t.Fatalf("config error: %v", err)
	}
	return dynamodb.NewFromConfig(cfg)
}

func createTestTable(t *testing.T, name string) {
	db := localClient(t)

	_, _ = db.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(name),
	})

	_, err := db.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(name),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("ID"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("ID"), KeyType: types.KeyTypeHash},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	// wait until active
	for {
		out, _ := db.DescribeTable(context.Background(),
			&dynamodb.DescribeTableInput{TableName: aws.String(name)})
		if out != nil && out.Table.TableStatus == types.TableStatusActive {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func newDBTable(t *testing.T, table string) *dbTable {
	return &dbTable{
		ctx:       context.Background(),
		ddb:       localClient(t),
		tableName: table,
	}
}

func TestPutAndGetItem(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	item := &TestItem{
		Name: "Alice",
		Age:  30,
	}
	item.SetID("user-1")

	if err := putItem(tbl, item); err != nil {
		t.Fatalf("putItem failed: %v", err)
	}

	got, err := getItem[*TestItem](tbl, "user-1")
	if err != nil {
		t.Fatalf("getItem failed: %v", err)
	}

	if got.GetID() != "user-1" || got.Name != "Alice" || got.Age != 30 {
		t.Fatalf("unexpected item: %#v", got)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	_, err := getItem[*TestItem](tbl, "missing")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
}

func TestQueryItems(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	users := []*TestItem{
		{DBItem: DBItem{id: "user-1"}, Name: "Alice", Age: 30},
		{DBItem: DBItem{id: "user-2"}, Name: "Bob", Age: 40},
	}

	for _, u := range users {
		_ = putItem(tbl, u)
	}

	q := &dynamodb.QueryInput{
		TableName:              aws.String(testTable),
		KeyConditionExpression: aws.String("ID = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: "user-1"},
		},
	}

	res, err := queryItems[*TestItem](tbl, q)
	if err != nil {
		t.Fatalf("queryItems failed: %v", err)
	}

	if len(res) != 1 || res[0].Name != "Alice" || res[0].GetID() != "user-1" {
		t.Fatalf("unexpected query result: %#v", res)
	}
}

func TestScanAllItems(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	u1 := &TestItem{Name: "A", Age: 1}
	u1.SetID("idA")

	u2 := &TestItem{Name: "B", Age: 2}
	u2.SetID("idB")

	_ = putItem(tbl, u1)
	_ = putItem(tbl, u2)

	items, err := scanAllItems[*TestItem](tbl)
	if err != nil {
		t.Fatalf("scanAllItems failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 results, got %d", len(items))
	}

	for _, item := range items {
		if item.Name != "A" && item.Name != "B" {
			t.Fatalf("unexpected result: %#v", items)
		}
		if item.GetID() != "id"+item.Name {
			t.Fatalf("unexpected result: %#v", items)
		}
	}
}

func TestUpdateItem(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	u := &TestItem{Name: "Alice", Age: 20}
	u.SetID("user-9")

	_ = putItem(tbl, u)

	u.Age = 50

	if err := updateItem(tbl, &u); err != nil {
		t.Fatalf("updateItem failed: %v", err)
	}

	got, _ := getItem[*TestItem](tbl, "user-9")
	if got.Age != 50 {
		t.Fatalf("expected age 50 got %d", got.Age)
	}
}

func TestUpdateAllItems(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	users := []*TestItem{
		{DBItem: DBItem{id: "1"}, Name: "A", Age: 10},
		{DBItem: DBItem{id: "2"}, Name: "B", Age: 20},
	}

	for _, u := range users {
		_ = putItem(tbl, u)
	}

	// modify
	users[0].Age = 99
	users[1].Age = 88

	if err := updateAllItems(tbl, users); err != nil {
		t.Fatalf("updateAllItems failed: %v", err)
	}

	got1, _ := getItem[*TestItem](tbl, "1")
	got2, _ := getItem[*TestItem](tbl, "2")

	if got1.Age != 99 || got2.Age != 88 {
		t.Fatalf("unexpected results: %#v %#v", got1, got2)
	}
}
