package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Dynamo is the DynamoDB Store: one table with string keys PK and SK.
type Dynamo struct {
	client *dynamodb.Client
	table  string
}

// NewDynamo uses client for the table named table.
func NewDynamo(client *dynamodb.Client, table string) *Dynamo {
	return &Dynamo{client: client, table: table}
}

func (d *Dynamo) key(pk, sk string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: pk},
		"SK": &types.AttributeValueMemberS{Value: sk},
	}
}

func (d *Dynamo) Get(ctx context.Context, pk, sk string, out any) (bool, error) {
	res, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &d.table,
		Key:            d.key(pk, sk),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return false, err
	}
	if res.Item == nil {
		return false, nil
	}
	return true, attributevalue.UnmarshalMap(res.Item, out)
}

func (d *Dynamo) Put(ctx context.Context, item any, cond Condition) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	if _, _, err := keyOf(av); err != nil {
		return err
	}
	in := &dynamodb.PutItemInput{TableName: &d.table, Item: av}
	in.ConditionExpression, in.ExpressionAttributeNames, in.ExpressionAttributeValues = expression(cond)
	_, err = d.client.PutItem(ctx, in)
	return conflict(err)
}

func (d *Dynamo) Query(ctx context.Context, pk, prefix string, out any) error {
	in := &dynamodb.QueryInput{
		TableName:                 &d.table,
		ConsistentRead:            aws.Bool(true),
		KeyConditionExpression:    aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":pk": &types.AttributeValueMemberS{Value: pk}},
	}
	if prefix != "" {
		in.KeyConditionExpression = aws.String("PK = :pk AND begins_with(SK, :prefix)")
		in.ExpressionAttributeValues[":prefix"] = &types.AttributeValueMemberS{Value: prefix}
	}
	var items []map[string]types.AttributeValue
	for {
		res, err := d.client.Query(ctx, in)
		if err != nil {
			return err
		}
		items = append(items, res.Items...)
		if len(res.LastEvaluatedKey) == 0 {
			break
		}
		in.ExclusiveStartKey = res.LastEvaluatedKey
	}
	return attributevalue.UnmarshalListOfMaps(items, out)
}

func (d *Dynamo) Delete(ctx context.Context, pk, sk string) error {
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &d.table, Key: d.key(pk, sk)})
	return err
}

func (d *Dynamo) Transact(ctx context.Context, ops ...Op) error {
	if len(ops) > MaxOps {
		return fmt.Errorf("store: %d ops in one transaction, at most %d", len(ops), MaxOps)
	}
	items := make([]types.TransactWriteItem, 0, len(ops))
	for _, op := range ops {
		expr, names, vals := expression(op.Cond)
		if op.Item != nil {
			av, err := attributevalue.MarshalMap(op.Item)
			if err != nil {
				return err
			}
			if _, _, err := keyOf(av); err != nil {
				return err
			}
			items = append(items, types.TransactWriteItem{Put: &types.Put{TableName: &d.table, Item: av,
				ConditionExpression: expr, ExpressionAttributeNames: names, ExpressionAttributeValues: vals}})
		} else {
			items = append(items, types.TransactWriteItem{Delete: &types.Delete{TableName: &d.table, Key: d.key(op.PK, op.SK),
				ConditionExpression: expr, ExpressionAttributeNames: names, ExpressionAttributeValues: vals}})
		}
	}
	_, err := d.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	return conflict(err)
}

// Ping reads one key that never exists: it proves the table is reachable and readable.
func (d *Dynamo) Ping(ctx context.Context) error {
	_, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: &d.table, Key: d.key("HEALTH", "-")})
	return err
}

// EnsureTable creates the table if it doesn't exist. It's for local runs against DynamoDB
// Local; on AWS the table comes from infra/, with its free-tier settings.
func (d *Dynamo) EnsureTable(ctx context.Context) error {
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: &d.table,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	var inUse *types.ResourceInUseException
	if err != nil && !errors.As(err, &inUse) {
		return err
	}
	waiter := dynamodb.NewTableExistsWaiter(d.client)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: &d.table}, 30*time.Second)
}

// expression turns a Condition into a DynamoDB condition expression (nil for Always).
func expression(c Condition) (*string, map[string]string, map[string]types.AttributeValue) {
	switch c.kind {
	case condNew:
		return aws.String("attribute_not_exists(PK)"), nil, nil
	case condVersion:
		names := map[string]string{"#v": VersionAttr}
		vals := map[string]types.AttributeValue{":v": &types.AttributeValueMemberN{Value: strconv.Itoa(c.version)}}
		if c.version == 0 {
			return aws.String("attribute_not_exists(PK) OR #v = :v"), names, vals
		}
		return aws.String("#v = :v"), names, vals
	}
	return nil, nil, nil
}

// conflict maps DynamoDB's "condition failed" errors, single or in a transaction, to ErrConflict.
func conflict(err error) error {
	var failed *types.ConditionalCheckFailedException
	if errors.As(err, &failed) {
		return ErrConflict
	}
	var canceled *types.TransactionCanceledException
	if errors.As(err, &canceled) {
		for _, r := range canceled.CancellationReasons {
			if code := aws.ToString(r.Code); code == "ConditionalCheckFailed" || code == "TransactionConflict" {
				return ErrConflict
			}
		}
	}
	return err
}
