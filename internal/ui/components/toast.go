package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/epel/limavm-tui/internal/theme"
)

// ToastLevel represents the severity of a toast notification.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast represents an active toast notification.
type Toast struct {
	Message string
	Level   ToastLevel
	Expires time.Time
}

// ClearToastMsg is dispatched to clear an expired toast.
type ClearToastMsg struct {
	ID time.Time
}

// ToastModel manages toast notifications.
type ToastModel struct {
	toast   *Toast
	width   int
	visible bool
}

// NewToastModel creates a ToastModel.
func NewToastModel() ToastModel {
	return ToastModel{width: 80}
}

// SetMessage displays a new toast and schedules automatic dismissal.
func (m *ToastModel) SetMessage(msg string, level ToastLevel, duration time.Duration) tea.Cmd {
	if duration <= 0 {
		duration = 4 * time.Second
	}
	exp := time.Now().Add(duration)
	m.toast = &Toast{
		Message: msg,
		Level:   level,
		Expires: exp,
	}
	m.visible = true

	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return ClearToastMsg{ID: exp}
	})
}

// Clear hides the toast immediately.
func (m *ToastModel) Clear() {
	m.toast = nil
	m.visible = false
}

// SetWidth sets the available render width.
func (m *ToastModel) SetWidth(w int) {
	m.width = w
}

// IsVisible returns whether a toast is currently active.
func (m ToastModel) IsVisible() bool {
	return m.visible && m.toast != nil
}

// Update handles ClearToastMsg.
func (m *ToastModel) Update(msg tea.Msg) {
	if clearMsg, ok := msg.(ClearToastMsg); ok {
		if m.toast != nil && !m.toast.Expires.After(clearMsg.ID) {
			m.Clear()
		}
	}
}

// View renders the current toast if active.
func (m ToastModel) View() string {
	if !m.IsVisible() {
		return ""
	}

	text := m.toast.Message
	switch m.toast.Level {
	case ToastSuccess:
		return theme.ToastSuccessStyle.Width(m.width).Render("✓ " + text)
	case ToastError:
		return theme.ToastErrorStyle.Width(m.width).Render("✗ " + text + "  (press Esc to dismiss)")
	case ToastWarning:
		return theme.ToastWarningStyle.Width(m.width).Render("! " + text)
	default:
		return theme.ToastInfoStyle.Width(m.width).Render("ℹ " + text)
	}
}
