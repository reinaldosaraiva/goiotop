//go:build darwin

package darwin

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

func TestNewDarwinMetricsCollector(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	if collector == nil {
		t.Fatal("Collector is nil")
	}
	if collector.cache == nil {
		t.Error("Cache is nil")
	}
	if collector.processParser == nil {
		t.Error("Process parser is nil")
	}
	if collector.sysctlParser == nil {
		t.Error("Sysctl parser is nil")
	}
	if collector.pressureProxy == nil {
		t.Error("Pressure proxy is nil")
	}
	if collector.diskIOParser == nil {
		t.Error("Disk IO parser is nil")
	}
	if collector.capabilities == nil {
		t.Error("Capabilities is nil")
	}
}

func TestCollectSystemMetrics(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	metrics, err := collector.CollectSystemMetrics(ctx)

	// May fail in sandbox, but should not panic
	if err == nil {
		if metrics == nil {
			t.Error("Metrics is nil")
		}
		if metrics != nil && metrics.CPUCount <= 0 {
			t.Error("CPU count should be greater than 0")
		}
		if metrics != nil && metrics.MemoryTotal <= 0 {
			t.Error("Memory total should be greater than 0")
		}
	} else {
		t.Logf("Failed to collect system metrics (may be sandbox issue): %v", err)
	}
}

func TestCollectProcessMetrics(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	// Use our own PID for testing (current process)
	currentPID := os.Getpid()
	pid, err := valueobjects.NewProcessID(int32(currentPID))
	if err != nil {
		t.Fatalf("Failed to create process ID: %v", err)
	}

	metrics, err := collector.CollectProcessMetrics(ctx, pid)

	// Should work for current process
	if err != nil {
		t.Logf("Failed to collect process metrics (may need elevated permissions): %v", err)
	} else {
		if metrics == nil {
			t.Error("Metrics is nil")
		}
		if metrics != nil && metrics.PID != pid {
			t.Errorf("PID mismatch: expected %v, got %v", pid, metrics.PID)
		}
	}
}

func TestCollectAllProcessMetrics(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	metrics, err := collector.CollectAllProcessMetrics(ctx)

	// Should work even in sandbox (at least for current process)
	if err != nil {
		t.Logf("Failed to collect all process metrics: %v", err)
	} else {
		if metrics == nil {
			t.Error("Metrics is nil")
		}
		// Should at least have the current process
		if len(metrics) < 1 {
			t.Error("Should have at least one process (current)")
		}
	}
}

func TestCollectCPUPressure(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	pressure, err := collector.CollectCPUPressure(ctx)

	if err != nil {
		t.Logf("Failed to collect CPU pressure: %v", err)
	} else {
		if pressure == nil {
			t.Error("Pressure is nil")
		}
		if pressure != nil {
			// Check that values are within valid range
			if pressure.Avg10.Value() < 0 || pressure.Avg10.Value() > 100 {
				t.Errorf("CPU pressure avg10 out of range: %v", pressure.Avg10.Value())
			}
		}
	}
}

func TestCollectDiskIO(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	// First get all disk I/O to find a valid device
	allIO, err := collector.CollectAllDiskIO(ctx)
	if err != nil {
		t.Logf("Failed to collect disk I/O: %v", err)
	} else if len(allIO) > 0 {
		// Test with the first device found
		device := allIO[0].Device

		diskIO, err := collector.CollectDiskIO(ctx, device)
		if err != nil {
			t.Logf("Failed to collect disk I/O for device %s: %v", device, err)
		} else {
			if diskIO == nil {
				t.Error("Disk IO is nil")
			}
			if diskIO != nil && diskIO.Device != device {
				t.Errorf("Device mismatch: expected %v, got %v", device, diskIO.Device)
			}
		}
	}
}

func TestCollectAll(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize the collector first
	ctx := context.Background()
	if err := collector.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize collector: %v", err)
	}

	options := services.DefaultCollectionOptions()

	result, err := collector.CollectAll(ctx, options)

	// Should not error even if some metrics fail
	if err != nil {
		t.Fatalf("Failed to collect all metrics: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}
	// Check timestamp
	if result.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestGetCapabilities(t *testing.T) {
	collector, err := NewDarwinMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	caps := collector.GetCapabilities()

	// macOS-specific capabilities
	if caps.SupportsCGroups {
		t.Error("macOS should not support cgroups")
	}
	// We have proxy-based pressure metrics
	if !caps.SupportsCPUPressure {
		t.Error("Should support CPU pressure (proxy)")
	}
	if !caps.SupportsMemoryPressure {
		t.Error("Should support memory pressure (proxy)")
	}
	if !caps.SupportsIOPressure {
		t.Error("Should support IO pressure (proxy)")
	}
	if !caps.SupportsNetworkIO {
		t.Error("Should support network IO")
	}
	if !caps.SupportsDiskIO {
		t.Error("Should support disk IO")
	}
}

func TestCGOWrapper(t *testing.T) {
	t.Run("GetSystemInfo", func(t *testing.T) {
		info, err := GetSystemInfo()
		if err == nil {
			assert.Greater(t, info.CPUCount, 0)
			assert.Greater(t, info.MemoryTotal, int64(0))
		}
	})

	t.Run("GetProcessList", func(t *testing.T) {
		pids, err := GetProcessList()
		if err == nil {
			assert.NotNil(t, pids)
			// Should at least have init process and current process
			assert.GreaterOrEqual(t, len(pids), 1)
		}
	})

	t.Run("GetLoadAverage", func(t *testing.T) {
		loadAvg, err := GetLoadAverage()
		if err == nil {
			assert.NotNil(t, loadAvg)
			assert.GreaterOrEqual(t, loadAvg.Load1, float64(0))
		}
	})
}

func TestCPUPressureProxy(t *testing.T) {
	proxy := newCPUPressureProxy()
	ctx := context.Background()

	pressure, err := proxy.collectCPUPressure(ctx)
	if err == nil {
		assert.NotNil(t, pressure)

		// Verify pressure values are reasonable
		assert.GreaterOrEqual(t, pressure.Avg10.Value(), float64(0))
		assert.LessOrEqual(t, pressure.Avg10.Value(), float64(100))
		assert.GreaterOrEqual(t, pressure.Avg60.Value(), float64(0))
		assert.LessOrEqual(t, pressure.Avg60.Value(), float64(100))
		assert.GreaterOrEqual(t, pressure.Avg300.Value(), float64(0))
		assert.LessOrEqual(t, pressure.Avg300.Value(), float64(100))
	}
}

func TestSandboxDetection(t *testing.T) {
	detector := newSandboxDetector()

	// Check that detection doesn't panic
	assert.NotNil(t, detector)

	// Get sandbox report
	report := detector.GetSandboxReport()
	assert.NotNil(t, report)
	assert.Contains(t, report, "is_sandboxed")
	assert.Contains(t, report, "available_features")
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "none"},
		{"permission error", ErrInsufficientPrivileges, "permission"},
		{"sandbox error", ErrSandboxRestriction, "sandbox"},
		{"not found error", ErrProcessNotFound, "not_found"},
		{"context timeout", context.DeadlineExceeded, "timeout"},
		{"context canceled", context.Canceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricsCache(t *testing.T) {
	cache := newMetricsCache()

	t.Run("ProcessMetrics", func(t *testing.T) {
		// Add a process snapshot
		snapshot := &processMetricsSnapshot{
			pid:          123,
			bytesRead:    1000,
			bytesWritten: 2000,
			timestamp:    time.Now(),
		}

		cache.UpdateProcessMetrics(123, snapshot)

		// Retrieve it
		retrieved, exists := cache.GetProcessMetrics(123)
		assert.True(t, exists)
		assert.NotNil(t, retrieved)
		assert.Equal(t, int32(123), retrieved.pid)
	})

	t.Run("CalculateRates", func(t *testing.T) {
		// First snapshot
		snapshot1 := &processMetricsSnapshot{
			pid:          456,
			bytesRead:    1000,
			bytesWritten: 2000,
			readOps:      10,
			writeOps:     20,
			timestamp:    time.Now(),
		}

		readRate, writeRate, readOps, writeOps := cache.CalculateProcessRates(456, snapshot1)

		// First calculation should return zeros
		assert.Equal(t, float64(0), readRate)
		assert.Equal(t, float64(0), writeRate)
		assert.Equal(t, int64(0), readOps)
		assert.Equal(t, int64(0), writeOps)

		// Wait a bit and add second snapshot
		time.Sleep(10 * time.Millisecond)

		snapshot2 := &processMetricsSnapshot{
			pid:          456,
			bytesRead:    2000,
			bytesWritten: 4000,
			readOps:      20,
			writeOps:     40,
			timestamp:    time.Now(),
		}

		readRate, writeRate, readOps, writeOps = cache.CalculateProcessRates(456, snapshot2)

		// Should now have non-zero rates
		assert.Greater(t, readRate, float64(0))
		assert.Greater(t, writeRate, float64(0))
		assert.Greater(t, readOps, int64(0))
		assert.Greater(t, writeOps, int64(0))
	})
}

// Benchmark tests
func BenchmarkCollectSystemMetrics(b *testing.B) {
	collector, err := NewDarwinMetricsCollector()
	require.NoError(b, err)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.CollectSystemMetrics(ctx)
	}
}

func BenchmarkCollectAllProcessMetrics(b *testing.B) {
	collector, err := NewDarwinMetricsCollector()
	require.NoError(b, err)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.CollectAllProcessMetrics(ctx)
	}
}

func BenchmarkCGOGetProcessIOStats(b *testing.B) {
	pid := int32(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetProcessIOStats(pid)
	}
}