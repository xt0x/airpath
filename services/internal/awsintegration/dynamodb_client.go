package awsintegration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient interface {
	PutItem(context.Context, string, map[string]any) error
	PutItemIfAbsent(context.Context, string, map[string]any, string) (bool, error)
	PutItemIfAbsentOrExpired(context.Context, string, map[string]any, string, string, int64) (bool, error)
	PutItemIfLeaseAvailable(context.Context, string, map[string]any, string, string) (bool, error)
	AcquireLeaseIfAvailable(context.Context, string, string, string, string, string) (map[string]any, bool, error)
	PutItemIfLeaseOwner(context.Context, string, map[string]any, string, string, string) (bool, error)
	PutItemIfLeaseUnchanged(context.Context, string, map[string]any, string, string) (bool, error)
	RemoveLeaseIfOwner(context.Context, string, string, string, string) (bool, error)
	PutItemIfOtherItemLeaseOwner(context.Context, string, map[string]any, string, map[string]any, string, string, string) (bool, error)
	PutItemsIfLeaseOwner(context.Context, string, map[string]any, string, map[string]any, string, string, string) (bool, error)
	PutItemCollectionIfLeaseOwner(context.Context, string, map[string]any, string, []map[string]any, string, string, string) (bool, error)
	PutItemIfCurrentAttributesMatch(context.Context, string, map[string]any, map[string]any, []string) (bool, error)
	PutItemIfCurrentAttributesEqual(context.Context, string, map[string]any, map[string]any) (bool, error)
	DeleteItemIfCurrentAttributesEqual(context.Context, string, map[string]any, map[string]any) (bool, error)
	AddUsageEstimate(context.Context, string, string, map[string]any, float64) (map[string]any, error)
	PutUsageEstimateIfHigher(context.Context, string, string, map[string]any, float64) (map[string]any, bool, error)
	DeleteItem(context.Context, string, map[string]any) error
	GetItem(context.Context, string, string, string) (map[string]any, bool, error)
	QueryStringPrefix(context.Context, string, string, string, string, string, string, int) ([]map[string]any, error)
	QueryRange(context.Context, string, string, string, string, string, *string, int, bool) ([]map[string]any, error)
	QueryRangeUntil(context.Context, string, string, string, string, string, string, int) ([]map[string]any, error)
	QueryRangeUntilPage(context.Context, string, string, string, string, string, string, int, map[string]any) ([]map[string]any, map[string]any, error)
}

type AWSDynamoDBClient struct {
	client *dynamodb.Client
}

func NewAWSDynamoDBClient(client *dynamodb.Client) *AWSDynamoDBClient {
	return &AWSDynamoDBClient{client: client}
}

func (c *AWSDynamoDBClient) PutItem(ctx context.Context, table string, item map[string]any) error {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
	})
	return err
}

func (c *AWSDynamoDBClient) PutItemIfAbsent(ctx context.Context, table string, item map[string]any, hashName string) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
		ExpressionAttributeNames: map[string]string{
			"#hash": hashName,
		},
		ConditionExpression: aws.String("attribute_not_exists(#hash)"),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) PutItemIfAbsentOrExpired(ctx context.Context, table string, item map[string]any, hashName string, expiresAtName string, nowUnix int64) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
		ExpressionAttributeNames: map[string]string{
			"#hash":      hashName,
			"#expiresAt": expiresAtName,
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":now": &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(nowUnix, 10)},
		},
		ConditionExpression: aws.String("attribute_not_exists(#hash) OR attribute_not_exists(#expiresAt) OR #expiresAt <= :now"),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) PutItemIfLeaseAvailable(ctx context.Context, table string, item map[string]any, _ string, now string) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
		ExpressionAttributeNames: map[string]string{
			"#flightId":        "flightId",
			"#fetchLeaseUntil": "fetchLeaseUntil",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":now": &ddbtypes.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_not_exists(#flightId) OR attribute_not_exists(#fetchLeaseUntil) OR #fetchLeaseUntil <= :now"),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) AcquireLeaseIfAvailable(ctx context.Context, table string, flightID string, owner string, leaseUntil string, now string) (map[string]any, bool, error) {
	nowTime, err := time.Parse(time.RFC3339, now)
	if err != nil {
		return nil, false, err
	}
	output, err := c.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			"flightId": &ddbtypes.AttributeValueMemberS{Value: flightID},
		},
		ExpressionAttributeNames: map[string]string{
			"#flightId":        "flightId",
			"#fetchLeaseUntil": "fetchLeaseUntil",
			"#fetchOwner":      "fetchOwner",
			"#ttl":             "ttl",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":leaseUntil": &ddbtypes.AttributeValueMemberS{Value: leaseUntil},
			":now":        &ddbtypes.AttributeValueMemberS{Value: now},
			":nowUnix":    &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(nowTime.UTC().Unix(), 10)},
			":owner":      &ddbtypes.AttributeValueMemberS{Value: owner},
		},
		ConditionExpression: aws.String("attribute_exists(#flightId) AND (attribute_not_exists(#ttl) OR #ttl > :nowUnix) AND (attribute_not_exists(#fetchLeaseUntil) OR #fetchLeaseUntil <= :now)"),
		UpdateExpression:    aws.String("SET #fetchOwner = :owner, #fetchLeaseUntil = :leaseUntil"),
		ReturnValues:        ddbtypes.ReturnValueAllNew,
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return decodeItem(output.Attributes), true, nil
}

func (c *AWSDynamoDBClient) PutItemIfLeaseOwner(ctx context.Context, table string, item map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	names, values, condition := leaseOwnerCondition(owner, leaseUntil, now)
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(table),
		Item:                      encoded,
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		ConditionExpression:       aws.String(condition),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) PutItemIfLeaseUnchanged(ctx context.Context, table string, item map[string]any, expectedOwner string, expectedLeaseUntil string) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	values := map[string]ddbtypes.AttributeValue{}
	ownerCondition := "attribute_not_exists(#fetchOwner)"
	if expectedOwner != "" {
		values[":expectedOwner"] = &ddbtypes.AttributeValueMemberS{Value: expectedOwner}
		ownerCondition = "#fetchOwner = :expectedOwner"
	}
	leaseCondition := "attribute_not_exists(#fetchLeaseUntil)"
	if expectedLeaseUntil != "" {
		values[":expectedLeaseUntil"] = &ddbtypes.AttributeValueMemberS{Value: expectedLeaseUntil}
		leaseCondition = "#fetchLeaseUntil = :expectedLeaseUntil"
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
		ExpressionAttributeNames: map[string]string{
			"#fetchLeaseUntil": "fetchLeaseUntil",
			"#fetchOwner":      "fetchOwner",
		},
		ConditionExpression: aws.String(ownerCondition + " AND " + leaseCondition),
	}
	if len(values) > 0 {
		input.ExpressionAttributeValues = values
	}
	_, err := c.client.PutItem(ctx, input)
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) RemoveLeaseIfOwner(ctx context.Context, table string, flightID string, owner string, leaseUntil string) (bool, error) {
	_, err := c.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			"flightId": &ddbtypes.AttributeValueMemberS{Value: flightID},
		},
		ExpressionAttributeNames: map[string]string{
			"#fetchLeaseUntil": "fetchLeaseUntil",
			"#fetchOwner":      "fetchOwner",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":leaseUntil": &ddbtypes.AttributeValueMemberS{Value: leaseUntil},
			":owner":      &ddbtypes.AttributeValueMemberS{Value: owner},
		},
		ConditionExpression: aws.String("#fetchOwner = :owner AND #fetchLeaseUntil = :leaseUntil"),
		UpdateExpression:    aws.String("REMOVE #fetchOwner, #fetchLeaseUntil"),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) PutItemIfOtherItemLeaseOwner(ctx context.Context, table string, item map[string]any, conditionTable string, conditionKey map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	encodedItem := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encodedItem[key] = encodeAttributeValue(value)
	}
	encodedConditionKey := make(map[string]ddbtypes.AttributeValue, len(conditionKey))
	for key, value := range conditionKey {
		encodedConditionKey[key] = encodeAttributeValue(value)
	}
	names, values, condition := leaseOwnerCondition(owner, leaseUntil, now)
	_, err := c.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []ddbtypes.TransactWriteItem{
			{
				ConditionCheck: &ddbtypes.ConditionCheck{
					TableName:                 aws.String(conditionTable),
					Key:                       encodedConditionKey,
					ExpressionAttributeNames:  names,
					ExpressionAttributeValues: values,
					ConditionExpression:       aws.String(condition),
				},
			},
			{
				Put: &ddbtypes.Put{
					TableName: aws.String(table),
					Item:      encodedItem,
				},
			},
		},
	})
	if err != nil {
		if transactionCanceledBecauseConditionFailed(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) PutItemsIfLeaseOwner(ctx context.Context, table string, item map[string]any, extraTable string, extraItem map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	return c.PutItemCollectionIfLeaseOwner(ctx, table, item, extraTable, []map[string]any{extraItem}, owner, leaseUntil, now)
}

func (c *AWSDynamoDBClient) PutItemCollectionIfLeaseOwner(ctx context.Context, table string, item map[string]any, extraTable string, extraItems []map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	if len(extraItems) > 99 {
		return false, fmt.Errorf("transaction item count %d exceeds DynamoDB limit", len(extraItems)+1)
	}
	encodedItem := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encodedItem[key] = encodeAttributeValue(value)
	}
	names, values, condition := leaseOwnerCondition(owner, leaseUntil, now)
	items := []ddbtypes.TransactWriteItem{
		{
			Put: &ddbtypes.Put{
				TableName:                 aws.String(table),
				Item:                      encodedItem,
				ExpressionAttributeNames:  names,
				ExpressionAttributeValues: values,
				ConditionExpression:       aws.String(condition),
			},
		},
	}
	for _, extraItem := range extraItems {
		encodedExtraItem := make(map[string]ddbtypes.AttributeValue, len(extraItem))
		for key, value := range extraItem {
			encodedExtraItem[key] = encodeAttributeValue(value)
		}
		items = append(items, ddbtypes.TransactWriteItem{
			Put: &ddbtypes.Put{
				TableName: aws.String(extraTable),
				Item:      encodedExtraItem,
			},
		})
	}
	_, err := c.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	if err != nil {
		if transactionCanceledBecauseConditionFailed(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func leaseOwnerCondition(owner string, leaseUntil string, now string) (map[string]string, map[string]ddbtypes.AttributeValue, string) {
	names := map[string]string{
		"#fetchLeaseUntil": "fetchLeaseUntil",
		"#fetchOwner":      "fetchOwner",
	}
	values := map[string]ddbtypes.AttributeValue{
		":leaseUntil": &ddbtypes.AttributeValueMemberS{Value: leaseUntil},
		":owner":      &ddbtypes.AttributeValueMemberS{Value: owner},
	}
	condition := "#fetchOwner = :owner AND #fetchLeaseUntil = :leaseUntil"
	if now != "" {
		values[":now"] = &ddbtypes.AttributeValueMemberS{Value: now}
		condition += " AND #fetchLeaseUntil > :now"
	}
	return names, values, condition
}

func transactionCanceledBecauseConditionFailed(err error) bool {
	var transactionCanceled *ddbtypes.TransactionCanceledException
	if !errors.As(err, &transactionCanceled) {
		return false
	}
	if len(transactionCanceled.CancellationReasons) == 0 {
		return false
	}
	conditionFailed := false
	for _, reason := range transactionCanceled.CancellationReasons {
		switch aws.ToString(reason.Code) {
		case "", "None":
			continue
		case "ConditionalCheckFailed":
			// A failed lease condition is a normal race between workers; any other
			// cancellation reason should surface as an infrastructure error.
			conditionFailed = true
		default:
			return false
		}
	}
	return conditionFailed
}

func (c *AWSDynamoDBClient) PutItemIfCurrentAttributesEqual(ctx context.Context, table string, item map[string]any, expected map[string]any) (bool, error) {
	return c.PutItemIfCurrentAttributesMatch(ctx, table, item, expected, nil)
}

func (c *AWSDynamoDBClient) PutItemIfCurrentAttributesMatch(ctx context.Context, table string, item map[string]any, expected map[string]any, absent []string) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	names, values, condition := currentAttributesCondition(expected, absent)
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(table),
		Item:                      encoded,
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		ConditionExpression:       aws.String(condition),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *AWSDynamoDBClient) DeleteItemIfCurrentAttributesEqual(ctx context.Context, table string, key map[string]any, expected map[string]any) (bool, error) {
	encoded := make(map[string]ddbtypes.AttributeValue, len(key))
	for name, value := range key {
		encoded[name] = encodeAttributeValue(value)
	}
	names, values, condition := currentAttributesCondition(expected, nil)
	_, err := c.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:                 aws.String(table),
		Key:                       encoded,
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		ConditionExpression:       aws.String(condition),
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func currentAttributesCondition(expected map[string]any, absent []string) (map[string]string, map[string]ddbtypes.AttributeValue, string) {
	names := make(map[string]string, len(expected)+len(absent))
	values := make(map[string]ddbtypes.AttributeValue, len(expected))
	parts := make([]string, 0, len(expected)+len(absent))
	index := 0
	for name, value := range expected {
		nameAlias := "#expected" + strconv.Itoa(index)
		valueAlias := ":expected" + strconv.Itoa(index)
		names[nameAlias] = name
		values[valueAlias] = encodeAttributeValue(value)
		parts = append(parts, nameAlias+" = "+valueAlias)
		index++
	}
	for _, name := range absent {
		nameAlias := "#absent" + strconv.Itoa(index)
		names[nameAlias] = name
		parts = append(parts, "attribute_not_exists("+nameAlias+")")
		index++
	}
	sort.Strings(parts)
	return names, values, strings.Join(parts, " AND ")
}

func (c *AWSDynamoDBClient) AddUsageEstimate(ctx context.Context, table string, budgetScope string, defaults map[string]any, delta float64) (map[string]any, error) {
	names := map[string]string{
		"#cost":              "estimatedMonthToDateCost",
		"#environment":       "environment",
		"#month":             "month",
		"#currency":          "currency",
		"#softStopThreshold": "softStopThreshold",
		"#fetchingEnabled":   "fetchingEnabled",
	}
	values := map[string]ddbtypes.AttributeValue{
		":delta":             &ddbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(delta, 'f', -1, 64)},
		":environment":       encodeAttributeValue(defaults["environment"]),
		":month":             encodeAttributeValue(defaults["month"]),
		":currency":          encodeAttributeValue(defaults["currency"]),
		":softStopThreshold": encodeAttributeValue(defaults["softStopThreshold"]),
		":fetchingEnabled":   encodeAttributeValue(defaults["fetchingEnabled"]),
	}
	output, err := c.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			"budgetScope": &ddbtypes.AttributeValueMemberS{Value: budgetScope},
		},
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		UpdateExpression: aws.String(
			"SET #environment = if_not_exists(#environment, :environment), #month = if_not_exists(#month, :month), #currency = if_not_exists(#currency, :currency), #softStopThreshold = if_not_exists(#softStopThreshold, :softStopThreshold), #fetchingEnabled = if_not_exists(#fetchingEnabled, :fetchingEnabled) ADD #cost :delta",
		),
		ReturnValues: ddbtypes.ReturnValueAllNew,
	})
	if err != nil {
		return nil, err
	}
	return decodeItem(output.Attributes), nil
}

func (c *AWSDynamoDBClient) PutUsageEstimateIfHigher(ctx context.Context, table string, budgetScope string, defaults map[string]any, estimate float64) (map[string]any, bool, error) {
	names := map[string]string{
		"#cost":              "estimatedMonthToDateCost",
		"#environment":       "environment",
		"#month":             "month",
		"#currency":          "currency",
		"#softStopThreshold": "softStopThreshold",
		"#fetchingEnabled":   "fetchingEnabled",
	}
	values := map[string]ddbtypes.AttributeValue{
		":estimate":          &ddbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(estimate, 'f', -1, 64)},
		":environment":       encodeAttributeValue(defaults["environment"]),
		":month":             encodeAttributeValue(defaults["month"]),
		":currency":          encodeAttributeValue(defaults["currency"]),
		":softStopThreshold": encodeAttributeValue(defaults["softStopThreshold"]),
		":fetchingEnabled":   encodeAttributeValue(defaults["fetchingEnabled"]),
	}
	output, err := c.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			"budgetScope": &ddbtypes.AttributeValueMemberS{Value: budgetScope},
		},
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		UpdateExpression: aws.String(
			"SET #environment = if_not_exists(#environment, :environment), #month = if_not_exists(#month, :month), #currency = if_not_exists(#currency, :currency), #softStopThreshold = if_not_exists(#softStopThreshold, :softStopThreshold), #fetchingEnabled = if_not_exists(#fetchingEnabled, :fetchingEnabled), #cost = :estimate",
		),
		ConditionExpression: aws.String("attribute_not_exists(#cost) OR #cost < :estimate"),
		ReturnValues:        ddbtypes.ReturnValueAllNew,
	})
	if err != nil {
		var conditionalFailed *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &conditionalFailed) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return decodeItem(output.Attributes), true, nil
}

func (c *AWSDynamoDBClient) DeleteItem(ctx context.Context, table string, key map[string]any) error {
	encoded := make(map[string]ddbtypes.AttributeValue, len(key))
	for name, value := range key {
		encoded[name] = encodeAttributeValue(value)
	}
	_, err := c.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key:       encoded,
	})
	return err
}

func (c *AWSDynamoDBClient) GetItem(ctx context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	output, err := c.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			hashName: &ddbtypes.AttributeValueMemberS{Value: hashValue},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if len(output.Item) == 0 {
		return nil, false, nil
	}
	return decodeItem(output.Item), true, nil
}

func (c *AWSDynamoDBClient) QueryStringPrefix(ctx context.Context, table string, indexName string, hashName string, hashValue string, rangeName string, prefix string, limit int) ([]map[string]any, error) {
	input := &dynamodb.QueryInput{
		TableName: aws.String(table),
		KeyConditionExpression: aws.String(
			"#hash = :hash AND begins_with(#range, :prefix)",
		),
		ExpressionAttributeNames: map[string]string{
			"#hash":  hashName,
			"#range": rangeName,
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":hash":   &ddbtypes.AttributeValueMemberS{Value: hashValue},
			":prefix": &ddbtypes.AttributeValueMemberS{Value: prefix},
		},
	}
	if indexName != "" {
		input.IndexName = aws.String(indexName)
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	return c.queryItems(ctx, input, limit)
}

func (c *AWSDynamoDBClient) QueryRange(ctx context.Context, table string, indexName string, hashName string, hashValue string, rangeName string, since *string, limit int, descending bool) ([]map[string]any, error) {
	names := map[string]string{"#hash": hashName, "#range": rangeName}
	values := map[string]ddbtypes.AttributeValue{":hash": &ddbtypes.AttributeValueMemberS{Value: hashValue}}
	keyCondition := "#hash = :hash"
	if since != nil {
		keyCondition += " AND #range >= :since"
		values[":since"] = &ddbtypes.AttributeValueMemberS{Value: *since}
	}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(table),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
		ScanIndexForward:          aws.Bool(!descending),
	}
	if indexName != "" {
		input.IndexName = aws.String(indexName)
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	return c.queryItems(ctx, input, limit)
}

func (c *AWSDynamoDBClient) QueryRangeUntil(ctx context.Context, table string, indexName string, hashName string, hashValue string, rangeName string, until string, limit int) ([]map[string]any, error) {
	items := []map[string]any{}
	var startKey map[string]any
	for {
		page, nextKey, err := c.QueryRangeUntilPage(ctx, table, indexName, hashName, hashValue, rangeName, until, limit, startKey)
		if err != nil {
			return nil, err
		}
		items = append(items, page...)
		if len(nextKey) == 0 || (limit > 0 && len(items) >= limit) {
			if limit > 0 && len(items) > limit {
				return items[:limit], nil
			}
			return items, nil
		}
		startKey = nextKey
	}
}

func (c *AWSDynamoDBClient) QueryRangeUntilPage(ctx context.Context, table string, indexName string, hashName string, hashValue string, rangeName string, until string, limit int, exclusiveStartKey map[string]any) ([]map[string]any, map[string]any, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("#hash = :hash AND #range <= :until"),
		ExpressionAttributeNames: map[string]string{
			"#hash":  hashName,
			"#range": rangeName,
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":hash":  &ddbtypes.AttributeValueMemberS{Value: hashValue},
			":until": &ddbtypes.AttributeValueMemberS{Value: until},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if indexName != "" {
		input.IndexName = aws.String(indexName)
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	if len(exclusiveStartKey) > 0 {
		input.ExclusiveStartKey = encodeItem(exclusiveStartKey)
	}
	output, err := c.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	items := make([]map[string]any, 0, len(output.Items))
	for _, item := range output.Items {
		items = append(items, decodeItem(item))
	}
	return items, decodeItem(output.LastEvaluatedKey), nil
}

func (c *AWSDynamoDBClient) queryItems(ctx context.Context, input *dynamodb.QueryInput, limit int) ([]map[string]any, error) {
	items := []map[string]any{}
	for {
		output, err := c.client.Query(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, item := range output.Items {
			items = append(items, decodeItem(item))
			if limit > 0 && len(items) >= limit {
				return items, nil
			}
		}
		if len(output.LastEvaluatedKey) == 0 {
			return items, nil
		}
		input.ExclusiveStartKey = output.LastEvaluatedKey
	}
}

type MemoryDynamoDBClient struct {
	mu              sync.RWMutex
	tables          map[string]map[string]map[string]any
	conditionalPuts map[string]struct{}
}

func NewMemoryDynamoDBClient() *MemoryDynamoDBClient {
	return &MemoryDynamoDBClient{
		tables:          map[string]map[string]map[string]any{},
		conditionalPuts: map[string]struct{}{},
	}
}

func (c *MemoryDynamoDBClient) PutItem(_ context.Context, table string, item map[string]any) error {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	c.tables[table][key] = cloneItem(item)
	return nil
}

func (c *MemoryDynamoDBClient) PutItemIfAbsent(_ context.Context, table string, item map[string]any, _ string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	if _, ok := c.tables[table][key]; ok {
		return false, nil
	}
	c.tables[table][key] = cloneItem(item)
	return true, nil
}

func (c *MemoryDynamoDBClient) PutItemIfAbsentOrExpired(_ context.Context, table string, item map[string]any, _ string, expiresAtName string, nowUnix int64) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	if current, ok := c.tables[table][key]; ok {
		expiresAt, hasExpiry := current[expiresAtName]
		if hasExpiry && int64(numberValue(expiresAt)) > nowUnix {
			return false, nil
		}
	}
	c.tables[table][key] = cloneItem(item)
	return true, nil
}

func (c *MemoryDynamoDBClient) PutItemIfLeaseAvailable(_ context.Context, table string, item map[string]any, _ string, now string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	if current, ok := c.tables[table][key]; ok {
		currentLeaseUntil := stringValue(current["fetchLeaseUntil"])
		if currentLeaseUntil != "" && currentLeaseUntil > now {
			return false, nil
		}
	}
	c.tables[table][key] = cloneItem(item)
	c.conditionalPuts[table+"#"+key] = struct{}{}
	return true, nil
}

func (c *MemoryDynamoDBClient) AcquireLeaseIfAvailable(_ context.Context, table string, flightID string, owner string, leaseUntil string, now string) (map[string]any, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[table][flightID]
	if !ok {
		return nil, false, nil
	}
	expired, err := itemTTLExpiredAtISO(current, now)
	if err != nil {
		return nil, false, err
	}
	if expired {
		return nil, false, nil
	}
	currentLeaseUntil := stringValue(current["fetchLeaseUntil"])
	if currentLeaseUntil != "" && currentLeaseUntil > now {
		return nil, false, nil
	}
	current["fetchOwner"] = owner
	current["fetchLeaseUntil"] = leaseUntil
	c.conditionalPuts[table+"#"+flightID] = struct{}{}
	return cloneItem(current), true, nil
}

func (c *MemoryDynamoDBClient) PutItemIfLeaseOwner(_ context.Context, table string, item map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	current, ok := c.tables[table][key]
	if !ok || !leaseOwnerMatches(current, owner, leaseUntil, now) {
		return false, nil
	}
	c.tables[table][key] = cloneItem(item)
	c.conditionalPuts[table+"#"+key] = struct{}{}
	return true, nil
}

func (c *MemoryDynamoDBClient) PutItemIfLeaseUnchanged(_ context.Context, table string, item map[string]any, expectedOwner string, expectedLeaseUntil string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	current, ok := c.tables[table][key]
	if !ok {
		return false, nil
	}
	if stringValue(current["fetchOwner"]) != expectedOwner || stringValue(current["fetchLeaseUntil"]) != expectedLeaseUntil {
		return false, nil
	}
	c.tables[table][key] = cloneItem(item)
	c.conditionalPuts[table+"#"+key] = struct{}{}
	return true, nil
}

func (c *MemoryDynamoDBClient) RemoveLeaseIfOwner(_ context.Context, table string, flightID string, owner string, leaseUntil string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[table][flightID]
	if !ok || stringValue(current["fetchOwner"]) != owner || stringValue(current["fetchLeaseUntil"]) != leaseUntil {
		return false, nil
	}
	delete(current, "fetchOwner")
	delete(current, "fetchLeaseUntil")
	c.conditionalPuts[table+"#"+flightID] = struct{}{}
	return true, nil
}

func (c *MemoryDynamoDBClient) PutItemIfOtherItemLeaseOwner(_ context.Context, table string, item map[string]any, conditionTable string, conditionKey map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	key := itemKey(item)
	conditionItemKey := itemKey(conditionKey)
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[conditionTable][conditionItemKey]
	if !ok || !leaseOwnerMatches(current, owner, leaseUntil, now) {
		return false, nil
	}
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	c.tables[table][key] = cloneItem(item)
	c.conditionalPuts[table+"#"+key] = struct{}{}
	return true, nil
}

func (c *MemoryDynamoDBClient) PutItemsIfLeaseOwner(_ context.Context, table string, item map[string]any, extraTable string, extraItem map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	return c.PutItemCollectionIfLeaseOwner(context.Background(), table, item, extraTable, []map[string]any{extraItem}, owner, leaseUntil, now)
}

func (c *MemoryDynamoDBClient) PutItemCollectionIfLeaseOwner(_ context.Context, table string, item map[string]any, extraTable string, extraItems []map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[table][key]
	if !ok || !leaseOwnerMatches(current, owner, leaseUntil, now) {
		return false, nil
	}
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	if c.tables[extraTable] == nil {
		c.tables[extraTable] = map[string]map[string]any{}
	}
	c.tables[table][key] = cloneItem(item)
	c.conditionalPuts[table+"#"+key] = struct{}{}
	for _, extraItem := range extraItems {
		extraKey := itemKey(extraItem)
		c.tables[extraTable][extraKey] = cloneItem(extraItem)
		c.conditionalPuts[extraTable+"#"+extraKey] = struct{}{}
	}
	return true, nil
}

func leaseOwnerMatches(item map[string]any, owner string, leaseUntil string, now string) bool {
	currentLeaseUntil := stringValue(item["fetchLeaseUntil"])
	if stringValue(item["fetchOwner"]) != owner || currentLeaseUntil != leaseUntil {
		return false
	}
	return now == "" || currentLeaseUntil > now
}

func (c *MemoryDynamoDBClient) PutItemIfCurrentAttributesEqual(_ context.Context, table string, item map[string]any, expected map[string]any) (bool, error) {
	return c.PutItemIfCurrentAttributesMatch(context.Background(), table, item, expected, nil)
}

func (c *MemoryDynamoDBClient) PutItemIfCurrentAttributesMatch(_ context.Context, table string, item map[string]any, expected map[string]any, absent []string) (bool, error) {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[table][key]
	if !ok || !attributesEqual(current, expected) || !attributesAbsent(current, absent) {
		return false, nil
	}
	c.tables[table][key] = cloneItem(item)
	return true, nil
}

func (c *MemoryDynamoDBClient) DeleteItemIfCurrentAttributesEqual(_ context.Context, table string, key map[string]any, expected map[string]any) (bool, error) {
	itemKey := itemKey(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.tables[table][itemKey]
	if !ok || !attributesEqual(current, expected) {
		return false, nil
	}
	delete(c.tables[table], itemKey)
	return true, nil
}

func attributesEqual(item map[string]any, expected map[string]any) bool {
	for name, value := range expected {
		if !reflect.DeepEqual(item[name], value) {
			return false
		}
	}
	return true
}

func attributesAbsent(item map[string]any, absent []string) bool {
	for _, name := range absent {
		if _, ok := item[name]; ok {
			return false
		}
	}
	return true
}

func (c *MemoryDynamoDBClient) AddUsageEstimate(_ context.Context, table string, budgetScope string, defaults map[string]any, delta float64) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	item := c.tables[table][budgetScope]
	if item == nil {
		item = map[string]any{"budgetScope": budgetScope}
		c.tables[table][budgetScope] = item
	}
	for key, value := range defaults {
		if _, ok := item[key]; !ok {
			item[key] = value
		}
	}
	item["estimatedMonthToDateCost"] = numberValue(item["estimatedMonthToDateCost"]) + delta
	return cloneItem(item), nil
}

func (c *MemoryDynamoDBClient) PutUsageEstimateIfHigher(_ context.Context, table string, budgetScope string, defaults map[string]any, estimate float64) (map[string]any, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	item := c.tables[table][budgetScope]
	if item != nil && numberValue(item["estimatedMonthToDateCost"]) >= estimate {
		return cloneItem(item), false, nil
	}
	if item == nil {
		item = map[string]any{"budgetScope": budgetScope}
		c.tables[table][budgetScope] = item
	}
	for key, value := range defaults {
		if _, ok := item[key]; !ok {
			item[key] = value
		}
	}
	item["estimatedMonthToDateCost"] = estimate
	return cloneItem(item), true, nil
}

func (c *MemoryDynamoDBClient) DeleteItem(_ context.Context, table string, key map[string]any) error {
	itemKey := itemKey(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tables[table], itemKey)
	return nil
}

func (c *MemoryDynamoDBClient) GetItem(_ context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, item := range c.tables[table] {
		if stringValue(item[hashName]) == hashValue {
			return cloneItem(item), true, nil
		}
	}
	return nil, false, nil
}

func (c *MemoryDynamoDBClient) QueryStringPrefix(_ context.Context, table string, _ string, hashName string, hashValue string, rangeName string, prefix string, limit int) ([]map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := []map[string]any{}
	for _, item := range c.tables[table] {
		if stringValue(item[hashName]) == hashValue && strings.HasPrefix(stringValue(item[rangeName]), prefix) {
			items = append(items, cloneItem(item))
		}
	}
	sort.Slice(items, func(left, right int) bool {
		return itemKey(items[left]) < itemKey(items[right])
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (c *MemoryDynamoDBClient) QueryRange(_ context.Context, table string, _ string, hashName string, hashValue string, rangeName string, since *string, limit int, descending bool) ([]map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := []map[string]any{}
	for _, item := range c.tables[table] {
		if stringValue(item[hashName]) != hashValue {
			continue
		}
		rangeValue := stringValue(item[rangeName])
		if since != nil && rangeValue < *since {
			continue
		}
		items = append(items, cloneItem(item))
	}
	sort.Slice(items, func(left, right int) bool {
		if descending {
			return stringValue(items[left][rangeName]) > stringValue(items[right][rangeName])
		}
		return stringValue(items[left][rangeName]) < stringValue(items[right][rangeName])
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (c *MemoryDynamoDBClient) QueryRangeUntil(_ context.Context, table string, _ string, hashName string, hashValue string, rangeName string, until string, limit int) ([]map[string]any, error) {
	items := []map[string]any{}
	var startKey map[string]any
	for {
		page, nextKey, err := c.QueryRangeUntilPage(context.Background(), table, "", hashName, hashValue, rangeName, until, limit, startKey)
		if err != nil {
			return nil, err
		}
		items = append(items, page...)
		if len(nextKey) == 0 || (limit > 0 && len(items) >= limit) {
			if limit > 0 && len(items) > limit {
				return items[:limit], nil
			}
			return items, nil
		}
		startKey = nextKey
	}
}

func (c *MemoryDynamoDBClient) QueryRangeUntilPage(_ context.Context, table string, _ string, hashName string, hashValue string, rangeName string, until string, limit int, exclusiveStartKey map[string]any) ([]map[string]any, map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := []map[string]any{}
	for _, item := range c.tables[table] {
		if stringValue(item[hashName]) != hashValue {
			continue
		}
		rangeValue := stringValue(item[rangeName])
		if rangeValue == "" || rangeValue > until {
			continue
		}
		items = append(items, cloneItem(item))
	}
	sort.Slice(items, func(left, right int) bool {
		leftRange := stringValue(items[left][rangeName])
		rightRange := stringValue(items[right][rangeName])
		if leftRange == rightRange {
			return itemKey(items[left]) < itemKey(items[right])
		}
		return leftRange < rightRange
	})
	if len(exclusiveStartKey) > 0 {
		startRange := stringValue(exclusiveStartKey[rangeName])
		startKey := itemKey(exclusiveStartKey)
		startIndex := 0
		for startIndex < len(items) {
			rangeValue := stringValue(items[startIndex][rangeName])
			if rangeValue > startRange || (rangeValue == startRange && itemKey(items[startIndex]) > startKey) {
				break
			}
			startIndex++
		}
		items = items[startIndex:]
	}
	var nextKey map[string]any
	if limit > 0 && len(items) > limit {
		nextKey = cloneItem(items[limit-1])
		items = items[:limit]
	}
	return items, nextKey, nil
}

func (c *MemoryDynamoDBClient) WasConditionalPutUsed(table string, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.conditionalPuts[table+"#"+key]
	return ok
}

func encodeAttributeValue(value any) ddbtypes.AttributeValue {
	switch typed := value.(type) {
	case string:
		return &ddbtypes.AttributeValueMemberS{Value: typed}
	case bool:
		return &ddbtypes.AttributeValueMemberBOOL{Value: typed}
	case int:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(typed)}
	case int64:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(typed, 10)}
	case float64:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(typed, 'f', -1, 64)}
	case nil:
		return &ddbtypes.AttributeValueMemberNULL{Value: true}
	default:
		return &ddbtypes.AttributeValueMemberS{Value: stringValue(value)}
	}
}

func encodeItem(item map[string]any) map[string]ddbtypes.AttributeValue {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	return encoded
}

func decodeItem(item map[string]ddbtypes.AttributeValue) map[string]any {
	decoded := make(map[string]any, len(item))
	for key, value := range item {
		decoded[key] = decodeAttributeValue(value)
	}
	return decoded
}

func decodeAttributeValue(value ddbtypes.AttributeValue) any {
	switch typed := value.(type) {
	case *ddbtypes.AttributeValueMemberS:
		return typed.Value
	case *ddbtypes.AttributeValueMemberN:
		if integer, err := strconv.ParseInt(typed.Value, 10, 64); err == nil {
			return integer
		}
		if number, err := strconv.ParseFloat(typed.Value, 64); err == nil {
			return number
		}
		return typed.Value
	case *ddbtypes.AttributeValueMemberBOOL:
		return typed.Value
	default:
		return nil
	}
}

func itemKey(item map[string]any) string {
	if lookupType := stringValue(item["lookupType"]); lookupType != "" {
		return lookupType + "#" + stringValue(item["lookupKey"])
	}
	for _, name := range []string{"flightId", "lookupKey", "budgetScope"} {
		if value := stringValue(item[name]); value != "" {
			if timestamp := stringValue(item["timestamp"]); timestamp != "" && name == "flightId" {
				return value + "#" + timestamp
			}
			return value
		}
	}
	return ""
}

func cloneItem(item map[string]any) map[string]any {
	cloned := make(map[string]any, len(item))
	for key, value := range item {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	if value == nil {
		return ""
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return ""
		}
		reflected = reflected.Elem()
	}
	if reflected.Kind() == reflect.String {
		return reflected.String()
	}
	return ""
}

func numberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case string:
		number, _ := strconv.ParseFloat(typed, 64)
		return number
	default:
		return 0
	}
}
