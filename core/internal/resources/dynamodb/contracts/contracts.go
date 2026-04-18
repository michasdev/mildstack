package contracts

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

const (
	BatchWriteItemLimit = 25
	BatchGetItemLimit   = 100
	TransactItemLimit   = 100
)

type BatchWriteItemRequest struct {
	Tables []BatchWriteTableRequest
}

type BatchWriteTableRequest struct {
	Table    string
	Requests []BatchWriteRequestItem
}

type BatchWriteRequestItem struct {
	PutItem   map[string]domain.AttributeValue
	DeleteKey map[string]domain.AttributeValue
}

type BatchWriteItemResult struct {
	Unprocessed []BatchWriteTableRequest
}

type BatchGetItemRequest struct {
	Tables []BatchGetTableRequest
}

type BatchGetTableRequest struct {
	Table          string
	Keys           []map[string]domain.AttributeValue
	ConsistentRead *bool
}

type BatchGetTableResponse struct {
	Table string
	Items []domain.Item
}

type BatchGetItemResult struct {
	Responses   []BatchGetTableResponse
	Unprocessed []BatchGetTableRequest
}

type TransactWriteItem struct {
	Table     string
	PutItem   map[string]domain.AttributeValue
	DeleteKey map[string]domain.AttributeValue
}

type TransactWriteItemsRequest struct {
	Items []TransactWriteItem
}

type TransactGetItem struct {
	Table string
	Key   map[string]domain.AttributeValue
}

type TransactGetItemsRequest struct {
	Items []TransactGetItem
}

type TransactGetItemResult struct {
	Item *domain.Item
}

type TransactGetItemsResult struct {
	Items []TransactGetItemResult
}

type TransactionCanceledReason struct {
	Code    string
	Message string
	Item    map[string]domain.AttributeValue
}

type TransactionCanceledError struct {
	Reasons []TransactionCanceledReason
}

func (e *TransactionCanceledError) Error() string {
	if e == nil {
		return "dynamodb: transaction canceled"
	}

	reasons := make([]string, 0, len(e.Reasons))
	for _, reason := range e.Reasons {
		if reason.Code == "" && reason.Message == "" {
			continue
		}
		parts := make([]string, 0, 2)
		if reason.Code != "" {
			parts = append(parts, reason.Code)
		}
		if reason.Message != "" {
			parts = append(parts, reason.Message)
		}
		reasons = append(reasons, strings.Join(parts, ": "))
	}
	if len(reasons) == 0 {
		return "dynamodb: transaction canceled"
	}
	return fmt.Sprintf("dynamodb: transaction canceled: %s", strings.Join(reasons, "; "))
}

func (e *TransactionCanceledError) CancellationReasons() []TransactionCanceledReason {
	if e == nil {
		return nil
	}
	copied := make([]TransactionCanceledReason, len(e.Reasons))
	for i, reason := range e.Reasons {
		copied[i] = TransactionCanceledReason{
			Code:    reason.Code,
			Message: reason.Message,
			Item:    cloneAttributeDocument(reason.Item),
		}
	}
	return copied
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
