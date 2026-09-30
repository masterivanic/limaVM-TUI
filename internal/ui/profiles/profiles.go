// Package profiles implements the TUI screen that lists the Lima harness
// profiles known to limavm-tui and runs their declared actions.
//
// Actions are external processes belonging to the harness, so they are run
// through the same terminal handoff used for the interactive shell: the TUI
// renderer is suspended, the process gets the real terminal (its output is
// therefore streamed live), and the TUI resumes when it exits.
package profiles

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epel/limavm-tui/internal/profile"
	"github.com/epel/limavm-tui/internal/store"
	"github.com/epel/limavm-tui/internal/theme"
	"github.com/epel/limavm-tui/internal/ui/common"
	"github.com/epel/limavm-tui/internal/ui/components"
)

// lastProfileKey is the store key holding the most recently used profile path.
const lastProfileKey = "last_profile"

// focusPane distinguishes the two columns of the screen.
type focusPane int

const (
	focusProfiles focusPane = iota
	focusModes
)

// Model is the Profiles screen.
type Model struct {
	store    *store.Store
	results  []profile.LoadResult
	profiles []*profile.Profile

	cursor     int
	modeCursor int
	focus      focusPane

	// queue holds the steps remaining in the running sequence, so that each
	// completion can trigger the next one.
	queue      []profile.Command
	queueLabel string
	queueTotal int
	queueOwner *profile.Profile

	width  int
	height int
}

// New creates a Profiles screen from already-resolved discovery results.
func New(st *store.Store, results []profile.LoadResult) Model {
	m := Model{
		store:   st,
		results: results,
	}
	m.profiles = profile.Valid(results)
	m.restoreLastUsed()
	return m
}

// restoreLastUsed places the cursor on the profile used last, if still present.
func (m *Model) restoreLastUsed() {
	if m.store == nil {
		return
	}
	last := m.store.GetSetting(lastProfileKey)
	if last == "" {
		return
	}
	for i, p := range m.profiles {
		if p.Path() == last {
			m.cursor = i
			return
		}
	}
}

// SetSize updates layout dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Count returns the number of successfully loaded profiles.
func (m Model) Count() int { return len(m.profiles) }

// Selected returns the highlighted profile, or nil when none is available.
func (m Model) Selected() *profile.Profile {
	if m.cursor >= 0 && m.cursor < len(m.profiles) {
		return m.profiles[m.cursor]
	}
	return nil
}

// Update handles input and step-sequencing messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.ProfileStepFinishedMsg:
		return m.advanceQueue(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return common.NavigateMsg{Target: common.ScreenDashboard}
			}

		case "tab":
			if m.focus == focusProfiles {
				m.focus = focusModes
			} else {
				m.focus = focusProfiles
			}
			return m, nil

		case "up", "k":
			if m.focus == focusModes {
				if m.modeCursor > 0 {
					m.modeCursor--
				}
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
				m.modeCursor = 0
			}
			return m, nil

		case "down", "j":
			if m.focus == focusModes {
				if sel := m.Selected(); sel != nil && m.modeCursor < len(sel.ProvisioningModes)-1 {
					m.modeCursor++
				}
				return m, nil
			}
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
				m.modeCursor = 0
			}
			return m, nil

		case "u":
			return m, m.runCreate()

		case "h":
			return m, m.runHealthCheck()

		case "p", "enter":
			return m, m.runSelectedMode()
		}
	}

	return m, nil
}

// rememberSelection persists the profile being acted upon.
func (m Model) rememberSelection(p *profile.Profile) {
	if m.store == nil || p == nil {
		return
	}
	_ = m.store.SetSetting(lastProfileKey, p.Path())
}

// launch builds the terminal-handoff command for a single profile command.
//
// The working directory is the profile root: harness scripts routinely invoke
// sibling scripts and resolve their own assets through relative paths, so
// running them from anywhere else would break them.
func launch(p *profile.Profile, c profile.Command, onDone func(error) tea.Msg) tea.Cmd {
	script, err := p.ResolveScript(c.Script)
	if err != nil {
		return func() tea.Msg {
			return common.ShowToastMsg{
				Message:  fmt.Sprintf("Profile %q: %v", p.Name, err),
				Level:    components.ToastError,
				Duration: 6 * time.Second,
			}
		}
	}

	cmd := exec.Command(script, p.ExpandArgs(c)...)
	cmd.Dir = p.Root()

	return func() tea.Msg {
		return common.LaunchExecMsg{
			Cmd:      cmd,
			Callback: func(err error) tea.Msg { return onDone(err) },
		}
	}
}

// runCreate brings the VM up using the profile's create hook.
func (m Model) runCreate() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	if sel.Lifecycle.Create == nil {
		return toast(fmt.Sprintf("Profile %q declares no lifecycle.create hook", sel.Name), components.ToastWarning)
	}
	m.rememberSelection(sel)

	name := sel.Name
	return tea.Sequence(
		notify(fmt.Sprintf("Starting VM for profile %q...", name)),
		launch(sel, *sel.Lifecycle.Create, func(err error) tea.Msg {
			if err != nil {
				return common.ShowToastMsg{
					Message:  fmt.Sprintf("Profile %q: VM start failed: %v", name, err),
					Level:    components.ToastError,
					Duration: 8 * time.Second,
				}
			}
			return common.ShowToastMsg{
				Message:  fmt.Sprintf("Profile %q: VM is up", name),
				Level:    components.ToastSuccess,
				Duration: 4 * time.Second,
			}
		}),
	)
}

// runHealthCheck runs the profile's health check hook.
func (m Model) runHealthCheck() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	if sel.Lifecycle.HealthCheck == nil {
		return toast(fmt.Sprintf("Profile %q declares no lifecycle.healthCheck hook", sel.Name), components.ToastWarning)
	}
	m.rememberSelection(sel)

	name := sel.Name
	return launch(sel, *sel.Lifecycle.HealthCheck, func(err error) tea.Msg {
		if err != nil {
			return common.ShowToastMsg{
				Message:  fmt.Sprintf("Profile %q: health check failed: %v", name, err),
				Level:    components.ToastError,
				Duration: 8 * time.Second,
			}
		}
		return common.ShowToastMsg{
			Message:  fmt.Sprintf("Profile %q: health check passed", name),
			Level:    components.ToastSuccess,
			Duration: 4 * time.Second,
		}
	})
}

// runSelectedMode starts the highlighted provisioning mode.
func (m *Model) runSelectedMode() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	if len(sel.ProvisioningModes) == 0 {
		return toast(fmt.Sprintf("Profile %q declares no provisioning modes", sel.Name), components.ToastWarning)
	}
	if m.modeCursor >= len(sel.ProvisioningModes) {
		m.modeCursor = 0
	}

	mode := sel.ProvisioningModes[m.modeCursor]
	m.rememberSelection(sel)

	// Seed the queue with every step, then fire the first one. Each completion
	// pops the next, so steps never overlap.
	m.queue = append([]profile.Command(nil), mode.Steps...)
	m.queueLabel = mode.Name
	m.queueTotal = len(mode.Steps)
	m.queueOwner = sel

	return tea.Sequence(
		notify(fmt.Sprintf("Provisioning %q: running mode %q (%d steps)...", sel.Name, mode.Name, len(mode.Steps))),
		m.nextStep(),
	)
}

// nextStep dequeues and launches the next pending step.
func (m *Model) nextStep() tea.Cmd {
	if len(m.queue) == 0 || m.queueOwner == nil {
		return nil
	}

	step := m.queue[0]
	m.queue = m.queue[1:]

	index := m.queueTotal - len(m.queue) - 1
	label := m.queueLabel
	total := m.queueTotal

	return launch(m.queueOwner, step, func(err error) tea.Msg {
		return common.ProfileStepFinishedMsg{
			Label:      label,
			StepIndex:  index,
			TotalSteps: total,
			Err:        err,
		}
	})
}

// advanceQueue reacts to a finished step: abort on failure, otherwise continue
// with the next step or report overall success.
func (m Model) advanceQueue(msg common.ProfileStepFinishedMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.queue = nil
		m.queueOwner = nil
		return m, toast(
			fmt.Sprintf("Mode %q: step %d/%d failed, sequence aborted: %v",
				msg.Label, msg.StepIndex+1, msg.TotalSteps, msg.Err),
			components.ToastError,
		)
	}

	if len(m.queue) > 0 {
		return m, tea.Sequence(
			notify(fmt.Sprintf("Mode %q: step %d/%d done", msg.Label, msg.StepIndex+1, msg.TotalSteps)),
			m.nextStep(),
		)
	}

	m.queueOwner = nil
	return m, toast(
		fmt.Sprintf("Mode %q completed (%d steps)", msg.Label, msg.TotalSteps),
		components.ToastSuccess,
	)
}

// toast emits a notification message.
func toast(text string, level components.ToastLevel) tea.Cmd {
	return func() tea.Msg {
		return common.ShowToastMsg{Message: text, Level: level, Duration: 5 * time.Second}
	}
}

// notify emits a short informational message.
func notify(text string) tea.Cmd {
	return func() tea.Msg {
		return common.ShowToastMsg{Message: text, Level: components.ToastInfo, Duration: 3 * time.Second}
	}
}

// KeyHelp returns contextual keybindings for this screen.
func (m Model) KeyHelp() []components.KeyHelp {
	return []components.KeyHelp{
		{Key: "↑/↓", Desc: "Select"},
		{Key: "Tab", Desc: "Profiles/Modes"},
		{Key: "u", Desc: "Start VM"},
		{Key: "h", Desc: "Health check"},
		{Key: "p", Desc: "Run mode"},
		{Key: "Esc", Desc: "Back"},
		{Key: "?", Desc: "Help"},
	}
}

// View renders the profile list, the details pane and any load failures.
func (m Model) View() string {
	title := theme.DialogTitleStyle.Render("Lima VM Profiles")

	if len(m.profiles) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title, m.renderEmpty())
	}

	listWidth := 32
	detailWidth := m.width - listWidth - 6
	if detailWidth < 30 {
		detailWidth = 30
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderList(listWidth),
		m.renderDetail(detailWidth),
	)

	parts := []string{title, body}
	if failures := profile.Failed(m.results); len(failures) > 0 {
		parts = append(parts, m.renderFailures(failures))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderEmpty explains how to make profiles visible.
func (m Model) renderEmpty() string {
	lines := []string{
		lipgloss.NewStyle().Foreground(theme.ColorTextDim).Render(
			"No profile loaded.\n\n" +
				"Pass one explicitly:\n" +
				"  limavm-tui --profile /path/to/" + profile.DefaultProfileFileName + "\n\n" +
				"or list search paths in ~/.config/limavm-tui/config.yaml:\n" +
				"  profilePaths:\n" +
				"    - ~/sources/pro/my-harness",
		),
	}
	if failures := profile.Failed(m.results); len(failures) > 0 {
		lines = append(lines, m.renderFailures(failures))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(lines, "\n\n"))
}

// renderList renders the left column.
func (m Model) renderList(width int) string {
	var rows []string
	header := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Profiles")
	rows = append(rows, header, "")

	for i, p := range m.profiles {
		line := p.Name
		if i == m.cursor {
			marker := "▶ "
			if m.focus == focusProfiles {
				rows = append(rows, theme.TableRowSelectedStyle.Render(marker+line))
			} else {
				rows = append(rows, lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(marker+line))
			}
			continue
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.ColorText).Render("  "+line))
	}

	style := theme.PanelStyle
	if m.focus == focusProfiles {
		style = theme.PanelActiveStyle
	}
	return style.Width(width).Render(strings.Join(rows, "\n"))
}

// renderDetail renders the right column for the highlighted profile.
func (m Model) renderDetail(width int) string {
	sel := m.Selected()
	if sel == nil {
		return ""
	}

	lbl := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Width(14)
	val := lipgloss.NewStyle().Foreground(theme.ColorText)
	row := func(k, v string) string { return lbl.Render(k) + " " + val.Render(v) }

	var rows []string
	rows = append(rows,
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render(sel.Name),
	)
	if sel.Description != "" {
		rows = append(rows, val.Render(sel.Description), "")
	}

	rows = append(rows,
		row("VM name:", sel.VMName),
		row("Profile:", sel.Path()),
	)

	specs := sel.Defaults
	var spec []string
	if specs.Arch != "" {
		spec = append(spec, specs.Arch)
	}
	if specs.VMType != "" {
		spec = append(spec, specs.VMType)
	}
	if specs.CPUs > 0 {
		spec = append(spec, fmt.Sprintf("%d vCPU", specs.CPUs))
	}
	if specs.Memory != "" {
		spec = append(spec, specs.Memory+" RAM")
	}
	if specs.Disk != "" {
		spec = append(spec, specs.Disk+" disk")
	}
	if len(spec) > 0 {
		rows = append(rows, row("Defaults:", strings.Join(spec, ", ")))
	}

	if len(sel.Mounts) > 0 {
		rows = append(rows, "", lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Mounts"))
		for _, mnt := range sel.Mounts {
			rows = append(rows, val.Render(fmt.Sprintf("  %s → %s", mnt.Host, mnt.Guest)))
		}
	}

	rows = append(rows, "", lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Provisioning modes"))
	if len(sel.ProvisioningModes) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.ColorTextDim).Render("  (none declared)"))
	}
	for i, mode := range sel.ProvisioningModes {
		text := fmt.Sprintf("%s (%d steps)", mode.Name, len(mode.Steps))
		if mode.Description != "" {
			text += " — " + mode.Description
		}
		if i == m.modeCursor {
			if m.focus == focusModes {
				rows = append(rows, theme.TableRowSelectedStyle.Render("▶ "+text))
			} else {
				rows = append(rows, lipgloss.NewStyle().Foreground(theme.ColorAccent).Render("▶ "+text))
			}
			continue
		}
		rows = append(rows, val.Render("  "+text))
	}

	style := theme.PanelStyle
	if m.focus == focusModes {
		style = theme.PanelActiveStyle
	}
	return style.Width(width).Render(strings.Join(rows, "\n"))
}

// renderFailures surfaces profiles that could not be loaded, rather than
// hiding them, so a typo or a broken profile is visible.
func (m Model) renderFailures(failures []profile.LoadResult) string {
	var rows []string
	rows = append(rows, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorStatusError).Render("Profiles that failed to load"))
	for _, f := range failures {
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.ColorStatusError).Render("  • "+f.Err.Error()))
	}
	return strings.Join(rows, "\n")
}
