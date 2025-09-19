//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// DarwinMetricsCollector implements the MetricsCollector interface for macOS
type DarwinMetricsCollector struct {
	mu sync.RWMutex

	// Cache for delta calculations
	cache *metricsCache

	// Configuration
	refreshRate time.Duration
	maxConcurrent int

	// State management
	lastCollectionTime time.Time
	isCollecting bool

	// Component parsers
	processParser *processParser
	sysctlParser  *sysctlInterface
	pressureProxy *cpuPressureProxy
	diskIOParser  *diskIOParser

	// Capability detection
	capabilities *collectorCapabilities
	sandboxMode  bool
}

// NewDarwinMetricsCollector creates a new instance of DarwinMetricsCollector
func NewDarwinMetricsCollector() (*DarwinMetricsCollector, error) {
	// Detect capabilities and sandbox restrictions
	caps := detectCapabilities()
	sandboxMode := detectSandboxMode()

	// Initialize cache
	cache := newMetricsCache()

	// Initialize parsers
	processParser := newProcessParser(sandboxMode)
	sysctlParser := newSysctlInterface()
	pressureProxy := newCPUPressureProxy()
	diskIOParser := newDiskIOParser()

	return &DarwinMetricsCollector{
		cache:          cache,
		refreshRate:    time.Second,
		maxConcurrent:  10,
		processParser:  processParser,
		sysctlParser:   sysctlParser,
		pressureProxy:  pressureProxy,
		diskIOParser:   diskIOParser,
		capabilities:   caps,
		sandboxMode:    sandboxMode,
	}, nil
}

// CollectSystemMetrics collects system-wide CPU, memory, and I/O metrics
func (c *DarwinMetricsCollector) CollectSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sysctlParser.collectSystemMetrics(ctx)
}

// CollectProcessMetrics collects metrics for a specific process
func (c *DarwinMetricsCollector) CollectProcessMetrics(ctx context.Context, pid valueobjects.ProcessID) (*entities.ProcessMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.processParser.collectProcessMetrics(ctx, pid)
}

// CollectAllProcessMetrics collects metrics for all processes
func (c *DarwinMetricsCollector) CollectAllProcessMetrics(ctx context.Context) ([]*entities.ProcessMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.processParser.collectAllProcessMetrics(ctx)
}

// CollectCPUPressure collects CPU pressure information
func (c *DarwinMetricsCollector) CollectCPUPressure(ctx context.Context) (*entities.CPUPressure, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.pressureProxy.collectCPUPressure(ctx)
}

// CollectDiskIO collects disk I/O statistics for a specific device
func (c *DarwinMetricsCollector) CollectDiskIO(ctx context.Context, device string) (*entities.DiskIO, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.diskIOParser.collectDiskIO(ctx, device)
}

// CollectAllDiskIO collects disk I/O statistics for all devices
func (c *DarwinMetricsCollector) CollectAllDiskIO(ctx context.Context) ([]*entities.DiskIO, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.diskIOParser.collectAllDiskIO(ctx)
}

// StartContinuousCollection starts continuous metric collection
func (c *DarwinMetricsCollector) StartContinuousCollection(ctx context.Context, options services.CollectionOptions, callback services.CollectionCallback) error {
	c.mu.Lock()
	if c.isCollecting {
		c.mu.Unlock()
		return ErrAlreadyCollecting
	}
	c.isCollecting = true
	c.refreshRate = interval
	c.mu.Unlock()

	go c.continuousCollectionLoop(ctx, callback)
	return nil
}

// StopContinuousCollection stops continuous metric collection
func (c *DarwinMetricsCollector) StopContinuousCollection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isCollecting {
		return ErrNotCollecting
	}

	c.isCollecting = false
	return nil
}

// CollectAll collects all available metrics based on options
func (c *DarwinMetricsCollector) CollectAll(ctx context.Context, options services.CollectionOptions) (*services.CollectionResult, error) {
	c.mu.Lock()
	c.lastCollectionTime = time.Now()
	c.mu.Unlock()

	var (
		systemMetrics   *entities.SystemMetrics
		processMetrics  []*entities.ProcessMetrics
		diskIOMetrics   []*entities.DiskIO
		networkIOMetrics []*entities.NetworkIO
		cpuPressure     *entities.CPUPressure
		memoryPressure  *entities.MemoryPressure
		ioPressure      *entities.IOPressure
		errors          []error
		wg              sync.WaitGroup
	)

	// Collect system metrics
	if options.IncludeSystem {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			systemMetrics, err = c.CollectSystemMetrics(ctx)
			if err != nil {
				c.errorMu.Lock()
				errors = append(errors, fmt.Errorf("system metrics: %w", err))
				c.errorMu.Unlock()
			}
		}()
	}

	// Collect process metrics
	if options.IncludeProcesses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if options.ProcessFilter != nil && options.ProcessFilter.PID != nil {
				process, err := c.CollectProcessMetrics(ctx, *options.ProcessFilter.PID)
				if err == nil {
					processMetrics = []*entities.ProcessMetrics{process}
				} else {
					errors = append(errors, fmt.Errorf("process metrics: %w", err))
				}
			} else {
				processMetrics, err = c.CollectAllProcessMetrics(ctx)
				if err != nil {
					errors = append(errors, fmt.Errorf("all process metrics: %w", err))
				}
			}
		}()
	}

	// Collect disk I/O metrics
	if options.IncludeDiskIO {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			diskIOMetrics, err = c.CollectAllDiskIO(ctx)
			if err != nil {
				errors = append(errors, fmt.Errorf("disk I/O metrics: %w", err))
			}
		}()
	}

	// Collect network I/O metrics
	if options.IncludeNetworkIO {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			networkIOMetrics, err = c.CollectAllNetworkIO(ctx)
			if err != nil {
				errors = append(errors, fmt.Errorf("network I/O metrics: %w", err))
			}
		}()
	}

	// Collect pressure metrics
	if options.IncludePressure {
		wg.Add(3)
		go func() {
			defer wg.Done()
			var err error
			cpuPressure, err = c.CollectCPUPressure(ctx)
			if err != nil {
				errors = append(errors, fmt.Errorf("CPU pressure: %w", err))
			}
		}()
		go func() {
			defer wg.Done()
			var err error
			memoryPressure, err = c.CollectMemoryPressure(ctx)
			if err != nil {
				errors = append(errors, fmt.Errorf("memory pressure: %w", err))
			}
		}()
		go func() {
			defer wg.Done()
			var err error
			ioPressure, err = c.CollectIOPressure(ctx)
			if err != nil {
				errors = append(errors, fmt.Errorf("I/O pressure: %w", err))
			}
		}()
	}

	wg.Wait()

	// Create timestamp
	timestamp, err := valueobjects.NewTimestamp(time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create timestamp: %w", err)
	}

	// Aggregate errors
	var aggregatedError error
	if len(errors) > 0 {
		aggregatedError = fmt.Errorf("collection completed with %d errors: %v", len(errors), errors)
	}

	return &services.CollectionResult{
		Timestamp:        timestamp,
		SystemMetrics:    systemMetrics,
		ProcessMetrics:   processMetrics,
		DiskIOMetrics:    diskIOMetrics,
		NetworkIOMetrics: networkIOMetrics,
		CPUPressure:      cpuPressure,
		MemoryPressure:   memoryPressure,
		IOPressure:       ioPressure,
		Errors:           aggregatedError,
	}, nil
}

// GetCapabilities returns the collector's capabilities
func (c *DarwinMetricsCollector) GetCapabilities() services.CollectorCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return services.CollectorCapabilities{
		SupportsProcessIO:       c.capabilities.hasProcessIO,
		SupportsCgroups:         false, // macOS doesn't have cgroups
		SupportsContainers:      c.capabilities.hasDocker,
		SupportsPressureMetrics: false, // No native PSI on macOS
		SupportsNetworkIO:       true,
		SupportsDiskIO:          true,
		MaxConcurrentCollections: c.maxConcurrent,
	}
}

// continuousCollectionLoop runs the continuous collection in a goroutine
func (c *DarwinMetricsCollector) continuousCollectionLoop(ctx context.Context, callback services.CollectionCallback) {
	ticker := time.NewTicker(c.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			if !c.isCollecting {
				c.mu.RUnlock()
				return
			}
			c.mu.RUnlock()

			// Collect all metrics
			result, err := c.CollectAll(ctx, services.CollectionOptions{
				IncludeSystem:    true,
				IncludeProcesses: true,
				IncludeDiskIO:    true,
				IncludeNetworkIO: true,
				IncludePressure:  true,
			})

			// Call the callback with results
			callback(result, err)
		}
	}
}

// sortAndLimitProcesses sorts processes by metric and returns top N
func sortAndLimitProcesses(processes []*entities.ProcessMetrics, metric services.MetricType, count int) []*entities.ProcessMetrics {
	// Implementation would sort processes by the specified metric
	// and return the top N processes
	// For now, return as-is with limit
	if len(processes) > count {
		return processes[:count]
	}
	return processes
}