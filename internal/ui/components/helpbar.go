package components

import (
	"strings"

	"github.com/masterivanic/limaVM-TUI/internal/theme"
)

// KeyHelp represents a single keybinding hint.
type KeyHelp struct {
	Key  string
	Desc string
}

// HelpBar renders a horizontal footer of keybinding hints.
type HelpBar struct {
	Width int
}

// NewHelpBar creates a HelpBar.
func NewHelpBar() HelpBar {
	return HelpBar{Width: 80}
}

// Render formats a slice of KeyHelp entries into a styled footer line.
func (h HelpBar) Render(keys []KeyHelp) string {
	var parts []string
	for _, k := range keys {
		part := theme.HelpKeyStyle.Render("["+k.Key+"]") + " " + theme.HelpDescStyle.Render(k.Desc)
		parts = append(parts, part)
	}

	content := strings.Join(parts, "  ")
	style := theme.HelpBarStyle.Width(h.Width)
	return style.Render(content)
}
