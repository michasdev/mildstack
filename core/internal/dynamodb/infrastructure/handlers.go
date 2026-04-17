package infrastructure

import "github.com/michasdev/mildstack/core/internal/dynamodb/domain"

type Service interface {
	ListTables() []domain.Table
	CreateTable(name, partitionKey, sortKey, billingMode string) (domain.Table, error)
	GetItem(table, key string) (domain.Item, error)
	PutItem(table, key string, attributes map[string]string) (domain.Item, error)
	DeleteItem(table, key string) error
}

type Handlers struct {
	service Service
}

type TablePayload struct {
	Name         string `json:"name"`
	PartitionKey string `json:"partition_key"`
	SortKey      string `json:"sort_key"`
	BillingMode  string `json:"billing_mode"`
}

type ItemPayload struct {
	Table      string         `json:"table"`
	Key        string         `json:"key"`
	Attributes map[string]any `json:"attributes"`
}

type ListTablesResponse struct {
	Tables []TablePayload `json:"tables"`
}

type CreateTableRequest struct {
	Name         string
	PartitionKey string
	SortKey      string
	BillingMode  string
}

type CreateTableResponse struct {
	Table TablePayload `json:"table"`
}

type GetItemRequest struct {
	Table string
	Key   string
}

type GetItemResponse struct {
	Item ItemPayload `json:"item"`
}

type PutItemRequest struct {
	Table      string
	Key        string
	Attributes map[string]string
}

type PutItemResponse struct {
	Item ItemPayload `json:"item"`
}

type DeleteItemRequest struct {
	Table string
	Key   string
}

type DeleteItemResponse struct {
	Deleted bool `json:"deleted"`
}

func NewHandlers(service Service) Handlers {
	return Handlers{service: service}
}

func (h Handlers) ListTables() ListTablesResponse {
	tables := h.service.ListTables()
	response := ListTablesResponse{
		Tables: make([]TablePayload, len(tables)),
	}
	for i, table := range tables {
		response.Tables[i] = TablePayload{
			Name:         table.Name,
			PartitionKey: table.PartitionKey,
			SortKey:      table.SortKey,
			BillingMode:  table.BillingMode,
		}
	}
	return response
}

func (h Handlers) CreateTable(request CreateTableRequest) (CreateTableResponse, error) {
	table, err := h.service.CreateTable(request.Name, request.PartitionKey, request.SortKey, request.BillingMode)
	if err != nil {
		return CreateTableResponse{}, err
	}
	return CreateTableResponse{
		Table: TablePayload{
			Name:         table.Name,
			PartitionKey: table.PartitionKey,
			SortKey:      table.SortKey,
			BillingMode:  table.BillingMode,
		},
	}, nil
}

func (h Handlers) GetItem(request GetItemRequest) (GetItemResponse, error) {
	item, err := h.service.GetItem(request.Table, request.Key)
	if err != nil {
		return GetItemResponse{}, err
	}
	return GetItemResponse{
		Item: ItemPayload{
			Table:      item.Table,
			Key:        item.Key,
			Attributes: copyAttributes(item.Attributes),
		},
	}, nil
}

func (h Handlers) PutItem(request PutItemRequest) (PutItemResponse, error) {
	item, err := h.service.PutItem(request.Table, request.Key, request.Attributes)
	if err != nil {
		return PutItemResponse{}, err
	}
	return PutItemResponse{
		Item: ItemPayload{
			Table:      item.Table,
			Key:        item.Key,
			Attributes: copyAttributes(item.Attributes),
		},
	}, nil
}

func (h Handlers) DeleteItem(request DeleteItemRequest) (DeleteItemResponse, error) {
	if err := h.service.DeleteItem(request.Table, request.Key); err != nil {
		return DeleteItemResponse{}, err
	}
	return DeleteItemResponse{Deleted: true}, nil
}

func copyAttributes(attributes map[string]string) map[string]any {
	if attributes == nil {
		return nil
	}

	copied := make(map[string]any, len(attributes))
	for key, value := range attributes {
		copied[key] = value
	}
	return copied
}
