package ui

import (
	"regexp"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestModelCopiesSnapshotAndHandlesEmptyState(t *testing.T) {
	t.Helper()

	snapshot := runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{Name: "alpha", Version: "v1"},
		},
		Ports: []int{8080},
	}

	m := NewModel(snapshot)
	snapshot.Services[0].Name = "mutated"
	snapshot.Ports[0] = 9090

	view := stripANSI(m.View())
	if !containsAll(view, []string{"MildStack UI", "alpha v1", "8080"}) {
		t.Fatalf("unexpected copied view:\n%s", view)
	}

	empty := NewModel(runtime.Snapshot{})
	emptyView := stripANSI(empty.View())
	if !containsAll(emptyView, []string{"(none)", "Services", "Ports"}) {
		t.Fatalf("unexpected empty view:\n%s", emptyView)
	}
}

func TestModelNavigatesServicesPortsAndBack(t *testing.T) {
	t.Helper()

	m := NewModel(runtime.Snapshot{
		Services: []orchestrator.Metadata{
			{Name: "alpha", Version: "v1"},
			{Name: "beta", Version: "v2"},
		},
		Ports: []int{8080, 9090},
	})

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := stripANSI(m.View()); !containsAll(got, []string{"> beta v2", "  8080"}) {
		t.Fatalf("expected service selection to move:\n%s", got)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := stripANSI(m.View()); !containsAll(got, []string{"Port: 9090", "Inspection: active runtime port"}) {
		t.Fatalf("expected port detail view:\n%s", got)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if got := stripANSI(m.View()); containsAll(got, []string{"Port: 9090", "Inspection: active runtime port"}) {
		t.Fatalf("expected esc to close detail view:\n%s", got)
	}

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.quitting {
		t.Fatalf("expected esc without detail to request quit")
	}

	m = NewModel(runtime.Snapshot{})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !m.quitting {
		t.Fatalf("expected q to request quit")
	}
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()

	next, _ := m.Update(msg)
	nextModel, ok := next.(model)
	if !ok {
		t.Fatalf("unexpected model type %T", next)
	}
	return nextModel
}

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !regexp.MustCompile(regexp.QuoteMeta(needle)).MatchString(haystack) {
			return false
		}
	}
	return true
}
