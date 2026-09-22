package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epel/limavm-tui/internal/limactl"
	"github.com/epel/limavm-tui/internal/theme"
)

// InfoModal displays diagnostic host information from limactl info.
type InfoModal struct {
	Active bool
	Info   *limactl.InfoDiagnostic
	Width  int
	Height int
}

// NewInfoModal creates an InfoModal.
func NewInfoModal() InfoModal {
	return InfoModal{Width: 70}
}

// Show activates the info modal with diagnostic data.
func (i *InfoModal) Show(info *limactl.InfoDiagnostic) {
	i.Info = info
	i.Active = true
}

// Hide closes the modal.
func (i *InfoModal) Hide() {
	i.Active = false
}

// Update processes key events to dismiss the modal.
func (i *InfoModal) Update(msg tea.Msg) bool {
	if !i.Active {
		return false
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "q", "enter", "i", "I":
			i.Hide()
			return true
		}
		return true
	}
	return false
}

// View renders the diagnostic info modal.
func (i InfoModal) View() string {
	if !i.Active || i.Info == nil {
		return ""
	}

	dialogWidth := 68
	if i.Width > 0 && i.Width < dialogWidth {
		dialogWidth = i.Width - 4
	}

	title := theme.DialogTitleStyle.Render("Lima Host Diagnostics & Info")

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Width(18)
	valStyle := lipgloss.NewStyle().Foreground(theme.ColorText)

	row := func(label, val string) string {
		return labelStyle.Render(label) + " " + valStyle.Render(val)
	}

	defaultTmpl := "default"
	if s, ok := i.Info.DefaultTemplate.(string); ok && s != "" {
		defaultTmpl = s
	}

	var guestAgentsList []string
	for k := range i.Info.GuestAgents {
		guestAgentsList = append(guestAgentsList, k)
	}
	agentStr := strings.Join(guestAgentsList, ", ")
	if agentStr == "" {
		agentStr = "none"
	}

	rows := []string{
		row("Lima Version:", i.Info.Version),
		row("Default Template:", defaultTmpl),
		row("Lima Home:", i.Info.LimaHome),
		row("Supported VM Types:", strings.Join(i.Info.VMTypes, ", ")),
		row("Guest Agents:", agentStr),
		row("Total Templates:", fmt.Sprintf("%d available", len(i.Info.Templates))),
	}

	note := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(theme.ColorTextDim).
		Render("Press [Esc] or [q] to close")

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"), note)
	return theme.DialogStyle.Width(dialogWidth).Render(content)
}
