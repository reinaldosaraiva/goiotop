package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/application/services"
)

// ViewType represents the different views in the TUI
type ViewType int

const (
	ProcessListView ViewType = iota
	SystemSummaryView
	HelpView
)

// App represents the main TUI application model
type App struct {
	orchestrator   *services.MetricsOrchestrator
	ctx            context.Context
	cancel         context.CancelFunc

	// View management
	currentView    ViewType
	processListModel ProcessListModel
	summaryModel   SummaryModel
	helpModel      HelpModel

	// Data
	displayResult  *dto.DisplayResultDTO
	lastUpdate     time.Time
	refreshRate    time.Duration

	// UI components
	width          int
	height         int
	help           help.Model
	keys           KeyMap

	// State
	err            error
	quitting       bool
}

// tickMsg is sent on refresh intervals
type tickMsg time.Time

// dataMsg contains updated metrics data
type dataMsg struct {
	result *dto.DisplayResultDTO
	err    error
}

// NewApp creates a new TUI application
func NewApp(orchestrator *services.MetricsOrchestrator, refreshRate time.Duration) *App {
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		orchestrator:    orchestrator,
		ctx:            ctx,
		cancel:         cancel,
		currentView:    ProcessListView,
		processListModel: NewProcessListModel(),
		summaryModel:   NewSummaryModel(),
		helpModel:      NewHelpModel(),
		refreshRate:    refreshRate,
		help:           help.New(),
		keys:           DefaultKeyMap(),
	}
}

// Init initializes the TUI
func (a *App) Init() tea.Cmd {
	// Initialize orchestrator
	if err := a.orchestrator.Initialize(a.ctx); err != nil {
		a.err = err
		return tea.Quit
	}

	// Start with initial data fetch and ticker
	return tea.Batch(
		a.fetchData(),
		a.tick(),
	)
}

// Update handles messages and updates the model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handling
		switch {
		case key.Matches(msg, a.keys.Quit):
			a.quitting = true
			a.cancel()
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			if a.currentView == HelpView {
				a.currentView = ProcessListView
			} else {
				a.currentView = HelpView
			}
			return a, nil

		case key.Matches(msg, a.keys.Tab):
			a.cycleView()
			return a, nil

		case key.Matches(msg, a.keys.ProcessView):
			a.currentView = ProcessListView
			return a, nil

		case key.Matches(msg, a.keys.SummaryView):
			a.currentView = SystemSummaryView
			return a, nil

		case key.Matches(msg, a.keys.Refresh):
			return a, a.fetchData()
		}

		// Pass to current view
		switch a.currentView {
		case ProcessListView:
			model, cmd := a.processListModel.Update(msg)
			a.processListModel = model.(ProcessListModel)
			cmds = append(cmds, cmd)

		case SystemSummaryView:
			model, cmd := a.summaryModel.Update(msg)
			a.summaryModel = model.(SummaryModel)
			cmds = append(cmds, cmd)

		case HelpView:
			model, cmd := a.helpModel.Update(msg)
			a.helpModel = model.(HelpModel)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update all views with new size
		a.processListModel.SetSize(msg.Width, msg.Height)
		a.summaryModel.SetSize(msg.Width, msg.Height)
		a.helpModel.SetSize(msg.Width, msg.Height)

	case tickMsg:
		cmds = append(cmds, a.fetchData())
		cmds = append(cmds, a.tick())

	case dataMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.displayResult = msg.result
			a.lastUpdate = time.Now()
			a.err = nil

			// Update views with new data
			if a.displayResult != nil {
				a.processListModel.SetData(a.displayResult)
				a.summaryModel.SetData(a.displayResult)
			}
		}
	}

	return a, tea.Batch(cmds...)
}

// View renders the current view
func (a *App) View() string {
	if a.quitting {
		return ""
	}

	// Error display
	if a.err != nil {
		return a.renderError()
	}

	// Header
	header := a.renderHeader()

	// Main content based on current view
	var content string
	switch a.currentView {
	case ProcessListView:
		content = a.processListModel.View()
	case SystemSummaryView:
		content = a.summaryModel.View()
	case HelpView:
		content = a.helpModel.View()
	}

	// Footer
	footer := a.renderFooter()

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		content,
		footer,
	)
}

// tick creates a ticker command
func (a *App) tick() tea.Cmd {
	return tea.Tick(a.refreshRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchData fetches metrics data
func (a *App) fetchData() tea.Cmd {
	return func() tea.Msg {
		options := dto.DisplayOptionsDTO{
			Mode:            "realtime",
			MaxRows:         100,
			ShowSystemStats: true,
			ShowProcesses:   true,
			SortBy:          a.processListModel.sortBy,
			SortOrder:       "desc",
		}

		result, err := a.orchestrator.DisplayMetrics(a.ctx, options)
		return dataMsg{result: result, err: err}
	}
}

// cycleView cycles through available views
func (a *App) cycleView() {
	switch a.currentView {
	case ProcessListView:
		a.currentView = SystemSummaryView
	case SystemSummaryView:
		a.currentView = HelpView
	case HelpView:
		a.currentView = ProcessListView
	}
}

// renderHeader renders the header section
func (a *App) renderHeader() string {
	title := HeaderStyle.Render("GoIOTop - System Monitor")

	viewIndicator := ""
	switch a.currentView {
	case ProcessListView:
		viewIndicator = "[Process List]"
	case SystemSummaryView:
		viewIndicator = "[System Summary]"
	case HelpView:
		viewIndicator = "[Help]"
	}

	updateTime := ""
	if !a.lastUpdate.IsZero() {
		updateTime = fmt.Sprintf("Last update: %s", a.lastUpdate.Format("15:04:05"))
	}

	headerLine := fmt.Sprintf("%s %s %s", title, viewIndicator, updateTime)
	return HeaderStyle.Width(a.width).Render(headerLine)
}

// renderFooter renders the footer section
func (a *App) renderFooter() string {
	helpKeys := []string{
		"TAB: Switch View",
		"h: Help",
		"q: Quit",
		"r: Refresh",
	}

	// Add view-specific keys
	if a.currentView == ProcessListView {
		helpKeys = append([]string{
			"c: Sort CPU",
			"m: Sort Memory",
			"i: Sort I/O",
			"p: Sort PID",
			"n: Sort Name",
			"f: Filter",
		}, helpKeys...)
	}

	helpText := lipgloss.JoinHorizontal(
		lipgloss.Left,
		helpKeys...,
	)

	return FooterStyle.Width(a.width).Render(helpText)
}

// renderError renders error messages
func (a *App) renderError() string {
	return ErrorStyle.Render(fmt.Sprintf("Error: %v", a.err))
}

// Run starts the TUI application
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}