package application

import (
	"fmt"
	"sort"
	"strings"

	ddbcontracts "github.com/michasdev/mildstack/core/internal/resources/dynamodb/contracts"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

type BatchWriteItemRequest = ddbcontracts.BatchWriteItemRequest
type BatchWriteTableRequest = ddbcontracts.BatchWriteTableRequest
type BatchWriteRequestItem = ddbcontracts.BatchWriteRequestItem
type BatchWriteItemResult = ddbcontracts.BatchWriteItemResult
type BatchGetItemRequest = ddbcontracts.BatchGetItemRequest
type BatchGetTableRequest = ddbcontracts.BatchGetTableRequest
type BatchGetTableResponse = ddbcontracts.BatchGetTableResponse
type BatchGetItemResult = ddbcontracts.BatchGetItemResult
type TransactWriteItem = ddbcontracts.TransactWriteItem
type TransactWriteItemsRequest = ddbcontracts.TransactWriteItemsRequest
type TransactGetItem = ddbcontracts.TransactGetItem
type TransactGetItemsRequest = ddbcontracts.TransactGetItemsRequest
type TransactGetItemResult = ddbcontracts.TransactGetItemResult
type TransactGetItemsResult = ddbcontracts.TransactGetItemsResult
type TransactionCanceledReason = ddbcontracts.TransactionCanceledReason
type TransactionCanceledError = ddbcontracts.TransactionCanceledError

const (
	batchWriteItemLimit     = ddbcontracts.BatchWriteItemLimit
	batchGetItemLimit       = ddbcontracts.BatchGetItemLimit
	transactItemLimit       = ddbcontracts.TransactItemLimit
	transactionConflictCode = "TransactionConflict"
)

func (s *Service) BatchWriteItem(request BatchWriteItemRequest) (BatchWriteItemResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := BatchWriteItemResult{}
	if len(request.Tables) == 0 {
		return result, fmt.Errorf("dynamodb: batch write requests are required")
	}

	next := s.state.Clone()
	processed := 0
	exhausted := false

	for tableIndex, tableRequest := range request.Tables {
		tableName := strings.TrimSpace(tableRequest.Table)
		if tableName == "" {
			return BatchWriteItemResult{}, fmt.Errorf("dynamodb: table name is required")
		}
		tableInfo, ok := next.Table(tableName)
		if !ok {
			return BatchWriteItemResult{}, fmt.Errorf("dynamodb: table %q not found", tableName)
		}

		if exhausted {
			result.Unprocessed = append(result.Unprocessed, cloneBatchWriteTableRequest(tableRequest))
			continue
		}

		if len(tableRequest.Requests) == 0 {
			continue
		}

		processedInTable := 0
		for _, itemRequest := range tableRequest.Requests {
			if processed >= batchWriteItemLimit {
				exhausted = true
				break
			}

			key, err := batchDocumentKey(tableInfo, itemRequest.PutItem, itemRequest.DeleteKey)
			if err != nil {
				return BatchWriteItemResult{}, err
			}

			if len(itemRequest.PutItem) > 0 {
				next.UpsertItem(domain.Item{
					Table:      tableName,
					Key:        key,
					Attributes: cloneAttributeDocument(itemRequest.PutItem),
				})
			} else {
				next.DeleteItem(tableName, key)
			}

			processed++
			processedInTable++
		}

		if exhausted && processedInTable < len(tableRequest.Requests) {
			result.Unprocessed = append(result.Unprocessed, BatchWriteTableRequest{
				Table:    tableName,
				Requests: cloneBatchWriteRequests(tableRequest.Requests[processedInTable:]),
			})
			for _, remaining := range request.Tables[tableIndex+1:] {
				result.Unprocessed = append(result.Unprocessed, cloneBatchWriteTableRequest(remaining))
			}
			break
		}
	}

	if err := s.commitStateLocked(next); err != nil {
		return BatchWriteItemResult{}, err
	}

	return result, nil
}

func (s *Service) BatchGetItem(request BatchGetItemRequest) (BatchGetItemResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := BatchGetItemResult{}
	if len(request.Tables) == 0 {
		return result, fmt.Errorf("dynamodb: batch get requests are required")
	}

	processed := 0
	exhausted := false

	for tableIndex, tableRequest := range request.Tables {
		tableName := strings.TrimSpace(tableRequest.Table)
		if tableName == "" {
			return BatchGetItemResult{}, fmt.Errorf("dynamodb: table name is required")
		}
		tableInfo, ok := s.state.Table(tableName)
		if !ok {
			return BatchGetItemResult{}, fmt.Errorf("dynamodb: table %q not found", tableName)
		}

		if exhausted {
			result.Unprocessed = append(result.Unprocessed, cloneBatchGetTableRequest(tableRequest))
			continue
		}

		processedInTable := 0
		tableResponse := BatchGetTableResponse{Table: tableName}
		for _, keyDocument := range tableRequest.Keys {
			if processed >= batchGetItemLimit {
				exhausted = true
				break
			}

			key, err := itemDocumentKey(tableInfo, keyDocument)
			if err != nil {
				return BatchGetItemResult{}, err
			}
			if item, ok := s.state.Item(tableName, key); ok {
				tableResponse.Items = append(tableResponse.Items, item)
			}
			processed++
			processedInTable++
		}

		if len(tableResponse.Items) > 0 {
			result.Responses = append(result.Responses, tableResponse)
		}

		if exhausted {
			if processedInTable < len(tableRequest.Keys) {
				result.Unprocessed = append(result.Unprocessed, BatchGetTableRequest{
					Table: tableName,
					Keys:  cloneKeyDocuments(tableRequest.Keys[processedInTable:]),
				})
			}
			for _, remainingTable := range request.Tables[tableIndex+1:] {
				result.Unprocessed = append(result.Unprocessed, cloneBatchGetTableRequest(remainingTable))
			}
			break
		}
	}

	return result, nil
}

func (s *Service) TransactWriteItems(request TransactWriteItemsRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(request.Items) == 0 {
		return fmt.Errorf("dynamodb: transaction items are required")
	}
	if len(request.Items) > transactItemLimit {
		return fmt.Errorf("dynamodb: transaction supports up to %d items in the local subset", transactItemLimit)
	}

	next := s.state.Clone()
	seen := make(map[string]int, len(request.Items))
	reasons := make([]TransactionCanceledReason, len(request.Items))

	for index, item := range request.Items {
		tableName := strings.TrimSpace(item.Table)
		if tableName == "" {
			return fmt.Errorf("dynamodb: table name is required")
		}
		tableInfo, ok := next.Table(tableName)
		if !ok {
			return fmt.Errorf("dynamodb: table %q not found", tableName)
		}

		key, err := transactDocumentKey(tableInfo, item)
		if err != nil {
			return err
		}

		if previous, ok := seen[tableName+"|"+key]; ok && len(item.ConditionCheckKey) == 0 && len(request.Items[previous].ConditionCheckKey) == 0 {
			reason := TransactionCanceledReason{
				Code:    transactionConflictCode,
				Message: "same item targeted more than once",
			}
			reasons[index] = reason
			reasons[previous] = reason
			return &TransactionCanceledError{Reasons: reasons}
		}
		seen[tableName+"|"+key] = index

		if len(item.PutItem) > 0 {
			next.UpsertItem(domain.Item{
				Table:      tableName,
				Key:        key,
				Attributes: cloneAttributeDocument(item.PutItem),
			})
			continue
		}

		if len(item.UpdateKey) > 0 {
			current, _ := next.Item(tableName, key)
			attrs := cloneDocument(current.Attributes)
			if attrs == nil {
				attrs = make(map[string]domain.AttributeValue)
			}

			if err := evaluateUpdateCondition(attrs, item.ConditionExpression, item.ExpressionAttributeNames, item.ExpressionAttributeValues); err != nil {
				return &TransactionCanceledError{Reasons: cancellationReasonsForTransaction(index, len(request.Items), "ConditionalCheckFailed", "The conditional request failed")}
			}

			operations, err := parseUpdateExpression(item.UpdateExpression, item.ExpressionAttributeNames, item.ExpressionAttributeValues)
			if err != nil {
				return err
			}
			for _, operation := range operations {
				path := operation.path
				if path == tableInfo.PartitionKey || (tableInfo.SortKey != "" && path == tableInfo.SortKey) {
					return fmt.Errorf("dynamodb: unsupported update to key attribute %q", path)
				}

				switch operation.kind {
				case "SET":
					attrs[path] = operation.value.Clone()
				case "REMOVE":
					delete(attrs, path)
				case "ADD":
					updated, err := addAttribute(attrs[path], operation.value)
					if err != nil {
						return err
					}
					attrs[path] = updated
				default:
					return fmt.Errorf("dynamodb: unsupported update operation %q", operation.kind)
				}
			}

			next.UpsertItem(domain.Item{
				Table:      tableName,
				Key:        key,
				Attributes: attrs,
			})
			continue
		}

		if len(item.ConditionCheckKey) > 0 {
			current, _ := next.Item(tableName, key)
			if err := evaluateUpdateCondition(current.Attributes, item.ConditionExpression, item.ExpressionAttributeNames, item.ExpressionAttributeValues); err != nil {
				return &TransactionCanceledError{Reasons: cancellationReasonsForTransaction(index, len(request.Items), "ConditionalCheckFailed", "The conditional request failed")}
			}
			continue
		}

		next.DeleteItem(tableName, key)
	}

	return s.commitStateLocked(next)
}

func (s *Service) TransactGetItems(request TransactGetItemsRequest) (TransactGetItemsResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := TransactGetItemsResult{}
	if len(request.Items) == 0 {
		return result, fmt.Errorf("dynamodb: transaction items are required")
	}
	if len(request.Items) > transactItemLimit {
		return result, fmt.Errorf("dynamodb: transaction supports up to %d items in the local subset", transactItemLimit)
	}

	for _, item := range request.Items {
		tableName := strings.TrimSpace(item.Table)
		if tableName == "" {
			return TransactGetItemsResult{}, fmt.Errorf("dynamodb: table name is required")
		}
		tableInfo, ok := s.state.Table(tableName)
		if !ok {
			return TransactGetItemsResult{}, fmt.Errorf("dynamodb: table %q not found", tableName)
		}

		key, err := itemDocumentKey(tableInfo, item.Key)
		if err != nil {
			return TransactGetItemsResult{}, err
		}

		if found, ok := s.state.Item(tableName, key); ok {
			copy := found
			result.Items = append(result.Items, TransactGetItemResult{Item: &copy})
			continue
		}
		result.Items = append(result.Items, TransactGetItemResult{})
	}

	return result, nil
}

func cloneBatchWriteTableRequest(request BatchWriteTableRequest) BatchWriteTableRequest {
	return BatchWriteTableRequest{
		Table:    request.Table,
		Requests: cloneBatchWriteRequests(request.Requests),
	}
}

func cloneBatchWriteRequests(requests []BatchWriteRequestItem) []BatchWriteRequestItem {
	if len(requests) == 0 {
		return nil
	}
	cloned := make([]BatchWriteRequestItem, len(requests))
	for i, request := range requests {
		cloned[i] = BatchWriteRequestItem{
			PutItem:   cloneAttributeDocument(request.PutItem),
			DeleteKey: cloneAttributeDocument(request.DeleteKey),
		}
	}
	return cloned
}

func cloneBatchGetTableRequest(request BatchGetTableRequest) BatchGetTableRequest {
	return BatchGetTableRequest{
		Table:          request.Table,
		Keys:           cloneKeyDocuments(request.Keys),
		ConsistentRead: request.ConsistentRead,
	}
}

func cloneKeyDocuments(keys []map[string]domain.AttributeValue) []map[string]domain.AttributeValue {
	if len(keys) == 0 {
		return nil
	}
	cloned := make([]map[string]domain.AttributeValue, len(keys))
	for i, key := range keys {
		cloned[i] = cloneAttributeDocument(key)
	}
	return cloned
}

func cloneAttributeDocument(values map[string]domain.AttributeValue) map[string]domain.AttributeValue {
	if values == nil {
		return nil
	}
	copied := make(map[string]domain.AttributeValue, len(values))
	for name, value := range values {
		copied[name] = value.Clone()
	}
	return copied
}

func batchDocumentKey(table domain.Table, putItem, deleteKey map[string]domain.AttributeValue) (string, error) {
	if len(putItem) > 0 {
		return itemRecordKey(table, putItem)
	}
	if len(deleteKey) > 0 {
		return itemDocumentKey(table, deleteKey)
	}
	return "", fmt.Errorf("dynamodb: batch request item is required")
}

func transactDocumentKey(table domain.Table, item TransactWriteItem) (string, error) {
	if len(item.PutItem) > 0 {
		return itemRecordKey(table, item.PutItem)
	}
	if len(item.DeleteKey) > 0 {
		return itemDocumentKey(table, item.DeleteKey)
	}
	if len(item.UpdateKey) > 0 {
		return itemDocumentKey(table, item.UpdateKey)
	}
	if len(item.ConditionCheckKey) > 0 {
		return itemDocumentKey(table, item.ConditionCheckKey)
	}
	return "", fmt.Errorf("dynamodb: transaction item is required")
}

func itemDocumentKey(table domain.Table, values map[string]domain.AttributeValue) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("dynamodb: item is required")
	}

	if strings.TrimSpace(table.PartitionKey) == "" {
		return "", fmt.Errorf("dynamodb: table %q has no partition key", table.Name)
	}

	expectedCount := 1
	if strings.TrimSpace(table.SortKey) != "" {
		expectedCount++
	}
	if len(values) != expectedCount {
		return "", fmt.Errorf("dynamodb: unsupported key attributes %q", strings.Join(sortedAttributeKeys(values), ", "))
	}

	partitionValue, ok := values[table.PartitionKey]
	if !ok {
		return "", fmt.Errorf("dynamodb: missing key attribute %q", table.PartitionKey)
	}

	partitionKey, err := attributeValueToKeyComponent(partitionValue)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(table.SortKey) == "" {
		return partitionKey, nil
	}

	sortValue, ok := values[table.SortKey]
	if !ok {
		return "", fmt.Errorf("dynamodb: missing key attribute %q", table.SortKey)
	}

	sortKey, err := attributeValueToKeyComponent(sortValue)
	if err != nil {
		return "", err
	}

	return partitionKey + "|" + sortKey, nil
}

func itemRecordKey(table domain.Table, values map[string]domain.AttributeValue) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("dynamodb: item is required")
	}
	if strings.TrimSpace(table.PartitionKey) == "" {
		return "", fmt.Errorf("dynamodb: table %q has no partition key", table.Name)
	}

	partitionValue, ok := values[table.PartitionKey]
	if !ok {
		return "", fmt.Errorf("dynamodb: missing key attribute %q", table.PartitionKey)
	}

	partitionKey, err := attributeValueToKeyComponent(partitionValue)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(table.SortKey) == "" {
		return partitionKey, nil
	}

	sortValue, ok := values[table.SortKey]
	if !ok {
		return "", fmt.Errorf("dynamodb: missing key attribute %q", table.SortKey)
	}

	sortKey, err := attributeValueToKeyComponent(sortValue)
	if err != nil {
		return "", err
	}

	return partitionKey + "|" + sortKey, nil
}

func sortedAttributeKeys(values map[string]domain.AttributeValue) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cancellationReasonsForTransaction(index, size int, code, message string) []TransactionCanceledReason {
	reasons := make([]TransactionCanceledReason, size)
	reasons[index] = TransactionCanceledReason{
		Code:    code,
		Message: message,
	}
	return reasons
}

func attributeValueToKeyComponent(value domain.AttributeValue) (string, error) {
	switch {
	case value.S != nil:
		return *value.S, nil
	case value.N != nil:
		return *value.N, nil
	case value.BOOL != nil:
		if *value.BOOL {
			return "true", nil
		}
		return "false", nil
	case value.NULL:
		return "", fmt.Errorf("dynamodb: null attribute values are not supported")
	default:
		return "", fmt.Errorf("dynamodb: only string and number attribute values are supported in the local subset")
	}
}
