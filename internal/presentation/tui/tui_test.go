package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/application/services"
	"github.com/reinaldosaraiva/goiotop/internal/application/config"
)

// MockOrchestrator implements a mock MetricsOrchestrator for testing
type MockOrchestrator struct {
	initialized bool
	shutdown    bool
}

func (m *MockOrchestrator) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *MockOrchestrator) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return nil
}

func (m *MockOrchestrator) DisplayMetrics(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	// Return mock data
	return &dto.DisplayResultDTO{
		Mode:      "realtime",
		Timestamp: time.Now(),
		SystemSummary: &dto.SystemSummaryDTO{
			CPUUsage:     45.5,
			MemoryUsage:  67.2,
			ProcessCount: 150,
		},
		Processes: []dto.ProcessMetricsDTO{
			{
				PID:  1234,
				Name: "test_process",
				User: "testuser",
				CPU: dto.CPUMetricsDTO{
					UsagePercent: 25.5,
				},
				Memory: dto.MemoryMetricsDTO{
					UsagePercent: 10.2,
					RSS:          1024 * 1024 * 100,
				},
			},
		},
	}, nil
}

func (m *MockOrchestrator) CollectMetrics(ctx context.Context, options *services.CollectionOptions) (*services.CollectionResult, error) {
	return nil, nil
}

func (m *MockOrchestrator) ExportMetrics(ctx context.Context, result *services.CollectionResult, options services.ExportOptions) error {
	return nil
}

func (m *MockOrchestrator) StartMonitoring(ctx context.Context, options services.MonitorOptions) error {
	return nil
}

func TestAppInit(t *testing.T) {
	orchestrator := &MockOrchestrator{}
	app := NewApp(orchestrator, 2*time.Second)

	// Test initialization
	cmd := app.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}

	if !orchestrator.initialized {
		t.Error("Expected orchestrator to be initialized")
	}
}

func TestAppViewSwitching(t *testing.T) {
	orchestrator := &MockOrchestrator{}
	app := NewApp(orchestrator, 2*time.Second)

	// Test initial view
	if app.currentView != ProcessListView {
		t.Error("Expected initial view to be ProcessListView")
	}

	// Test tab key to switch views
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	updatedApp := model.(*App)
	if updatedApp.currentView != SystemSummaryView {
		t.Error("Expected view to switch to SystemSummaryView")
	}

	// Test switching to help view
	model, _ = updatedApp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updatedApp = model.(*App)
	if updatedApp.currentView != HelpView {
		t.Error("Expected view to switch to HelpView")
	}
}

func TestAppWindowResize(t *testing.T) {
	orchestrator := &MockOrchestrator{}
	app := NewApp(orchestrator, 2*time.Second)

	// Test window resize
	model, _ := app.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})
	updatedApp := model.(*App)

	if updatedApp.width != 120 || updatedApp.height != 40 {
		t.Errorf("Expected window size to be 120x40, got %dx%d",
			updatedApp.width, updatedApp.height)
	}
}

func TestProcessListModel(t *testing.T) {
	model := NewProcessListModel()

	// Test initial state
	if model.sortBy != "cpu" {
		t.Errorf("Expected initial sort to be 'cpu', got %s", model.sortBy)
	}

	// Test setting data
	result := &dto.DisplayResultDTO{
		Processes: []dto.ProcessMetricsDTO{
			{
				PID:  1,
				Name: "init",
				CPU:  dto.CPUMetricsDTO{UsagePercent: 0.1},
			},
			{
				PID:  100,
				Name: "systemd",
				CPU:  dto.CPUMetricsDTO{UsagePercent: 2.5},
			},
		},
	}
	model.SetData(result)

	if len(model.processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(model.processes))
	}

	// Test sorting
	model.sortBy = "pid"
	model.sortProcesses()
	if model.filtered[0].PID != 1 {
		t.Error("Expected processes to be sorted by PID")
	}
}

func TestSummaryModel(t *testing.T) {
	model := NewSummaryModel()

	// Test setting data
	result := &dto.DisplayResultDTO{
		SystemSummary: &dto.SystemSummaryDTO{
			CPUUsage:     75.5,
			MemoryUsage:  60.2,
			ProcessCount: 200,
		},
	}
	model.SetData(result)

	if model.systemSummary == nil {
		t.Error("Expected system summary to be set")
	}

	if model.systemSummary.AvgCPUPercent != 75.5 {
		t.Errorf("Expected CPU usage to be 75.5, got %f",
			model.systemSummary.AvgCPUPercent)
	}
}

func TestHelpModel(t *testing.T) {
	model := NewHelpModel()

	// Test initial content
	if len(model.content) == 0 {
		t.Error("Expected help content to be populated")
	}

	// Test scrolling
	initialOffset := model.scrollOffset
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.scrollOffset <= initialOffset {
		t.Error("Expected scroll offset to increase")
	}
}

func TestKeyMap(t *testing.T) {
	keys := DefaultKeyMap()

	// Test that key bindings are defined
	if keys.Quit.Keys() == nil {
		t.Error("Expected Quit key to be defined")
	}

	if keys.SortCPU.Keys() == nil {
		t.Error("Expected SortCPU key to be defined")
	}

	// Test help generation
	shortHelp := keys.ShortHelp()
	if len(shortHelp) == 0 {
		t.Error("Expected short help to contain keys")
	}

	fullHelp := keys.FullHelp()
	if len(fullHelp) == 0 {
		t.Error("Expected full help to contain key groups")
	}
}

func TestAppState(t *testing.T) {
	state := NewAppState()

	// Test initial state
	if state.CurrentView != ProcessListView {
		t.Error("Expected initial view to be ProcessListView")
	}

	// Test view switching
	state.SetView(SystemSummaryView)
	if state.CurrentView != SystemSummaryView {
		t.Error("Expected view to be SystemSummaryView")
	}
	if state.PreviousView != ProcessListView {
		t.Error("Expected previous view to be ProcessListView")
	}

	// Test toggle
	state.ToggleView()
	if state.CurrentView != ProcessListView {
		t.Error("Expected view to toggle back to ProcessListView")
	}

	// Test sorting
	state.SetSort("memory", "asc")
	if state.SortBy != "memory" || state.SortOrder != "asc" {
		t.Error("Expected sort to be updated")
	}

	// Test filtering
	state.SetFilter("chrome", 10.0, 50.0, 20.0)
	if state.ProcessFilter != "chrome" {
		t.Error("Expected process filter to be set")
	}

	// Test display options generation
	opts := state.GetDisplayOptions()
	if opts.SortBy != "memory" {
		t.Errorf("Expected sort by memory, got %s", opts.SortBy)
	}
}

func TestStateManager(t *testing.T) {
	orchestrator := &MockOrchestrator{}
	config := &config.Config{
		Display: config.DisplayConfig{
			RefreshInterval: 2 * time.Second,
		},
	}

	// Create a real orchestrator for testing
	// Note: In a real test, you'd use a proper mock
	realOrchestrator := &services.MetricsOrchestrator{}
	_ = realOrchestrator

	manager := NewStateManager(orchestrator)

	// Test start
	err := manager.Start()
	if err != nil {
		t.Errorf("Expected start to succeed: %v", err)
	}

	if !orchestrator.initialized {
		t.Error("Expected orchestrator to be initialized")
	}

	// Test stop
	err = manager.Stop()
	if err != nil {
		t.Errorf("Expected stop to succeed: %v", err)
	}

	if !orchestrator.shutdown {
		t.Error("Expected orchestrator to be shut down")
	}
}

func TestUtils(t *testing.T) {
	// Test FormatBytes
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{500, "500 B"},
	}

	for _, test := range tests {
		result := FormatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s",
				test.bytes, result, test.expected)
		}
	}

	// Test FormatRate
	rateTests := []struct {
		rate     float64
		expected string
	}{
		{1024, "1.00 KB/s"},
		{1024 * 1024, "1.00 MB/s"},
		{500, "500 B/s"},
	}

	for _, test := range rateTests {
		result := FormatRate(test.rate)
		if result != test.expected {
			t.Errorf("FormatRate(%f) = %s; want %s",
				test.rate, result, test.expected)
		}
	}

	// Test SortProcesses
	processes := []dto.ProcessDisplayDTO{
		{PID: 3, CPUPercent: 10.0, Name: "c"},
		{PID: 1, CPUPercent: 50.0, Name: "a"},
		{PID: 2, CPUPercent: 25.0, Name: "b"},
	}

	sorted := SortProcesses(processes, "cpu", false)
	if sorted[0].CPUPercent != 50.0 {
		t.Error("Expected processes to be sorted by CPU descending")
	}

	sorted = SortProcesses(processes, "pid", true)
	if sorted[0].PID != 1 {
		t.Error("Expected processes to be sorted by PID ascending")
	}

	// Test FilterProcesses
	filtered := FilterProcesses(processes, "a", 0, 0, 0)
	if len(filtered) != 1 || filtered[0].Name != "a" {
		t.Error("Expected to filter by name")
	}

	filtered = FilterProcesses(processes, "", 30.0, 0, 0)
	if len(filtered) != 1 || filtered[0].CPUPercent != 50.0 {
		t.Error("Expected to filter by CPU threshold")
	}
}