package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func Routes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/dynamodb/tables",
			Name:   "dynamodb.tables.index",
		},
		{
			Method: "POST",
			Path:   "/dynamodb/tables",
			Name:   "dynamodb.tables.create",
		},
		{
			Method: "GET",
			Path:   "/dynamodb/tables/:table/items/:item",
			Name:   "dynamodb.items.show",
		},
		{
			Method: "PUT",
			Path:   "/dynamodb/tables/:table/items/:item",
			Name:   "dynamodb.items.update",
		},
		{
			Method: "DELETE",
			Path:   "/dynamodb/tables/:table/items/:item",
			Name:   "dynamodb.items.delete",
		},
	}
}
