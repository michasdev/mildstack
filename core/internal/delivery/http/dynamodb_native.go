package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	dynamodbdomain "github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

type DynamoDBNativeService interface {
	ListTables() []dynamodbdomain.Table
	CreateTable(name, partitionKey, sortKey, billingMode string) (dynamodbdomain.Table, error)
	DescribeTable(name string) (dynamodbdomain.Table, error)
	DeleteTable(name string) (dynamodbdomain.Table, error)
	GetItem(table, key string) (dynamodbdomain.Item, error)
	PutItem(table, key string, attributes map[string]string) (dynamodbdomain.Item, error)
	DeleteItem(table, key string) error
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
		"DeleteItem": {
			supported: true,
			execute:   (*dynamoDBNativeHandler).handleDeleteItem,
		},
		"UpdateTable":        {supported: false},
		"UpdateItem":         {supported: false},
		"Query":              {supported: false},
		"Scan":               {supported: false},
		"BatchGetItem":       {supported: false},
		"BatchWriteItem":     {supported: false},
		"TransactGetItems":   {supported: false},
		"TransactWriteItems": {supported: false},
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
	case strings.Contains(message, "already exists"):
		return "ResourceInUseException"
	case strings.Contains(message, "still creating"):
		return "ResourceInUseException"
	case strings.Contains(message, "not found"):
		return "ResourceNotFoundException"
	case strings.Contains(message, "required"):
		return "ValidationException"
	case strings.Contains(message, "invalid"):
		return "ValidationException"
	default:
		return "InternalServerError"
	}
}

type dynamoDBErrorResponse struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
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

type deleteItemResponse struct{}

type dynamoAttributeValue struct {
	S    string `json:"S,omitempty"`
	N    string `json:"N,omitempty"`
	BOOL *bool  `json:"BOOL,omitempty"`
	NULL bool   `json:"NULL,omitempty"`
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
	return tableDescription{
		TableName:          table.Name,
		TableStatus:        table.Status,
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

	if value, ok := values["id"]; ok {
		return attributeValueToString(value)
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

func itemFromAttributeValueMap(values map[string]dynamoAttributeValue) (string, map[string]string, error) {
	if len(values) == 0 {
		return "", nil, fmt.Errorf("dynamodb: item is required")
	}

	attributes := make(map[string]string, len(values))
	key := ""
	for name, value := range values {
		copied, err := attributeValueToString(value)
		if err != nil {
			return "", nil, err
		}
		attributes[name] = copied
		if key == "" && strings.EqualFold(name, "id") {
			key = copied
		}
	}

	if key == "" {
		if copied, ok := attributes["id"]; ok {
			key = copied
		}
	}
	if key == "" {
		for _, copied := range attributes {
			key = copied
			break
		}
	}
	if strings.TrimSpace(key) == "" {
		return "", nil, fmt.Errorf("dynamodb: item key is required")
	}

	return key, attributes, nil
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

func attributeValueMapFromDomain(attributes map[string]string) map[string]dynamoAttributeValue {
	if attributes == nil {
		return nil
	}

	copied := make(map[string]dynamoAttributeValue, len(attributes))
	for name, value := range attributes {
		copied[name] = dynamoAttributeValueFromString(value)
	}
	return copied
}

func dynamoAttributeValueFromString(value string) dynamoAttributeValue {
	if strings.TrimSpace(value) == "" {
		return dynamoAttributeValue{S: value}
	}

	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return dynamoAttributeValue{N: value}
	}

	return dynamoAttributeValue{S: value}
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
