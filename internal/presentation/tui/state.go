package tui

import (
	"context"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/application/services"
)

// AppState represents the application state
type AppState struct {
	mu sync.RWMutex

	// View state
	CurrentView      ViewType
	PreviousView     ViewType

	// Data state
	LastUpdate       time.Time
	DisplayResult    *dto.DisplayResultDTO
	CollectionResult *dto.CollectionResultDTO

	// Sorting preferences
	SortBy           string
	SortOrder        string

	// Filter settings
	ProcessFilter    string
	IOThreshold      float64
	CPUThreshold     float64
	MemoryThreshold  float64

	// Display preferences
	ShowIdle         bool
	ShowKernel       bool
	ShowSystemProcs  bool
	RefreshRate      time.Duration

	// Session management
	SessionStartTime time.Time
	UpdateCount      int64
	ErrorCount       int64

	// User preferences (persisted)
	Preferences      UserPreferences
}

// UserPreferences represents user preferences that can be persisted
type UserPreferences struct {
	Theme            string        `json:"theme" yaml:"theme"`
	RefreshRate      time.Duration `json:"refresh_rate" yaml:"refresh_rate"`
	DefaultView      string        `json:"default_view" yaml:"default_view"`
	DefaultSort      string        `json:"default_sort" yaml:"default_sort"`
	ShowIdle         bool          `json:"show_idle" yaml:"show_idle"`
	ShowKernel       bool          `json:"show_kernel" yaml:"show_kernel"`
	ShowSystemProcs  bool          `json:"show_system_procs" yaml:"show_system_procs"`
	ProcessLimit     int           `json:"process_limit" yaml:"process_limit"`
}

// NewAppState creates a new application state
func NewAppState() *AppState {
	return &AppState{
		CurrentView:      ProcessListView,
		SortBy:          "cpu",
		SortOrder:       "desc",
		ShowIdle:        false,
		ShowKernel:      false,
		ShowSystemProcs: false,
		RefreshRate:     2 * time.Second,
		SessionStartTime: time.Now(),
		Preferences:     DefaultPreferences(),
	}
}

// DefaultPreferences returns default user preferences
func DefaultPreferences() UserPreferences {
	return UserPreferences{
		Theme:           "dark",
		RefreshRate:     2 * time.Second,
		DefaultView:     "processes",
		DefaultSort:     "cpu",
		ShowIdle:        false,
		ShowKernel:      false,
		ShowSystemProcs: false,
		ProcessLimit:    100,
	}
}

// UpdateData updates the data in the state
func (s *AppState) UpdateData(result *dto.DisplayResultDTO) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DisplayResult = result
	s.LastUpdate = time.Now()
	s.UpdateCount++
}

// SetView changes the current view
func (s *AppState) SetView(view ViewType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.PreviousView = s.CurrentView
	s.CurrentView = view
}

// ToggleView switches between current and previous view
func (s *AppState) ToggleView() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentView, s.PreviousView = s.PreviousView, s.CurrentView
}

// SetSort updates sorting preferences
func (s *AppState) SetSort(sortBy, sortOrder string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SortBy = sortBy
	s.SortOrder = sortOrder
}

// SetFilter updates filter settings
func (s *AppState) SetFilter(processFilter string, ioThreshold, cpuThreshold, memThreshold float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ProcessFilter = processFilter
	s.IOThreshold = ioThreshold
	s.CPUThreshold = cpuThreshold
	s.MemoryThreshold = memThreshold
}

// GetDisplayOptions returns display options based on current state
func (s *AppState) GetDisplayOptions() dto.DisplayOptionsDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return dto.DisplayOptionsDTO{
		Mode:            "realtime",
		RefreshInterval: s.RefreshRate,
		MaxRows:         s.Preferences.ProcessLimit,
		ShowSystemStats: true,
		ShowProcesses:   true,
		SortBy:          s.SortBy,
		SortOrder:       s.SortOrder,
		Filters: dto.DisplayFiltersDTO{
			MinCPU:        s.CPUThreshold,
			MinMemory:     s.MemoryThreshold,
			ProcessFilter: s.ProcessFilter,
		},
	}
}

// GetCollectionOptions returns collection options based on current state
func (s *AppState) GetCollectionOptions() dto.CollectionOptionsDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return dto.CollectionOptionsDTO{
		IncludeIdle:        s.ShowIdle,
		IncludeKernel:      s.ShowKernel,
		IncludeSystemProcs: s.ShowSystemProcs,
		ProcessLimit:       s.Preferences.ProcessLimit,
		SortBy:            s.SortBy,
		MinCPUThreshold:    s.CPUThreshold,
		MinMemThreshold:    s.MemoryThreshold,
		SampleDuration:     100 * time.Millisecond,
	}
}

// IncrementError increments the error counter
func (s *AppState) IncrementError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
}

// GetSessionDuration returns the current session duration
func (s *AppState) GetSessionDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.SessionStartTime)
}

// GetUpdateRate returns the current update rate (updates per second)
func (s *AppState) GetUpdateRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	duration := time.Since(s.SessionStartTime).Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.UpdateCount) / duration
}

// StateManager manages state coordination with the orchestrator
type StateManager struct {
	state        *AppState
	orchestrator *services.MetricsOrchestrator
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewStateManager creates a new state manager
func NewStateManager(orchestrator *services.MetricsOrchestrator) *StateManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &StateManager{
		state:        NewAppState(),
		orchestrator: orchestrator,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the state manager
func (sm *StateManager) Start() error {
	return sm.orchestrator.Initialize(sm.ctx)
}

// Stop stops the state manager
func (sm *StateManager) Stop() error {
	sm.cancel()
	return sm.orchestrator.Shutdown(sm.ctx)
}

// FetchData fetches new data from the orchestrator
func (sm *StateManager) FetchData() (*dto.DisplayResultDTO, error) {
	options := sm.state.GetDisplayOptions()
	result, err := sm.orchestrator.DisplayMetrics(sm.ctx, options)
	if err != nil {
		sm.state.IncrementError()
		return nil, err
	}

	sm.state.UpdateData(result)
	return result, nil
}

// GetState returns the current application state
func (sm *StateManager) GetState() *AppState {
	return sm.state
}

// SavePreferences saves user preferences to disk
func (sm *StateManager) SavePreferences() error {
	// TODO: Implement preference persistence
	// This would typically save to a config file
	return nil
}

// LoadPreferences loads user preferences from disk
func (sm *StateManager) LoadPreferences() error {
	// TODO: Implement preference loading
	// This would typically load from a config file
	return nil
}