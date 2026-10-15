package detail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/limactl"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
	"github.com/masterivanic/limaVM-TUI/internal/ui/common"
	"github.com/masterivanic/limaVM-TUI/internal/ui/components"
)

type tabIndex int

const (
	tabOverview tabIndex = iota
	tabDisks
	tabNetwork
	tabSnapshots
	tabYAML
)

var tabTitles = []string{
	"1. Overview",
	"2. Disks",
	"3. Network & Tunnels",
	"4. Snapshots",
	"5. YAML",
}

// Model represents the VM Detail screen with 5 tabs.
type Model struct {
	client     *limactl.Client
	vmName     string
	instance   *limactl.Instance
	activeTab  tabIndex
	viewport   viewport.Model
	disks      []limactl.Disk
	snapshots  []limactl.Snapshot
	sshConfig  string
	yamlText   string
	width      int
	height     int
	loading    bool
	cursorDisk int
	cursorSnap int
}

// New creates a new Detail Model for a VM instance.
func New(client *limactl.Client) Model {
	vp := viewport.New(80, 20)
	return Model{
		client:    client,
		activeTab: tabOverview,
		viewport:  vp,
		loading:   true,
	}
}

// SetVM sets the target VM to inspect and loads its details.
func (m *Model) SetVM(name string) tea.Cmd {
	m.vmName = name
	m.loading = true
	m.activeTab = tabOverview
	m.cursorDisk = 0
	m.cursorSnap = 0
	return m.RefreshCmd()
}

// SetSize updates layout dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	contentHeight := h - 6
	if contentHeight < 4 {
		contentHeight = 4
	}
	m.viewport.Width = w - 4
	m.viewport.Height = contentHeight
}

// RefreshCmd fetches full instance data, snapshots, disks, SSH info, and YAML.
func (m Model) RefreshCmd() tea.Cmd {
	name := m.vmName
	if name == "" {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		inst, err := m.client.Get(ctx, name)
		if err != nil {
			return detailLoadedMsg{Err: err}
		}

		disks, _ := m.client.DiskList(ctx)
		snaps, _ := m.client.SnapshotList(ctx, name)
		sshCfg, _ := m.client.ShowSSH(ctx, name, "config")
		yamlStr, _ := m.client.GetYAML(ctx, name)

		return detailLoadedMsg{
			Instance:  inst,
			Disks:     disks,
			Snapshots: snaps,
			SSHConfig: sshCfg,
			YAML:      yamlStr,
		}
	}
}

type detailLoadedMsg struct {
	Instance  *limactl.Instance
	Disks     []limactl.Disk
	Snapshots []limactl.Snapshot
	SSHConfig string
	YAML      string
	Err       error
}

// Update handles user input and async events on the detail screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case detailLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, func() tea.Msg {
				return common.ShowToastMsg{
					Message:  "Failed to load VM details: " + msg.Err.Error(),
					Level:    components.ToastError,
					Duration: 5 * time.Second,
				}
			}
		}

		m.instance = msg.Instance
		m.disks = msg.Disks
		m.snapshots = msg.Snapshots
		m.sshConfig = msg.SSHConfig
		m.yamlText = msg.YAML
		m.updateViewportContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return common.NavigateMsg{Target: common.ScreenDashboard}
			}

		case "tab", "right", "l":
			m.activeTab = (m.activeTab + 1) % tabIndex(len(tabTitles))
			m.updateViewportContent()
			return m, nil

		case "shift+tab", "left", "h":
			if m.activeTab == 0 {
				m.activeTab = tabIndex(len(tabTitles) - 1)
			} else {
				m.activeTab--
			}
			m.updateViewportContent()
			return m, nil

		case "1":
			m.activeTab = tabOverview
			m.updateViewportContent()
			return m, nil
		case "2":
			m.activeTab = tabDisks
			m.updateViewportContent()
			return m, nil
		case "3":
			m.activeTab = tabNetwork
			m.updateViewportContent()
			return m, nil
		case "4":
			m.activeTab = tabSnapshots
			m.updateViewportContent()
			return m, nil
		case "5":
			m.activeTab = tabYAML
			m.updateViewportContent()
			return m, nil

		// Tab-specific or global detail shortcuts
		case "s":
			if m.instance != nil {
				return m, func() tea.Msg {
					cmd := m.client.CommandContext(context.Background(), "shell", m.instance.Name)
					return common.LaunchExecMsg{
						Cmd: cmd,
						Callback: func(err error) tea.Msg {
							return common.ExecFinishedMsg{Err: err}
						},
					}
				}
			}

		case "t":
			// Launch SOCKS tunnel via ExecProcess
			if m.instance != nil {
				return m, func() tea.Msg {
					cmd := m.client.CommandContext(context.Background(), "tunnel", m.instance.Name)
					return common.LaunchExecMsg{
						Cmd: cmd,
						Callback: func(err error) tea.Msg {
							return common.ExecFinishedMsg{Err: err}
						},
					}
				}
			}

		case "e":
			// Edit YAML in $EDITOR
			if m.activeTab == tabYAML && m.instance != nil {
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}
				yamlFile := filepath.Join(m.instance.Dir, "lima.yaml")
				cmd := exec.Command(editor, yamlFile)

				return m, func() tea.Msg {
					return common.LaunchExecMsg{
						Cmd: cmd,
						Callback: func(err error) tea.Msg {
							// Upon return, reload YAML and re-validate
							return detailLoadedMsg{Err: err}
						},
					}
				}
			}

		// Disks actions
		case "c":
			if m.activeTab == tabDisks {
				return m, func() tea.Msg {
					return common.OpenPromptMsg{
						Title:        "Create New Virtual Disk",
						Prompt:       "Enter disk name and size (e.g. data-disk:20GiB):",
						DefaultValue: m.vmName + "-disk:10GiB",
						Placeholder:  "disk-name:size",
						ActionTag:    "disk-create:" + m.vmName,
					}
				}
			}
			if m.activeTab == tabSnapshots {
				return m, func() tea.Msg {
					return common.OpenPromptMsg{
						Title:        "Create Snapshot",
						Prompt:       fmt.Sprintf("Enter snapshot tag for %q:", m.vmName),
						DefaultValue: fmt.Sprintf("snap-%s", time.Now().Format("20060102-150405")),
						Placeholder:  "snapshot-tag",
						ActionTag:    "snapshot-create:" + m.vmName,
					}
				}
			}

		case "d":
			if m.activeTab == tabSnapshots && len(m.snapshots) > 0 && m.cursorSnap < len(m.snapshots) {
				tag := m.snapshots[m.cursorSnap].Tag
				return m, func() tea.Msg {
					return common.OpenConfirmMsg{
						Title:     "Delete Snapshot",
						Message:   fmt.Sprintf("Delete snapshot %q for instance %q?", tag, m.vmName),
						ActionTag: fmt.Sprintf("snapshot-delete:%s:%s", m.vmName, tag),
					}
				}
			}
			if m.activeTab == tabDisks && len(m.disks) > 0 && m.cursorDisk < len(m.disks) {
				diskName := m.disks[m.cursorDisk].Name
				return m, func() tea.Msg {
					return common.OpenConfirmMsg{
						Title:     "Delete Disk",
						Message:   fmt.Sprintf("Permanently delete disk %q?", diskName),
						ActionTag: fmt.Sprintf("disk-delete:%s", diskName),
					}
				}
			}

		case "r":
			if m.activeTab == tabDisks && len(m.disks) > 0 && m.cursorDisk < len(m.disks) {
				diskName := m.disks[m.cursorDisk].Name
				return m, func() tea.Msg {
					return common.OpenPromptMsg{
						Title:        "Resize Disk",
						Prompt:       fmt.Sprintf("Enter new size for disk %q (e.g. 30GiB):", diskName),
						DefaultValue: "20GiB",
						Placeholder:  "new-size",
						ActionTag:    fmt.Sprintf("disk-resize:%s", diskName),
					}
				}
			}

		case "u":
			if m.activeTab == tabDisks && len(m.disks) > 0 && m.cursorDisk < len(m.disks) {
				diskName := m.disks[m.cursorDisk].Name
				return m, m.unlockDiskCmd(diskName)
			}

		case "a":
			if m.activeTab == tabSnapshots && len(m.snapshots) > 0 && m.cursorSnap < len(m.snapshots) {
				tag := m.snapshots[m.cursorSnap].Tag
				return m, func() tea.Msg {
					return common.OpenConfirmMsg{
						Title:     "Apply Snapshot",
						Message:   fmt.Sprintf("Restore VM %q to snapshot %q? Current memory/disk state will be replaced.", m.vmName, tag),
						ActionTag: fmt.Sprintf("snapshot-apply:%s:%s", m.vmName, tag),
					}
				}
			}

		case "j", "down":
			if m.activeTab == tabDisks && m.cursorDisk < len(m.disks)-1 {
				m.cursorDisk++
				m.updateViewportContent()
				return m, nil
			}
			if m.activeTab == tabSnapshots && m.cursorSnap < len(m.snapshots)-1 {
				m.cursorSnap++
				m.updateViewportContent()
				return m, nil
			}

		case "k", "up":
			if m.activeTab == tabDisks && m.cursorDisk > 0 {
				m.cursorDisk--
				m.updateViewportContent()
				return m, nil
			}
			if m.activeTab == tabSnapshots && m.cursorSnap > 0 {
				m.cursorSnap--
				m.updateViewportContent()
				return m, nil
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewportContent() {
	var content string
	switch m.activeTab {
	case tabOverview:
		content = m.renderOverviewTab()
	case tabDisks:
		content = m.renderDisksTab()
	case tabNetwork:
		content = m.renderNetworkTab()
	case tabSnapshots:
		content = m.renderSnapshotsTab()
	case tabYAML:
		content = m.renderYAMLTab()
	}

	m.viewport.SetContent(content)
}

func (m Model) renderOverviewTab() string {
	if m.instance == nil {
		return "Loading VM metadata..."
	}

	inst := m.instance
	lbl := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Width(18)
	val := lipgloss.NewStyle().Foreground(theme.ColorText)

	row := func(label, value string) string {
		return lbl.Render(label) + " " + val.Render(value)
	}

	lines := []string{
		row("Name:", inst.Name),
		row("Status:", theme.StatusBadge(inst.Status)),
		row("Hostname:", inst.Hostname),
		row("VM Type:", inst.VMType),
		row("Architecture:", fmt.Sprintf("%s (Host: %s)", inst.Arch, inst.HostArch)),
		row("CPUs:", fmt.Sprintf("%d cores (%s)", inst.CPUs, inst.CPUType)),
		row("Memory:", limactl.HumanBytes(inst.Memory)),
		row("Disk:", limactl.HumanBytes(inst.Disk)),
		row("Directory:", inst.Dir),
		row("SSH Config:", inst.SSHConfigFile),
		row("SSH Address:", fmt.Sprintf("%s:%d", inst.SSHAddress, inst.SSHLocalPort)),
		row("Protected:", fmt.Sprintf("%t", inst.Protected)),
		row("Lima Version:", inst.LimaVersion),
	}

	if len(inst.Errors) > 0 {
		errHdr := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorStatusError).Render("\nErrors:")
		lines = append(lines, errHdr)
		for _, e := range inst.Errors {
			lines = append(lines, "  • "+val.Render(e))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderDisksTab() string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Attached & Lima Disks:"))
	lines = append(lines, "Use [j/k] to navigate, [c] create disk, [r] resize, [u] unlock, [d] delete\n")

	if len(m.disks) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorTextDim).Render("No virtual disks found. Press [c] to create one."))
		return strings.Join(lines, "\n")
	}

	for i, d := range m.disks {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(theme.ColorText)
		if i == m.cursorDisk {
			prefix = "▶ "
			style = theme.TableRowSelectedStyle
		}

		info := fmt.Sprintf("%-20s  Size: %-10s  Format: %-6s  Instance: %-15s",
			d.Name,
			limactl.HumanBytes(d.Size),
			d.Format,
			d.Instance,
		)
		if d.Locked {
			info += " [LOCKED]"
		}

		lines = append(lines, prefix+style.Render(info))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderNetworkTab() string {
	var lines []string
	lbl := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Width(18)
	val := lipgloss.NewStyle().Foreground(theme.ColorText)

	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Network Configuration:"))

	if m.instance != nil {
		lines = append(lines, lbl.Render("SSH Endpoint:")+" "+val.Render(fmt.Sprintf("%s:%d", m.instance.SSHAddress, m.instance.SSHLocalPort)))
		lines = append(lines, lbl.Render("SSH Config:")+" "+val.Render(m.instance.SSHConfigFile))
		if len(m.instance.Networks) > 0 {
			lines = append(lines, "\nAttached VM Networks:")
			for _, n := range m.instance.Networks {
				lines = append(lines, fmt.Sprintf("  • Lima: %s (MAC: %s, Interface: %s)", n.Lima, n.MAC, n.Interface))
			}
		}
	}

	lines = append(lines, "\n"+lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("SOCKS Tunnel:"))
	lines = append(lines, val.Render("Press [t] to launch a SOCKS proxy tunnel to this VM's network."))

	if m.sshConfig != "" {
		lines = append(lines, "\n"+lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("SSH Config snippet:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorTextDim).Render(m.sshConfig))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderSnapshotsTab() string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render(fmt.Sprintf("Snapshots for %s:", m.vmName)))
	lines = append(lines, "Use [j/k] to navigate, [c] create snapshot, [a] apply, [d] delete\n")

	if len(m.snapshots) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorTextDim).Render("No snapshots taken yet. Press [c] to save a snapshot."))
		return strings.Join(lines, "\n")
	}

	for i, s := range m.snapshots {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(theme.ColorText)
		if i == m.cursorSnap {
			prefix = "▶ "
			style = theme.TableRowSelectedStyle
		}

		info := fmt.Sprintf("%-25s  Size: %s", s.Tag, s.Size)
		lines = append(lines, prefix+style.Render(info))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderYAMLTab() string {
	if m.yamlText == "" {
		return "No lima.yaml found for this instance."
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent).
		Render(fmt.Sprintf("lima.yaml for %s (Press [e] to edit in $EDITOR):\n\n", m.vmName))

	return header + m.yamlText
}

func (m Model) unlockDiskCmd(diskName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		err := m.client.DiskUnlock(ctx, diskName)
		if err != nil {
			return common.ShowToastMsg{
				Message:  fmt.Sprintf("Unlock disk %q failed: %v", diskName, err),
				Level:    components.ToastError,
				Duration: 5 * time.Second,
			}
		}
		return common.ShowToastMsg{
			Message:  fmt.Sprintf("Disk %q unlocked", diskName),
			Level:    components.ToastSuccess,
			Duration: 3 * time.Second,
		}
	}
}

// HandleConfirmResult handles confirmation callbacks for disks and snapshots.
func (m Model) HandleConfirmResult(actionTag string, confirmed bool) tea.Cmd {
	if !confirmed {
		return nil
	}

	if strings.HasPrefix(actionTag, "snapshot-delete:") {
		parts := strings.Split(actionTag, ":")
		if len(parts) == 3 {
			inst, tag := parts[1], parts[2]
			return tea.Batch(
				func() tea.Msg {
					return common.OpStartedMsg{Message: fmt.Sprintf("Deleting snapshot %q...", tag)}
				},
				func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
					defer cancel()

					err := m.client.SnapshotDelete(ctx, inst, tag)
					return common.OpFinishedMsg{
						Message: fmt.Sprintf("Snapshot %q deleted", tag),
						Err:     err,
					}
				},
			)
		}
	}

	if strings.HasPrefix(actionTag, "snapshot-apply:") {
		parts := strings.Split(actionTag, ":")
		if len(parts) == 3 {
			inst, tag := parts[1], parts[2]
			return tea.Batch(
				func() tea.Msg {
					return common.OpStartedMsg{Message: fmt.Sprintf("Restoring snapshot %q to %q...", tag, inst)}
				},
				func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
					defer cancel()

					err := m.client.SnapshotApply(ctx, inst, tag)
					return common.OpFinishedMsg{
						Message: fmt.Sprintf("Restored snapshot %q successfully", tag),
						Err:     err,
					}
				},
			)
		}
	}

	if strings.HasPrefix(actionTag, "disk-delete:") {
		diskName := strings.TrimPrefix(actionTag, "disk-delete:")
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Deleting disk %q...", diskName)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
				defer cancel()

				err := m.client.DiskDelete(ctx, diskName, true)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Disk %q deleted", diskName),
					Err:     err,
				}
			},
		)
	}

	return nil
}

// HandlePromptResult handles user inputs for creating disks, snapshots, or resizing disks.
func (m Model) HandlePromptResult(actionTag, val string, canceled bool) tea.Cmd {
	if canceled || strings.TrimSpace(val) == "" {
		return nil
	}

	if strings.HasPrefix(actionTag, "snapshot-create:") {
		inst := strings.TrimPrefix(actionTag, "snapshot-create:")
		tag := val
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Creating snapshot %q...", tag)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
				defer cancel()

				err := m.client.SnapshotCreate(ctx, inst, tag)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Snapshot %q created", tag),
					Err:     err,
				}
			},
		)
	}

	if strings.HasPrefix(actionTag, "disk-create:") {
		parts := strings.Split(val, ":")
		diskName := parts[0]
		diskSize := "10GiB"
		if len(parts) > 1 {
			diskSize = parts[1]
		}

		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Creating disk %q (%s)...", diskName, diskSize)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
				defer cancel()

				err := m.client.DiskCreate(ctx, diskName, diskSize, "qcow2")
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Disk %q created", diskName),
					Err:     err,
				}
			},
		)
	}

	if strings.HasPrefix(actionTag, "disk-resize:") {
		diskName := strings.TrimPrefix(actionTag, "disk-resize:")
		newSize := val
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Resizing disk %q to %s...", diskName, newSize)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
				defer cancel()

				err := m.client.DiskResize(ctx, diskName, newSize)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Disk %q resized to %s", diskName, newSize),
					Err:     err,
				}
			},
		)
	}

	return nil
}

// KeyHelp returns contextual keybindings for the detail screen.
func (m Model) KeyHelp() []components.KeyHelp {
	base := []components.KeyHelp{
		{Key: "Tab / 1-5", Desc: "Switch Tab"},
		{Key: "s", Desc: "Shell"},
		{Key: "Esc / q", Desc: "Back to VMs"},
	}

	switch m.activeTab {
	case tabYAML:
		base = append(base, components.KeyHelp{Key: "e", Desc: "Edit YAML in $EDITOR"})
	case tabNetwork:
		base = append(base, components.KeyHelp{Key: "t", Desc: "Launch Tunnel"})
	case tabDisks:
		base = append(base,
			components.KeyHelp{Key: "c", Desc: "New Disk"},
			components.KeyHelp{Key: "r", Desc: "Resize"},
			components.KeyHelp{Key: "u", Desc: "Unlock"},
			components.KeyHelp{Key: "d", Desc: "Delete"},
		)
	case tabSnapshots:
		base = append(base,
			components.KeyHelp{Key: "c", Desc: "New Snap"},
			components.KeyHelp{Key: "a", Desc: "Apply Snap"},
			components.KeyHelp{Key: "d", Desc: "Delete Snap"},
		)
	}

	return base
}

// View renders the detail screen tabs and active tab viewport.
func (m Model) View() string {
	var tabs []string
	for i, t := range tabTitles {
		if tabIndex(i) == m.activeTab {
			tabs = append(tabs, theme.TabActiveStyle.Render(t))
		} else {
			tabs = append(tabs, theme.TabInactiveStyle.Render(t))
		}
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent).
		Render(fmt.Sprintf("VM: %s", m.vmName))

	divider := lipgloss.NewStyle().
		Foreground(theme.ColorBorder).
		Width(m.width).
		Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabRow,
		divider,
		m.viewport.View(),
	)
}
