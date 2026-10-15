package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
)

// OpSpinner represents an active asynchronous operation with spinner.
type OpSpinner struct {
	Spinner spinner.Model
	Message string
	Active  bool
}

// NewOpSpinner initializes a spinner with dark theme styling.
func NewOpSpinner() OpSpinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorAccent)
	return OpSpinner{
		Spinner: s,
	}
}

// Start activates the spinner with a descriptive operation message.
func (o *OpSpinner) Start(msg string) tea.Cmd {
	o.Message = msg
	o.Active = true
	return o.Spinner.Tick
}

// Stop deactivates the spinner.
func (o *OpSpinner) Stop() {
	o.Active = false
	o.Message = ""
}

// Update advances the spinner animation.
func (o *OpSpinner) Update(msg tea.Msg) tea.Cmd {
	if !o.Active {
		return nil
	}
	var cmd tea.Cmd
	o.Spinner, cmd = o.Spinner.Update(msg)
	return cmd
}

// View renders the spinner and current operation message.
func (o OpSpinner) View() string {
	if !o.Active {
		return ""
	}
	msg := lipgloss.NewStyle().Foreground(theme.ColorText).Render(o.Message)
	return o.Spinner.View() + " " + msg
}
