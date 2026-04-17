package domain

import "sort"

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

func (s State) ListTables() []Table {
	tables := make([]Table, len(s.Tables))
	copy(tables, s.Tables)
	sort.SliceStable(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	return tables
}

func (s State) ListItems(table string) []Item {
	items := make([]Item, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Table == table {
			items = append(items, Item{
				Table:      item.Table,
				Key:        item.Key,
				Attributes: cloneAttributes(item.Attributes),
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}

func (s State) Table(name string) (Table, bool) {
	for _, table := range s.Tables {
		if table.Name == name {
			return table, true
		}
	}
	return Table{}, false
}

func (s State) Item(table, key string) (Item, bool) {
	for _, item := range s.Items {
		if item.Table == table && item.Key == key {
			return Item{
				Table:      item.Table,
				Key:        item.Key,
				Attributes: cloneAttributes(item.Attributes),
			}, true
		}
	}
	return Item{}, false
}

func (s State) HasTable(name string) bool {
	_, ok := s.Table(name)
	return ok
}

func (s State) HasItem(table, key string) bool {
	_, ok := s.Item(table, key)
	return ok
}

func (s *State) UpsertTable(name, partitionKey, sortKey, billingMode string) Table {
	for i := range s.Tables {
		if s.Tables[i].Name == name {
			s.Tables[i].PartitionKey = partitionKey
			s.Tables[i].SortKey = sortKey
			s.Tables[i].BillingMode = billingMode
			return s.Tables[i]
		}
	}

	table := Table{
		Name:         name,
		PartitionKey: partitionKey,
		SortKey:      sortKey,
		BillingMode:  billingMode,
	}
	s.Tables = append(s.Tables, table)
	return table
}

func (s *State) UpsertItem(item Item) Item {
	cloned := Item{
		Table:      item.Table,
		Key:        item.Key,
		Attributes: cloneAttributes(item.Attributes),
	}

	for i := range s.Items {
		if s.Items[i].Table == cloned.Table && s.Items[i].Key == cloned.Key {
			s.Items[i] = cloned
			return Item{
				Table:      s.Items[i].Table,
				Key:        s.Items[i].Key,
				Attributes: cloneAttributes(s.Items[i].Attributes),
			}
		}
	}

	s.Items = append(s.Items, cloned)
	return Item{
		Table:      cloned.Table,
		Key:        cloned.Key,
		Attributes: cloneAttributes(cloned.Attributes),
	}
}

func (s *State) DeleteItem(table, key string) bool {
	for i := range s.Items {
		if s.Items[i].Table == table && s.Items[i].Key == key {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return true
		}
	}
	return false
}

func (s State) Snapshot() map[string]any {
	tables := make([]any, 0, len(s.Tables))
	for _, table := range s.ListTables() {
		tables = append(tables, map[string]any{
			"name":          table.Name,
			"partition_key": table.PartitionKey,
			"sort_key":      table.SortKey,
			"billing_mode":  table.BillingMode,
		})
	}

	items := make([]any, 0, len(s.Items))
	for _, item := range s.sortedItems() {
		items = append(items, map[string]any{
			"table":      item.Table,
			"key":        item.Key,
			"attributes": copyAttributesAny(item.Attributes),
		})
	}

	return map[string]any{
		"service": s.Service,
		"tables":  tables,
		"items":   items,
	}
}

func (s State) sortedItems() []Item {
	items := make([]Item, len(s.Items))
	for i, item := range s.Items {
		items[i] = Item{
			Table:      item.Table,
			Key:        item.Key,
			Attributes: cloneAttributes(item.Attributes),
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Table == items[j].Table {
			return items[i].Key < items[j].Key
		}
		return items[i].Table < items[j].Table
	})
	return items
}

func cloneAttributes(attributes map[string]string) map[string]string {
	if attributes == nil {
		return nil
	}

	cloned := make(map[string]string, len(attributes))
	for key, value := range attributes {
		cloned[key] = value
	}
	return cloned
}

func copyAttributesAny(attributes map[string]string) map[string]any {
	if attributes == nil {
		return nil
	}

	copied := make(map[string]any, len(attributes))
	for key, value := range attributes {
		copied[key] = value
	}
	return copied
}
