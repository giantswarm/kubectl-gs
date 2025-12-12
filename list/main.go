package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

var (
	selectedItemStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	normalItemStyle    = lipgloss.NewStyle()
	promptStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	matchStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Underline(true)
	selectedMatchStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("211")).Underline(true)
)

type item struct {
	title string
}

func (i item) String() string {
	return i.title
}

type itemMatch struct {
	item      item
	positions []int
}

type model struct {
	items    []item
	filtered []itemMatch
	cursor   int
	filter   textinput.Model
	width    int
	height   int
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) filterItems() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = make([]itemMatch, len(m.items))
		for i, item := range m.items {
			m.filtered[i] = itemMatch{item: item, positions: nil}
		}
	} else {
		// Use fuzzy matching library
		matches := fuzzy.FindFrom(query, itemSource(m.items))
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
type itemSource []item

func (s itemSource) String(i int) string {
	return s[i].title
}

func (s itemSource) Len() int {
	return len(s)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				fmt.Println(m.filtered[m.cursor].item.title)
			}
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
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
				result.WriteString(matchStyle.Render(string(char)))
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

func (m model) View() string {
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
		s.WriteString(highlightMatches(m.filtered[i].item.title, m.filtered[i].positions, i == m.cursor))
		s.WriteString("\n")
	}

	// Render prompt at the bottom
	s.WriteString(promptStyle.Render("> "))
	s.WriteString(m.filter.View())

	return s.String()
}

func main() {
	items := []item{
		{title: "Raspberry Pi's"},
		{title: "Nutella"},
		{title: "Bitter melon"},
		{title: "Nice socks"},
		{title: "Eight hours of sleep"},
		{title: "Cats"},
		{title: "Plantasia, the album"},
		{title: "Pour over coffee"},
		{title: "VR"},
		{title: "Noguchi Lamps"},
		{title: "Linux"},
		{title: "Business school"},
		{title: "Pottery"},
		{title: "Shampoo"},
		{title: "Table tennis"},
		{title: "Milk crates"},
		{title: "Afternoon tea"},
		{title: "Stickers"},
		{title: "20° Weather"},
		{title: "Warm light"},
		{title: "The vernal equinox"},
		{title: "Gaffer's tape"},
		{title: "Terrycloth"},
	}

	// Initialize text input for filtering
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

	m := model{
		items:    items,
		filtered: filtered,
		cursor:   0,
		filter:   ti,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
