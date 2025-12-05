package deploy

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectorModel_FilterItems(t *testing.T) {
	tests := []struct {
		name          string
		items         []interface{}
		filterValue   string
		expectedCount int
	}{
		{
			name: "no filter returns all items",
			items: []interface{}{
				catalogItem{name: "control-plane-catalog"},
				catalogItem{name: "giantswarm-catalog"},
				catalogItem{name: "test-catalog"},
			},
			filterValue:   "",
			expectedCount: 3,
		},
		{
			name: "filter matches partial string",
			items: []interface{}{
				catalogItem{name: "control-plane-catalog"},
				catalogItem{name: "giantswarm-catalog"},
				catalogItem{name: "test-catalog"},
			},
			filterValue:   "control",
			expectedCount: 1,
		},
		{
			name: "fuzzy matching works",
			items: []interface{}{
				catalogItem{name: "control-plane-catalog"},
				catalogItem{name: "giantswarm-catalog"},
				catalogItem{name: "test-catalog"},
			},
			filterValue:   "ctlg",
			expectedCount: 3, // Fuzzy should match all that contain these letters
		},
		{
			name: "catalog entry items",
			items: []interface{}{
				catalogEntryItem{
					appName: "my-app",
					version: "1.0.0",
					catalog: "test-catalog",
					title:   "my-app                                   1.0.0                2024-01-01 00:00:00",
				},
				catalogEntryItem{
					appName: "other-app",
					version: "2.0.0",
					catalog: "test-catalog",
					title:   "other-app                                2.0.0                2024-01-02 00:00:00",
				},
			},
			filterValue:   "my",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newSelectorModel(tt.items, "test > ")
			model.filter.SetValue(tt.filterValue)
			model.filterItems()

			if len(model.filtered) != tt.expectedCount {
				t.Errorf("expected %d filtered items, got %d", tt.expectedCount, len(model.filtered))
			}
		})
	}
}

func TestSelectorModel_Navigation(t *testing.T) {
	items := []interface{}{
		catalogItem{name: "catalog-1"},
		catalogItem{name: "catalog-2"},
		catalogItem{name: "catalog-3"},
	}

	model := newSelectorModel(items, "test > ")

	// Test cursor starts at 0 (which renders at bottom due to reverse rendering)
	if model.cursor != 0 {
		t.Errorf("expected cursor to start at 0, got %d", model.cursor)
	}

	// Test moving up (increases cursor since items render in reverse)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(selectorModel)

	if model.cursor != 1 {
		t.Errorf("expected cursor to be at 1 after moving up, got %d", model.cursor)
	}

	// Test moving down (decreases cursor)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(selectorModel)

	if model.cursor != 0 {
		t.Errorf("expected cursor to be at 0 after moving down, got %d", model.cursor)
	}

	// Test cursor doesn't go below 0
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(selectorModel)

	if model.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", model.cursor)
	}
}

func TestSelectorModel_Selection(t *testing.T) {
	items := []interface{}{
		catalogItem{name: "catalog-1"},
		catalogItem{name: "catalog-2"},
		catalogItem{name: "catalog-3"},
	}

	model := newSelectorModel(items, "test > ")

	// Move to second item (up increases cursor due to reverse rendering)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ := model.Update(msg)
	model = updatedModel.(selectorModel)

	// Select the item
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(selectorModel)

	if model.selected == nil {
		t.Error("expected an item to be selected")
	}

	selectedItem, ok := model.selected.(catalogItem)
	if !ok {
		t.Error("expected selected item to be catalogItem")
	}

	if selectedItem.name != "catalog-2" {
		t.Errorf("expected selected item to be 'catalog-2', got '%s'", selectedItem.name)
	}
}

func TestItemSource(t *testing.T) {
	items := []interface{}{
		catalogItem{name: "test-catalog"},
		catalogEntryItem{
			appName: "my-app",
			version: "1.0.0",
			title:   "my-app 1.0.0",
		},
	}

	source := itemSource(items)

	if source.Len() != 2 {
		t.Errorf("expected length 2, got %d", source.Len())
	}

	if source.String(0) != "test-catalog" {
		t.Errorf("expected 'test-catalog', got '%s'", source.String(0))
	}

	if source.String(1) != "my-app 1.0.0" {
		t.Errorf("expected 'my-app 1.0.0', got '%s'", source.String(1))
	}
}
