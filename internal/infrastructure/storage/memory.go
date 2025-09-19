package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// MemoryRepository implements the MetricsRepository interface with in-memory storage
type MemoryRepository struct {
	mu sync.RWMutex

	// Ring buffers for each metric type
	systemMetrics  *RingBuffer[*entities.SystemMetrics]
	processMetrics *RingBuffer[*entities.ProcessMetrics]
	cpuPressure    *RingBuffer[*entities.CPUPressure]
	diskIO         *RingBuffer[*entities.DiskIO]

	// Indexes for efficient querying
	processIndex map[valueobjects.PID]*RingBuffer[*entities.ProcessMetrics]
	diskIndex    map[string]*RingBuffer[*entities.DiskIO]


	// Statistics
	totalWrites atomic.Int64
	totalReads  atomic.Int64

	// Configuration
	config MemoryRepositoryConfig
}

// MemoryRepositoryConfig holds configuration for the memory repository
type MemoryRepositoryConfig struct {
	SystemMetricsCapacity  int
	ProcessMetricsCapacity int
	CPUPressureCapacity    int
	DiskIOCapacity         int
	RetentionPeriod        time.Duration
	CleanupInterval        time.Duration
}

// DefaultMemoryRepositoryConfig returns default configuration
func DefaultMemoryRepositoryConfig() MemoryRepositoryConfig {
	return MemoryRepositoryConfig{
		SystemMetricsCapacity:  10000,
		ProcessMetricsCapacity: 50000,
		CPUPressureCapacity:    10000,
		DiskIOCapacity:         10000,
		RetentionPeriod:        24 * time.Hour,
		CleanupInterval:        1 * time.Hour,
	}
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository(config MemoryRepositoryConfig) *MemoryRepository {
	repo := &MemoryRepository{
		systemMetrics:  NewRingBuffer[*entities.SystemMetrics](config.SystemMetricsCapacity),
		processMetrics: NewRingBuffer[*entities.ProcessMetrics](config.ProcessMetricsCapacity),
		cpuPressure:    NewRingBuffer[*entities.CPUPressure](config.CPUPressureCapacity),
		diskIO:         NewRingBuffer[*entities.DiskIO](config.DiskIOCapacity),
		processIndex:   make(map[valueobjects.PID]*RingBuffer[*entities.ProcessMetrics]),
		diskIndex:      make(map[string]*RingBuffer[*entities.DiskIO]),
		config:         config,
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go repo.startCleanup()
	}

	return repo
}

// SaveSystemMetrics saves system metrics to the repository
func (r *MemoryRepository) SaveSystemMetrics(ctx context.Context, metrics *entities.SystemMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	r.systemMetrics.PushWithTime(metrics, metrics.Timestamp.Time())
	r.totalWrites.Add(1)
	return nil
}

// SaveProcessMetrics saves process metrics to the repository
func (r *MemoryRepository) SaveProcessMetrics(ctx context.Context, metrics *entities.ProcessMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Add to main buffer
	r.processMetrics.PushWithTime(metrics, metrics.Timestamp.Time())

	// Update process index
	if _, exists := r.processIndex[metrics.PID]; !exists {
		r.processIndex[metrics.PID] = NewRingBuffer[*entities.ProcessMetrics](1000)
	}
	r.processIndex[metrics.PID].PushWithTime(metrics, metrics.Timestamp.Time())

	r.totalWrites.Add(1)
	return nil
}

// SaveCPUPressure saves CPU pressure metrics to the repository
func (r *MemoryRepository) SaveCPUPressure(ctx context.Context, pressure *entities.CPUPressure) error {
	if pressure == nil {
		return fmt.Errorf("pressure cannot be nil")
	}

	r.cpuPressure.PushWithTime(pressure, pressure.Timestamp.Time())
	r.totalWrites.Add(1)
	return nil
}

// SaveDiskIO saves disk I/O metrics to the repository
func (r *MemoryRepository) SaveDiskIO(ctx context.Context, diskIO *entities.DiskIO) error {
	if diskIO == nil {
		return fmt.Errorf("diskIO cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Add to main buffer
	r.diskIO.PushWithTime(diskIO, diskIO.Timestamp.Time())

	// Update disk index
	device := diskIO.Device.String()
	if _, exists := r.diskIndex[device]; !exists {
		r.diskIndex[device] = NewRingBuffer[*entities.DiskIO](1000)
	}
	r.diskIndex[device].PushWithTime(diskIO, diskIO.Timestamp.Time())

	r.totalWrites.Add(1)
	return nil
}

// QuerySystemMetrics queries system metrics from the repository
func (r *MemoryRepository) QuerySystemMetrics(ctx context.Context, filter repositories.MetricsFilter) ([]*entities.SystemMetrics, error) {
	items, timestamps := r.systemMetrics.GetByTimeRange(filter.StartTime, filter.EndTime)

	results := make([]*entities.SystemMetrics, 0, len(items))
	for i, item := range items {
		if filter.MatchesSystemMetrics(item) {
			// Clone to prevent external modification
			clone := *item
			clone.Timestamp = valueobjects.NewTimestamp(timestamps[i])
			results = append(results, &clone)
		}
	}

	r.totalReads.Add(int64(len(results)))
	return results, nil
}

// QueryProcessMetrics queries process metrics from the repository
func (r *MemoryRepository) QueryProcessMetrics(ctx context.Context, filter repositories.MetricsFilter) ([]*entities.ProcessMetrics, error) {
	var items []*entities.ProcessMetrics
	var timestamps []time.Time

	r.mu.RLock()
	if filter.PID != nil {
		// Query from process index if PID is specified
		if buffer, exists := r.processIndex[*filter.PID]; exists {
			i, t := buffer.GetByTimeRange(filter.StartTime, filter.EndTime)
			items = i
			timestamps = t
		}
	} else {
		// Query from main buffer
		i, t := r.processMetrics.GetByTimeRange(filter.StartTime, filter.EndTime)
		items = i
		timestamps = t
	}
	r.mu.RUnlock()

	results := make([]*entities.ProcessMetrics, 0, len(items))
	for i, item := range items {
		if filter.MatchesProcessMetrics(item) {
			// Clone to prevent external modification
			clone := *item
			clone.Timestamp = valueobjects.NewTimestamp(timestamps[i])
			results = append(results, &clone)
		}
	}

	r.totalReads.Add(int64(len(results)))
	return results, nil
}

// QueryCPUPressure queries CPU pressure metrics from the repository
func (r *MemoryRepository) QueryCPUPressure(ctx context.Context, filter repositories.MetricsFilter) ([]*entities.CPUPressure, error) {
	items, timestamps := r.cpuPressure.GetByTimeRange(filter.StartTime, filter.EndTime)

	results := make([]*entities.CPUPressure, 0, len(items))
	for i, item := range items {
		// Clone to prevent external modification
		clone := *item
		clone.Timestamp = valueobjects.NewTimestamp(timestamps[i])
		results = append(results, &clone)
	}

	r.totalReads.Add(int64(len(results)))
	return results, nil
}

// QueryDiskIO queries disk I/O metrics from the repository
func (r *MemoryRepository) QueryDiskIO(ctx context.Context, filter repositories.MetricsFilter) ([]*entities.DiskIO, error) {
	var items []*entities.DiskIO
	var timestamps []time.Time

	r.mu.RLock()
	if filter.Device != nil {
		// Query from disk index if device is specified
		if buffer, exists := r.diskIndex[*filter.Device]; exists {
			i, t := buffer.GetByTimeRange(filter.StartTime, filter.EndTime)
			items = i
			timestamps = t
		}
	} else {
		// Query from main buffer
		i, t := r.diskIO.GetByTimeRange(filter.StartTime, filter.EndTime)
		items = i
		timestamps = t
	}
	r.mu.RUnlock()

	results := make([]*entities.DiskIO, 0, len(items))
	for i, item := range items {
		// Clone to prevent external modification
		clone := *item
		clone.Timestamp = valueobjects.NewTimestamp(timestamps[i])
		results = append(results, &clone)
	}

	r.totalReads.Add(int64(len(results)))
	return results, nil
}

// GetLatestSystemMetrics returns the most recent system metrics
func (r *MemoryRepository) GetLatestSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error) {
	items, _ := r.systemMetrics.GetLatest(1)
	if len(items) == 0 {
		return nil, fmt.Errorf("no system metrics found")
	}
	r.totalReads.Add(1)
	return items[0], nil
}

// GetLatestProcessMetrics returns the most recent process metrics for a PID
func (r *MemoryRepository) GetLatestProcessMetrics(ctx context.Context, pid valueobjects.PID) (*entities.ProcessMetrics, error) {
	r.mu.RLock()
	buffer, exists := r.processIndex[pid]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no process metrics found for PID %d", pid.Value())
	}

	items, _ := buffer.GetLatest(1)
	if len(items) == 0 {
		return nil, fmt.Errorf("no process metrics found for PID %d", pid.Value())
	}

	r.totalReads.Add(1)
	return items[0], nil
}

// GetTopProcesses returns the top N processes by a specified metric
func (r *MemoryRepository) GetTopProcesses(ctx context.Context, metric string, limit int) ([]*entities.ProcessMetrics, error) {
	// Get latest metrics for all processes
	items, _ := r.processMetrics.GetLatest(r.processMetrics.Size())

	// Sort by the specified metric
	sorted := sortProcessesByMetric(items, metric)

	// Return top N
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	r.totalReads.Add(int64(len(sorted)))
	return sorted, nil
}

// AggregateSystemMetrics aggregates system metrics over a time period
func (r *MemoryRepository) AggregateSystemMetrics(ctx context.Context, start, end time.Time, aggType repositories.AggregationType) (*entities.SystemMetrics, error) {
	filter := repositories.MetricsFilter{
		StartTime: start,
		EndTime:   end,
	}

	metrics, err := r.QuerySystemMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found in the specified time range")
	}

	return aggregateSystemMetrics(metrics, aggType), nil
}

// AggregateProcessMetrics aggregates process metrics over a time period
func (r *MemoryRepository) AggregateProcessMetrics(ctx context.Context, pid valueobjects.PID, start, end time.Time, aggType repositories.AggregationType) (*entities.ProcessMetrics, error) {
	filter := repositories.MetricsFilter{
		StartTime: start,
		EndTime:   end,
		PID:       &pid,
	}

	metrics, err := r.QueryProcessMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found for PID %d in the specified time range", pid.Value())
	}

	return aggregateProcessMetrics(metrics, aggType), nil
}

// DeleteOldMetrics removes metrics older than the specified time
func (r *MemoryRepository) DeleteOldMetrics(ctx context.Context, before time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	evicted := 0
	evicted += r.systemMetrics.EvictOlderThan(before)
	evicted += r.processMetrics.EvictOlderThan(before)
	evicted += r.cpuPressure.EvictOlderThan(before)
	evicted += r.diskIO.EvictOlderThan(before)

	// Clean up process index
	for pid, buffer := range r.processIndex {
		buffer.EvictOlderThan(before)
		if buffer.Size() == 0 {
			delete(r.processIndex, pid)
		}
	}

	// Clean up disk index
	for device, buffer := range r.diskIndex {
		buffer.EvictOlderThan(before)
		if buffer.Size() == 0 {
			delete(r.diskIndex, device)
		}
	}

	return nil
}

// BeginTransaction starts a new transaction (not implemented)
func (r *MemoryRepository) BeginTransaction(ctx context.Context) (repositories.Transaction, error) {
	return nil, fmt.Errorf("transactions not supported in memory repository")
}

// GetStatistics returns repository statistics
func (r *MemoryRepository) GetStatistics(ctx context.Context) (repositories.RepositoryStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := repositories.RepositoryStats{
		TotalWrites:    r.totalWrites.Load(),
		TotalReads:     r.totalReads.Load(),
		SystemMetrics:  int64(r.systemMetrics.Size()),
		ProcessMetrics: int64(r.processMetrics.Size()),
		CPUPressure:    int64(r.cpuPressure.Size()),
		DiskIO:         int64(r.diskIO.Size()),
		MemoryUsage:    r.estimateMemoryUsage(),
	}

	return stats, nil
}

// HealthCheck performs a health check on the repository
func (r *MemoryRepository) HealthCheck(ctx context.Context) error {
	// Check if buffers are accessible
	if r.systemMetrics == nil || r.processMetrics == nil {
		return fmt.Errorf("repository not properly initialized")
	}

	// Check memory usage
	memUsage := r.estimateMemoryUsage()
	if memUsage > 1024*1024*1024 { // 1GB limit
		return fmt.Errorf("memory usage too high: %d bytes", memUsage)
	}

	return nil
}

// Initialize initializes the repository
func (r *MemoryRepository) Initialize(ctx context.Context) error {
	// Repository is already initialized
	return nil
}

// Close closes the repository and releases resources (parameterless version)
func (r *MemoryRepository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear all buffers
	r.systemMetrics.Clear()
	r.processMetrics.Clear()
	r.cpuPressure.Clear()
	r.diskIO.Clear()

	// Clear indexes
	r.processIndex = make(map[valueobjects.PID]*RingBuffer[*entities.ProcessMetrics])
	r.diskIndex = make(map[string]*RingBuffer[*entities.DiskIO])

	return nil
}

// CloseWithContext closes the repository with context
func (r *MemoryRepository) CloseWithContext(ctx context.Context) error {
	return r.Close()
}

// Ping checks if the repository is responsive
func (r *MemoryRepository) Ping(ctx context.Context) error {
	return r.HealthCheck(ctx)
}

// PurgeOldMetrics deletes metrics older than the specified time
func (r *MemoryRepository) PurgeOldMetrics(ctx context.Context, before time.Time) error {
	return r.DeleteOldMetrics(ctx, before)
}

// startCleanup runs periodic cleanup of old data
func (r *MemoryRepository) startCleanup() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-r.config.RetentionPeriod)
		r.DeleteOldMetrics(context.Background(), cutoff)
	}
}

// estimateMemoryUsage estimates the total memory usage
func (r *MemoryRepository) estimateMemoryUsage() int64 {
	total := r.systemMetrics.MemoryUsage()
	total += r.processMetrics.MemoryUsage()
	total += r.cpuPressure.MemoryUsage()
	total += r.diskIO.MemoryUsage()

	// Add index memory estimation
	for _, buffer := range r.processIndex {
		total += buffer.MemoryUsage()
	}
	for _, buffer := range r.diskIndex {
		total += buffer.MemoryUsage()
	}

	return total
}

// Helper functions for aggregation and sorting
func sortProcessesByMetric(processes []*entities.ProcessMetrics, metric string) []*entities.ProcessMetrics {
	// Create a copy to avoid modifying the original
	sorted := make([]*entities.ProcessMetrics, len(processes))
	copy(sorted, processes)

	switch metric {
	case "cpu":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].CPUPercent.Value() > sorted[j].CPUPercent.Value()
		})
	case "memory":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].MemoryRSS.Value() > sorted[j].MemoryRSS.Value()
		})
	case "io_read":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].IOCounters == nil || sorted[j].IOCounters == nil {
				return false
			}
			return sorted[i].IOCounters.ReadBytes() > sorted[j].IOCounters.ReadBytes()
		})
	case "io_write":
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].IOCounters == nil || sorted[j].IOCounters == nil {
				return false
			}
			return sorted[i].IOCounters.WriteBytes() > sorted[j].IOCounters.WriteBytes()
		})
	case "threads":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].NumThreads.Value() > sorted[j].NumThreads.Value()
		})
	default:
		// Default to sorting by PID
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].PID.Value() < sorted[j].PID.Value()
		})
	}

	return sorted
}

func aggregateSystemMetrics(metrics []*entities.SystemMetrics, aggType repositories.AggregationType) *entities.SystemMetrics {
	if len(metrics) == 0 {
		return nil
	}

	// Create aggregated result
	result := &entities.SystemMetrics{
		Timestamp: metrics[0].Timestamp, // Use first timestamp
	}

	switch aggType {
	case repositories.AggregationTypeAverage:
		var totalCPU, totalMemUsed, totalMemAvail float64
		var totalLoadAvg1, totalLoadAvg5, totalLoadAvg15 float64
		var totalSwapUsed, totalSwapFree float64

		for _, m := range metrics {
			totalCPU += m.CPUUsage.Value()
			totalMemUsed += float64(m.MemoryUsed.Value())
			totalMemAvail += float64(m.MemoryAvailable.Value())
			totalLoadAvg1 += m.LoadAverage1.Value()
			totalLoadAvg5 += m.LoadAverage5.Value()
			totalLoadAvg15 += m.LoadAverage15.Value()
			totalSwapUsed += float64(m.SwapUsed.Value())
			totalSwapFree += float64(m.SwapFree.Value())
		}

		count := float64(len(metrics))
		result.CPUUsage = valueobjects.NewCPUUsage(totalCPU / count)
		result.MemoryUsed = valueobjects.NewMemorySize(uint64(totalMemUsed / count))
		result.MemoryAvailable = valueobjects.NewMemorySize(uint64(totalMemAvail / count))
		result.LoadAverage1 = valueobjects.NewLoadAverage(totalLoadAvg1 / count)
		result.LoadAverage5 = valueobjects.NewLoadAverage(totalLoadAvg5 / count)
		result.LoadAverage15 = valueobjects.NewLoadAverage(totalLoadAvg15 / count)
		result.SwapUsed = valueobjects.NewMemorySize(uint64(totalSwapUsed / count))
		result.SwapFree = valueobjects.NewMemorySize(uint64(totalSwapFree / count))

	case repositories.AggregationTypeMin:
		result.CPUUsage = metrics[0].CPUUsage
		result.MemoryUsed = metrics[0].MemoryUsed
		result.MemoryAvailable = metrics[0].MemoryAvailable
		result.LoadAverage1 = metrics[0].LoadAverage1
		result.LoadAverage5 = metrics[0].LoadAverage5
		result.LoadAverage15 = metrics[0].LoadAverage15
		result.SwapUsed = metrics[0].SwapUsed
		result.SwapFree = metrics[0].SwapFree

		for _, m := range metrics[1:] {
			if m.CPUUsage.Value() < result.CPUUsage.Value() {
				result.CPUUsage = m.CPUUsage
			}
			if m.MemoryUsed.Value() < result.MemoryUsed.Value() {
				result.MemoryUsed = m.MemoryUsed
			}
			if m.MemoryAvailable.Value() < result.MemoryAvailable.Value() {
				result.MemoryAvailable = m.MemoryAvailable
			}
			if m.LoadAverage1.Value() < result.LoadAverage1.Value() {
				result.LoadAverage1 = m.LoadAverage1
			}
			if m.LoadAverage5.Value() < result.LoadAverage5.Value() {
				result.LoadAverage5 = m.LoadAverage5
			}
			if m.LoadAverage15.Value() < result.LoadAverage15.Value() {
				result.LoadAverage15 = m.LoadAverage15
			}
			if m.SwapUsed.Value() < result.SwapUsed.Value() {
				result.SwapUsed = m.SwapUsed
			}
			if m.SwapFree.Value() < result.SwapFree.Value() {
				result.SwapFree = m.SwapFree
			}
		}

	case repositories.AggregationTypeMax:
		result.CPUUsage = metrics[0].CPUUsage
		result.MemoryUsed = metrics[0].MemoryUsed
		result.MemoryAvailable = metrics[0].MemoryAvailable
		result.LoadAverage1 = metrics[0].LoadAverage1
		result.LoadAverage5 = metrics[0].LoadAverage5
		result.LoadAverage15 = metrics[0].LoadAverage15
		result.SwapUsed = metrics[0].SwapUsed
		result.SwapFree = metrics[0].SwapFree

		for _, m := range metrics[1:] {
			if m.CPUUsage.Value() > result.CPUUsage.Value() {
				result.CPUUsage = m.CPUUsage
			}
			if m.MemoryUsed.Value() > result.MemoryUsed.Value() {
				result.MemoryUsed = m.MemoryUsed
			}
			if m.MemoryAvailable.Value() > result.MemoryAvailable.Value() {
				result.MemoryAvailable = m.MemoryAvailable
			}
			if m.LoadAverage1.Value() > result.LoadAverage1.Value() {
				result.LoadAverage1 = m.LoadAverage1
			}
			if m.LoadAverage5.Value() > result.LoadAverage5.Value() {
				result.LoadAverage5 = m.LoadAverage5
			}
			if m.LoadAverage15.Value() > result.LoadAverage15.Value() {
				result.LoadAverage15 = m.LoadAverage15
			}
			if m.SwapUsed.Value() > result.SwapUsed.Value() {
				result.SwapUsed = m.SwapUsed
			}
			if m.SwapFree.Value() > result.SwapFree.Value() {
				result.SwapFree = m.SwapFree
			}
		}

	case repositories.AggregationTypeSum:
		var totalCPU float64
		var totalMemUsed, totalMemAvail uint64
		var totalLoadAvg1, totalLoadAvg5, totalLoadAvg15 float64
		var totalSwapUsed, totalSwapFree uint64

		for _, m := range metrics {
			totalCPU += m.CPUUsage.Value()
			totalMemUsed += m.MemoryUsed.Value()
			totalMemAvail += m.MemoryAvailable.Value()
			totalLoadAvg1 += m.LoadAverage1.Value()
			totalLoadAvg5 += m.LoadAverage5.Value()
			totalLoadAvg15 += m.LoadAverage15.Value()
			totalSwapUsed += m.SwapUsed.Value()
			totalSwapFree += m.SwapFree.Value()
		}

		result.CPUUsage = valueobjects.NewCPUUsage(totalCPU)
		result.MemoryUsed = valueobjects.NewMemorySize(totalMemUsed)
		result.MemoryAvailable = valueobjects.NewMemorySize(totalMemAvail)
		result.LoadAverage1 = valueobjects.NewLoadAverage(totalLoadAvg1)
		result.LoadAverage5 = valueobjects.NewLoadAverage(totalLoadAvg5)
		result.LoadAverage15 = valueobjects.NewLoadAverage(totalLoadAvg15)
		result.SwapUsed = valueobjects.NewMemorySize(totalSwapUsed)
		result.SwapFree = valueobjects.NewMemorySize(totalSwapFree)

	default:
		// Default to returning the first metric
		return metrics[0]
	}

	// Copy other fields from the first metric
	if len(metrics) > 0 {
		result.ProcessCount = metrics[0].ProcessCount
		result.RunningProcesses = metrics[0].RunningProcesses
		result.ThreadCount = metrics[0].ThreadCount
		result.KernelThreads = metrics[0].KernelThreads
		result.NetworkConnections = metrics[0].NetworkConnections
		result.Uptime = metrics[0].Uptime
		result.BootTime = metrics[0].BootTime
		result.DiskPartitions = metrics[0].DiskPartitions
	}

	return result
}

func aggregateProcessMetrics(metrics []*entities.ProcessMetrics, aggType repositories.AggregationType) *entities.ProcessMetrics {
	if len(metrics) == 0 {
		return nil
	}

	// Create aggregated result using first metric as template
	result := &entities.ProcessMetrics{
		PID:       metrics[0].PID,
		Name:      metrics[0].Name,
		Username:  metrics[0].Username,
		State:     metrics[0].State,
		Timestamp: metrics[0].Timestamp,
	}

	switch aggType {
	case repositories.AggregationTypeAverage:
		var totalCPU, totalMemRSS, totalMemVMS, totalMemShared float64
		var totalThreads, totalFiles int32
		var totalReadBytes, totalWriteBytes uint64

		for _, m := range metrics {
			totalCPU += m.CPUPercent.Value()
			totalMemRSS += float64(m.MemoryRSS.Value())
			totalMemVMS += float64(m.MemoryVMS.Value())
			totalMemShared += float64(m.MemoryShared.Value())
			totalThreads += m.NumThreads.Value()
			totalFiles += m.OpenFiles.Value()
			if m.IOCounters != nil {
				totalReadBytes += m.IOCounters.ReadBytes()
				totalWriteBytes += m.IOCounters.WriteBytes()
			}
		}

		count := float64(len(metrics))
		result.CPUPercent = valueobjects.NewCPUUsage(totalCPU / count)
		result.MemoryRSS = valueobjects.NewMemorySize(uint64(totalMemRSS / count))
		result.MemoryVMS = valueobjects.NewMemorySize(uint64(totalMemVMS / count))
		result.MemoryShared = valueobjects.NewMemorySize(uint64(totalMemShared / count))
		result.NumThreads = valueobjects.NewThreadCount(int32(float64(totalThreads) / count))
		result.OpenFiles = valueobjects.NewFileCount(int32(float64(totalFiles) / count))

		if totalReadBytes > 0 || totalWriteBytes > 0 {
			result.IOCounters = &valueobjects.IOCounters{}
			result.IOCounters.SetReadBytes(uint64(float64(totalReadBytes) / count))
			result.IOCounters.SetWriteBytes(uint64(float64(totalWriteBytes) / count))
		}

	case repositories.AggregationTypeMin:
		result.CPUPercent = metrics[0].CPUPercent
		result.MemoryRSS = metrics[0].MemoryRSS
		result.MemoryVMS = metrics[0].MemoryVMS
		result.MemoryShared = metrics[0].MemoryShared
		result.NumThreads = metrics[0].NumThreads
		result.OpenFiles = metrics[0].OpenFiles
		result.IOCounters = metrics[0].IOCounters

		for _, m := range metrics[1:] {
			if m.CPUPercent.Value() < result.CPUPercent.Value() {
				result.CPUPercent = m.CPUPercent
			}
			if m.MemoryRSS.Value() < result.MemoryRSS.Value() {
				result.MemoryRSS = m.MemoryRSS
			}
			if m.MemoryVMS.Value() < result.MemoryVMS.Value() {
				result.MemoryVMS = m.MemoryVMS
			}
			if m.MemoryShared.Value() < result.MemoryShared.Value() {
				result.MemoryShared = m.MemoryShared
			}
			if m.NumThreads.Value() < result.NumThreads.Value() {
				result.NumThreads = m.NumThreads
			}
			if m.OpenFiles.Value() < result.OpenFiles.Value() {
				result.OpenFiles = m.OpenFiles
			}
		}

	case repositories.AggregationTypeMax:
		result.CPUPercent = metrics[0].CPUPercent
		result.MemoryRSS = metrics[0].MemoryRSS
		result.MemoryVMS = metrics[0].MemoryVMS
		result.MemoryShared = metrics[0].MemoryShared
		result.NumThreads = metrics[0].NumThreads
		result.OpenFiles = metrics[0].OpenFiles
		result.IOCounters = metrics[0].IOCounters

		for _, m := range metrics[1:] {
			if m.CPUPercent.Value() > result.CPUPercent.Value() {
				result.CPUPercent = m.CPUPercent
			}
			if m.MemoryRSS.Value() > result.MemoryRSS.Value() {
				result.MemoryRSS = m.MemoryRSS
			}
			if m.MemoryVMS.Value() > result.MemoryVMS.Value() {
				result.MemoryVMS = m.MemoryVMS
			}
			if m.MemoryShared.Value() > result.MemoryShared.Value() {
				result.MemoryShared = m.MemoryShared
			}
			if m.NumThreads.Value() > result.NumThreads.Value() {
				result.NumThreads = m.NumThreads
			}
			if m.OpenFiles.Value() > result.OpenFiles.Value() {
				result.OpenFiles = m.OpenFiles
			}
		}

	default:
		// Default to returning the first metric
		return metrics[0]
	}

	// Copy other fields from the first metric
	if len(metrics) > 0 {
		result.Command = metrics[0].Command
		result.Nice = metrics[0].Nice
		result.Priority = metrics[0].Priority
		// Use the first metric's timestamp as representative
		result.CreateTime = metrics[0].CreateTime
	}

	return result
}