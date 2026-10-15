package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/masterivanic/limaVM-TUI/internal/limactl"
	"github.com/masterivanic/limaVM-TUI/internal/profile"
	"github.com/masterivanic/limaVM-TUI/internal/store"
	"github.com/masterivanic/limaVM-TUI/internal/theme"
	"github.com/masterivanic/limaVM-TUI/internal/ui/common"
	"github.com/masterivanic/limaVM-TUI/internal/ui/components"
	"github.com/masterivanic/limaVM-TUI/internal/ui/createvm"
	"github.com/masterivanic/limaVM-TUI/internal/ui/dashboard"
	"github.com/masterivanic/limaVM-TUI/internal/ui/detail"
	"github.com/masterivanic/limaVM-TUI/internal/ui/networks"
	"github.com/masterivanic/limaVM-TUI/internal/ui/profiles"
)

// App is the root Bubble Tea model managing all screens and overlays.
type App struct {
	client       *limactl.Client
	store        *store.Store
	activeScreen common.Screen

	// Sub-screens
	dashboardScreen dashboard.Model
	detailScreen    detail.Model
	createVMScreen  createvm.Model
	networksScreen  networks.Model
	profilesScreen  profiles.Model

	// Overlays & Components
	helpBar       components.HelpBar
	toast         components.ToastModel
	confirmDialog components.ConfirmDialog
	promptDialog  components.PromptDialog
	spinner       components.OpSpinner
	helpModal     components.HelpModal
	infoModal     components.InfoModal

	width    int
	height   int
	quitting bool
}

// NewApp instantiates the root application model.
//
// profileResults carries the outcome of profile discovery, including sources
// that failed to load so the Profiles screen can surface them.
func NewApp(client *limactl.Client, st *store.Store, pollInterval time.Duration, profileResults []profile.LoadResult) App {
	return App{
		client:          client,
		store:           st,
		activeScreen:    common.ScreenDashboard,
		dashboardScreen: dashboard.New(client, st, pollInterval),
		detailScreen:    detail.New(client),
		createVMScreen:  createvm.New(client, st),
		networksScreen:  networks.New(client),
		profilesScreen:  profiles.New(st, profileResults),
		helpBar:         components.NewHelpBar(),
		toast:           components.NewToastModel(),
		confirmDialog:   components.NewConfirmDialog(),
		promptDialog:    components.NewPromptDialog(),
		spinner:         components.NewOpSpinner(),
		helpModal:       components.NewHelpModal(),
		infoModal:       components.NewInfoModal(),
	}
}

// Init initializes child components and starts initial operations.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.dashboardScreen.Init(),
		a.createVMScreen.Init(),
		a.networksScreen.Init(),
	)
}

// Update coordinates messages, key events, and view routing.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.helpBar.Width = msg.Width
		a.toast.SetWidth(msg.Width)
		a.confirmDialog.Width = msg.Width
		a.promptDialog.Width = msg.Width
		a.helpModal.Width = msg.Width
		a.infoModal.Width = msg.Width

		a.dashboardScreen.SetSize(msg.Width, msg.Height)
		a.detailScreen.SetSize(msg.Width, msg.Height)
		a.createVMScreen.SetSize(msg.Width, msg.Height)
		a.networksScreen.SetSize(msg.Width, msg.Height)
		a.profilesScreen.SetSize(msg.Width, msg.Height)
		return a, nil

	case common.LaunchExecMsg:
		// Suspend the Bubble Tea renderer, hand over terminal to pty session, resume on exit
		return a, tea.ExecProcess(msg.Cmd, func(err error) tea.Msg {
			if msg.Callback != nil {
				return msg.Callback(err)
			}
			return common.ExecFinishedMsg{Err: err}
		})

	case common.ExecFinishedMsg:
		if msg.Err != nil {
			cmds = append(cmds, a.toast.SetMessage(fmt.Sprintf("Interactive session exited with error: %v", msg.Err), components.ToastError, 5*time.Second))
		} else {
			cmds = append(cmds, a.toast.SetMessage("Session finished", components.ToastInfo, 2*time.Second))
		}
		// Refresh dashboard or details after returning from shell/editor
		cmds = append(cmds, a.dashboardScreen.FetchVMsCmd())
		if a.activeScreen == common.ScreenDetail {
			cmds = append(cmds, a.detailScreen.RefreshCmd())
		}
		return a, tea.Batch(cmds...)

	case common.NavigateMsg:
		a.activeScreen = msg.Target
		if msg.Target == common.ScreenDetail && msg.VMName != "" {
			cmds = append(cmds, a.detailScreen.SetVM(msg.VMName))
		}
		if msg.Target == common.ScreenDashboard {
			cmds = append(cmds, a.dashboardScreen.FetchVMsCmd())
		}
		if msg.Target == common.ScreenCreateVM {
			if len(a.createVMScreen.Templates()) == 0 {
				cmds = append(cmds, a.createVMScreen.LoadTemplatesCmd())
			}
		}
		return a, tea.Batch(cmds...)

	case common.TemplatesLoadedMsg:
		var cmd tea.Cmd
		a.createVMScreen, cmd = a.createVMScreen.Update(msg)
		return a, cmd

	case common.ShowToastMsg:
		cmds = append(cmds, a.toast.SetMessage(msg.Message, msg.Level, msg.Duration))
		return a, tea.Batch(cmds...)

	case common.OpStartedMsg:
		cmds = append(cmds, a.spinner.Start(msg.Message))
		return a, tea.Batch(cmds...)

	case common.OpFinishedMsg:
		a.spinner.Stop()
		if msg.Err != nil {
			cmds = append(cmds, a.toast.SetMessage(msg.Err.Error(), components.ToastError, 6*time.Second))
		} else if msg.Message != "" {
			cmds = append(cmds, a.toast.SetMessage(msg.Message, components.ToastSuccess, 3*time.Second))
		}
		// Refresh dashboard & detail after state-changing operation
		cmds = append(cmds, a.dashboardScreen.FetchVMsCmd())
		if a.activeScreen == common.ScreenDetail {
			cmds = append(cmds, a.detailScreen.RefreshCmd())
		}
		return a, tea.Batch(cmds...)

	case common.OpenConfirmMsg:
		a.confirmDialog.Show(msg.Title, msg.Message, msg.ActionTag)
		return a, nil

	case common.OpenPromptMsg:
		cmds = append(cmds, a.promptDialog.Show(msg.Title, msg.Prompt, msg.DefaultValue, msg.Placeholder, msg.ActionTag))
		return a, tea.Batch(cmds...)

	case components.ConfirmResultMsg:
		if msg.ActionTag == "prune-garbage" && msg.Confirmed {
			return a, a.pruneCmd()
		}
		// Route confirm results to active screen
		switch a.activeScreen {
		case common.ScreenDashboard:
			cmds = append(cmds, a.dashboardScreen.HandleConfirmResult(msg.ActionTag, msg.Confirmed))
		case common.ScreenDetail:
			cmds = append(cmds, a.detailScreen.HandleConfirmResult(msg.ActionTag, msg.Confirmed))
		case common.ScreenNetworks:
			cmds = append(cmds, a.networksScreen.HandleConfirmResult(msg.ActionTag, msg.Confirmed))
		}
		return a, tea.Batch(cmds...)

	case components.PromptResultMsg:
		switch a.activeScreen {
		case common.ScreenDashboard:
			cmds = append(cmds, a.dashboardScreen.HandlePromptResult(msg.ActionTag, msg.Value, msg.Canceled))
		case common.ScreenDetail:
			cmds = append(cmds, a.detailScreen.HandlePromptResult(msg.ActionTag, msg.Value, msg.Canceled))
		case common.ScreenNetworks:
			cmds = append(cmds, a.networksScreen.HandlePromptResult(msg.ActionTag, msg.Value, msg.Canceled))
		}
		return a, tea.Batch(cmds...)

	case infoLoadedMsg:
		if msg.Err != nil {
			cmds = append(cmds, a.toast.SetMessage("Failed to fetch info: "+msg.Err.Error(), components.ToastError, 5*time.Second))
		} else {
			a.infoModal.Show(msg.Info)
		}
		return a, tea.Batch(cmds...)

	case components.ClearToastMsg:
		a.toast.Update(msg)
		return a, nil
	}

	// Update active overlays
	if a.helpModal.Active {
		if a.helpModal.Update(msg) {
			return a, nil
		}
	}
	if a.infoModal.Active {
		if a.infoModal.Update(msg) {
			return a, nil
		}
	}
	if a.confirmDialog.Active {
		if cmd, handled := a.confirmDialog.Update(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}
	}
	if a.promptDialog.Active {
		if cmd, handled := a.promptDialog.Update(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}
	}

	// Spinner animation update
	if a.spinner.Active {
		if cmd := a.spinner.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Global key handling when modals are not intercepting
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// When the active screen owns a text input (e.g. Create VM form),
		// keys belong to the field; only ctrl+c stays global.
		screenCaptures := a.activeScreen == common.ScreenCreateVM && a.createVMScreen.CapturesKeys()

		if keyMsg.String() == "ctrl+c" {
			a.quitting = true
			return a, tea.Quit
		}

		if !screenCaptures {
			switch keyMsg.String() {
			case "?":
				a.helpModal.Toggle()
				return a, nil

			case "1":
				if a.activeScreen != common.ScreenDashboard {
					a.activeScreen = common.ScreenDashboard
					cmds = append(cmds, a.dashboardScreen.FetchVMsCmd())
					return a, tea.Batch(cmds...)
				}

			case "2":
				if a.activeScreen != common.ScreenNetworks {
					a.activeScreen = common.ScreenNetworks
					cmds = append(cmds, a.networksScreen.FetchNetworksCmd())
					return a, tea.Batch(cmds...)
				}

			case "3":
				if a.activeScreen != common.ScreenCreateVM {
					a.activeScreen = common.ScreenCreateVM
					if len(a.createVMScreen.Templates()) == 0 {
						cmds = append(cmds, a.createVMScreen.LoadTemplatesCmd())
					}
					return a, tea.Batch(cmds...)
				}

			case "4":
				if a.activeScreen != common.ScreenProfiles {
					a.activeScreen = common.ScreenProfiles
					return a, tea.Batch(cmds...)
				}

			case "I":
				return a, a.loadInfoCmd()
			case "P":
				a.confirmDialog.Show(
					"Prune Garbage Objects",
					"Prune unused Lima caches and temporary files?\n(Keeps objects referred by existing instances)",
					"prune-garbage",
				)
				return a, nil

			case "q":
				// If on a sub-screen, return to dashboard; if on dashboard, quit
				if a.activeScreen != common.ScreenDashboard {
					a.activeScreen = common.ScreenDashboard
					cmds = append(cmds, a.dashboardScreen.FetchVMsCmd())
					return a, tea.Batch(cmds...)
				}
				a.quitting = true
				return a, tea.Quit
			}
		}
	}

	// Delegate to active screen
	var screenCmd tea.Cmd
	switch a.activeScreen {
	case common.ScreenDashboard:
		a.dashboardScreen, screenCmd = a.dashboardScreen.Update(msg)
	case common.ScreenDetail:
		a.detailScreen, screenCmd = a.detailScreen.Update(msg)
	case common.ScreenCreateVM:
		a.createVMScreen, screenCmd = a.createVMScreen.Update(msg)
	case common.ScreenNetworks:
		a.networksScreen, screenCmd = a.networksScreen.Update(msg)
	case common.ScreenProfiles:
		a.profilesScreen, screenCmd = a.profilesScreen.Update(msg)
	}
	if screenCmd != nil {
		cmds = append(cmds, screenCmd)
	}

	return a, tea.Batch(cmds...)
}

type infoLoadedMsg struct {
	Info *limactl.InfoDiagnostic
	Err  error
}

func (a App) loadInfoCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		info, err := a.client.Info(ctx)
		return infoLoadedMsg{Info: info, Err: err}
	}
}

func (a App) pruneCmd() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return common.OpStartedMsg{Message: "Pruning unused Lima objects..."}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
			defer cancel()

			out, err := a.client.Prune(ctx, true)
			msg := "Prune completed"
			if out != "" {
				msg = fmt.Sprintf("Prune: %s", out)
			}
			return common.OpFinishedMsg{
				Message: msg,
				Err:     err,
			}
		},
	)
}

// View renders the full application layout.
func (a App) View() string {
	if a.quitting {
		return ""
	}

	// 1. Header Navigation Bar
	navItems := []string{
		a.renderNavTab("1: VMs", a.activeScreen == common.ScreenDashboard),
		a.renderNavTab("2: Networks", a.activeScreen == common.ScreenNetworks),
		a.renderNavTab("3: Create VM", a.activeScreen == common.ScreenCreateVM),
		a.renderNavTab("4: Profiles", a.activeScreen == common.ScreenProfiles),
	}
	navBar := lipgloss.JoinHorizontal(lipgloss.Top, navItems...)

	appHeader := theme.AppTitleStyle.Render(" LimaVM TUI Manager ")
	var rightHeader string
	if a.spinner.Active {
		rightHeader = a.spinner.View()
	} else {
		rightHeader = theme.AppVersionStyle.Render("press [?] for help")
	}

	headerWidth := a.width
	if headerWidth <= 0 {
		headerWidth = 80
	}
	middleSpace := headerWidth - lipgloss.Width(appHeader) - lipgloss.Width(navBar) - lipgloss.Width(rightHeader) - 2
	if middleSpace < 1 {
		middleSpace = 1
	}
	spacer := strings.Repeat(" ", middleSpace)
	topBar := lipgloss.JoinHorizontal(lipgloss.Center, appHeader, " ", navBar, spacer, rightHeader)

	// 2. Active Screen View
	var screenView string
	switch a.activeScreen {
	case common.ScreenDashboard:
		screenView = a.dashboardScreen.View()
	case common.ScreenDetail:
		screenView = a.detailScreen.View()
	case common.ScreenCreateVM:
		screenView = a.createVMScreen.View()
	case common.ScreenNetworks:
		screenView = a.networksScreen.View()
	case common.ScreenProfiles:
		screenView = a.profilesScreen.View()
	}

	// 3. Modal Overlays (if active)
	if a.helpModal.Active {
		screenView = a.placeOverlay(screenView, a.helpModal.View())
	} else if a.infoModal.Active {
		screenView = a.placeOverlay(screenView, a.infoModal.View())
	} else if a.confirmDialog.Active {
		screenView = a.placeOverlay(screenView, a.confirmDialog.View())
	} else if a.promptDialog.Active {
		screenView = a.placeOverlay(screenView, a.promptDialog.View())
	}

	// 4. Toast notification line
	var toastLine string
	if a.toast.IsVisible() {
		toastLine = a.toast.View()
	}

	// 5. Bottom Keybinding Help Bar
	var keys []components.KeyHelp
	switch a.activeScreen {
	case common.ScreenDashboard:
		keys = a.dashboardScreen.KeyHelp()
	case common.ScreenDetail:
		keys = a.detailScreen.KeyHelp()
	case common.ScreenCreateVM:
		keys = a.createVMScreen.KeyHelp()
	case common.ScreenNetworks:
		keys = a.networksScreen.KeyHelp()
	case common.ScreenProfiles:
		keys = a.profilesScreen.KeyHelp()
	}
	footer := a.helpBar.Render(keys)

	var layout []string
	layout = append(layout, topBar, screenView)
	if toastLine != "" {
		layout = append(layout, toastLine)
	}
	layout = append(layout, footer)

	return lipgloss.JoinVertical(lipgloss.Left, layout...)
}

func (a App) renderNavTab(title string, active bool) string {
	if active {
		return theme.TabActiveStyle.Render(title)
	}
	return theme.TabInactiveStyle.Render(title)
}

func (a App) placeOverlay(baseView, overlayView string) string {
	// Simple overlay placement centered over base view
	w := a.width
	if w <= 0 {
		w = 80
	}
	h := a.height
	if h <= 0 {
		h = 24
	}

	return lipgloss.Place(
		w, h-4,
		lipgloss.Center, lipgloss.Center,
		overlayView,
		lipgloss.WithWhitespaceChars(" "),
	)
}
