package domain

const StateKey = "services/dynamodb"

type State struct {
	Service string
	Tables  []Table
	Items   []Item
}

type Table struct {
	Name         string
	PartitionKey string
	SortKey      string
	BillingMode  string
}

type Item struct {
	Table      string
	Key        string
	Attributes map[string]string
}

func NewState() State {
	return State{
		Service: "dynamodb",
		Tables: []Table{
			{
				Name:         "mildstack-records",
				PartitionKey: "id",
				SortKey:      "version",
				BillingMode:  "PAY_PER_REQUEST",
			},
		},
		Items: []Item{
			{
				Table: "mildstack-records",
				Key:   "example#1",
				Attributes: map[string]string{
					"id":      "example#1",
					"version": "1",
					"title":   "bootstrap item",
				},
			},
		},
	}
}

func (s State) Snapshot() map[string]any {
	tables := make([]any, 0, len(s.Tables))
	for _, table := range s.Tables {
		tables = append(tables, map[string]any{
			"name":          table.Name,
			"partition_key": table.PartitionKey,
			"sort_key":      table.SortKey,
			"billing_mode":  table.BillingMode,
		})
	}

	items := make([]any, 0, len(s.Items))
	for _, item := range s.Items {
		attributes := make(map[string]any, len(item.Attributes))
		for key, value := range item.Attributes {
			attributes[key] = value
		}

		items = append(items, map[string]any{
			"table":      item.Table,
			"key":        item.Key,
			"attributes": attributes,
		})
	}

	return map[string]any{
		"service": s.Service,
		"tables":  tables,
		"items":   items,
	}
}
