package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
)

// ConfirmResultMsg is dispatched when a confirm dialog is answered.
type ConfirmResultMsg struct {
	ActionTag string
	Confirmed bool
}

// ConfirmDialog represents a modal confirmation popup.
type ConfirmDialog struct {
	Title     string
	Message   string
	ActionTag string
	Active    bool
	Width     int
	Height    int
}

// NewConfirmDialog creates an inactive ConfirmDialog.
func NewConfirmDialog() ConfirmDialog {
	return ConfirmDialog{Width: 50}
}

// Show activates the dialog with given parameters.
func (c *ConfirmDialog) Show(title, message, actionTag string) {
	c.Title = title
	c.Message = message
	c.ActionTag = actionTag
	c.Active = true
}

// Hide dismisses the dialog.
func (c *ConfirmDialog) Hide() {
	c.Active = false
	c.Title = ""
	c.Message = ""
	c.ActionTag = ""
}

// Update processes key events when active.
func (c *ConfirmDialog) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !c.Active {
		return nil, false
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(keyMsg.String()) {
		case "y", "enter":
			tag := c.ActionTag
			c.Hide()
			return func() tea.Msg {
				return ConfirmResultMsg{ActionTag: tag, Confirmed: true}
			}, true
		case "n", "esc", "q":
			tag := c.ActionTag
			c.Hide()
			return func() tea.Msg {
				return ConfirmResultMsg{ActionTag: tag, Confirmed: false}
			}, true
		}
		// Absorb any other keys while modal is active
		return nil, true
	}

	return nil, false
}

// View renders the confirmation dialog centered or styled.
func (c ConfirmDialog) View() string {
	if !c.Active {
		return ""
	}

	dialogWidth := 50
	if c.Width > 0 && c.Width < dialogWidth {
		dialogWidth = c.Width - 4
	}

	title := theme.DialogTitleStyle.Render(c.Title)
	msg := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Width(dialogWidth - 4).
		Render(c.Message)

	actions := lipgloss.NewStyle().MarginTop(1).Render(
		theme.HelpKeyStyle.Render("[y]") + " " + theme.HelpDescStyle.Render("Confirm") +
			"    " +
			theme.HelpKeyStyle.Render("[n/Esc]") + " " + theme.HelpDescStyle.Render("Cancel"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, title, msg, actions)
	return theme.DialogStyle.Width(dialogWidth).Render(content)
}
