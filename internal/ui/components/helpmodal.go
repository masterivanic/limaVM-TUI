package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epel/limavm-tui/internal/theme"
)

// HelpModal displays an overlay with all application keybindings.
type HelpModal struct {
	Active bool
	Width  int
	Height int
}

// NewHelpModal creates a HelpModal.
func NewHelpModal() HelpModal {
	return HelpModal{Width: 70}
}

// Show activates the help modal.
func (h *HelpModal) Show() {
	h.Active = true
}

// Hide closes the help modal.
func (h *HelpModal) Hide() {
	h.Active = false
}

// Toggle toggles modal visibility.
func (h *HelpModal) Toggle() {
	h.Active = !h.Active
}

// Update processes key events to dismiss the help modal.
func (h *HelpModal) Update(msg tea.Msg) bool {
	if !h.Active {
		return false
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "?", "esc", "q", "enter":
			h.Hide()
			return true
		}
		// Block all other keys while modal is open
		return true
	}
	return false
}

// View renders the help overlay.
func (h HelpModal) View() string {
	if !h.Active {
		return ""
	}

	dialogWidth := 68
	if h.Width > 0 && h.Width < dialogWidth {
		dialogWidth = h.Width - 4
	}

	title := theme.DialogTitleStyle.Render("Keyboard Shortcuts & Help")

	sections := []struct {
		header string
		keys   [][2]string
	}{
		{
			header: "Global Navigation",
			keys: [][2]string{
				{"1", "Dashboard (VM List)"},
				{"2", "Networks Screen"},
				{"3", "Create VM Form"},
				{"I", "Host Info & Diagnostics"},
				{"P", "Prune Unused Resources"},
				{"?", "Toggle this Help overlay"},
				{"q / Esc", "Back / Quit"},
				{"Ctrl+C", "Force Quit"},
			},
		},
		{
			header: "Dashboard (VM List)",
			keys: [][2]string{
				{"Enter", "View VM Details"},
				{"s", "Open Interactive Shell in VM"},
				{"x", "Start / Stop VM (toggle)"},
				{"d", "Delete VM (with confirm)"},
				{"c", "Create New VM"},
				{"r", "Rename VM (clone + delete)"},
				{"C", "Clone VM"},
				{"p", "Protect / Unprotect VM"},
				{"f", "Toggle Favorite VM"},
				{"/", "Filter VM list"},
			},
		},
		{
			header: "VM Detail Screen",
			keys: [][2]string{
				{"Left / Right", "Switch tabs (Overview/Disks/Net/Snaps/YAML)"},
				{"1 - 5", "Jump directly to tab index"},
				{"e", "Edit YAML in $EDITOR"},
				{"s", "Open Shell in this VM"},
				{"t", "Launch SOCKS Tunnel"},
			},
		},
		{
			header: "Disks & Snapshots & Networks",
			keys: [][2]string{
				{"c", "Create (Disk / Snapshot / Network)"},
				{"d", "Delete (Disk / Snapshot / Network)"},
				{"r", "Resize (Disk)"},
				{"u", "Unlock (Disk)"},
				{"a", "Apply / Restore (Snapshot)"},
			},
		},
	}

	var renderedSections []string
	for _, sec := range sections {
		hdr := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorSecondary).
			MarginTop(1).
			Render(sec.header)

		var rows []string
		for _, k := range sec.keys {
			keyCol := theme.HelpKeyStyle.Width(16).Render("[" + k[0] + "]")
			descCol := theme.HelpDescStyle.Render(k[1])
			rows = append(rows, keyCol+" "+descCol)
		}
		renderedSections = append(renderedSections, hdr, strings.Join(rows, "\n"))
	}

	footer := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(theme.ColorTextDim).
		Render("Press [?] or [Esc] to close")

	all := append([]string{title}, renderedSections...)
	all = append(all, footer)
	content := strings.Join(all, "\n")

	return theme.DialogStyle.Width(dialogWidth).Render(content)
}
