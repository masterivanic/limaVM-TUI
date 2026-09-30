package createvm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/epel/limavm-tui/internal/limactl"
	"github.com/epel/limavm-tui/internal/store"
	"github.com/epel/limavm-tui/internal/theme"
	"github.com/epel/limavm-tui/internal/ui/common"
	"github.com/epel/limavm-tui/internal/ui/components"
)

type fieldIndex int

const (
	fieldMode fieldIndex = iota
	fieldName
	fieldTemplate
	fieldCustomPath
	fieldCPUs
	fieldMemory
	fieldDisk
	fieldSubmit
	fieldCount
)

type creationMode int

const (
	modeTemplate creationMode = iota
	modeCustomYAML
)

// Model represents the Create VM form screen.
type Model struct {
	client       *limactl.Client
	store        *store.Store
	templates    []limactl.Template
	tmplCursor   int
	focusedField fieldIndex
	mode         creationMode

	inputName       textinput.Model
	inputCustomPath textinput.Model
	inputCPUs       textinput.Model
	inputMemory     textinput.Model
	inputDisk       textinput.Model

	width   int
	height  int
	loading bool
	err     error
}

// New creates a new Create VM Model.
func New(client *limactl.Client, st *store.Store) Model {
	inName := textinput.New()
	inName.Placeholder = "my-vm"
	inName.CharLimit = 64

	inCustom := textinput.New()
	inCustom.Placeholder = "/path/to/lima.yaml or template://docker"
	inCustom.CharLimit = 256

	inCPUs := textinput.New()
	inCPUs.Placeholder = "2 (leave empty for template default)"
	inCPUs.CharLimit = 4

	inMem := textinput.New()
	inMem.Placeholder = "4 (GiB, leave empty for template default)"
	inMem.CharLimit = 6

	inDisk := textinput.New()
	inDisk.Placeholder = "20 (GiB, leave empty for template default)"
	inDisk.CharLimit = 6

	m := Model{
		client:          client,
		store:           st,
		focusedField:    fieldName,
		mode:            modeTemplate,
		inputName:       inName,
		inputCustomPath: inCustom,
		inputCPUs:       inCPUs,
		inputMemory:     inMem,
		inputDisk:       inDisk,
		loading:         true,
	}

	m.focusCurrentField()
	return m
}

// Init loads the template catalog.
func (m Model) Init() tea.Cmd {
	return m.LoadTemplatesCmd()
}

// LoadTemplatesCmd discovers templates asynchronously.
func (m Model) LoadTemplatesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), limactl.DefaultCommandTimeout)
		defer cancel()

		tmpls, err := m.client.ListTemplates(ctx)
		return common.TemplatesLoadedMsg{Templates: tmpls, Err: err}
	}
}

// Templates returns the loaded templates list.
func (m Model) Templates() []limactl.Template {
	return m.templates
}

// IsLoading returns whether templates are currently being loaded.
func (m Model) IsLoading() bool {
	return m.loading
}

// CapturesKeys reports whether a text input field currently owns the
// keyboard, so the root app must not intercept global shortcuts (digits, q,
// ?...) that would otherwise be typed into the field.
func (m Model) CapturesKeys() bool {
	switch m.focusedField {
	case fieldName, fieldCustomPath, fieldCPUs, fieldMemory, fieldDisk:
		return true
	}
	return false
}

// SetSize updates layout dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) focusCurrentField() {
	m.inputName.Blur()
	m.inputCustomPath.Blur()
	m.inputCPUs.Blur()
	m.inputMemory.Blur()
	m.inputDisk.Blur()

	switch m.focusedField {
	case fieldName:
		m.inputName.Focus()
	case fieldCustomPath:
		m.inputCustomPath.Focus()
	case fieldCPUs:
		m.inputCPUs.Focus()
	case fieldMemory:
		m.inputMemory.Focus()
	case fieldDisk:
		m.inputDisk.Focus()
	}
}

// Update processes keyboard input and commands.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case common.TemplatesLoadedMsg:
		m.loading = false
		m.err = msg.Err
		if msg.Err == nil {
			m.templates = msg.Templates
			// Match last-used template from store if available
			last := m.store.GetLastUsedTemplate()
			if last != "" {
				for i, t := range m.templates {
					if t.Name == last {
						m.tmplCursor = i
						break
					}
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return common.NavigateMsg{Target: common.ScreenDashboard}
			}

		case "tab", "down":
			m.nextField()
			m.focusCurrentField()
			return m, nil

		case "shift+tab", "up":
			m.prevField()
			m.focusCurrentField()
			return m, nil

		case "m", "M":
			// Toggle mode if not typing in text field
			if m.focusedField == fieldMode || m.focusedField == fieldSubmit {
				if m.mode == modeTemplate {
					m.mode = modeCustomYAML
				} else {
					m.mode = modeTemplate
				}
				return m, nil
			}

		case "r":
			if m.focusedField == fieldTemplate || len(m.templates) == 0 {
				return m, m.LoadTemplatesCmd()
			}

		case "enter":
			if m.focusedField == fieldSubmit {
				return m, m.submitCreateCmd()
			}
			m.nextField()
			m.focusCurrentField()
			return m, nil
		}

		// When template list is focused, arrow keys navigate templates
		if m.focusedField == fieldTemplate && m.mode == modeTemplate {
			switch msg.String() {
			case "left", "k":
				if m.tmplCursor > 0 {
					m.tmplCursor--
				}
				return m, nil
			case "right", "j":
				if m.tmplCursor < len(m.templates)-1 {
					m.tmplCursor++
				}
				return m, nil
			}
		}
	}

	// Update active textinput
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldName:
		m.inputName, cmd = m.inputName.Update(msg)
		cmds = append(cmds, cmd)
	case fieldCustomPath:
		m.inputCustomPath, cmd = m.inputCustomPath.Update(msg)
		cmds = append(cmds, cmd)
	case fieldCPUs:
		m.inputCPUs, cmd = m.inputCPUs.Update(msg)
		cmds = append(cmds, cmd)
	case fieldMemory:
		m.inputMemory, cmd = m.inputMemory.Update(msg)
		cmds = append(cmds, cmd)
	case fieldDisk:
		m.inputDisk, cmd = m.inputDisk.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) nextField() {
	m.focusedField++
	if m.mode == modeTemplate && m.focusedField == fieldCustomPath {
		m.focusedField++
	} else if m.mode == modeCustomYAML && m.focusedField == fieldTemplate {
		m.focusedField++
	}
	if m.focusedField >= fieldCount {
		m.focusedField = fieldMode
	}
}

func (m *Model) prevField() {
	if m.focusedField == 0 {
		m.focusedField = fieldCount - 1
	} else {
		m.focusedField--
	}
	if m.mode == modeTemplate && m.focusedField == fieldCustomPath {
		m.focusedField--
	} else if m.mode == modeCustomYAML && m.focusedField == fieldTemplate {
		m.focusedField--
	}
}

func (m Model) submitCreateCmd() tea.Cmd {
	name := strings.TrimSpace(m.inputName.Value())
	if name != "" {
		if err := limactl.ValidateName(name); err != nil {
			return func() tea.Msg {
				return common.ShowToastMsg{
					Message:  err.Error(),
					Level:    components.ToastError,
					Duration: 4 * time.Second,
				}
			}
		}
	}

	var target string
	if m.mode == modeTemplate {
		if len(m.templates) == 0 {
			return func() tea.Msg {
				return common.ShowToastMsg{
					Message:  "No templates available",
					Level:    components.ToastError,
					Duration: 3 * time.Second,
				}
			}
		}
		target = fmt.Sprintf("template://%s", m.templates[m.tmplCursor].Name)
	} else {
		target = strings.TrimSpace(m.inputCustomPath.Value())
		if target == "" {
			return func() tea.Msg {
				return common.ShowToastMsg{
					Message:  "Custom YAML path or template URL is required",
					Level:    components.ToastError,
					Duration: 4 * time.Second,
				}
			}
		}
	}

	var cpus int
	if v := strings.TrimSpace(m.inputCPUs.Value()); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cpus = n
		}
	}

	var memGiB float64
	if v := strings.TrimSpace(m.inputMemory.Value()); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			memGiB = f
		}
	}

	var diskGiB float64
	if v := strings.TrimSpace(m.inputDisk.Value()); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			diskGiB = f
		}
	}

	opts := limactl.CreateOpts{
		Name:           name,
		TemplateOrPath: target,
		CPUs:           cpus,
		MemoryGiB:      memGiB,
		DiskGiB:        diskGiB,
	}

	// Persist last used template
	if m.mode == modeTemplate && len(m.templates) > m.tmplCursor {
		_ = m.store.SetLastUsedTemplate(m.templates[m.tmplCursor].Name)
	}

	displayTarget := name
	if displayTarget == "" {
		displayTarget = target
	}

	return tea.Batch(
		func() tea.Msg {
			return common.OpStartedMsg{Message: fmt.Sprintf("Creating VM %q...", displayTarget)}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), limactl.LifecycleCommandTimeout)
			defer cancel()

			// Auto-validate custom YAML file if it's a local file path
			if m.mode == modeCustomYAML && strings.HasSuffix(target, ".yaml") || strings.HasSuffix(target, ".yml") {
				if _, err := os.Stat(target); err == nil {
					if err := m.client.ValidateYAML(ctx, target); err != nil {
						return common.OpFinishedMsg{
							Message: fmt.Sprintf("Validation failed for %s", target),
							Err:     err,
						}
					}
				}
			}

			err := m.client.Create(ctx, opts)
			return common.OpFinishedMsg{
				Message: fmt.Sprintf("VM %q created successfully", displayTarget),
				Err:     err,
			}
		},
		func() tea.Msg {
			// Navigate back to dashboard on creation kickoff
			return common.NavigateMsg{Target: common.ScreenDashboard}
		},
	)
}

// KeyHelp returns contextual keybindings for Create VM form.
func (m Model) KeyHelp() []components.KeyHelp {
	return []components.KeyHelp{
		{Key: "Tab / Enter", Desc: "Next Field"},
		{Key: "Shift+Tab", Desc: "Prev Field"},
		{Key: "Left / Right", Desc: "Browse Templates"},
		{Key: "m", Desc: "Toggle Mode"},
		{Key: "Esc", Desc: "Cancel & Return"},
	}
}

// View renders the Create VM form and template preview pane.
func (m Model) View() string {
	title := theme.DialogTitleStyle.Render("Create New Lima Virtual Machine")

	fieldLabel := func(f fieldIndex, label string) string {
		s := lipgloss.NewStyle().Bold(true).Width(18)
		if m.focusedField == f {
			s = s.Foreground(theme.ColorAccent)
		} else {
			s = s.Foreground(theme.ColorTextDim)
		}
		return s.Render(label)
	}

	// Mode toggle line
	modeTmpl := " [•] Template Picker "
	modeYAML := " [ ] Custom YAML / Path "
	if m.mode == modeCustomYAML {
		modeTmpl = " [ ] Template Picker "
		modeYAML = " [•] Custom YAML / Path "
	}
	modeStr := modeTmpl + modeYAML + " (Press [m] to switch)"
	if m.focusedField == fieldMode {
		modeStr = theme.TableRowSelectedStyle.Render(modeStr)
	}

	formRows := []string{
		fieldLabel(fieldMode, "Source Mode:") + " " + modeStr,
		fieldLabel(fieldName, "Instance Name:") + " " + m.inputName.View(),
	}

	if m.mode == modeTemplate {
		tmplSummary := "Loading templates..."
		if len(m.templates) > 0 {
			cur := m.templates[m.tmplCursor]
			tmplSummary = fmt.Sprintf("◀ %s (%d/%d) ▶", cur.Name, m.tmplCursor+1, len(m.templates))
			if cur.IsDefault {
				tmplSummary += " [DEFAULT]"
			}
		} else if !m.loading {
			if m.err != nil {
				tmplSummary = fmt.Sprintf("Error: %v (press [r] to retry)", m.err)
			} else {
				tmplSummary = "No templates found (press [r] to reload)"
			}
		}
		if m.focusedField == fieldTemplate {
			tmplSummary = theme.TableRowSelectedStyle.Render(tmplSummary)
		}
		formRows = append(formRows, fieldLabel(fieldTemplate, "Template:")+" "+tmplSummary)
	} else {
		formRows = append(formRows, fieldLabel(fieldCustomPath, "YAML / URL:")+" "+m.inputCustomPath.View())
	}

	formRows = append(formRows,
		fieldLabel(fieldCPUs, "Override CPUs:")+" "+m.inputCPUs.View(),
		fieldLabel(fieldMemory, "Override Memory:")+" "+m.inputMemory.View(),
		fieldLabel(fieldDisk, "Override Disk:")+" "+m.inputDisk.View(),
	)

	submitBtn := "[ Create Virtual Machine ]"
	if m.focusedField == fieldSubmit {
		submitBtn = theme.TableRowSelectedStyle.Render("▶ " + submitBtn + " ◀")
	} else {
		submitBtn = lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(submitBtn)
	}
	formRows = append(formRows, "\n"+submitBtn)

	formContent := lipgloss.JoinVertical(lipgloss.Left, formRows...)

	// Template Preview Pane (when in template mode)
	if m.mode == modeTemplate && len(m.templates) > m.tmplCursor {
		tmpl := m.templates[m.tmplCursor]
		previewTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorSecondary).
			Render(fmt.Sprintf("Template Details: %s", tmpl.Name))

		loc := lipgloss.NewStyle().
			Foreground(theme.ColorTextDim).
			Render("File: " + filepath.Base(tmpl.Location))

		desc := lipgloss.NewStyle().
			Foreground(theme.ColorText).
			Width(38).
			Render(tmpl.Description)

		previewBox := theme.PanelStyle.
			Width(42).
			Render(lipgloss.JoinVertical(lipgloss.Left, previewTitle, loc, "\n", desc))

		combined := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(m.width-46).Render(formContent),
			previewBox,
		)
		return lipgloss.JoinVertical(lipgloss.Left, title, combined)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, formContent)
}
