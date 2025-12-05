package deploy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/sahilm/fuzzy"
)

var (
	selectedItemStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	normalItemStyle    = lipgloss.NewStyle()
	promptStyleSelect  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	matchStyleSelect   = lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Underline(true)
	selectedMatchStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("211")).Underline(true)
)

// SelectionResult holds the result of the interactive selection
type SelectionResult struct {
	AppName  string
	Version  string
	Catalog  string
	Canceled bool
}

// catalogItem represents an item in the catalog selector
type catalogItem struct {
	name string
}

func (i catalogItem) String() string {
	return i.name
}

// catalogEntryItem represents an item in the catalog entry selector
type catalogEntryItem struct {
	appName   string
	version   string
	catalog   string
	namespace string
	date      string
	title     string // Combined string for display and fuzzy matching
}

func (i catalogEntryItem) String() string {
	return i.title
}

type itemMatch struct {
	item      interface{}
	positions []int
}

// selectorModel is the bubbletea model for the interactive selector
type selectorModel struct {
	items    []interface{}
	filtered []itemMatch
	cursor   int
	filter   textinput.Model
	width    int
	height   int
	prompt   string
	selected interface{}
	err      error
}

func newSelectorModel(items []interface{}, prompt string) selectorModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	// Initialize filtered list
	filtered := make([]itemMatch, len(items))
	for i, item := range items {
		filtered[i] = itemMatch{item: item, positions: nil}
	}

	return selectorModel{
		items:    items,
		filtered: filtered,
		cursor:   0,
		filter:   ti,
		prompt:   prompt,
	}
}

func (m selectorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *selectorModel) filterItems() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = make([]itemMatch, len(m.items))
		for i, item := range m.items {
			m.filtered[i] = itemMatch{item: item, positions: nil}
		}
	} else {
		// Use fuzzy matching library
		matches := fuzzy.FindFromNoSort(query, itemSource(m.items))
		m.filtered = make([]itemMatch, len(matches))
		for i, match := range matches {
			m.filtered[i] = itemMatch{
				item:      m.items[match.Index],
				positions: match.MatchedIndexes,
			}
		}
	}
	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 && len(m.filtered) > 0 {
		m.cursor = 0
	}
}

// itemSource implements fuzzy.Source interface
type itemSource []interface{}

func (s itemSource) String(i int) string {
	switch item := s[i].(type) {
	case catalogItem:
		return item.String()
	case catalogEntryItem:
		return item.String()
	default:
		return ""
	}
}

func (s itemSource) Len() int {
	return len(s)
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item
			}
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "down", "ctrl+j":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		default:
			// Update the filter input
			m.filter, cmd = m.filter.Update(msg)
			m.filterItems()
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// highlightMatches highlights the matching characters at given positions
func highlightMatches(text string, positions []int, isSelected bool) string {
	if len(positions) == 0 {
		if isSelected {
			return selectedItemStyle.Render(text)
		}
		return normalItemStyle.Render(text)
	}

	// Create a set of matched positions for quick lookup
	matchedPos := make(map[int]bool)
	for _, pos := range positions {
		matchedPos[pos] = true
	}

	var result strings.Builder
	textRunes := []rune(text)

	for i, char := range textRunes {
		if matchedPos[i] {
			if isSelected {
				result.WriteString(selectedMatchStyle.Render(string(char)))
			} else {
				result.WriteString(matchStyleSelect.Render(string(char)))
			}
		} else {
			if isSelected {
				result.WriteString(selectedItemStyle.Render(string(char)))
			} else {
				result.WriteString(normalItemStyle.Render(string(char)))
			}
		}
	}

	return result.String()
}

func (m selectorModel) View() string {
	var s strings.Builder

	// Calculate how many items we can show
	availableLines := m.height - 2 // Reserve 2 lines for prompt

	// Determine which items to show (centered around cursor)
	start := 0
	end := len(m.filtered)

	if end > availableLines {
		// Center the view around the cursor
		start = m.cursor - availableLines/2
		if start < 0 {
			start = 0
		}
		end = start + availableLines
		if end > len(m.filtered) {
			end = len(m.filtered)
			start = end - availableLines
			if start < 0 {
				start = 0
			}
		}
	}

	// Add empty lines at the top to push items to bottom
	itemCount := end - start
	for i := itemCount; i < availableLines; i++ {
		s.WriteString("\n")
	}

	// Render items in reverse order (bottom-up, with first item closest to prompt)
	for i := end - 1; i >= start; i-- {
		var displayText string
		switch item := m.filtered[i].item.(type) {
		case catalogItem:
			displayText = item.String()
		case catalogEntryItem:
			displayText = item.String()
		}
		s.WriteString(highlightMatches(displayText, m.filtered[i].positions, i == m.cursor))
		s.WriteString("\n")
	}

	// Render prompt at the bottom
	s.WriteString(promptStyleSelect.Render(m.prompt))
	s.WriteString(m.filter.View())

	return s.String()
}

// selectCatalog presents a catalog selector to the user and returns the selected catalog name
func (r *runner) selectCatalog(ctx context.Context) (string, error) {
	// Get all catalogs
	catalogs, err := r.listAllCatalogs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list catalogs: %w", err)
	}

	if len(catalogs.Items) == 0 {
		return "", fmt.Errorf("no catalogs found")
	}

	// Convert to catalog items
	items := make([]interface{}, len(catalogs.Items))
	for i, catalog := range catalogs.Items {
		items[i] = catalogItem{name: catalog.Name}
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select catalog > ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running catalog selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return "", fmt.Errorf("no catalog selected")
	}

	catalogItem, ok := m.selected.(catalogItem)
	if !ok {
		return "", fmt.Errorf("invalid selection")
	}

	return catalogItem.name, nil
}

// selectCatalogEntry presents a catalog entry selector to the user
func (r *runner) selectCatalogEntry(ctx context.Context, appNameFilter string, catalogFilter string) (*SelectionResult, error) {
	// Get catalog data service
	catalogDataService, err := r.getCatalogService()
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog service: %w", err)
	}

	// Build selector for listing catalog entries
	var selectors []string
	if appNameFilter != "" {
		selectors = append(selectors, fmt.Sprintf("app.kubernetes.io/name=%s", appNameFilter))
	}
	if catalogFilter != "" {
		selectors = append(selectors, fmt.Sprintf("application.giantswarm.io/catalog=%s", catalogFilter))
	}

	selector := strings.Join(selectors, ",")

	// List catalog entries
	entries, err := catalogDataService.GetEntries(ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog entries: %w", err)
	}

	if len(entries.Items) == 0 {
		return nil, fmt.Errorf("no catalog entries found")
	}

	// Sort entries by date (newest first in array, so after reverse rendering newest appears at bottom)
	sortedEntries := make([]applicationv1alpha1.AppCatalogEntry, len(entries.Items))
	copy(sortedEntries, entries.Items)
	slices.SortFunc(sortedEntries, func(a, b applicationv1alpha1.AppCatalogEntry) int {
		// Sort by date descending (newest first)
		if a.Spec.DateUpdated.Time.After(b.Spec.DateUpdated.Time) {
			return -1
		}
		if a.Spec.DateUpdated.Time.Before(b.Spec.DateUpdated.Time) {
			return 1
		}
		return 0
	})

	// Convert to catalog entry items
	items := make([]interface{}, len(sortedEntries))
	for i, entry := range sortedEntries {
		// Format: name version date [catalog]
		title := fmt.Sprintf("%-40s %-20s %s",
			entry.Spec.AppName,
			entry.Spec.Version,
			entry.Spec.DateUpdated.Format("2006-01-02 15:04:05"))

		items[i] = catalogEntryItem{
			appName:   entry.Spec.AppName,
			version:   entry.Spec.Version,
			catalog:   entry.Spec.Catalog.Name,
			namespace: entry.Namespace,
			date:      entry.Spec.DateUpdated.Format("2006-01-02 15:04:05"),
			title:     title,
		}
	}

	// Create and run thse selector
	model := newSelectorModel(items, "Select app and version ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running catalog entry selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return &SelectionResult{Canceled: true}, nil
	}

	entryItem, ok := m.selected.(catalogEntryItem)
	if !ok {
		return nil, fmt.Errorf("invalid selection")
	}

	return &SelectionResult{
		AppName:  entryItem.appName,
		Version:  entryItem.version,
		Catalog:  entryItem.catalog,
		Canceled: false,
	}, nil
}

// listAllCatalogs gets all catalogs in the cluster
func (r *runner) listAllCatalogs(ctx context.Context) (*applicationv1alpha1.CatalogList, error) {
	catalogs := &applicationv1alpha1.CatalogList{}
	err := r.ctrlClient.List(ctx, catalogs)
	return catalogs, err
}
