package deploy

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type spinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	err      error
	quitting bool
}

type operationCompleteMsg struct {
	err error
}

func initialSpinnerModel(message string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00aaff"))
	return spinnerModel{
		spinner: s,
		message: message,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case operationCompleteMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m spinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return errorStyle.Render("✗ ") + m.message + "\n"
		}
		return successStyle.Render("✓ ") + m.message + "\n"
	}

	if m.quitting {
		return mutedStyle.Render("Operation cancelled\n")
	}

	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// RunWithSpinner runs a function with a spinner display
func RunWithSpinner(message string, fn func() error) error {
	p := tea.NewProgram(initialSpinnerModel(message))

	// Run the operation in a goroutine
	go func() {
		// Add a small delay to ensure the spinner starts
		time.Sleep(100 * time.Millisecond)
		err := fn()
		p.Send(operationCompleteMsg{err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	// Get the operation error from the final model
	if m, ok := finalModel.(spinnerModel); ok {
		return m.err
	}

	return nil
}
