package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epel/limavm-tui/internal/theme"
)

// PromptResultMsg is dispatched when an input prompt is submitted or canceled.
type PromptResultMsg struct {
	ActionTag string
	Value     string
	Canceled  bool
}

// PromptDialog represents a single-input modal dialog.
type PromptDialog struct {
	Title     string
	Prompt    string
	ActionTag string
	Input     textinput.Model
	Active    bool
	Width     int
}

// NewPromptDialog creates an inactive PromptDialog.
func NewPromptDialog() PromptDialog {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40
	return PromptDialog{
		Input: ti,
		Width: 54,
	}
}

// Show activates the prompt modal with initial parameters.
func (p *PromptDialog) Show(title, prompt, defaultValue, placeholder, actionTag string) tea.Cmd {
	p.Title = title
	p.Prompt = prompt
	p.ActionTag = actionTag
	p.Active = true
	p.Input.SetValue(defaultValue)
	p.Input.Placeholder = placeholder
	p.Input.Focus()
	return textinput.Blink
}

// Hide closes the prompt dialog.
func (p *PromptDialog) Hide() {
	p.Active = false
	p.Title = ""
	p.Prompt = ""
	p.ActionTag = ""
	p.Input.Blur()
}

// Update processes events for the prompt dialog.
func (p *PromptDialog) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !p.Active {
		return nil, false
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			val := strings.TrimSpace(p.Input.Value())
			tag := p.ActionTag
			p.Hide()
			return func() tea.Msg {
				return PromptResultMsg{ActionTag: tag, Value: val, Canceled: false}
			}, true
		case "esc":
			tag := p.ActionTag
			p.Hide()
			return func() tea.Msg {
				return PromptResultMsg{ActionTag: tag, Value: "", Canceled: true}
			}, true
		}
	}

	var cmd tea.Cmd
	p.Input, cmd = p.Input.Update(msg)
	return cmd, true
}

// View renders the prompt dialog.
func (p PromptDialog) View() string {
	if !p.Active {
		return ""
	}

	dialogWidth := 54
	if p.Width > 0 && p.Width < dialogWidth {
		dialogWidth = p.Width - 4
	}

	title := theme.DialogTitleStyle.Render(p.Title)
	prompt := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		MarginBottom(1).
		Render(p.Prompt)

	inputView := p.Input.View()

	actions := lipgloss.NewStyle().MarginTop(1).Render(
		theme.HelpKeyStyle.Render("[Enter]") + " " + theme.HelpDescStyle.Render("Submit") +
			"    " +
			theme.HelpKeyStyle.Render("[Esc]") + " " + theme.HelpDescStyle.Render("Cancel"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, title, prompt, inputView, actions)
	return theme.DialogStyle.Width(dialogWidth).Render(content)
}
