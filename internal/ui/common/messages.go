package common

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/epel/limavm-tui/internal/limactl"
	"github.com/epel/limavm-tui/internal/ui/components"
)

// Screen identifies the currently active screen.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenDetail
	ScreenCreateVM
	ScreenNetworks
)

// NavigateMsg requests switching the active screen.
type NavigateMsg struct {
	Target Screen
	VMName string // For navigating to details
}

// ShowToastMsg requests displaying a notification.
type ShowToastMsg struct {
	Message  string
	Level    components.ToastLevel
	Duration time.Duration
}

// OpStartedMsg indicates an asynchronous operation has started.
type OpStartedMsg struct {
	Message string
}

// OpFinishedMsg indicates an asynchronous operation has completed.
type OpFinishedMsg struct {
	Message string
	Err     error
}

// OpenConfirmMsg requests displaying a confirmation dialog.
type OpenConfirmMsg struct {
	Title     string
	Message   string
	ActionTag string
}

// OpenPromptMsg requests displaying a text input prompt modal.
type OpenPromptMsg struct {
	Title        string
	Prompt       string
	DefaultValue string
	Placeholder  string
	ActionTag    string
}

// LaunchExecMsg requests suspending the TUI to run an external interactive command.
type LaunchExecMsg struct {
	Cmd      *exec.Cmd
	Callback func(err error) tea.Msg
}

// ExecFinishedMsg indicates an external command has finished.
type ExecFinishedMsg struct {
	Err error
}

// RefreshVMsMsg requests refreshing the VM list.
type RefreshVMsMsg struct {
	Instances []limactl.Instance
	Err       error
}

// TemplatesLoadedMsg delivers discovered templates to CreateVM screen.
type TemplatesLoadedMsg struct {
	Templates []limactl.Template
	Err       error
}
