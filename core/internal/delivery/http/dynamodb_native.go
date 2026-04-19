package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	ddbcontracts "github.com/michasdev/mildstack/core/internal/resources/dynamodb/contracts"
	dynamodbdomain "github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

type DynamoDBNativeService interface {
	ListTables() []dynamodbdomain.Table
	CreateTable(name, partitionKey, sortKey, billingMode string) (dynamodbdomain.Table, error)
	DescribeTable(name string) (dynamodbdomain.Table, error)
	DeleteTable(name string) (dynamodbdomain.Table, error)
	GetItem(table, key string) (dynamodbdomain.Item, error)
	PutItem(table, key string, attributes map[string]dynamodbdomain.AttributeValue) (dynamodbdomain.Item, error)
	UpdateItem(table, key, updateExpression, conditionExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]dynamodbdomain.AttributeValue) (dynamodbdomain.Item, error)
	Query(table, keyConditionExpression, filterExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]dynamodbdomain.AttributeValue, limit *int, exclusiveStartKey map[string]dynamodbdomain.AttributeValue, scanIndexForward *bool) (dynamodbdomain.ReadPage, error)
	Scan(table, filterExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]dynamodbdomain.AttributeValue, limit *int, exclusiveStartKey map[string]dynamodbdomain.AttributeValue) (dynamodbdomain.ReadPage, error)
	DeleteItem(table, key string) error
	BatchWriteItem(request ddbcontracts.BatchWriteItemRequest) (ddbcontracts.BatchWriteItemResult, error)
	BatchGetItem(request ddbcontracts.BatchGetItemRequest) (ddbcontracts.BatchGetItemResult, error)
	TransactWriteItems(request ddbcontracts.TransactWriteItemsRequest) error
	TransactGetItems(request ddbcontracts.TransactGetItemsRequest) (ddbcontracts.TransactGetItemsResult, error)
}

const (
	dynamoDBJSONContentType = "application/x-amz-json-1.0"
	dynamoDBTargetPrefix    = "DynamoDB_20120810."
	dynamoDBErrorPrefix     = "com.amazonaws.dynamodb.v20120810#"
)

func RegisterDynamoDBNativeRoutes(engine *gin.Engine, service DynamoDBNativeService) {
	if engine == nil || service == nil {
		return
	}

	handler := newDynamoDBNativeHandler(service)
	engine.Use(func(c *gin.Context) {
		if handled := handler.dispatch(c); handled {
			c.Abort()
			return
		}
		c.Next()
	})
}

type dynamoDBNativeHandler struct {
	service  DynamoDBNativeService
	registry map[string]dynamoTargetSpec
}

type dynamoTargetSpec struct {
	supported bool
	execute   func(*dynamoDBNativeHandler, *gin.Context, []byte) error
}

func newDynamoDBNativeHandler(service DynamoDBNativeService) dynamoDBNativeHandler {
	return dynamoDBNativeHandler{
		service:  service,
		registry: newDynamoDBTargetRegistry(),
	}
}

func newDynamoDBTargetRegistry() map[string]dynamoTargetSpec {
	return map[string]dynamoTargetSpec{
		"ListTables": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleListTables,
		},
		"CreateTable": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleCreateTable,
		},
		"DescribeTable": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleDescribeTable,
		},
		"DeleteTable": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleDeleteTable,
		},
		"GetItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleGetItem,
		},
		"PutItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handlePutItem,
		},
		"UpdateItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleUpdateItem,
		},
		"Query": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleQuery,
		},
		"Scan": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleScan,
		},
		"DeleteItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleDeleteItem,
		},
		"BatchGetItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleBatchGetItem,
		},
		"BatchWriteItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleBatchWriteItem,
		},
		"TransactGetItems": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleTransactGetItems,
		},
		"TransactWriteItems": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleTransactWriteItems,
		},
		"UpdateTable": {supported: false},
	}
}

func (h dynamoDBNativeHandler) dispatch(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}

	if c.Request.Method != http.MethodPost {
		return false
	}

	requestPath := strings.TrimSpace(c.Request.URL.Path)
	if requestPath == "" || strings.HasPrefix(requestPath, "/api/") {
		return false
	}
	if requestPath != "/" {
		return false
	}

	if !isDynamoDBJSONRequest(c.Request.Header.Get("Content-Type")) {
		return false
	}

	targetName, err := parseDynamoDBTarget(c.Request.Header.Get("X-Amz-Target"))
	if err != nil {
		writeDynamoDBError(c, http.StatusBadRequest, "ValidationException", err.Error())
		return true
	}

	spec, ok := h.registry[targetName]
	if !ok {
		writeDynamoDBError(c, http.StatusNotFound, "UnknownOperationException", fmt.Sprintf("Unknown DynamoDB target %q", targetName))
		return true
	}
	if !spec.supported {
		writeDynamoDBError(c, http.StatusNotFound, "UnknownOperationException", fmt.Sprintf("DynamoDB target %q is not supported by the current local subset", targetName))
		return true
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeDynamoDBError(c, http.StatusBadRequest, "ValidationException", fmt.Sprintf("dynamodb: read request body: %v", err))
		return true
	}

	if err := spec.execute(&h, c, body); err != nil {
		writeDynamoDBError(c, http.StatusBadRequest, dynamoDBErrorCode(err), err.Error())
	}
	return true
}

func (h *dynamoDBNativeHandler) handleListTables(c *gin.Context, body []byte) error {
	request := listTablesRequest{}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &request); err != nil {
			return fmt.Errorf("dynamodb: invalid ListTables request: %w", err)
		}
	}

	tableNames := make([]string, 0, len(h.service.ListTables()))
	for _, table := range h.service.ListTables() {
		tableNames = append(tableNames, table.Name)
	}

	names, lastEvaluated := paginateTableNames(tableNames, request.Limit, request.ExclusiveStartTableName)
	writeDynamoDBJSON(c, http.StatusOK, listTablesResponse{
		TableNames:             names,
		LastEvaluatedTableName: lastEvaluated,
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleCreateTable(c *gin.Context, body []byte) error {
	request := createTableRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid CreateTable request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	partitionKey, sortKey := partitionAndSortKeys(request.KeySchema)
	billingMode := strings.TrimSpace(request.BillingMode)

	table, err := h.service.CreateTable(tableName, partitionKey, sortKey, billingMode)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, createTableResponse{
		TableDescription: tableDescription{
			TableName:            table.Name,
			TableStatus:          table.Status,
			TableArn:             awscontext.Default().DynamoDBTableARN(table.Name),
			CreationDateTime:     awsTimestamp(table.CreatedAt),
			KeySchema:            cloneKeySchema(request.KeySchema, partitionKey, sortKey),
			AttributeDefinitions: cloneAttributeDefinitions(request.AttributeDefinitions),
			BillingModeSummary:   billingModeSummaryFor(table.BillingMode),
		},
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleDescribeTable(c *gin.Context, body []byte) error {
	request := describeTableRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid DescribeTable request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	table, err := h.service.DescribeTable(tableName)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, describeTableResponse{
		Table: tableDescriptionFromDomain(table),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleDeleteTable(c *gin.Context, body []byte) error {
	request := deleteTableRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid DeleteTable request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	table, err := h.service.DeleteTable(tableName)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, deleteTableResponse{
		TableDescription: tableDescriptionFromDomain(table),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleGetItem(c *gin.Context, body []byte) error {
	request := getItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid GetItem request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	key, err := keyFromAttributeValueMap(request.Key)
	if err != nil {
		return err
	}

	item, err := h.service.GetItem(tableName, key)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, getItemResponse{
		Item: attributeValueMapFromDomain(item.Attributes),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handlePutItem(c *gin.Context, body []byte) error {
	request := putItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid PutItem request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	key, attributes, err := itemFromAttributeValueMap(request.Item)
	if err != nil {
		return err
	}

	item, err := h.service.PutItem(tableName, key, attributes)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, putItemResponse{
		Attributes: attributeValueMapFromDomain(item.Attributes),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleDeleteItem(c *gin.Context, body []byte) error {
	request := deleteItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid DeleteItem request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	key, err := keyFromAttributeValueMap(request.Key)
	if err != nil {
		return err
	}

	if err := h.service.DeleteItem(tableName, key); err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, deleteItemResponse{})
	return nil
}

func (h *dynamoDBNativeHandler) handleUpdateItem(c *gin.Context, body []byte) error {
	request := updateItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid UpdateItem request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}

	key, err := keyFromAttributeValueMap(request.Key)
	if err != nil {
		return err
	}

	if rv := strings.ToUpper(strings.TrimSpace(request.ReturnValues)); rv != "" && rv != "NONE" && rv != "ALL_NEW" {
		return fmt.Errorf("dynamodb: unsupported return values %q", request.ReturnValues)
	}

	expressionAttributeValues := make(map[string]dynamodbdomain.AttributeValue, len(request.ExpressionAttributeValues))
	for name, value := range request.ExpressionAttributeValues {
		converted, err := attributeValueToDomain(value)
		if err != nil {
			return err
		}
		expressionAttributeValues[name] = converted
	}

	item, err := h.service.UpdateItem(
		tableName,
		key,
		request.UpdateExpression,
		request.ConditionExpression,
		request.ExpressionAttributeNames,
		expressionAttributeValues,
	)
	if err != nil {
		return err
	}

	response := updateItemResponse{}
	if strings.EqualFold(strings.TrimSpace(request.ReturnValues), "ALL_NEW") {
		response.Attributes = attributeValueMapFromDomain(item.Attributes)
	}
	writeDynamoDBJSON(c, http.StatusOK, response)
	return nil
}

func (h *dynamoDBNativeHandler) handleQuery(c *gin.Context, body []byte) error {
	request := queryRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid Query request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}
	if strings.TrimSpace(request.KeyConditionExpression) == "" {
		return fmt.Errorf("dynamodb: key condition expression is required")
	}
	if strings.TrimSpace(request.IndexName) != "" {
		return fmt.Errorf("dynamodb: index queries are not supported")
	}
	if strings.TrimSpace(request.ProjectionExpression) != "" {
		return fmt.Errorf("dynamodb: projection expressions are not supported")
	}
	if selectValue := strings.ToUpper(strings.TrimSpace(request.Select)); selectValue != "" && selectValue != "ALL_ATTRIBUTES" {
		return fmt.Errorf("dynamodb: unsupported select value %q", request.Select)
	}

	expressionAttributeValues := make(map[string]dynamodbdomain.AttributeValue, len(request.ExpressionAttributeValues))
	for name, value := range request.ExpressionAttributeValues {
		converted, err := attributeValueToDomain(value)
		if err != nil {
			return err
		}
		expressionAttributeValues[name] = converted
	}

	exclusiveStartKey, err := attributeValueMapToDomain(request.ExclusiveStartKey)
	if err != nil {
		return err
	}

	result, err := h.service.Query(
		tableName,
		request.KeyConditionExpression,
		request.FilterExpression,
		request.ExpressionAttributeNames,
		expressionAttributeValues,
		request.Limit,
		exclusiveStartKey,
		request.ScanIndexForward,
	)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, queryResponse{
		Items:            attributeValueListFromDomain(result.Items),
		Count:            result.Count,
		ScannedCount:     result.ScannedCount,
		LastEvaluatedKey: attributeValueMapFromDomain(result.LastEvaluatedKey),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleScan(c *gin.Context, body []byte) error {
	request := scanRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid Scan request: %w", err)
	}

	tableName := strings.TrimSpace(request.TableName)
	if tableName == "" {
		return fmt.Errorf("dynamodb: table name is required")
	}
	if strings.TrimSpace(request.IndexName) != "" {
		return fmt.Errorf("dynamodb: index scans are not supported")
	}
	if strings.TrimSpace(request.ProjectionExpression) != "" {
		return fmt.Errorf("dynamodb: projection expressions are not supported")
	}
	if selectValue := strings.ToUpper(strings.TrimSpace(request.Select)); selectValue != "" && selectValue != "ALL_ATTRIBUTES" {
		return fmt.Errorf("dynamodb: unsupported select value %q", request.Select)
	}
	if request.Segment != nil || request.TotalSegments != nil {
		return fmt.Errorf("dynamodb: parallel scan segments are not supported")
	}

	expressionAttributeValues := make(map[string]dynamodbdomain.AttributeValue, len(request.ExpressionAttributeValues))
	for name, value := range request.ExpressionAttributeValues {
		converted, err := attributeValueToDomain(value)
		if err != nil {
			return err
		}
		expressionAttributeValues[name] = converted
	}

	exclusiveStartKey, err := attributeValueMapToDomain(request.ExclusiveStartKey)
	if err != nil {
		return err
	}

	result, err := h.service.Scan(
		tableName,
		request.FilterExpression,
		request.ExpressionAttributeNames,
		expressionAttributeValues,
		request.Limit,
		exclusiveStartKey,
	)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, scanResponse{
		Items:            attributeValueListFromDomain(result.Items),
		Count:            result.Count,
		ScannedCount:     result.ScannedCount,
		LastEvaluatedKey: attributeValueMapFromDomain(result.LastEvaluatedKey),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleBatchWriteItem(c *gin.Context, body []byte) error {
	request := batchWriteItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid BatchWriteItem request: %w", err)
	}
	if strings.TrimSpace(request.ReturnConsumedCapacity) != "" {
		return fmt.Errorf("dynamodb: return consumed capacity is not supported")
	}
	if strings.TrimSpace(request.ReturnItemCollectionMetrics) != "" {
		return fmt.Errorf("dynamodb: return item collection metrics is not supported")
	}
	if len(request.RequestItems) == 0 {
		return fmt.Errorf("dynamodb: request items are required")
	}

	tableNames := make([]string, 0, len(request.RequestItems))
	for tableName := range request.RequestItems {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	appRequest := ddbcontracts.BatchWriteItemRequest{Tables: make([]ddbcontracts.BatchWriteTableRequest, 0, len(tableNames))}
	for _, tableName := range tableNames {
		writeRequests := request.RequestItems[tableName]
		tableRequest := ddbcontracts.BatchWriteTableRequest{
			Table:    tableName,
			Requests: make([]ddbcontracts.BatchWriteRequestItem, 0, len(writeRequests)),
		}
		for _, writeRequest := range writeRequests {
			putSet := writeRequest.PutRequest != nil
			deleteSet := writeRequest.DeleteRequest != nil
			if putSet == deleteSet {
				return fmt.Errorf("dynamodb: each batch write item must contain exactly one PutRequest or DeleteRequest")
			}

			if putSet {
				item, err := attributeValueMapToDomain(writeRequest.PutRequest.Item)
				if err != nil {
					return err
				}
				tableRequest.Requests = append(tableRequest.Requests, ddbcontracts.BatchWriteRequestItem{
					PutItem: item,
				})
				continue
			}

			key, err := attributeValueMapToDomain(writeRequest.DeleteRequest.Key)
			if err != nil {
				return err
			}
			tableRequest.Requests = append(tableRequest.Requests, ddbcontracts.BatchWriteRequestItem{
				DeleteKey: key,
			})
		}
		appRequest.Tables = append(appRequest.Tables, tableRequest)
	}

	result, err := h.service.BatchWriteItem(appRequest)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, batchWriteItemResponse{
		UnprocessedItems: batchWriteUnprocessedItemsFromDomain(result.Unprocessed),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleBatchGetItem(c *gin.Context, body []byte) error {
	request := batchGetItemRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid BatchGetItem request: %w", err)
	}
	if strings.TrimSpace(request.ReturnConsumedCapacity) != "" {
		return fmt.Errorf("dynamodb: return consumed capacity is not supported")
	}
	if len(request.RequestItems) == 0 {
		return fmt.Errorf("dynamodb: request items are required")
	}

	tableNames := make([]string, 0, len(request.RequestItems))
	for tableName := range request.RequestItems {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	appRequest := ddbcontracts.BatchGetItemRequest{Tables: make([]ddbcontracts.BatchGetTableRequest, 0, len(tableNames))}
	for _, tableName := range tableNames {
		tableRequest := request.RequestItems[tableName]
		if strings.TrimSpace(tableRequest.ProjectionExpression) != "" {
			return fmt.Errorf("dynamodb: projection expressions are not supported")
		}
		if len(tableRequest.ExpressionAttributeNames) > 0 {
			return fmt.Errorf("dynamodb: expression attribute names are not supported")
		}
		keys := make([]map[string]dynamodbdomain.AttributeValue, 0, len(tableRequest.Keys))
		for _, keyDocument := range tableRequest.Keys {
			key, err := attributeValueMapToDomain(keyDocument)
			if err != nil {
				return err
			}
			keys = append(keys, key)
		}
		appRequest.Tables = append(appRequest.Tables, ddbcontracts.BatchGetTableRequest{
			Table:          tableName,
			Keys:           keys,
			ConsistentRead: tableRequest.ConsistentRead,
		})
	}

	result, err := h.service.BatchGetItem(appRequest)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, batchGetItemResponse{
		Responses:       batchGetResponsesFromDomain(result.Responses),
		UnprocessedKeys: batchGetUnprocessedKeysFromDomain(result.Unprocessed),
	})
	return nil
}

func (h *dynamoDBNativeHandler) handleTransactWriteItems(c *gin.Context, body []byte) error {
	request := transactWriteItemsRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid TransactWriteItems request: %w", err)
	}
	if strings.TrimSpace(request.ReturnConsumedCapacity) != "" {
		return fmt.Errorf("dynamodb: return consumed capacity is not supported")
	}
	if strings.TrimSpace(request.ReturnItemCollectionMetrics) != "" {
		return fmt.Errorf("dynamodb: return item collection metrics is not supported")
	}
	if len(request.TransactItems) == 0 {
		return fmt.Errorf("dynamodb: transaction items are required")
	}

	appRequest := ddbcontracts.TransactWriteItemsRequest{Items: make([]ddbcontracts.TransactWriteItem, 0, len(request.TransactItems))}
	for _, item := range request.TransactItems {
		switch {
		case item.Put != nil:
			if item.Delete != nil || item.Update != nil || item.ConditionCheck != nil {
				return fmt.Errorf("dynamodb: each transaction item must contain exactly one operation")
			}
			if strings.TrimSpace(item.Put.ConditionExpression) != "" || len(item.Put.ExpressionAttributeNames) > 0 || len(item.Put.ExpressionAttributeValues) > 0 || strings.TrimSpace(item.Put.ReturnValuesOnConditionCheckFailure) != "" {
				return fmt.Errorf("dynamodb: condition expressions and return values on check failure are not supported")
			}
			if strings.TrimSpace(item.Put.TableName) == "" {
				return fmt.Errorf("dynamodb: table name is required")
			}
			putItem, err := attributeValueMapToDomain(item.Put.Item)
			if err != nil {
				return err
			}
			appRequest.Items = append(appRequest.Items, ddbcontracts.TransactWriteItem{
				Table:   item.Put.TableName,
				PutItem: putItem,
			})
		case item.Delete != nil:
			if item.Put != nil || item.Update != nil || item.ConditionCheck != nil {
				return fmt.Errorf("dynamodb: each transaction item must contain exactly one operation")
			}
			if strings.TrimSpace(item.Delete.ConditionExpression) != "" || len(item.Delete.ExpressionAttributeNames) > 0 || len(item.Delete.ExpressionAttributeValues) > 0 || strings.TrimSpace(item.Delete.ReturnValuesOnConditionCheckFailure) != "" {
				return fmt.Errorf("dynamodb: condition expressions and return values on check failure are not supported")
			}
			if strings.TrimSpace(item.Delete.TableName) == "" {
				return fmt.Errorf("dynamodb: table name is required")
			}
			deleteKey, err := attributeValueMapToDomain(item.Delete.Key)
			if err != nil {
				return err
			}
			appRequest.Items = append(appRequest.Items, ddbcontracts.TransactWriteItem{
				Table:     item.Delete.TableName,
				DeleteKey: deleteKey,
			})
		case item.Update != nil || item.ConditionCheck != nil:
			return fmt.Errorf("dynamodb: update and condition check transaction items are not supported")
		default:
			return fmt.Errorf("dynamodb: each transaction item must contain exactly one operation")
		}
	}

	err := h.service.TransactWriteItems(appRequest)
	if err != nil {
		var canceled interface {
			CancellationReasons() []ddbcontracts.TransactionCanceledReason
		}
		if errors.As(err, &canceled) {
			writeDynamoDBTransactionCanceledError(c, err)
			return nil
		}
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, transactWriteItemsResponse{})
	return nil
}

func (h *dynamoDBNativeHandler) handleTransactGetItems(c *gin.Context, body []byte) error {
	request := transactGetItemsRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return fmt.Errorf("dynamodb: invalid TransactGetItems request: %w", err)
	}
	if strings.TrimSpace(request.ReturnConsumedCapacity) != "" {
		return fmt.Errorf("dynamodb: return consumed capacity is not supported")
	}
	if len(request.TransactItems) == 0 {
		return fmt.Errorf("dynamodb: transaction items are required")
	}

	appRequest := ddbcontracts.TransactGetItemsRequest{Items: make([]ddbcontracts.TransactGetItem, 0, len(request.TransactItems))}
	for _, item := range request.TransactItems {
		if item.Get == nil {
			return fmt.Errorf("dynamodb: each transact get item must contain a Get operation")
		}
		if strings.TrimSpace(item.Get.TableName) == "" {
			return fmt.Errorf("dynamodb: table name is required")
		}
		if strings.TrimSpace(item.Get.ProjectionExpression) != "" || len(item.Get.ExpressionAttributeNames) > 0 {
			return fmt.Errorf("dynamodb: projection expressions are not supported")
		}
		key, err := attributeValueMapToDomain(item.Get.Key)
		if err != nil {
			return err
		}
		appRequest.Items = append(appRequest.Items, ddbcontracts.TransactGetItem{
			Table: item.Get.TableName,
			Key:   key,
		})
	}

	result, err := h.service.TransactGetItems(appRequest)
	if err != nil {
		return err
	}

	writeDynamoDBJSON(c, http.StatusOK, transactGetItemsResponse{
		Responses: transactGetResponsesFromDomain(result.Items),
	})
	return nil
}

func isDynamoDBJSONRequest(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return false
	}
	return strings.EqualFold(mediaType, dynamoDBJSONContentType)
}

func parseDynamoDBTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "", errors.New("dynamodb: X-Amz-Target header is required")
	}
	if !strings.HasPrefix(target, dynamoDBTargetPrefix) {
		return "", fmt.Errorf("dynamodb: X-Amz-Target %q must start with %q", target, dynamoDBTargetPrefix)
	}

	operation := strings.TrimSpace(strings.TrimPrefix(target, dynamoDBTargetPrefix))
	if operation == "" {
		return "", fmt.Errorf("dynamodb: X-Amz-Target %q is missing an operation name", target)
	}
	return operation, nil
}

func writeDynamoDBJSON(c *gin.Context, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		writeDynamoDBError(c, http.StatusInternalServerError, "InternalServerError", fmt.Sprintf("dynamodb: marshal response: %v", err))
		return
	}

	c.Data(status, dynamoDBJSONContentType, data)
}

func writeDynamoDBError(c *gin.Context, status int, code, message string) {
	c.Header("x-amzn-errortype", code)
	data, err := json.Marshal(dynamoDBErrorResponse{
		Type:    dynamoDBErrorPrefix + code,
		Message: message,
	})
	if err != nil {
		data = []byte(`{"__type":"` + dynamoDBErrorPrefix + `InternalServerError","message":"dynamodb: marshal error response"}`)
		status = http.StatusInternalServerError
	}

	c.Data(status, dynamoDBJSONContentType, data)
}

func dynamoDBErrorCode(err error) string {
	if err == nil {
		return "InternalServerError"
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "transaction canceled"):
		return "TransactionCanceledException"
	case strings.Contains(message, "already exists"):
		return "ResourceInUseException"
	case strings.Contains(message, "still creating"):
		return "ResourceInUseException"
	case strings.Contains(message, "not found"):
		return "ResourceNotFoundException"
	case strings.Contains(message, "conditional check failed"):
		return "ConditionalCheckFailedException"
	case strings.Contains(message, "required"):
		return "ValidationException"
	case strings.Contains(message, "invalid"):
		return "ValidationException"
	case strings.Contains(message, "unsupported"):
		return "ValidationException"
	case strings.Contains(message, "not supported"):
		return "ValidationException"
	default:
		return "InternalServerError"
	}
}

func writeDynamoDBTransactionCanceledError(c *gin.Context, err error) {
	var canceled interface {
		CancellationReasons() []ddbcontracts.TransactionCanceledReason
	}
	if !errors.As(err, &canceled) {
		writeDynamoDBError(c, http.StatusBadRequest, dynamoDBErrorCode(err), err.Error())
		return
	}

	reasons := canceled.CancellationReasons()
	payload := dynamoDBTransactionCanceledErrorResponse{
		Type:    dynamoDBErrorPrefix + "TransactionCanceledException",
		Message: err.Error(),
	}
	if len(reasons) > 0 {
		payload.CancellationReasons = make([]dynamoCancellationReason, len(reasons))
		for i, reason := range reasons {
			payload.CancellationReasons[i] = dynamoCancellationReason{
				Code:    reason.Code,
				Message: reason.Message,
				Item:    attributeValueMapFromDomain(reason.Item),
			}
		}
	}

	c.Header("x-amzn-errortype", "TransactionCanceledException")
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		writeDynamoDBError(c, http.StatusInternalServerError, "InternalServerError", fmt.Sprintf("dynamodb: marshal error response: %v", marshalErr))
		return
	}

	c.Data(http.StatusBadRequest, dynamoDBJSONContentType, data)
}

type dynamoDBErrorResponse struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

type dynamoDBTransactionCanceledErrorResponse struct {
	Type                string                     `json:"__type"`
	Message             string                     `json:"message"`
	CancellationReasons []dynamoCancellationReason `json:"CancellationReasons,omitempty"`
}

type dynamoCancellationReason struct {
	Code    string                          `json:"Code,omitempty"`
	Message string                          `json:"Message,omitempty"`
	Item    map[string]dynamoAttributeValue `json:"Item,omitempty"`
}

type listTablesRequest struct {
	Limit                   *int   `json:"Limit,omitempty"`
	ExclusiveStartTableName string `json:"ExclusiveStartTableName,omitempty"`
}

type listTablesResponse struct {
	TableNames             []string `json:"TableNames"`
	LastEvaluatedTableName string   `json:"LastEvaluatedTableName,omitempty"`
}

type createTableRequest struct {
	TableName            string                      `json:"TableName"`
	BillingMode          string                      `json:"BillingMode,omitempty"`
	KeySchema            []dynamoKeySchemaElement    `json:"KeySchema,omitempty"`
	AttributeDefinitions []dynamoAttributeDefinition `json:"AttributeDefinitions,omitempty"`
}

type createTableResponse struct {
	TableDescription tableDescription `json:"TableDescription"`
}

type describeTableRequest struct {
	TableName string `json:"TableName"`
}

type describeTableResponse struct {
	Table tableDescription `json:"Table"`
}

type deleteTableRequest struct {
	TableName string `json:"TableName"`
}

type deleteTableResponse struct {
	TableDescription tableDescription `json:"TableDescription"`
}

type getItemRequest struct {
	TableName string                          `json:"TableName"`
	Key       map[string]dynamoAttributeValue `json:"Key"`
}

type getItemResponse struct {
	Item map[string]dynamoAttributeValue `json:"Item,omitempty"`
}

type putItemRequest struct {
	TableName string                          `json:"TableName"`
	Item      map[string]dynamoAttributeValue `json:"Item"`
}

type putItemResponse struct {
	Attributes map[string]dynamoAttributeValue `json:"Attributes,omitempty"`
}

type deleteItemRequest struct {
	TableName string                          `json:"TableName"`
	Key       map[string]dynamoAttributeValue `json:"Key"`
}

type updateItemRequest struct {
	TableName                 string                          `json:"TableName"`
	Key                       map[string]dynamoAttributeValue `json:"Key"`
	UpdateExpression          string                          `json:"UpdateExpression"`
	ConditionExpression       string                          `json:"ConditionExpression,omitempty"`
	ExpressionAttributeNames  map[string]string               `json:"ExpressionAttributeNames,omitempty"`
	ExpressionAttributeValues map[string]dynamoAttributeValue `json:"ExpressionAttributeValues,omitempty"`
	ReturnValues              string                          `json:"ReturnValues,omitempty"`
}

type deleteItemResponse struct{}

type updateItemResponse struct {
	Attributes map[string]dynamoAttributeValue `json:"Attributes,omitempty"`
}

type queryRequest struct {
	TableName                 string                          `json:"TableName"`
	KeyConditionExpression    string                          `json:"KeyConditionExpression"`
	FilterExpression          string                          `json:"FilterExpression,omitempty"`
	ExpressionAttributeNames  map[string]string               `json:"ExpressionAttributeNames,omitempty"`
	ExpressionAttributeValues map[string]dynamoAttributeValue `json:"ExpressionAttributeValues,omitempty"`
	Limit                     *int                            `json:"Limit,omitempty"`
	ExclusiveStartKey         map[string]dynamoAttributeValue `json:"ExclusiveStartKey,omitempty"`
	ScanIndexForward          *bool                           `json:"ScanIndexForward,omitempty"`
	IndexName                 string                          `json:"IndexName,omitempty"`
	ProjectionExpression      string                          `json:"ProjectionExpression,omitempty"`
	Select                    string                          `json:"Select,omitempty"`
}

type scanRequest struct {
	TableName                 string                          `json:"TableName"`
	FilterExpression          string                          `json:"FilterExpression,omitempty"`
	ExpressionAttributeNames  map[string]string               `json:"ExpressionAttributeNames,omitempty"`
	ExpressionAttributeValues map[string]dynamoAttributeValue `json:"ExpressionAttributeValues,omitempty"`
	Limit                     *int                            `json:"Limit,omitempty"`
	ExclusiveStartKey         map[string]dynamoAttributeValue `json:"ExclusiveStartKey,omitempty"`
	IndexName                 string                          `json:"IndexName,omitempty"`
	ProjectionExpression      string                          `json:"ProjectionExpression,omitempty"`
	Select                    string                          `json:"Select,omitempty"`
	Segment                   *int                            `json:"Segment,omitempty"`
	TotalSegments             *int                            `json:"TotalSegments,omitempty"`
}

type queryResponse struct {
	Items            []map[string]dynamoAttributeValue `json:"Items,omitempty"`
	Count            int                               `json:"Count,omitempty"`
	ScannedCount     int                               `json:"ScannedCount,omitempty"`
	LastEvaluatedKey map[string]dynamoAttributeValue   `json:"LastEvaluatedKey,omitempty"`
}

type scanResponse struct {
	Items            []map[string]dynamoAttributeValue `json:"Items,omitempty"`
	Count            int                               `json:"Count,omitempty"`
	ScannedCount     int                               `json:"ScannedCount,omitempty"`
	LastEvaluatedKey map[string]dynamoAttributeValue   `json:"LastEvaluatedKey,omitempty"`
}

type batchWriteItemRequest struct {
	RequestItems                map[string][]batchWriteRequest `json:"RequestItems"`
	ReturnConsumedCapacity      string                         `json:"ReturnConsumedCapacity,omitempty"`
	ReturnItemCollectionMetrics string                         `json:"ReturnItemCollectionMetrics,omitempty"`
}

type batchWriteRequest struct {
	PutRequest    *batchWritePutRequest    `json:"PutRequest,omitempty"`
	DeleteRequest *batchWriteDeleteRequest `json:"DeleteRequest,omitempty"`
}

type batchWritePutRequest struct {
	Item map[string]dynamoAttributeValue `json:"Item"`
}

type batchWriteDeleteRequest struct {
	Key map[string]dynamoAttributeValue `json:"Key"`
}

type batchWriteItemResponse struct {
	UnprocessedItems map[string][]batchWriteRequest `json:"UnprocessedItems,omitempty"`
}

type batchGetItemRequest struct {
	RequestItems           map[string]batchGetTableRequest `json:"RequestItems"`
	ReturnConsumedCapacity string                          `json:"ReturnConsumedCapacity,omitempty"`
}

type batchGetTableRequest struct {
	Keys                     []map[string]dynamoAttributeValue `json:"Keys"`
	ConsistentRead           *bool                             `json:"ConsistentRead,omitempty"`
	ProjectionExpression     string                            `json:"ProjectionExpression,omitempty"`
	ExpressionAttributeNames map[string]string                 `json:"ExpressionAttributeNames,omitempty"`
}

type batchGetItemResponse struct {
	Responses       map[string][]map[string]dynamoAttributeValue `json:"Responses,omitempty"`
	UnprocessedKeys map[string]batchGetTableRequest              `json:"UnprocessedKeys,omitempty"`
}

type transactWriteItemsRequest struct {
	TransactItems               []transactWriteItem `json:"TransactItems"`
	ClientRequestToken          string              `json:"ClientRequestToken,omitempty"`
	ReturnConsumedCapacity      string              `json:"ReturnConsumedCapacity,omitempty"`
	ReturnItemCollectionMetrics string              `json:"ReturnItemCollectionMetrics,omitempty"`
}

type transactWriteItem struct {
	Put            *transactWritePutRequest            `json:"Put,omitempty"`
	Delete         *transactWriteDeleteRequest         `json:"Delete,omitempty"`
	Update         *transactWriteUpdateRequest         `json:"Update,omitempty"`
	ConditionCheck *transactWriteConditionCheckRequest `json:"ConditionCheck,omitempty"`
}

type transactWritePutRequest struct {
	TableName                           string                          `json:"TableName"`
	Item                                map[string]dynamoAttributeValue `json:"Item"`
	ConditionExpression                 string                          `json:"ConditionExpression,omitempty"`
	ExpressionAttributeNames            map[string]string               `json:"ExpressionAttributeNames,omitempty"`
	ExpressionAttributeValues           map[string]dynamoAttributeValue `json:"ExpressionAttributeValues,omitempty"`
	ReturnValuesOnConditionCheckFailure string                          `json:"ReturnValuesOnConditionCheckFailure,omitempty"`
}

type transactWriteDeleteRequest struct {
	TableName                           string                          `json:"TableName"`
	Key                                 map[string]dynamoAttributeValue `json:"Key"`
	ConditionExpression                 string                          `json:"ConditionExpression,omitempty"`
	ExpressionAttributeNames            map[string]string               `json:"ExpressionAttributeNames,omitempty"`
	ExpressionAttributeValues           map[string]dynamoAttributeValue `json:"ExpressionAttributeValues,omitempty"`
	ReturnValuesOnConditionCheckFailure string                          `json:"ReturnValuesOnConditionCheckFailure,omitempty"`
}

type transactWriteUpdateRequest struct {
	TableName string `json:"TableName"`
}

type transactWriteConditionCheckRequest struct {
	TableName string `json:"TableName"`
}

type transactWriteItemsResponse struct{}

type transactGetItemsRequest struct {
	TransactItems          []transactGetItem `json:"TransactItems"`
	ReturnConsumedCapacity string            `json:"ReturnConsumedCapacity,omitempty"`
}

type transactGetItem struct {
	Get *transactGetGetRequest `json:"Get,omitempty"`
}

type transactGetGetRequest struct {
	TableName                string                          `json:"TableName"`
	Key                      map[string]dynamoAttributeValue `json:"Key"`
	ProjectionExpression     string                          `json:"ProjectionExpression,omitempty"`
	ExpressionAttributeNames map[string]string               `json:"ExpressionAttributeNames,omitempty"`
}

type transactGetItemsResponse struct {
	Responses []transactGetItemResponse `json:"Responses,omitempty"`
}

type transactGetItemResponse struct {
	Item map[string]dynamoAttributeValue `json:"Item,omitempty"`
}

type dynamoAttributeValue struct {
	S    string                          `json:"S,omitempty"`
	N    string                          `json:"N,omitempty"`
	BOOL *bool                           `json:"BOOL,omitempty"`
	NULL bool                            `json:"NULL,omitempty"`
	M    map[string]dynamoAttributeValue `json:"M,omitempty"`
	L    []dynamoAttributeValue          `json:"L,omitempty"`
}

type dynamoKeySchemaElement struct {
	AttributeName string `json:"AttributeName"`
	KeyType       string `json:"KeyType"`
}

type dynamoAttributeDefinition struct {
	AttributeName string `json:"AttributeName"`
	AttributeType string `json:"AttributeType"`
}

type tableDescription struct {
	TableName            string                      `json:"TableName"`
	TableStatus          string                      `json:"TableStatus"`
	TableArn             string                      `json:"TableArn,omitempty"`
	CreationDateTime     int64                       `json:"CreationDateTime,omitempty"`
	KeySchema            []dynamoKeySchemaElement    `json:"KeySchema,omitempty"`
	AttributeDefinitions []dynamoAttributeDefinition `json:"AttributeDefinitions,omitempty"`
	BillingModeSummary   *billingModeSummary         `json:"BillingModeSummary,omitempty"`
}

type billingModeSummary struct {
	BillingMode string `json:"BillingMode"`
}

func billingModeSummaryFor(billingMode string) *billingModeSummary {
	billingMode = strings.TrimSpace(billingMode)
	if billingMode == "" {
		return nil
	}
	return &billingModeSummary{BillingMode: billingMode}
}

func partitionAndSortKeys(keySchema []dynamoKeySchemaElement) (string, string) {
	partitionKey := defaultPartitionKeyName()
	sortKey := ""
	for _, element := range keySchema {
		switch strings.ToUpper(strings.TrimSpace(element.KeyType)) {
		case "HASH":
			if name := strings.TrimSpace(element.AttributeName); name != "" {
				partitionKey = name
			}
		case "RANGE":
			if name := strings.TrimSpace(element.AttributeName); name != "" {
				sortKey = name
			}
		}
	}
	return partitionKey, sortKey
}

func defaultPartitionKeyName() string {
	return "id"
}

func cloneKeySchema(source []dynamoKeySchemaElement, partitionKey, sortKey string) []dynamoKeySchemaElement {
	if len(source) > 0 {
		cloned := make([]dynamoKeySchemaElement, len(source))
		copy(cloned, source)
		return cloned
	}

	keySchema := []dynamoKeySchemaElement{
		{
			AttributeName: partitionKey,
			KeyType:       "HASH",
		},
	}
	if strings.TrimSpace(sortKey) != "" {
		keySchema = append(keySchema, dynamoKeySchemaElement{
			AttributeName: sortKey,
			KeyType:       "RANGE",
		})
	}
	return keySchema
}

func cloneAttributeDefinitions(source []dynamoAttributeDefinition) []dynamoAttributeDefinition {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]dynamoAttributeDefinition, len(source))
	copy(cloned, source)
	return cloned
}

func tableDescriptionFromDomain(table dynamodbdomain.Table) tableDescription {
	aws := awscontext.Default()
	return tableDescription{
		TableName:          table.Name,
		TableStatus:        table.Status,
		TableArn:           aws.DynamoDBTableARN(table.Name),
		CreationDateTime:   awsTimestamp(table.CreatedAt),
		KeySchema:          cloneKeySchema(nil, table.PartitionKey, table.SortKey),
		BillingModeSummary: billingModeSummaryFor(table.BillingMode),
	}
}

func awsTimestamp(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func keyFromAttributeValueMap(values map[string]dynamoAttributeValue) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("dynamodb: key is required")
	}

	if key, ok, err := syntheticItemKey(values); ok || err != nil {
		return key, err
	}

	if len(values) == 1 {
		for _, value := range values {
			return attributeValueToString(value)
		}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return "", fmt.Errorf("dynamodb: unsupported key attributes %q", strings.Join(keys, ", "))
}

func itemFromAttributeValueMap(values map[string]dynamoAttributeValue) (string, map[string]dynamodbdomain.AttributeValue, error) {
	if len(values) == 0 {
		return "", nil, fmt.Errorf("dynamodb: item is required")
	}

	attributes := make(map[string]dynamodbdomain.AttributeValue, len(values))
	key := ""
	var err error
	for name, value := range values {
		copied, err := attributeValueToDomain(value)
		if err != nil {
			return "", nil, err
		}
		attributes[name] = copied
	}

	key, _, err = syntheticItemKey(values)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(key) == "" {
		return "", nil, fmt.Errorf("dynamodb: item key is required")
	}

	return key, attributes, nil
}

func syntheticItemKey(values map[string]dynamoAttributeValue) (string, bool, error) {
	if len(values) == 0 {
		return "", false, nil
	}

	if idValue, ok := values["id"]; ok {
		id, err := attributeValueToString(idValue)
		if err != nil {
			return "", false, err
		}
		if skValue, ok := values["sk"]; ok {
			sk, err := attributeValueToString(skValue)
			if err != nil {
				return "", false, err
			}
			if strings.TrimSpace(sk) != "" {
				return id + "|" + sk, true, nil
			}
		}
		return id, true, nil
	}

	if len(values) == 1 {
		for _, value := range values {
			key, err := attributeValueToString(value)
			return key, true, err
		}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return "", false, fmt.Errorf("dynamodb: unsupported key attributes %q", strings.Join(keys, ", "))
}

func attributeValueToString(value dynamoAttributeValue) (string, error) {
	switch {
	case value.S != "":
		return value.S, nil
	case value.N != "":
		return value.N, nil
	case value.BOOL != nil:
		return strconv.FormatBool(*value.BOOL), nil
	case value.NULL:
		return "", fmt.Errorf("dynamodb: null attribute values are not supported")
	default:
		return "", fmt.Errorf("dynamodb: only string and number attribute values are supported in the local subset")
	}
}

func attributeValueToDomain(value dynamoAttributeValue) (dynamodbdomain.AttributeValue, error) {
	switch {
	case value.S != "":
		return dynamodbdomain.StringValue(value.S), nil
	case value.N != "":
		return dynamodbdomain.NumberValue(value.N), nil
	case value.BOOL != nil:
		return dynamodbdomain.BoolValue(*value.BOOL), nil
	case value.NULL:
		return dynamodbdomain.NullValue(), nil
	case value.M != nil:
		copied := make(map[string]dynamodbdomain.AttributeValue, len(value.M))
		for name, child := range value.M {
			converted, err := attributeValueToDomain(child)
			if err != nil {
				return dynamodbdomain.AttributeValue{}, err
			}
			copied[name] = converted
		}
		return dynamodbdomain.MapValue(copied), nil
	case value.L != nil:
		copied := make([]dynamodbdomain.AttributeValue, len(value.L))
		for i, child := range value.L {
			converted, err := attributeValueToDomain(child)
			if err != nil {
				return dynamodbdomain.AttributeValue{}, err
			}
			copied[i] = converted
		}
		return dynamodbdomain.ListValue(copied), nil
	default:
		return dynamodbdomain.AttributeValue{}, fmt.Errorf("dynamodb: only string, number, bool, null, map, and list attribute values are supported in the local subset")
	}
}

func attributeValueMapFromDomain(attributes map[string]dynamodbdomain.AttributeValue) map[string]dynamoAttributeValue {
	if attributes == nil {
		return nil
	}

	copied := make(map[string]dynamoAttributeValue, len(attributes))
	for name, value := range attributes {
		copied[name] = dynamoAttributeValueFromDomain(value)
	}
	return copied
}

func dynamoAttributeValueFromDomain(value dynamodbdomain.AttributeValue) dynamoAttributeValue {
	switch {
	case value.S != nil:
		return dynamoAttributeValue{S: *value.S}
	case value.N != nil:
		return dynamoAttributeValue{N: *value.N}
	case value.BOOL != nil:
		return dynamoAttributeValue{BOOL: value.BOOL}
	case value.NULL:
		return dynamoAttributeValue{NULL: true}
	case value.M != nil:
		copied := make(map[string]dynamoAttributeValue, len(*value.M))
		for name, child := range *value.M {
			copied[name] = dynamoAttributeValueFromDomain(child)
		}
		return dynamoAttributeValue{M: copied}
	case value.L != nil:
		copied := make([]dynamoAttributeValue, len(*value.L))
		for i, child := range *value.L {
			copied[i] = dynamoAttributeValueFromDomain(child)
		}
		return dynamoAttributeValue{L: copied}
	default:
		return dynamoAttributeValue{S: ""}
	}
}

func attributeValueListFromDomain(items []dynamodbdomain.Item) []map[string]dynamoAttributeValue {
	if len(items) == 0 {
		return nil
	}

	copied := make([]map[string]dynamoAttributeValue, len(items))
	for i, item := range items {
		copied[i] = attributeValueMapFromDomain(item.Attributes)
	}
	return copied
}

func attributeValueMapToDomain(values map[string]dynamoAttributeValue) (map[string]dynamodbdomain.AttributeValue, error) {
	if len(values) == 0 {
		return nil, nil
	}

	converted := make(map[string]dynamodbdomain.AttributeValue, len(values))
	for name, value := range values {
		parsed, err := attributeValueToDomain(value)
		if err != nil {
			return nil, err
		}
		converted[name] = parsed
	}
	return converted, nil
}

func batchWriteUnprocessedItemsFromDomain(unprocessed []ddbcontracts.BatchWriteTableRequest) map[string][]batchWriteRequest {
	if len(unprocessed) == 0 {
		return nil
	}

	copied := make(map[string][]batchWriteRequest, len(unprocessed))
	for _, tableRequest := range unprocessed {
		requests := make([]batchWriteRequest, 0, len(tableRequest.Requests))
		for _, itemRequest := range tableRequest.Requests {
			requests = append(requests, batchWriteRequestFromDomain(itemRequest))
		}
		copied[tableRequest.Table] = requests
	}
	return copied
}

func batchWriteRequestFromDomain(request ddbcontracts.BatchWriteRequestItem) batchWriteRequest {
	if len(request.PutItem) > 0 {
		return batchWriteRequest{
			PutRequest: &batchWritePutRequest{Item: attributeValueMapFromDomain(request.PutItem)},
		}
	}
	return batchWriteRequest{
		DeleteRequest: &batchWriteDeleteRequest{Key: attributeValueMapFromDomain(request.DeleteKey)},
	}
}

func batchGetResponsesFromDomain(responses []ddbcontracts.BatchGetTableResponse) map[string][]map[string]dynamoAttributeValue {
	if len(responses) == 0 {
		return nil
	}

	copied := make(map[string][]map[string]dynamoAttributeValue, len(responses))
	for _, tableResponse := range responses {
		copied[tableResponse.Table] = attributeValueListFromDomain(tableResponse.Items)
	}
	return copied
}

func batchGetUnprocessedKeysFromDomain(unprocessed []ddbcontracts.BatchGetTableRequest) map[string]batchGetTableRequest {
	if len(unprocessed) == 0 {
		return nil
	}

	copied := make(map[string]batchGetTableRequest, len(unprocessed))
	for _, tableRequest := range unprocessed {
		copied[tableRequest.Table] = batchGetTableRequest{
			Keys:           cloneDynamoAttributeDocuments(tableRequest.Keys),
			ConsistentRead: tableRequest.ConsistentRead,
		}
	}
	return copied
}

func transactGetResponsesFromDomain(items []ddbcontracts.TransactGetItemResult) []transactGetItemResponse {
	if len(items) == 0 {
		return nil
	}

	responses := make([]transactGetItemResponse, len(items))
	for i, item := range items {
		if item.Item == nil {
			continue
		}
		responses[i] = transactGetItemResponse{
			Item: attributeValueMapFromDomain(item.Item.Attributes),
		}
	}
	return responses
}

func cloneDynamoAttributeDocuments(keys []map[string]dynamodbdomain.AttributeValue) []map[string]dynamoAttributeValue {
	if len(keys) == 0 {
		return nil
	}

	copied := make([]map[string]dynamoAttributeValue, len(keys))
	for i, key := range keys {
		copied[i] = attributeValueMapFromDomain(key)
	}
	return copied
}

func paginateTableNames(names []string, limit *int, exclusiveStart string) ([]string, string) {
	if len(names) == 0 {
		return nil, ""
	}

	start := 0
	if exclusiveStart = strings.TrimSpace(exclusiveStart); exclusiveStart != "" {
		for i, name := range names {
			if name == exclusiveStart {
				start = i + 1
				break
			}
		}
	}
	if start >= len(names) {
		return nil, ""
	}

	end := len(names)
	if limit != nil && *limit >= 0 && *limit < end-start {
		end = start + *limit
	}

	page := append([]string(nil), names[start:end]...)
	lastEvaluated := ""
	if end < len(names) && len(page) > 0 {
		lastEvaluated = page[len(page)-1]
	}
	return page, lastEvaluated
}
