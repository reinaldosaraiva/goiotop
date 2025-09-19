//go:build linux

package linux

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

func TestNewLinuxMetricsCollector(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	// Check capabilities
	caps := collector.GetCapabilities()
	if !caps.SupportsSystemMetrics {
		t.Error("Expected system metrics support")
	}
	if !caps.SupportsProcessMetrics {
		t.Error("Expected process metrics support")
	}
}

func TestCollectAll(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	err = collector.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize collector: %v", err)
	}

	options := services.DefaultCollectionOptions()

	result, err := collector.CollectAll(ctx, options)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that we have some metrics
	if result.SystemMetrics == nil {
		t.Error("Expected system metrics")
	}

	if len(result.ProcessMetrics) == 0 {
		t.Error("Expected at least one process")
	}

	if len(result.DiskIO) == 0 {
		t.Error("Expected at least one disk")
	}
}

func TestCollectSystemMetrics(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	metrics, err := collector.CollectSystemMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to collect system metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Validate CPU usage is in valid range
	if metrics.CPUUsage().Value() < 0 || metrics.CPUUsage().Value() > 100 {
		t.Errorf("Invalid CPU usage: %f", metrics.CPUUsage().Value())
	}

	// Validate memory usage is in valid range
	if metrics.MemoryUsage().Value() < 0 || metrics.MemoryUsage().Value() > 100 {
		t.Errorf("Invalid memory usage: %f", metrics.MemoryUsage().Value())
	}

	// Check that disk rates are non-negative
	if metrics.DiskReadRate().BytesPerSecond() < 0 {
		t.Error("Invalid disk read rate")
	}
	if metrics.DiskWriteRate().BytesPerSecond() < 0 {
		t.Error("Invalid disk write rate")
	}
}

func TestCollectCPUPressure(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Skip if PSI is not supported
	caps := collector.GetCapabilities()
	if !caps.SupportsCPUPressure {
		t.Skip("PSI not supported on this system")
	}

	ctx := context.Background()
	pressure, err := collector.CollectCPUPressure(ctx)
	if err != nil {
		t.Fatalf("Failed to collect CPU pressure: %v", err)
	}

	if pressure == nil {
		t.Fatal("Expected non-nil pressure")
	}

	// Validate pressure values are in valid range
	if pressure.Some10s().Value() < 0 || pressure.Some10s().Value() > 100 {
		t.Errorf("Invalid Some10s pressure: %f", pressure.Some10s().Value())
	}
}

func TestCollectProcessMetrics(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()

	// Collect metrics for our own process
	pid := os.Getpid()
	procID, _ := valueobjects.NewProcessID(uint32(pid))

	metrics, err := collector.CollectProcessMetrics(ctx, procID)
	if err != nil {
		t.Fatalf("Failed to collect process metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Should have our process name
	if metrics.Name() == "" {
		t.Error("Expected non-empty process name")
	}

	// CPU usage should be valid
	if metrics.CPUUsage().Value() < 0 || metrics.CPUUsage().Value() > 100 {
		t.Errorf("Invalid CPU usage: %f", metrics.CPUUsage().Value())
	}

	// Memory percentage should be valid
	if metrics.MemoryUsage().Value() < 0 || metrics.MemoryUsage().Value() > 100 {
		t.Errorf("Invalid memory usage percentage: %f", metrics.MemoryUsage().Value())
	}
}

func TestCollectAllProcessMetrics(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	processes, err := collector.CollectAllProcessMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to collect all process metrics: %v", err)
	}

	if len(processes) == 0 {
		t.Fatal("Expected at least one process")
	}

	// Find our own process
	pid := os.Getpid()
	found := false
	for _, proc := range processes {
		if int(proc.ProcessID().Value()) == pid {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find current process in list")
	}
}

func TestCollectFilteredProcessMetrics(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()

	// Filter for processes using more than 0% CPU
	filter := services.ProcessFilter{
		MinCPUThreshold: 0.0,
	}

	processes, err := collector.CollectFilteredProcessMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to collect filtered process metrics: %v", err)
	}

	// Should have some processes
	if len(processes) == 0 {
		t.Error("Expected at least one process matching filter")
	}
}

func TestCollectDiskIO(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	disks, err := collector.CollectAllDiskIO(ctx)
	if err != nil {
		t.Fatalf("Failed to collect disk I/O: %v", err)
	}

	// Should have at least one disk
	if len(disks) == 0 {
		t.Skip("No physical disks found")
	}

	// Test collecting specific disk
	firstDisk := disks[0].DeviceName()
	diskIO, err := collector.CollectDiskIO(ctx, firstDisk)
	if err != nil {
		t.Fatalf("Failed to collect disk I/O for %s: %v", firstDisk, err)
	}

	if diskIO == nil {
		t.Fatal("Expected non-nil disk I/O")
	}

	if diskIO.DeviceName() != firstDisk {
		t.Errorf("Expected device %s, got %s", firstDisk, diskIO.DeviceName())
	}
}

func TestStartStopContinuousCollection(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	results := make(chan *services.CollectionResult, 10)

	// Initialize collector first
	err = collector.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize collector: %v", err)
	}

	// Start continuous collection
	options := services.DefaultCollectionOptions()
	options.Interval = 100 * time.Millisecond
	err = collector.StartContinuousCollection(ctx, options, func(result *services.CollectionResult) {
		select {
		case results <- result:
		default:
		}
	})

	if err != nil {
		t.Fatalf("Failed to start continuous collection: %v", err)
	}

	// Wait for at least 2 collections
	time.Sleep(250 * time.Millisecond)

	// Stop collection
	err = collector.StopContinuousCollection(ctx)
	if err != nil {
		t.Fatalf("Failed to stop continuous collection: %v", err)
	}

	// Check we got some results
	close(results)
	count := 0
	for range results {
		count++
	}

	if count < 2 {
		t.Errorf("Expected at least 2 collections, got %d", count)
	}
}

func TestGetCapabilities(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	caps := collector.GetCapabilities()

	// Should support system metrics on Linux
	if !caps.SupportsSystemMetrics {
		t.Error("Expected system metrics support on Linux")
	}

	// Should support process metrics on Linux
	if !caps.SupportsProcessMetrics {
		t.Error("Expected process metrics support on Linux")
	}

	// Should support disk I/O on Linux
	if !caps.SupportsDiskIO {
		t.Error("Expected disk I/O support on Linux")
	}

	// Check platform-specific info exists
	if caps.PlatformSpecific == nil {
		t.Error("Expected platform-specific information")
	}
}

func TestResetAndCleanup(t *testing.T) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	// Initialize and collect some metrics first
	ctx := context.Background()
	err = collector.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize collector: %v", err)
	}

	_, err = collector.CollectAll(ctx, services.DefaultCollectionOptions())
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Get statistics before reset
	statsBefore := collector.GetStatistics()
	if statsBefore.TotalCollections == 0 {
		t.Error("Expected non-zero collection count")
	}

	// Reset
	err = collector.Reset()
	if err != nil {
		t.Fatalf("Failed to reset collector: %v", err)
	}

	// Get statistics after reset
	statsAfter := collector.GetStatistics()
	if statsAfter.TotalCollections != 0 {
		t.Error("Expected zero collection count after reset")
	}

	// Cleanup
	err = collector.Cleanup()
	if err != nil {
		t.Fatalf("Failed to cleanup collector: %v", err)
	}
}

// Benchmark tests

func BenchmarkCollectAll(b *testing.B) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		b.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	err = collector.Initialize(ctx)
	if err != nil {
		b.Fatalf("Failed to initialize collector: %v", err)
	}

	options := services.DefaultCollectionOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectAll(ctx, options)
		if err != nil {
			b.Fatalf("Failed to collect metrics: %v", err)
		}
	}
}

func BenchmarkCollectSystemMetrics(b *testing.B) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		b.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectSystemMetrics(ctx)
		if err != nil {
			b.Fatalf("Failed to collect system metrics: %v", err)
		}
	}
}

func BenchmarkCollectProcessMetrics(b *testing.B) {
	collector, err := NewLinuxMetricsCollector()
	if err != nil {
		b.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	pid := os.Getpid()
	procID, _ := valueobjects.NewProcessID(uint32(pid))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectProcessMetrics(ctx, procID)
		if err != nil {
			b.Fatalf("Failed to collect process metrics: %v", err)
		}
	}
}

// PSI parsing tests

func TestPSIParser(t *testing.T) {
	// Use test fixtures
	testdataPath := filepath.Join("..", "..", "..", "..", "testdata", "linux")
	parser := &psiParser{
		cpuPath:    filepath.Join(testdataPath, "proc_pressure_cpu"),
		ioPath:     filepath.Join(testdataPath, "proc_pressure_io"),
		memoryPath: filepath.Join(testdataPath, "proc_pressure_memory"),
	}

	t.Run("ParseCPUPressure", func(t *testing.T) {
		pressure, err := parser.collectCPUPressure()
		if err != nil {
			// File might not exist in test environment
			if os.IsNotExist(err) {
				t.Skip("Test fixture not found")
			}
			t.Fatalf("Failed to parse CPU pressure: %v", err)
		}

		if pressure == nil {
			t.Fatal("Expected non-nil pressure")
		}

		// Check values from fixture
		if pressure.Some10s().Value() != 0.0 {
			t.Errorf("Expected Some10s=0.0, got %f", pressure.Some10s().Value())
		}
	})

	t.Run("ParseIOPressure", func(t *testing.T) {
		pressure, err := parser.collectIOPressure()
		if err != nil {
			// File might not exist in test environment
			if os.IsNotExist(err) {
				t.Skip("Test fixture not found")
			}
			t.Fatalf("Failed to parse I/O pressure: %v", err)
		}

		if pressure == nil {
			t.Fatal("Expected non-nil pressure")
		}

		// Check values from fixture
		if pressure.Some10s().Value() != 2.50 {
			t.Errorf("Expected Some10s=2.50, got %f", pressure.Some10s().Value())
		}
		if pressure.Full10s().Value() != 1.00 {
			t.Errorf("Expected Full10s=1.00, got %f", pressure.Full10s().Value())
		}
	})
}

// Test error handling

func TestErrorHandling(t *testing.T) {
	t.Run("CollectorError", func(t *testing.T) {
		err := &CollectorError{
			Type:    ErrorTypePermission,
			Op:      "read",
			Path:    "/proc/1/io",
			Err:     ErrPermissionDenied,
			Message: "insufficient permissions",
		}

		// Test Error() method
		errStr := err.Error()
		if errStr == "" {
			t.Error("Expected non-empty error string")
		}

		// Test Is() method
		if !err.Is(ErrPermissionDenied) {
			t.Error("Expected error to match ErrPermissionDenied")
		}

		// Test Unwrap() method
		if err.Unwrap() != ErrPermissionDenied {
			t.Error("Expected unwrapped error to be ErrPermissionDenied")
		}
	})

	t.Run("ErrorAggregator", func(t *testing.T) {
		agg := NewErrorAggregator()

		// Add some errors
		agg.Add(ErrPermissionDenied)
		agg.AddWithContext("read", "/proc/1/io", ErrPermissionDenied)

		if !agg.HasErrors() {
			t.Error("Expected aggregator to have errors")
		}

		if agg.Count() != 2 {
			t.Errorf("Expected 2 errors, got %d", agg.Count())
		}

		// Test FirstError
		first := agg.FirstError()
		if first != ErrPermissionDenied {
			t.Error("Expected first error to be ErrPermissionDenied")
		}

		// Test Error() string
		errStr := agg.Error()
		if errStr == "" {
			t.Error("Expected non-empty error string")
		}
	})
}