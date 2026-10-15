package common

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/masterivanic/limaVM-TUI/internal/limactl"
	"github.com/masterivanic/limaVM-TUI/internal/ui/components"
)

// Screen identifies the currently active screen.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenDetail
	ScreenCreateVM
	ScreenNetworks
	ScreenProfiles
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

// ProfileStepFinishedMsg reports the completion of a single profile step.
//
// Steps of a provisioning mode run one after another rather than in a single
// shell invocation: each step is a separate argv-only process, so a profile
// cannot inject shell syntax, and a failing step stops the sequence.
type ProfileStepFinishedMsg struct {
	// Label identifies the sequence, e.g. a provisioning mode name.
	Label string
	// StepIndex is the zero-based index of the step that just finished.
	StepIndex int
	// TotalSteps is the number of steps in the sequence.
	TotalSteps int
	// Err is non-nil when the step failed; the sequence is then aborted.
	Err error
}
