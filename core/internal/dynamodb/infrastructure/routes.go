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
			Method: "GET",
			Path:   "/dynamodb/tables/:table",
			Name:   "dynamodb.tables.show",
		},
		{
			Method: "GET",
			Path:   "/dynamodb/tables/:table/items",
			Name:   "dynamodb.items.index",
		},
		{
			Method: "GET",
			Path:   "/dynamodb/tables/:table/items/:item",
			Name:   "dynamodb.items.show",
		},
	}
}
