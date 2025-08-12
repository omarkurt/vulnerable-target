package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/dockercompose"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/registry"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
	banner "github.com/happyhackingspace/vulnerable-target/internal/utils"
)

// Styles
var (
	// Compact banner/title
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		PaddingLeft(2)

	statusRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	statusStoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	statusHealthyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575"))

	statusUnhealthyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500"))

	portStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498db"))

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Italic(true)

	commandBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#FFFFFF"))
)

// keyMap defines all the keys used in the TUI
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Stop     key.Binding
	Start    key.Binding
	StartAlt key.Binding
	Info     key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "select"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop selected"),
	),
	Start: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart selected"),
	),
	StartAlt: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "start selected"),
	),
	Info: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "info for selected"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q/esc", "quit"),
	),
}

// TargetStatus represents the status of a running target
type TargetStatus struct {
	TemplateID   string
	TemplateName string
	Author       string
	Provider     string
	Status       string
	Health       string
	Containers   int
	Ports        []string
	Uptime       time.Duration
	LastChecked  time.Time
}

// Model represents the TUI model
type Model struct {
	table        table.Model
	targets      []TargetStatus
	help         help.Model
	spinner      spinner.Model
	loading      bool
	watchMode    bool
	showHelp     bool
	showInfo     bool
	infoText     string
	toast        string
	toastSuccess bool
	toastUntil   time.Time
	lastUpdate   time.Time
	width        int
	height       int
	selectedRow  int
	err          error
}

// StatusTUI manages the status display
type StatusTUI struct {
	watchMode bool
	interval  time.Duration
}

// NewStatusTUI creates a new status TUI
func NewStatusTUI() *StatusTUI {
	return &StatusTUI{
		watchMode: false,
		interval:  2 * time.Second,
	}
}

// SetWatchMode enables or disables watch mode
func (s *StatusTUI) SetWatchMode(watch bool) {
	s.watchMode = watch
}

// Run starts the TUI
func (s *StatusTUI) Run(ctx context.Context) error {
	model := s.initialModel()
	
	var opts []tea.ProgramOption
	opts = append(opts, tea.WithAltScreen())
	opts = append(opts, tea.WithMouseAllMotion())
	
	if s.watchMode {
		opts = append(opts, tea.WithoutCatchPanics())
	}

	p := tea.NewProgram(model, opts...)
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// initialModel creates the initial model
func (s *StatusTUI) initialModel() Model {
	// Create help model
	h := help.New()
	h.ShowAll = false

	// Create spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create initial table (compact widths)
	columns := []table.Column{
		{Title: "Template", Width: 16},
		{Title: "Name", Width: 28},
		{Title: "Prov", Width: 6},
		{Title: "Stat", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(tableStyle)

	return Model{
		table:     t,
		targets:   []TargetStatus{},
		help:      h,
		spinner:   sp,
		loading:   true,
		watchMode: s.watchMode,
		showHelp:  false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchStatus,
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
		case key.Matches(msg, keys.Info):
			m.toggleInfo()
			return m, nil
		case key.Matches(msg, keys.Refresh):
			m.loading = true
			return m, fetchStatus
		case key.Matches(msg, keys.Stop):
			if len(m.targets) > 0 && m.selectedRow < len(m.targets) {
				target := m.targets[m.selectedRow]
				return m, stopTarget(target)
			}
		case key.Matches(msg, keys.Start), key.Matches(msg, keys.StartAlt):
			if len(m.targets) > 0 && m.selectedRow < len(m.targets) {
				target := m.targets[m.selectedRow]
				return m, startTarget(target)
			}
		case key.Matches(msg, keys.Select):
			if len(m.targets) > 0 && m.selectedRow < len(m.targets) {
				target := m.targets[m.selectedRow]
				if strings.EqualFold(target.Status, "running") {
					return m, stopTarget(target)
				}
				return m, startTarget(target)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
		m.table.SetWidth(msg.Width)

	case statusUpdateMsg:
		m.loading = false
		m.targets = msg.targets
		m.lastUpdate = time.Now()
		m.updateTable()
		
		// If in watch mode, schedule next update
		if m.watchMode {
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}

	case tickMsg:
		if m.watchMode {
			return m, fetchStatus
		}

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case errorMsg:
		m.err = msg.err
		m.loading = false
		m.toast = fmt.Sprintf("%v", msg.err)
		m.toastSuccess = false
		m.toastUntil = time.Now().Add(3 * time.Second)
	case toastMsg:
		m.toast = msg.text
		m.toastSuccess = msg.success
		m.toastUntil = time.Now().Add(3 * time.Second)
		return m, fetchStatus
	}

	// Update table
	if !m.loading {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		
		// Track selected row
		m.selectedRow = m.table.Cursor()
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	var s strings.Builder

	// Compact banner centered
	bannerLine := fmt.Sprintf("%s  %s", banner.RainbowText("vt"), banner.AppVersion)
	s.WriteString(lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, bannerLine))
	s.WriteString("\n")
	// Title centered
	s.WriteString(lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, titleStyle.Render("🚀 Status Monitor")))
	s.WriteString("\n")

	// Loading state
	if m.loading {
		s.WriteString(fmt.Sprintf("%s Loading status...\n", m.spinner.View()))
		return s.String()
	}

	// Error state
	if m.err != nil {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render(
			fmt.Sprintf("❌ Error: %v\n", m.err),
		))
		s.WriteString("\nPress 'q' to quit or 'ctrl+r' to retry.\n")
		return s.String()
	}

	// No targets running
	if len(m.targets) == 0 {
		s.WriteString(infoStyle.Render("No vulnerable targets are currently running.\n\n"))
		s.WriteString("Start a target with: vt start --id <template-id> --provider docker-compose\n")
		s.WriteString("\nPress 'q' to quit.\n")
		return s.String()
	}
	// Table view centered
	tableView := m.table.View()
	s.WriteString(lipgloss.Place(m.width, m.height-6, lipgloss.Center, lipgloss.Top, tableView))
	s.WriteString("\n")

	// Toast notification centered
	if m.toast != "" && time.Now().Before(m.toastUntil) {
		style := errorStyle
		if m.toastSuccess {
			style = successStyle
		}
		s.WriteString(lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, style.Render(m.toast)))
		s.WriteString("\n")
	}

	// Command bar at bottom only
	cmds := "⌘: enter=toggle  a=start  s=stop  r=restart  i=info  ctrl+r=refresh  ?=help  q=quit"
	s.WriteString(lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, commandBarStyle.Render(cmds)))
	s.WriteString("\n")

	// Info panel
	if m.showInfo && m.infoText != "" {
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(m.width-10).Render(m.infoText)
		s.WriteString(lipgloss.Place(m.width, 6, lipgloss.Center, lipgloss.Top, box))
		s.WriteString("\n")
	}

	// Help
	if m.showHelp {
		helpText := []string{
			"Navigation:",
			"  ↑/k: Move up",
			"  ↓/j: Move down",
			"",
			"Actions:",
			"  enter: Toggle start/stop",
			"  a: Start selected target",
			"  s: Stop selected target",
			"  r: Restart selected target", 
			"  i: Show info for selected",
			"  ctrl+r: Refresh status",
			"",
			"  ?: Toggle this help",
			"  q/esc: Quit",
		}
		s.WriteString(helpStyle.Render(strings.Join(helpText, "\n")))
	} else {
		s.WriteString(helpStyle.Render("Press ? for help"))
	}

	return s.String()
}

// updateTable updates the table with current target data
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, target := range m.targets {
		// Stat column text
		var statTxt string
		if strings.EqualFold(target.Status, "running") {
			if target.Health == "healthy" {
				statTxt = "Running (healthy)"
			} else if target.Health == "partial" {
				statTxt = "Running (partial)"
			} else if target.Health == "unhealthy" {
				statTxt = "Running (unhealthy)"
			} else {
				statTxt = "Running"
			}
		} else {
			statTxt = "Stopped"
		}
		status := statTxt
		if strings.HasPrefix(statTxt, "Running") {
			status = statusRunningStyle.Render(statTxt)
		} else {
			status = statusStoppedStyle.Render(statTxt)
		}

		rows = append(rows, table.Row{
			target.TemplateID,
			target.TemplateName,
			shortProvider(target.Provider),
			status,
		})
	}
	m.table.SetRows(rows)
}

// Messages
type statusUpdateMsg struct {
	targets []TargetStatus
}

type tickMsg time.Time

type errorMsg struct {
	err error
}

type toastMsg struct {
	text    string
	success bool
}

// Commands
func fetchStatus() tea.Msg {
	targets := []TargetStatus{}
	
	// Docker client
	dc, err := dockercompose.NewDockerClient()
	if err != nil {
		// If Docker not available, return empty list gracefully
		return statusUpdateMsg{targets: targets}
	}
	defer dc.Close()
	
	// Get all templates
	allTemplates := templates.Templates
	
	// For each template, show status by querying Docker (running/stopped)
	for templateID, template := range allTemplates {
		if templateID == "example-template" {
			continue
		}
		projectName := projectNameFromTemplateID(templateID)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		containers, err := dc.GetContainersByProject(ctx, projectName)
		cancel()
		status := "stopped"
		health := "unknown"
		ports := []string{}
		containersCount := 0
		if err == nil && len(containers) > 0 {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
			svcStatus, _ := dc.GetContainerStatus(ctx2, projectName)
			cancel2()
			containersCount = len(svcStatus)
			runningCount := 0
			healthyCount := 0
			for _, s := range svcStatus {
				if s.State == "running" {
					runningCount++
				}
				if s.Health == "healthy" {
					healthyCount++
				}
				ports = append(ports, s.Ports...)
			}
			if runningCount > 0 {
				status = "running"
				if healthyCount == containersCount {
					health = "healthy"
				} else if healthyCount > 0 {
					health = "partial"
				} else {
					health = "unhealthy"
				}
			}
		}
		targets = append(targets, TargetStatus{
			TemplateID:   templateID,
			TemplateName: template.Info.Name,
			Author:       template.Info.Author,
			Provider:     "docker-compose",
			Status:       status,
			Health:       health,
			Containers:   containersCount,
			Ports:        ports,
			LastChecked:  time.Now(),
		})
	}
	
return statusUpdateMsg{targets: targets}
}

func (m *Model) toggleInfo() {
	if len(m.targets) == 0 || m.selectedRow >= len(m.targets) {
		m.showInfo = false
		return
	}
	m.showInfo = !m.showInfo
	if !m.showInfo {
		m.infoText = ""
		return
	}
	t := m.targets[m.selectedRow]
	lines := []string{}
	lines = append(lines, fmt.Sprintf("Template: %s (%s)", t.TemplateName, t.TemplateID))
	lines = append(lines, fmt.Sprintf("Author: %s", t.Author))
	lines = append(lines, fmt.Sprintf("Provider: %s", t.Provider))
	lines = append(lines, fmt.Sprintf("Status: %s", t.Status))
	if strings.EqualFold(t.Status, "running") {
		endpoints := endpointsFromPorts(t.Ports)
		if len(endpoints) > 0 {
			lines = append(lines, "Endpoints:")
			for _, e := range endpoints {
				lines = append(lines, "  - "+e)
			}
		}
	}
	// Add template tags/tech if available
	if tpl, err := templates.GetByID(t.TemplateID); err == nil {
		if len(tpl.Info.Technologies) > 0 {
			lines = append(lines, "Tech: "+strings.Join(tpl.Info.Technologies, ", "))
		}
		if len(tpl.Info.Tags) > 0 {
			lines = append(lines, "Tags: "+strings.Join(tpl.Info.Tags, ", "))
		}
	}
	m.infoText = strings.Join(lines, "\n")
}

func projectNameFromTemplateID(id string) string {
	return "vt-" + strings.ReplaceAll(id, "_", "-")
}

func endpointsFromPorts(ports []string) []string {
	endpoints := []string{}
	for _, p := range ports {
		// format expected: public:private/proto
		parts := strings.Split(p, ":")
		if len(parts) >= 2 {
			pub := parts[0]
			endpoints = append(endpoints, fmt.Sprintf("http://127.0.0.1:%s", pub))
		}
	}
	return endpoints
}

func shortProvider(p string) string {
	if p == "docker-compose" {
		return "dc"
	}
	return p
}

func stopTarget(target TargetStatus) tea.Cmd {
	return func() tea.Msg {
		// Get the template
		template, err := templates.GetByID(target.TemplateID)
		if err != nil {
			return errorMsg{err: err}
		}
		
		// Get the provider
		provider := registry.GetProvider(target.Provider)
		if provider == nil {
			return errorMsg{err: fmt.Errorf("provider %s not found", target.Provider)}
		}
		
		// Indicate action running
		
		// Stop the target
		if err := provider.Stop(template); err != nil {
			return errorMsg{err: err}
		}
		
		// Show success toast and trigger refresh
		return toastMsg{text: fmt.Sprintf("🛑 Stopped %s", target.TemplateName), success: true}
	}
}

func startTarget(target TargetStatus) tea.Cmd {
	return func() tea.Msg {
		// Get the template
		template, err := templates.GetByID(target.TemplateID)
		if err != nil {
			return errorMsg{err: err}
		}
		
		// Get the provider
		provider := registry.GetProvider(target.Provider)
		if provider == nil {
			return errorMsg{err: fmt.Errorf("provider %s not found", target.Provider)}
		}
		
		// Start the target
		if err := provider.Start(template); err != nil {
			return errorMsg{err: err}
		}
		
		// After start, resolve endpoints
		dc, err := dockercompose.NewDockerClient()
		if err != nil {
			return toastMsg{text: fmt.Sprintf("▶ Started %s", target.TemplateName), success: true}
		}
		defer dc.Close()
		projectName := projectNameFromTemplateID(target.TemplateID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		svcStatus, _ := dc.GetContainerStatus(ctx, projectName)
		cancel()
		allPorts := []string{}
		for _, s := range svcStatus {
			allPorts = append(allPorts, s.Ports...)
		}
		endpoints := endpointsFromPorts(allPorts)
		msg := fmt.Sprintf("▶ Started %s", target.TemplateName)
		if len(endpoints) > 0 {
			msg += " • " + strings.Join(endpoints, ", ")
		}
		// Show success toast and trigger refresh
		return toastMsg{text: msg, success: true}
	}
}
