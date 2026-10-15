package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/limactl"
	"github.com/masterivanic/limaVM-TUI/internal/store"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
	"github.com/masterivanic/limaVM-TUI/internal/ui/common"
	"github.com/masterivanic/limaVM-TUI/internal/ui/components"
)

type tickPollMsg time.Time

// Model represents the VM Dashboard screen.
type Model struct {
	client       *limactl.Client
	store        *store.Store
	table        table.Model
	instances    []limactl.Instance
	filtered     []limactl.Instance
	filterInput  textinput.Model
	filtering    bool
	filterQuery  string
	width        int
	height       int
	pollInterval time.Duration
	loading      bool
}

// New creates a new Dashboard Model.
func New(client *limactl.Client, st *store.Store, pollInterval time.Duration) Model {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter VMs (press Esc to clear)..."
	ti.CharLimit = 32
	ti.Width = 40

	tbl := table.New(
		table.WithFocused(true),
	)

	// Apply dark styling to table
	s := table.DefaultStyles()
	s.Header = theme.TableHeaderStyle
	s.Selected = theme.TableRowSelectedStyle
	s.Cell = theme.TableCellStyle
	tbl.SetStyles(s)

	m := Model{
		client:       client,
		store:        st,
		table:        tbl,
		filterInput:  ti,
		pollInterval: pollInterval,
		loading:      true,
	}

	m.setupColumns(80)
	return m
}

func (m *Model) setupColumns(width int) {
	if width <= 0 {
		width = 80
	}

	cols := []table.Column{
		{Title: "★", Width: 3},
		{Title: "NAME", Width: 20},
		{Title: "STATUS", Width: 12},
		{Title: "ARCH", Width: 10},
		{Title: "CPUS", Width: 6},
		{Title: "MEMORY", Width: 12},
		{Title: "DISK", Width: 12},
		{Title: "SSH / IP", Width: 18},
	}

	m.table.SetColumns(cols)
}

// Init triggers initial VM fetch and starts polling ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.FetchVMsCmd(),
		m.tickCmd(),
	)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.pollInterval, func(t time.Time) tea.Msg {
		return tickPollMsg(t)
	})
}

// FetchVMsCmd queries limactl for instance list asynchronously.
func (m Model) FetchVMsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		vms, err := m.client.List(ctx)
		return common.RefreshVMsMsg{
			Instances: vms,
			Err:       err,
		}
	}
}

// SetSize updates screen dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.setupColumns(w)

	// Leave room for header, filter line, and footer
	tableHeight := h - 7
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)
	m.table.SetWidth(w - 2)
}

// SelectedInstance returns the currently selected Instance or nil.
func (m Model) SelectedInstance() *limactl.Instance {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.filtered) {
		return &m.filtered[idx]
	}
	return nil
}

// Update handles messages for the dashboard.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickPollMsg:
		cmds = append(cmds, m.FetchVMsCmd(), m.tickCmd())
		return m, tea.Batch(cmds...)

	case common.RefreshVMsMsg:
		m.loading = false
		if msg.Err != nil {
			cmds = append(cmds, func() tea.Msg {
				return common.ShowToastMsg{
					Message:  "Failed to refresh VMs: " + msg.Err.Error(),
					Level:    components.ToastError,
					Duration: 5 * time.Second,
				}
			})
			return m, tea.Batch(cmds...)
		}

		m.instances = msg.Instances
		m.applyFilter()
		return m, nil

	case tea.KeyMsg:
		// If filter input is active
		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filtering = false
				m.filterInput.Blur()
				return m, nil
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filterQuery = ""
				m.applyFilter()
				return m, nil
			}

			var tiCmd tea.Cmd
			m.filterInput, tiCmd = m.filterInput.Update(msg)
			m.filterQuery = m.filterInput.Value()
			m.applyFilter()
			return m, tiCmd
		}

		// Dashboard table navigation and hotkeys
		switch msg.String() {
		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink

		case "enter":
			if sel := m.SelectedInstance(); sel != nil {
				return m, func() tea.Msg {
					return common.NavigateMsg{
						Target: common.ScreenDetail,
						VMName: sel.Name,
					}
				}
			}

		case "s":
			if sel := m.SelectedInstance(); sel != nil {
				return m, m.launchShellCmd(sel.Name)
			}

		case "x":
			if sel := m.SelectedInstance(); sel != nil {
				return m, m.toggleStartStopCmd(sel)
			}

		case "d":
			if sel := m.SelectedInstance(); sel != nil {
				return m, func() tea.Msg {
					return common.OpenConfirmMsg{
						Title:     "Delete Virtual Machine",
						Message:   fmt.Sprintf("Are you sure you want to delete instance %q?\nThis will destroy the VM disk and cannot be undone.", sel.Name),
						ActionTag: "delete-vm:" + sel.Name,
					}
				}
			}

		case "c":
			return m, func() tea.Msg {
				return common.NavigateMsg{
					Target: common.ScreenCreateVM,
				}
			}

		case "r":
			if sel := m.SelectedInstance(); sel != nil {
				return m, func() tea.Msg {
					return common.OpenPromptMsg{
						Title:        "Rename Virtual Machine",
						Prompt:       fmt.Sprintf("Enter new name for %q:\n(Note: Emulated via clone to new name, then delete original)", sel.Name),
						DefaultValue: sel.Name + "-renamed",
						Placeholder:  "new-instance-name",
						ActionTag:    "rename-vm:" + sel.Name,
					}
				}
			}

		case "C":
			if sel := m.SelectedInstance(); sel != nil {
				return m, func() tea.Msg {
					return common.OpenPromptMsg{
						Title:        "Clone Virtual Machine",
						Prompt:       fmt.Sprintf("Enter clone name for %q:", sel.Name),
						DefaultValue: sel.Name + "-clone",
						Placeholder:  "cloned-instance-name",
						ActionTag:    "clone-vm:" + sel.Name,
					}
				}
			}

		case "p":
			if sel := m.SelectedInstance(); sel != nil {
				return m, m.toggleProtectCmd(sel)
			}

		case "f":
			if sel := m.SelectedInstance(); sel != nil {
				isFav, err := m.store.ToggleFavorite(sel.Name)
				if err == nil {
					msg := "Added to favorites"
					if !isFav {
						msg = "Removed from favorites"
					}
					m.rebuildRows()
					return m, func() tea.Msg {
						return common.ShowToastMsg{
							Message:  fmt.Sprintf("%s: %s", sel.Name, msg),
							Level:    components.ToastInfo,
							Duration: 2 * time.Second,
						}
					}
				}
			}
		}
	}

	var tableCmd tea.Cmd
	m.table, tableCmd = m.table.Update(msg)
	cmds = append(cmds, tableCmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) applyFilter() {
	query := strings.TrimSpace(strings.ToLower(m.filterQuery))
	if query == "" {
		m.filtered = m.instances
	} else {
		var matched []limactl.Instance
		for _, inst := range m.instances {
			if strings.Contains(strings.ToLower(inst.Name), query) ||
				strings.Contains(strings.ToLower(inst.Status), query) ||
				strings.Contains(strings.ToLower(inst.Arch), query) {
				matched = append(matched, inst)
			}
		}
		m.filtered = matched
	}

	m.rebuildRows()
}

func (m *Model) rebuildRows() {
	rows := make([]table.Row, len(m.filtered))
	for i, inst := range m.filtered {
		fav := " "
		if m.store.IsFavorite(inst.Name) {
			fav = "★"
		}

		sshTarget := "-"
		if inst.SSHAddress != "" && inst.SSHLocalPort > 0 {
			sshTarget = fmt.Sprintf("%s:%d", inst.SSHAddress, inst.SSHLocalPort)
		} else if inst.SSHAddress != "" {
			sshTarget = inst.SSHAddress
		}

		statusBadge := inst.Status
		if inst.Protected {
			statusBadge += " 🔒"
		}

		rows[i] = table.Row{
			fav,
			inst.Name,
			statusBadge,
			inst.Arch,
			fmt.Sprintf("%d", inst.CPUs),
			limactl.HumanBytes(inst.Memory),
			limactl.HumanBytes(inst.Disk),
			sshTarget,
		}
	}

	cur := m.table.Cursor()
	m.table.SetRows(rows)
	if cur >= len(rows) && len(rows) > 0 {
		m.table.SetCursor(len(rows) - 1)
	}
}

func (m Model) launchShellCmd(vmName string) tea.Cmd {
	return func() tea.Msg {
		cmd := m.client.CommandContext(context.Background(), "shell", vmName)
		return common.LaunchExecMsg{
			Cmd: cmd,
			Callback: func(err error) tea.Msg {
				return common.ExecFinishedMsg{Err: err}
			},
		}
	}
}

func (m Model) toggleStartStopCmd(inst *limactl.Instance) tea.Cmd {
	name := inst.Name
	isRunning := strings.EqualFold(inst.Status, "Running")

	return tea.Batch(
		func() tea.Msg {
			action := "Starting"
			if isRunning {
				action = "Stopping"
			}
			return common.OpStartedMsg{Message: fmt.Sprintf("%s instance %q...", action, name)}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
			defer cancel()

			var err error
			if isRunning {
				err = m.client.Stop(ctx, name, false)
			} else {
				err = m.client.Start(ctx, name)
			}

			action := "started"
			if isRunning {
				action = "stopped"
			}

			msg := fmt.Sprintf("Instance %q %s successfully", name, action)
			return common.OpFinishedMsg{
				Message: msg,
				Err:     err,
			}
		},
	)
}

func (m Model) toggleProtectCmd(inst *limactl.Instance) tea.Cmd {
	name := inst.Name
	isProt := inst.Protected

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		var err error
		var action string
		if isProt {
			err = m.client.Unprotect(ctx, name)
			action = "unprotected"
		} else {
			err = m.client.Protect(ctx, name)
			action = "protected"
		}

		if err != nil {
			return common.ShowToastMsg{
				Message:  fmt.Sprintf("Failed to change protection for %q: %v", name, err),
				Level:    components.ToastError,
				Duration: 5 * time.Second,
			}
		}

		return common.OpFinishedMsg{
			Message: fmt.Sprintf("Instance %q %s", name, action),
			Err:     nil,
		}
	}
}

// HandleConfirmResult handles confirmation callbacks for VM deletion.
func (m Model) HandleConfirmResult(actionTag string, confirmed bool) tea.Cmd {
	if !confirmed {
		return nil
	}

	if strings.HasPrefix(actionTag, "delete-vm:") {
		name := strings.TrimPrefix(actionTag, "delete-vm:")
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Deleting instance %q...", name)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
				defer cancel()

				err := m.client.Delete(ctx, name, true)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Instance %q deleted", name),
					Err:     err,
				}
			},
		)
	}

	return nil
}

// HandlePromptResult handles user input for rename or clone.
func (m Model) HandlePromptResult(actionTag, value string, canceled bool) tea.Cmd {
	if canceled || value == "" {
		return nil
	}

	if strings.HasPrefix(actionTag, "rename-vm:") {
		oldName := strings.TrimPrefix(actionTag, "rename-vm:")
		newName := value
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Renaming %q to %q (cloning and removing original)...", oldName, newName)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
				defer cancel()

				err := m.client.Rename(ctx, oldName, newName)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Instance renamed to %q", newName),
					Err:     err,
				}
			},
		)
	}

	if strings.HasPrefix(actionTag, "clone-vm:") {
		oldName := strings.TrimPrefix(actionTag, "clone-vm:")
		newName := value
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Cloning %q to %q...", oldName, newName)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
				defer cancel()

				err := m.client.Clone(ctx, oldName, newName)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Cloned to %q", newName),
					Err:     err,
				}
			},
		)
	}

	return nil
}

// KeyHelp returns contextual keybindings for the dashboard.
func (m Model) KeyHelp() []components.KeyHelp {
	if m.filtering {
		return []components.KeyHelp{
			{Key: "Enter", Desc: "Apply filter"},
			{Key: "Esc", Desc: "Clear filter"},
		}
	}

	return []components.KeyHelp{
		{Key: "Enter", Desc: "Detail"},
		{Key: "s", Desc: "Shell"},
		{Key: "x", Desc: "Start/Stop"},
		{Key: "d", Desc: "Delete"},
		{Key: "c", Desc: "Create"},
		{Key: "r", Desc: "Rename"},
		{Key: "C", Desc: "Clone"},
		{Key: "p", Desc: "Protect"},
		{Key: "f", Desc: "Favorite"},
		{Key: "/", Desc: "Filter"},
		{Key: "?", Desc: "Help"},
		{Key: "q", Desc: "Quit"},
	}
}

// View renders the dashboard table.
func (m Model) View() string {
	var parts []string

	if m.filtering {
		filterLine := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true).
			Render("Filter: ") + m.filterInput.View()
		parts = append(parts, filterLine)
	} else if m.filterQuery != "" {
		activeFilter := lipgloss.NewStyle().
			Foreground(theme.ColorTextDim).
			Render(fmt.Sprintf("Filtered by %q (press / to edit, Esc to clear)", m.filterQuery))
		parts = append(parts, activeFilter)
	}

	if len(m.filtered) == 0 && !m.loading {
		emptyMsg := lipgloss.NewStyle().
			Foreground(theme.ColorTextDim).
			Padding(2, 4).
			Render("No Lima virtual machines found. Press [c] to create one!")
		parts = append(parts, emptyMsg)
	} else {
		parts = append(parts, m.table.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
