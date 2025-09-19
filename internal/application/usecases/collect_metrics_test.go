package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/config"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// MockMetricsCollector is a mock implementation of MetricsCollector
type MockMetricsCollector struct {
	InitializeFunc    func(ctx context.Context) error
	CleanupFunc       func(ctx context.Context) error
	CollectSystemFunc func(ctx context.Context, opts *services.CollectionOptions) (*entities.SystemMetrics, error)
	CollectProcessFunc func(ctx context.Context, opts *services.CollectionOptions) ([]*entities.ProcessMetrics, error)
	CollectAllFunc    func(ctx context.Context, opts *services.CollectionOptions) (*services.CollectionResult, error)
	StartContinuousFunc func(ctx context.Context, interval time.Duration, opts *services.CollectionOptions, callback services.CollectionCallback) error
	StopContinuousFunc func(ctx context.Context) error
	HealthCheckFunc   func(ctx context.Context) error
	GetCapabilitiesFunc func(ctx context.Context) (map[string]interface{}, error)
	GetSystemInfoFunc func(ctx context.Context) (map[string]interface{}, error)
}

func (m *MockMetricsCollector) Initialize(ctx context.Context) error {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx)
	}
	return nil
}

func (m *MockMetricsCollector) Cleanup(ctx context.Context) error {
	if m.CleanupFunc != nil {
		return m.CleanupFunc(ctx)
	}
	return nil
}

func (m *MockMetricsCollector) CollectSystemMetrics(ctx context.Context, opts *services.CollectionOptions) (*entities.SystemMetrics, error) {
	if m.CollectSystemFunc != nil {
		return m.CollectSystemFunc(ctx, opts)
	}
	return createMockSystemMetrics(), nil
}

func (m *MockMetricsCollector) CollectProcessMetrics(ctx context.Context, opts *services.CollectionOptions) ([]*entities.ProcessMetrics, error) {
	if m.CollectProcessFunc != nil {
		return m.CollectProcessFunc(ctx, opts)
	}
	return createMockProcessMetrics(), nil
}

func (m *MockMetricsCollector) CollectAll(ctx context.Context, opts *services.CollectionOptions) (*services.CollectionResult, error) {
	if m.CollectAllFunc != nil {
		return m.CollectAllFunc(ctx, opts)
	}
	return &services.CollectionResult{
		System:    createMockSystemMetrics(),
		Processes: createMockProcessMetrics(),
	}, nil
}

func (m *MockMetricsCollector) StartContinuousCollection(ctx context.Context, interval time.Duration, opts *services.CollectionOptions, callback services.CollectionCallback) error {
	if m.StartContinuousFunc != nil {
		return m.StartContinuousFunc(ctx, interval, opts, callback)
	}
	// Simulate continuous collection with callback
	go func() {
		callback(createMockSystemMetrics(), createMockProcessMetrics(), nil)
	}()
	return nil
}

func (m *MockMetricsCollector) StopContinuousCollection(ctx context.Context) error {
	if m.StopContinuousFunc != nil {
		return m.StopContinuousFunc(ctx)
	}
	return nil
}

func (m *MockMetricsCollector) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

func (m *MockMetricsCollector) GetCapabilities(ctx context.Context) (map[string]interface{}, error) {
	if m.GetCapabilitiesFunc != nil {
		return m.GetCapabilitiesFunc(ctx)
	}
	return map[string]interface{}{
		"cpu":        true,
		"memory":     true,
		"disk":       true,
		"network":    true,
		"process":    true,
		"gpu":        false,
		"pressure":   true,
		"continuous": true,
	}, nil
}

func (m *MockMetricsCollector) GetSystemInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.GetSystemInfoFunc != nil {
		return m.GetSystemInfoFunc(ctx)
	}
	return map[string]interface{}{
		"hostname":         "test-host",
		"platform":         "linux",
		"platform_version": "5.10",
		"architecture":     "x86_64",
		"cpu_model":        "Intel Core i7",
		"cpu_cores":        8,
		"cpu_threads":      16,
		"total_memory":     uint64(16 * 1024 * 1024 * 1024),
		"boot_time":        time.Now().Add(-24 * time.Hour),
		"backend":          "gopsutil",
	}, nil
}

// Implement missing interface methods with default behavior
func (m *MockMetricsCollector) CollectCPUMetrics(ctx context.Context, opts *services.CollectionOptions) (*entities.CPUMetrics, error) {
	return &entities.CPUMetrics{
		UsagePercent: 50.0,
		CoreCount:    8,
	}, nil
}

func (m *MockMetricsCollector) CollectMemoryMetrics(ctx context.Context, opts *services.CollectionOptions) (*entities.MemoryMetrics, error) {
	return &entities.MemoryMetrics{
		Total:        16 * 1024 * 1024 * 1024,
		Used:         8 * 1024 * 1024 * 1024,
		UsagePercent: 50.0,
	}, nil
}

func (m *MockMetricsCollector) CollectDiskMetrics(ctx context.Context, opts *services.CollectionOptions) ([]entities.DiskMetrics, error) {
	return []entities.DiskMetrics{
		{
			Device:       "/dev/sda1",
			MountPoint:   "/",
			Total:        500 * 1024 * 1024 * 1024,
			Used:         250 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
		},
	}, nil
}

func (m *MockMetricsCollector) CollectNetworkMetrics(ctx context.Context, opts *services.CollectionOptions) ([]entities.NetworkMetrics, error) {
	return []entities.NetworkMetrics{
		{
			Interface:     "eth0",
			BytesSent:     1024000,
			BytesReceived: 2048000,
		},
	}, nil
}

func (m *MockMetricsCollector) CollectLoadAvg(ctx context.Context) (*entities.LoadAvg, error) {
	return &entities.LoadAvg{
		Load1:  1.5,
		Load5:  1.2,
		Load15: 1.0,
	}, nil
}

func (m *MockMetricsCollector) CollectUptime(ctx context.Context) (int64, error) {
	return 86400, nil // 1 day in seconds
}

func (m *MockMetricsCollector) CollectGPUMetrics(ctx context.Context, opts *services.CollectionOptions) ([]entities.GPUMetrics, error) {
	return []entities.GPUMetrics{}, nil
}

func (m *MockMetricsCollector) CollectCPUPressure(ctx context.Context) (*entities.CPUPressure, error) {
	return &entities.CPUPressure{
		Some10:  10.0,
		Some60:  8.0,
		Some300: 5.0,
	}, nil
}

func (m *MockMetricsCollector) CollectDiskIO(ctx context.Context, device string) (*entities.DiskIO, error) {
	return &entities.DiskIO{
		Device:     device,
		ReadBytes:  1024000,
		WriteBytes: 512000,
	}, nil
}

// MockMetricsRepository is a mock implementation of MetricsRepository
type MockMetricsRepository struct {
	SaveSystemFunc    func(ctx context.Context, metrics *entities.SystemMetrics) error
	SaveProcessFunc   func(ctx context.Context, metrics []*entities.ProcessMetrics) error
	InitializeFunc    func(ctx context.Context) error
	CloseFunc         func(ctx context.Context) error
	HealthCheckFunc   func(ctx context.Context) error
}

func (m *MockMetricsRepository) Initialize(ctx context.Context) error {
	if m.InitializeFunc != nil {
		return m.InitializeFunc(ctx)
	}
	return nil
}

func (m *MockMetricsRepository) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

func (m *MockMetricsRepository) SaveSystemMetrics(ctx context.Context, metrics *entities.SystemMetrics) error {
	if m.SaveSystemFunc != nil {
		return m.SaveSystemFunc(ctx, metrics)
	}
	return nil
}

func (m *MockMetricsRepository) SaveProcessMetrics(ctx context.Context, metrics []*entities.ProcessMetrics) error {
	if m.SaveProcessFunc != nil {
		return m.SaveProcessFunc(ctx, metrics)
	}
	return nil
}

func (m *MockMetricsRepository) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

// Implement other required interface methods with default behavior
func (m *MockMetricsRepository) QuerySystemMetrics(ctx context.Context, filter interface{}) ([]*entities.SystemMetrics, error) {
	return []*entities.SystemMetrics{createMockSystemMetrics()}, nil
}

func (m *MockMetricsRepository) QueryProcessMetrics(ctx context.Context, filter interface{}) ([]*entities.ProcessMetrics, error) {
	return createMockProcessMetrics(), nil
}

func (m *MockMetricsRepository) GetLatestSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error) {
	return createMockSystemMetrics(), nil
}

func (m *MockMetricsRepository) GetSystemMetricsAggregated(ctx context.Context, start, end time.Time, opts interface{}) ([]*entities.SystemMetrics, error) {
	return []*entities.SystemMetrics{createMockSystemMetrics()}, nil
}

func (m *MockMetricsRepository) GetTopProcessesByCPU(ctx context.Context, limit int) ([]*entities.ProcessMetrics, error) {
	return createMockProcessMetrics(), nil
}

func (m *MockMetricsRepository) GetTopProcessesByMemory(ctx context.Context, limit int) ([]*entities.ProcessMetrics, error) {
	return createMockProcessMetrics(), nil
}

func (m *MockMetricsRepository) CountSystemMetrics(ctx context.Context, filter interface{}) (int, error) {
	return 1, nil
}

func (m *MockMetricsRepository) CountProcessMetrics(ctx context.Context, filter interface{}) (int, error) {
	return 2, nil
}

func (m *MockMetricsRepository) DeleteOldMetrics(ctx context.Context, before time.Time) error {
	return nil
}

func (m *MockMetricsRepository) BeginTransaction(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *MockMetricsRepository) CommitTransaction(ctx context.Context, tx interface{}) error {
	return nil
}

func (m *MockMetricsRepository) RollbackTransaction(ctx context.Context, tx interface{}) error {
	return nil
}

// Helper functions
func createMockSystemMetrics() *entities.SystemMetrics {
	return &entities.SystemMetrics{
		Timestamp: time.Now(),
		Hostname:  "test-host",
		Platform:  "linux",
		CPU: entities.CPUMetrics{
			UsagePercent: 50.0,
			CoreCount:    8,
			ThreadCount:  16,
		},
		Memory: entities.MemoryMetrics{
			Total:        16 * 1024 * 1024 * 1024,
			Used:         8 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
		},
		LoadAvg: entities.LoadAvg{
			Load1:  1.5,
			Load5:  1.2,
			Load15: 1.0,
		},
		Uptime: 86400,
	}
}

func createMockProcessMetrics() []*entities.ProcessMetrics {
	return []*entities.ProcessMetrics{
		{
			PID:           1234,
			Name:          "test-process-1",
			Username:      "user1",
			State:         "running",
			CPUPercent:    25.0,
			MemoryPercent: 10.0,
			CreateTime:    time.Now().Add(-1 * time.Hour),
		},
		{
			PID:           5678,
			Name:          "test-process-2",
			Username:      "user2",
			State:         "sleeping",
			CPUPercent:    5.0,
			MemoryPercent: 20.0,
			CreateTime:    time.Now().Add(-2 * time.Hour),
		},
	}
}

// Test cases
func TestCollectMetrics_Execute_SingleCollection(t *testing.T) {
	ctx := context.Background()

	mockCollector := &MockMetricsCollector{}
	mockRepo := &MockMetricsRepository{}
	logger := config.NewNopLogger()
	cfg := &config.CollectionConfig{
		Interval:  2 * time.Second,
		Timeout:   5 * time.Second,
		BatchSize: 100,
		CollectorOptions: config.CollectorConfig{
			ProcessLimit: 100,
		},
	}

	uc := NewCollectMetrics(mockCollector, mockRepo, logger, cfg)

	request := dto.CollectionRequestDTO{
		Mode: "single",
		Options: dto.CollectionOptionsDTO{
			ProcessLimit: 100,
		},
	}

	result, err := uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.SystemMetrics == nil {
		t.Error("Expected system metrics, got nil")
	}

	if len(result.Processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(result.Processes))
	}
}

func TestCollectMetrics_Execute_SingleCollection_WithError(t *testing.T) {
	ctx := context.Background()

	expectedError := errors.New("collection failed")
	mockCollector := &MockMetricsCollector{
		CollectSystemFunc: func(ctx context.Context, opts *services.CollectionOptions) (*entities.SystemMetrics, error) {
			return nil, expectedError
		},
	}
	mockRepo := &MockMetricsRepository{}
	logger := config.NewNopLogger()
	cfg := &config.CollectionConfig{
		CollectorOptions: config.CollectorConfig{
			ProcessLimit: 100,
		},
	}

	uc := NewCollectMetrics(mockCollector, mockRepo, logger, cfg)

	request := dto.CollectionRequestDTO{
		Mode: "single",
		Options: dto.CollectionOptionsDTO{
			ProcessLimit: 100,
		},
	}

	result, err := uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error from Execute, got: %v", err)
	}

	if result.Success {
		t.Error("Expected success to be false")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors in result")
	}
}

func TestCollectMetrics_Execute_BatchCollection(t *testing.T) {
	ctx := context.Background()

	mockCollector := &MockMetricsCollector{}
	mockRepo := &MockMetricsRepository{}
	logger := config.NewNopLogger()
	cfg := &config.CollectionConfig{
		BatchSize: 10,
		CollectorOptions: config.CollectorConfig{
			ProcessLimit: 100,
		},
	}

	uc := NewCollectMetrics(mockCollector, mockRepo, logger, cfg)

	request := dto.CollectionRequestDTO{
		Mode: "batch",
		Options: dto.CollectionOptionsDTO{
			ProcessLimit: 10,
		},
	}

	result, err := uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.SystemMetrics == nil {
		t.Error("Expected system metrics")
	}

	if len(result.Processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(result.Processes))
	}
}

func TestCollectMetrics_ValidateCollectionCapabilities(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		capabilities  map[string]interface{}
		request       dto.CollectionRequestDTO
		expectError   bool
	}{
		{
			name: "continuous mode supported",
			capabilities: map[string]interface{}{
				"continuous": true,
				"gpu":        false,
				"pressure":   true,
			},
			request: dto.CollectionRequestDTO{
				Mode: "continuous",
				Options: dto.CollectionOptionsDTO{
					EnableGPU:      false,
					EnablePressure: true,
				},
			},
			expectError: false,
		},
		{
			name: "continuous mode not supported",
			capabilities: map[string]interface{}{
				"continuous": false,
			},
			request: dto.CollectionRequestDTO{
				Mode: "continuous",
			},
			expectError: true,
		},
		{
			name: "GPU not supported",
			capabilities: map[string]interface{}{
				"gpu": false,
			},
			request: dto.CollectionRequestDTO{
				Mode: "single",
				Options: dto.CollectionOptionsDTO{
					EnableGPU: true,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCollector := &MockMetricsCollector{
				GetCapabilitiesFunc: func(ctx context.Context) (map[string]interface{}, error) {
					return tt.capabilities, nil
				},
			}
			mockRepo := &MockMetricsRepository{}
			logger := config.NewNopLogger()
			cfg := &config.CollectionConfig{}

			uc := NewCollectMetrics(mockCollector, mockRepo, logger, cfg)

			err := uc.ValidateCollectionCapabilities(tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestCollectMetrics_ProcessFiltering(t *testing.T) {
	ctx := context.Background()

	processes := createMockProcessMetrics()
	mockCollector := &MockMetricsCollector{
		CollectProcessFunc: func(ctx context.Context, opts *services.CollectionOptions) ([]*entities.ProcessMetrics, error) {
			return processes, nil
		},
	}
	mockRepo := &MockMetricsRepository{}
	logger := config.NewNopLogger()
	cfg := &config.CollectionConfig{
		CollectorOptions: config.CollectorConfig{
			ProcessLimit: 100,
		},
	}

	uc := NewCollectMetrics(mockCollector, mockRepo, logger, cfg)

	// Test with process name filter
	request := dto.CollectionRequestDTO{
		Mode: "single",
		Options: dto.CollectionOptionsDTO{
			ProcessLimit: 100,
		},
		Filters: dto.CollectionFiltersDTO{
			ProcessNames: []string{"test-process-1"},
		},
	}

	result, err := uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Processes) != 1 {
		t.Errorf("Expected 1 process after filtering, got %d", len(result.Processes))
	}

	if result.Processes[0].Name != "test-process-1" {
		t.Errorf("Expected process name 'test-process-1', got '%s'", result.Processes[0].Name)
	}

	// Test with exclude filter
	request.Filters = dto.CollectionFiltersDTO{
		ExcludeNames: []string{"test-process-1"},
	}

	result, err = uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Processes) != 1 {
		t.Errorf("Expected 1 process after filtering, got %d", len(result.Processes))
	}

	if result.Processes[0].Name != "test-process-2" {
		t.Errorf("Expected process name 'test-process-2', got '%s'", result.Processes[0].Name)
	}

	// Test with user filter
	request.Filters = dto.CollectionFiltersDTO{
		UserFilter: "user1",
	}

	result, err = uc.Execute(ctx, request)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Processes) != 1 {
		t.Errorf("Expected 1 process after filtering, got %d", len(result.Processes))
	}

	if result.Processes[0].Username != "user1" {
		t.Errorf("Expected username 'user1', got '%s'", result.Processes[0].Username)
	}
}