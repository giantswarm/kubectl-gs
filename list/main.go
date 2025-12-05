package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedItemStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	normalItemStyle   = lipgloss.NewStyle()
	promptStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	matchStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Underline(true)
	selectedMatchStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("211")).Underline(true)
)

type item struct {
	title string
}

type model struct {
	items         []item
	filtered      []item
	cursor        int
	filter        textinput.Model
	width         int
	height        int
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) filterItems() {
	if m.filter.Value() == "" {
		m.filtered = m.items
	} else {
		m.filtered = []item{}
		query := strings.ToLower(m.filter.Value())
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item.title), query) {
				m.filtered = append(m.filtered, item)
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				fmt.Println(m.filtered[m.cursor].title)
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

// highlightMatches highlights the matching portion of the text
func highlightMatches(text, query string, isSelected bool) string {
	if query == "" {
		if isSelected {
			return selectedItemStyle.Render(text)
		}
		return normalItemStyle.Render(text)
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerText, lowerQuery)
	if idx == -1 {
		if isSelected {
			return selectedItemStyle.Render(text)
		}
		return normalItemStyle.Render(text)
	}

	// Split the text into parts: before, match, after
	before := text[:idx]
	match := text[idx : idx+len(query)]
	after := text[idx+len(query):]

	var result strings.Builder

	if isSelected {
		result.WriteString(selectedItemStyle.Render(before))
		result.WriteString(selectedMatchStyle.Render(match))
		result.WriteString(selectedItemStyle.Render(after))
	} else {
		result.WriteString(normalItemStyle.Render(before))
		result.WriteString(matchStyle.Render(match))
		result.WriteString(normalItemStyle.Render(after))
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
	query := m.filter.Value()
	for i := end - 1; i >= start; i-- {
		s.WriteString(highlightMatches(m.filtered[i].title, query, i == m.cursor))
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

	m := model{
		items:    items,
		filtered: items,
		cursor:   0,
		filter:   ti,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
