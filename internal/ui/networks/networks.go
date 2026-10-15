package networks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/limactl"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
	"github.com/masterivanic/limaVM-TUI/internal/ui/common"
	"github.com/masterivanic/limaVM-TUI/internal/ui/components"
)

// Model represents the dedicated Lima Networks management screen.
type Model struct {
	client   *limactl.Client
	table    table.Model
	networks []limactl.Network
	width    int
	height   int
	loading  bool
}

// New creates a new Networks Model.
func New(client *limactl.Client) Model {
	tbl := table.New(
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = theme.TableHeaderStyle
	s.Selected = theme.TableRowSelectedStyle
	s.Cell = theme.TableCellStyle
	tbl.SetStyles(s)

	m := Model{
		client:  client,
		table:   tbl,
		loading: true,
	}

	m.setupColumns()
	return m
}

func (m *Model) setupColumns() {
	cols := []table.Column{
		{Title: "NAME", Width: 18},
		{Title: "MODE", Width: 14},
		{Title: "GATEWAY", Width: 20},
		{Title: "NETMASK", Width: 18},
		{Title: "DHCP RANGE", Width: 24},
		{Title: "INTERFACE", Width: 14},
	}
	m.table.SetColumns(cols)
}

// Init loads the networks list.
func (m Model) Init() tea.Cmd {
	return m.FetchNetworksCmd()
}

// FetchNetworksCmd queries limactl network list.
func (m Model) FetchNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		nets, err := m.client.NetworkList(ctx)
		return networksLoadedMsg{Networks: nets, Err: err}
	}
}

type networksLoadedMsg struct {
	Networks []limactl.Network
	Err      error
}

// SetSize updates dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	tableHeight := h - 6
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)
	m.table.SetWidth(w - 2)
}

// SelectedNetwork returns the currently highlighted Network or nil.
func (m Model) SelectedNetwork() *limactl.Network {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.networks) {
		return &m.networks[idx]
	}
	return nil
}

// Update handles events for the Networks screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case networksLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, func() tea.Msg {
				return common.ShowToastMsg{
					Message:  "Failed to load networks: " + msg.Err.Error(),
					Level:    components.ToastError,
					Duration: 5 * time.Second,
				}
			}
		}

		m.networks = msg.Networks
		m.rebuildRows()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "1":
			return m, func() tea.Msg {
				return common.NavigateMsg{Target: common.ScreenDashboard}
			}

		case "c":
			return m, func() tea.Msg {
				return common.OpenPromptMsg{
					Title:        "Create Lima Network",
					Prompt:       "Enter network name and gateway CIDR (e.g. dev-net:192.168.42.1/24):",
					DefaultValue: "mynet:192.168.50.1/24",
					Placeholder:  "netname:gateway",
					ActionTag:    "network-create",
				}
			}

		case "d":
			if sel := m.SelectedNetwork(); sel != nil && sel.Name != "" {
				return m, func() tea.Msg {
					return common.OpenConfirmMsg{
						Title:     "Delete Network",
						Message:   fmt.Sprintf("Permanently delete Lima network %q?", sel.Name),
						ActionTag: "network-delete:" + sel.Name,
					}
				}
			}

		case "r":
			return m, m.FetchNetworksCmd()
		}
	}

	var tableCmd tea.Cmd
	m.table, tableCmd = m.table.Update(msg)
	cmds = append(cmds, tableCmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) rebuildRows() {
	rows := make([]table.Row, len(m.networks))
	for i, n := range m.networks {
		name := n.Name
		if name == "" {
			name = "(default)"
		}

		dhcp := "-"
		if n.DHCPStart != "" || n.DHCPEnd != "" {
			dhcp = fmt.Sprintf("%s - %s", n.DHCPStart, n.DHCPEnd)
		}

		iface := n.Interface
		if iface == "" {
			iface = "-"
		}

		rows[i] = table.Row{
			name,
			n.Mode,
			n.Gateway,
			n.Netmask,
			dhcp,
			iface,
		}
	}

	cur := m.table.Cursor()
	m.table.SetRows(rows)
	if cur >= len(rows) && len(rows) > 0 {
		m.table.SetCursor(len(rows) - 1)
	}
}

// HandleConfirmResult processes deletion confirmation.
func (m Model) HandleConfirmResult(actionTag string, confirmed bool) tea.Cmd {
	if !confirmed {
		return nil
	}

	if strings.HasPrefix(actionTag, "network-delete:") {
		netName := strings.TrimPrefix(actionTag, "network-delete:")
		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Deleting network %q...", netName)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
				defer cancel()

				err := m.client.NetworkDelete(ctx, netName, true)
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Network %q deleted", netName),
					Err:     err,
				}
			},
			m.FetchNetworksCmd(),
		)
	}

	return nil
}

// HandlePromptResult processes network creation input.
func (m Model) HandlePromptResult(actionTag, val string, canceled bool) tea.Cmd {
	if canceled || strings.TrimSpace(val) == "" {
		return nil
	}

	if actionTag == "network-create" {
		parts := strings.Split(val, ":")
		name := strings.TrimSpace(parts[0])
		gw := "192.168.42.1/24"
		if len(parts) > 1 {
			gw = strings.TrimSpace(parts[1])
		}

		return tea.Batch(
			func() tea.Msg {
				return common.OpStartedMsg{Message: fmt.Sprintf("Creating network %q...", name)}
			},
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
				defer cancel()

				err := m.client.NetworkCreate(ctx, name, gw, "user-v2", "")
				return common.OpFinishedMsg{
					Message: fmt.Sprintf("Network %q created", name),
					Err:     err,
				}
			},
			m.FetchNetworksCmd(),
		)
	}

	return nil
}

// KeyHelp returns contextual keybindings for the networks screen.
func (m Model) KeyHelp() []components.KeyHelp {
	return []components.KeyHelp{
		{Key: "c", Desc: "Create Network"},
		{Key: "d", Desc: "Delete Network"},
		{Key: "r", Desc: "Refresh"},
		{Key: "1 / Esc", Desc: "Back to VMs"},
	}
}

// View renders the networks table.
func (m Model) View() string {
	title := theme.DialogTitleStyle.Render("Lima Virtual Networks")
	if len(m.networks) == 0 && !m.loading {
		empty := lipgloss.NewStyle().
			Foreground(theme.ColorTextDim).
			Padding(2, 4).
			Render("No custom networks configured. Press [c] to create one.")
		return lipgloss.JoinVertical(lipgloss.Left, title, empty)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, m.table.View())
}
