//go:build linux

package linux

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
	"github.com/shirou/gopsutil/v4/cpu"
)

// LinuxMetricsCollector implements MetricsCollector for Linux systems
type LinuxMetricsCollector struct {
	mu         sync.RWMutex
	cache      *metricsCache
	psiParser  *psiParser
	procParser *procfsParser
	diskParser *diskStatsParser
	sysParser  *systemParser

	// Configuration
	interval   time.Duration
	continuous bool
	stopChan   chan struct{}
	doneChan   chan struct{}

	// State tracking
	lastCollectionTime time.Time
	collectionCount    uint64
	errorCount         uint64
	capabilities       services.CollectorCapabilities
	lastError          error
	initialized        bool
}

// NewLinuxMetricsCollector creates a new Linux-specific metrics collector
func NewLinuxMetricsCollector() (*LinuxMetricsCollector, error) {
	// Initialize CPU info for accurate per-CPU metrics
	if _, err := cpu.Info(); err != nil {
		return nil, fmt.Errorf("failed to initialize CPU info: %w", err)
	}

	c := &LinuxMetricsCollector{
		cache:      newMetricsCache(),
		psiParser:  newPSIParser(),
		procParser: newProcfsParser(),
		diskParser: newDiskStatsParser(),
		sysParser:  newSystemParser(),
		interval:   time.Second,
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}

	// Detect capabilities
	c.capabilities = c.detectCapabilities()

	return c, nil
}

// Initialize initializes the collector
func (c *LinuxMetricsCollector) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	// Initialize parsers if needed
	if err := c.sysParser.initialize(); err != nil {
		return fmt.Errorf("failed to initialize system parser: %w", err)
	}

	c.initialized = true
	return nil
}

// CollectAll collects all available metrics
func (c *LinuxMetricsCollector) CollectAll(ctx context.Context, options services.CollectionOptions) (*services.CollectionResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, services.ErrCollectorNotInitialized
	}

	startTime := time.Now()
	result := &services.CollectionResult{
		Timestamp:        time.Now(),
		ProcessMetrics:   []*entities.ProcessMetrics{},
		DiskIO:           []*entities.DiskIO{},
		Errors:           []error{},
		ProcessCount:     0,
		SkippedProcesses: 0,
	}

	// Collect system metrics
	if options.IncludeSystemMetrics {
		systemMetrics, err := c.sysParser.collectSystemMetrics()
		if err != nil {
			c.errorCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect system metrics: %w", err))
			c.lastError = err
		} else {
			result.SystemMetrics = systemMetrics
		}
	}

	// Collect CPU pressure if available and requested
	if options.IncludeCPUPressure && c.capabilities.SupportsCPUPressure {
		cpuPressure, err := c.psiParser.collectCPUPressure()
		if err != nil {
			// PSI might not be available, log but don't fail
			c.errorCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect CPU pressure: %w", err))
		} else {
			result.CPUPressure = cpuPressure
		}
	}

	// Collect process metrics
	if options.IncludeProcessMetrics {
		processList, skipped, err := c.procParser.collectAllProcessMetricsWithStats(options.ProcessFilter)
		if err != nil {
			c.errorCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect process metrics: %w", err))
			c.lastError = err
		} else {
			result.ProcessMetrics = processList
			result.ProcessCount = len(processList)
			result.SkippedProcesses = skipped
		}
	}

	// Collect disk I/O metrics
	if options.IncludeDiskIO {
		diskIOList, err := c.diskParser.collectAllDiskIO()
		if err != nil {
			c.errorCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect disk I/O metrics: %w", err))
			c.lastError = err
		} else {
			result.DiskIO = diskIOList
		}
	}

	c.lastCollectionTime = time.Now()
	c.collectionCount++
	result.CollectionTime = time.Since(startTime)

	return result, nil
}

// CollectSystemMetrics collects only system-wide metrics
func (c *LinuxMetricsCollector) CollectSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error) {
	return c.sysParser.collectSystemMetrics()
}

// CollectCPUPressure collects CPU pressure information
func (c *LinuxMetricsCollector) CollectCPUPressure(ctx context.Context) (*entities.CPUPressure, error) {
	if !c.capabilities.SupportsCPUPressure {
		return nil, ErrPSINotSupported
	}
	return c.psiParser.collectCPUPressure()
}

// CollectProcessMetrics collects metrics for a specific process
func (c *LinuxMetricsCollector) CollectProcessMetrics(ctx context.Context, pid valueobjects.ProcessID) (*entities.ProcessMetrics, error) {
	return c.procParser.collectProcessMetrics(int(pid.Value()))
}

// CollectAllProcessMetrics collects metrics for all processes
func (c *LinuxMetricsCollector) CollectAllProcessMetrics(ctx context.Context) ([]*entities.ProcessMetrics, error) {
	metrics, _, err := c.procParser.collectAllProcessMetricsWithStats(nil)
	return metrics, err
}

// CollectFilteredProcessMetrics collects metrics for filtered processes
func (c *LinuxMetricsCollector) CollectFilteredProcessMetrics(ctx context.Context, filter services.ProcessFilter) ([]*entities.ProcessMetrics, error) {
	metrics, _, err := c.procParser.collectAllProcessMetricsWithStats(&filter)
	return metrics, err
}

// CollectTopProcessesByCPU collects top N processes by CPU usage
func (c *LinuxMetricsCollector) CollectTopProcessesByCPU(ctx context.Context, n int) ([]*entities.ProcessMetrics, error) {
	processes, _, err := c.procParser.collectAllProcessMetricsWithStats(nil)
	if err != nil {
		return nil, err
	}

	// Sort and truncate to top N
	return c.procParser.getTopByCPU(processes, n), nil
}

// CollectTopProcessesByMemory collects top N processes by memory usage
func (c *LinuxMetricsCollector) CollectTopProcessesByMemory(ctx context.Context, n int) ([]*entities.ProcessMetrics, error) {
	processes, _, err := c.procParser.collectAllProcessMetricsWithStats(nil)
	if err != nil {
		return nil, err
	}

	// Sort and truncate to top N
	return c.procParser.getTopByMemory(processes, n), nil
}

// CollectTopProcessesByDiskIO collects top N processes by disk I/O
func (c *LinuxMetricsCollector) CollectTopProcessesByDiskIO(ctx context.Context, n int) ([]*entities.ProcessMetrics, error) {
	processes, _, err := c.procParser.collectAllProcessMetricsWithStats(nil)
	if err != nil {
		return nil, err
	}

	// Sort and truncate to top N
	return c.procParser.getTopByDiskIO(processes, n), nil
}

// CollectDiskIO collects I/O metrics for a specific disk
func (c *LinuxMetricsCollector) CollectDiskIO(ctx context.Context, device string) (*entities.DiskIO, error) {
	return c.diskParser.collectDiskIO(device)
}

// CollectAllDiskIO collects I/O metrics for all disks
func (c *LinuxMetricsCollector) CollectAllDiskIO(ctx context.Context) ([]*entities.DiskIO, error) {
	return c.diskParser.collectAllDiskIO()
}

// StartContinuousCollection starts continuous metrics collection
func (c *LinuxMetricsCollector) StartContinuousCollection(ctx context.Context, options services.CollectionOptions, callback services.CollectionCallback) error {
	c.mu.Lock()
	if c.continuous {
		c.mu.Unlock()
		return services.ErrCollectorBusy
	}
	c.continuous = true
	c.interval = options.Interval
	if c.interval == 0 {
		c.interval = time.Second
	}
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		defer close(c.doneChan)

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			case <-ticker.C:
				result, _ := c.CollectAll(ctx, options)
				if callback != nil {
					callback(result)
				}
			}
		}
	}()

	return nil
}

// StopContinuousCollection stops continuous metrics collection
func (c *LinuxMetricsCollector) StopContinuousCollection(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.continuous {
		return fmt.Errorf("continuous collection not running")
	}

	close(c.stopChan)
	<-c.doneChan

	c.continuous = false
	c.stopChan = make(chan struct{})
	c.doneChan = make(chan struct{})

	return nil
}

// GetCapabilities returns the collector's capabilities
func (c *LinuxMetricsCollector) GetCapabilities() services.CollectorCapabilities {
	return c.capabilities
}

// GetActiveProcesses returns active process IDs
func (c *LinuxMetricsCollector) GetActiveProcesses(ctx context.Context) ([]valueobjects.ProcessID, error) {
	return c.procParser.getActiveProcesses()
}

// GetProcessInfo returns detailed process information
func (c *LinuxMetricsCollector) GetProcessInfo(ctx context.Context, pid valueobjects.ProcessID) (*services.ProcessInfo, error) {
	return c.procParser.getProcessInfo(int(pid.Value()))
}

// GetSystemInfo returns system information
func (c *LinuxMetricsCollector) GetSystemInfo(ctx context.Context) (*services.SystemInfo, error) {
	return c.sysParser.getSystemInfo()
}

// GetDiskDevices returns available disk devices
func (c *LinuxMetricsCollector) GetDiskDevices(ctx context.Context) ([]string, error) {
	return c.diskParser.getAvailableDevices()
}

// IsCollecting returns whether continuous collection is running
func (c *LinuxMetricsCollector) IsCollecting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.continuous
}

// GetLastError returns the last error encountered
func (c *LinuxMetricsCollector) GetLastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// GetStatistics returns collection statistics
func (c *LinuxMetricsCollector) GetStatistics() services.CollectionStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return services.CollectionStatistics{
		TotalCollections:      c.collectionCount,
		SuccessfulCollections: c.collectionCount - c.errorCount,
		FailedCollections:     c.errorCount,
		LastCollectionTime:    c.lastCollectionTime,
	}
}

// Reset resets the collector state
func (c *LinuxMetricsCollector) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.reset()
	c.collectionCount = 0
	c.errorCount = 0
	c.lastCollectionTime = time.Time{}

	return nil
}

// Cleanup performs cleanup operations
func (c *LinuxMetricsCollector) Cleanup() error {
	if c.continuous {
		ctx := context.Background()
		return c.StopContinuousCollection(ctx)
	}
	return nil
}

// detectCapabilities detects available features on the system
func (c *LinuxMetricsCollector) detectCapabilities() services.CollectorCapabilities {
	caps := detectSystemCapabilities()
	return caps
}