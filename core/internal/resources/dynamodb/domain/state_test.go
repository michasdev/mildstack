package domain

import "testing"

func TestStateSnapshotCopiesLiveData(t *testing.T) {
	t.Helper()

	state := NewState()
	table := state.UpsertTable("mildstack-archive", "pk", "sk", "PAY_PER_REQUEST")
	state.UpsertItem(Item{
		Table: table.Name,
		Key:   "item#1",
		Attributes: map[string]string{
			"id":    "item#1",
			"title": "archive item",
		},
	})

	snapshot := state.Snapshot()

	tables := snapshot["tables"].([]any)
	tables[0].(map[string]any)["name"] = "mutated"
	items := snapshot["items"].([]any)
	items[0].(map[string]any)["attributes"].(map[string]any)["title"] = "changed"

	originalTable, ok := state.Table("mildstack-records")
	if !ok {
		t.Fatal("expected bootstrap table to remain present")
	}
	if got, want := originalTable.Name, "mildstack-records"; got != want {
		t.Fatalf("unexpected table name: got %q want %q", got, want)
	}
	originalItem, ok := state.Item(table.Name, "item#1")
	if !ok {
		t.Fatal("expected item to remain present")
	}
	if got, want := originalItem.Attributes["title"], "archive item"; got != want {
		t.Fatalf("unexpected item title: got %q want %q", got, want)
	}
}

func TestStateMutationHelpersReturnCopiesAndUpdateState(t *testing.T) {
	t.Helper()

	state := NewState()

	tables := state.ListTables()
	tables[0].Name = "mutated"
	if got, want := state.Tables[0].Name, "mildstack-records"; got != want {
		t.Fatalf("table slice aliased live state: got %q want %q", got, want)
	}

	items := state.ListItems("mildstack-records")
	items[0].Key = "mutated"
	items[0].Attributes["title"] = "changed"
	if got, want := state.Items[0].Key, "example#1"; got != want {
		t.Fatalf("item slice aliased live state: got %q want %q", got, want)
	}
	if got, want := state.Items[0].Attributes["title"], "bootstrap item"; got != want {
		t.Fatalf("item attributes aliased live state: got %q want %q", got, want)
	}

	table := state.UpsertTable("mildstack-logs", "pk", "", "PAY_PER_REQUEST")
	if got, want := table.Name, "mildstack-logs"; got != want {
		t.Fatalf("unexpected table name: got %q want %q", got, want)
	}
	if !state.HasTable("mildstack-logs") {
		t.Fatal("expected new table to be present")
	}

	item := state.UpsertItem(Item{
		Table: table.Name,
		Key:   "item#1",
		Attributes: map[string]string{
			"id":    "item#1",
			"title": "logs item",
		},
	})
	if got, want := item.Attributes["title"], "logs item"; got != want {
		t.Fatalf("unexpected item title: got %q want %q", got, want)
	}
	if !state.HasItem(table.Name, "item#1") {
		t.Fatal("expected new item to be present")
	}

	if deleted := state.DeleteItem(table.Name, "item#1"); !deleted {
		t.Fatal("expected item delete to report success")
	}
	if state.HasItem(table.Name, "item#1") {
		t.Fatal("expected deleted item to be removed")
	}
}
