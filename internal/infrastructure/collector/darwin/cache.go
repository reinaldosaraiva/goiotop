//go:build darwin

package darwin

import (
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// metricsCache provides thread-safe caching for metric snapshots
type metricsCache struct {
	mu sync.RWMutex

	// Process metrics cache
	processes map[int32]*processMetricsSnapshot

	// Disk I/O cache
	diskIO map[string]*diskIOSnapshot

	// Network I/O cache
	networkIO map[string]*networkIOSnapshot

	// System metrics cache
	system *systemMetricsSnapshot

	// Cache expiration settings
	maxAge time.Duration
}

// processMetricsSnapshot stores a snapshot of process metrics
type processMetricsSnapshot struct {
	pid              int32
	bytesRead        int64
	bytesWritten     int64
	readOps          int64
	writeOps         int64
	cpuTime          int64
	networkBytesRecv uint64
	networkBytesSent uint64
	timestamp        time.Time
}

// diskIOSnapshot stores a snapshot of disk I/O metrics
type diskIOSnapshot struct {
	device       string
	readBytes    uint64
	writeBytes   uint64
	readOps      uint64
	writeOps     uint64
	ioTime       uint64
	timestamp    time.Time
}

// networkIOSnapshot stores a snapshot of network I/O metrics
type networkIOSnapshot struct {
	interface_   string
	bytesRecv    uint64
	bytesSent    uint64
	packetsRecv  uint64
	packetsSent  uint64
	timestamp    time.Time
}

// systemMetricsSnapshot stores a snapshot of system metrics
type systemMetricsSnapshot struct {
	cpuTime      uint64
	diskReadBytes  uint64
	diskWriteBytes uint64
	timestamp    time.Time
}

// newMetricsCache creates a new metrics cache
func newMetricsCache() *metricsCache {
	return &metricsCache{
		processes: make(map[int32]*processMetricsSnapshot),
		diskIO:    make(map[string]*diskIOSnapshot),
		networkIO: make(map[string]*networkIOSnapshot),
		maxAge:    5 * time.Minute,
	}
}

// UpdateProcessMetrics updates the cache with new process metrics
func (c *metricsCache) UpdateProcessMetrics(pid int32, snapshot *processMetricsSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processes[pid] = snapshot
}

// GetProcessMetrics retrieves cached process metrics
func (c *metricsCache) GetProcessMetrics(pid int32) (*processMetricsSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot, exists := c.processes[pid]
	if !exists {
		return nil, false
	}

	// Check if cache entry is expired
	if time.Since(snapshot.timestamp) > c.maxAge {
		return nil, false
	}

	return snapshot, true
}

// CalculateProcessRates calculates rates from cached and current process metrics
func (c *metricsCache) CalculateProcessRates(pid int32, current *processMetricsSnapshot) (readRate, writeRate float64, readOps, writeOps int64) {
	c.mu.RLock()
	cached, exists := c.processes[pid]
	c.mu.RUnlock()

	if !exists || cached == nil {
		// No cached data, store current and return zeros
		c.UpdateProcessMetrics(pid, current)
		return 0, 0, 0, 0
	}

	// Calculate time delta
	timeDelta := current.timestamp.Sub(cached.timestamp).Seconds()
	if timeDelta <= 0 {
		// No time has passed or negative time (clock adjustment)
		return 0, 0, 0, 0
	}

	// Calculate byte rates
	bytesReadDelta := current.bytesRead - cached.bytesRead
	bytesWrittenDelta := current.bytesWritten - cached.bytesWritten

	if bytesReadDelta < 0 {
		bytesReadDelta = 0 // Counter may have reset
	}
	if bytesWrittenDelta < 0 {
		bytesWrittenDelta = 0
	}

	readRate = float64(bytesReadDelta) / timeDelta
	writeRate = float64(bytesWrittenDelta) / timeDelta

	// Calculate operation counts
	readOps = current.readOps - cached.readOps
	writeOps = current.writeOps - cached.writeOps

	if readOps < 0 {
		readOps = 0
	}
	if writeOps < 0 {
		writeOps = 0
	}

	// Update cache with current snapshot
	c.UpdateProcessMetrics(pid, current)

	return readRate, writeRate, readOps, writeOps
}

// UpdateDiskIO updates the cache with new disk I/O metrics
func (c *metricsCache) UpdateDiskIO(device string, snapshot *diskIOSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.diskIO[device] = snapshot
}

// GetDiskIO retrieves cached disk I/O metrics
func (c *metricsCache) GetDiskIO(device string) (*diskIOSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot, exists := c.diskIO[device]
	if !exists {
		return nil, false
	}

	// Check if cache entry is expired
	if time.Since(snapshot.timestamp) > c.maxAge {
		return nil, false
	}

	return snapshot, true
}

// CalculateDiskIORates calculates rates from cached and current disk I/O metrics
func (c *metricsCache) CalculateDiskIORates(device string, current *diskIOSnapshot) (readRate, writeRate, readIOPS, writeIOPS float64) {
	c.mu.RLock()
	cached, exists := c.diskIO[device]
	c.mu.RUnlock()

	if !exists || cached == nil {
		// No cached data, store current and return zeros
		c.UpdateDiskIO(device, current)
		return 0, 0, 0, 0
	}

	// Calculate time delta
	timeDelta := current.timestamp.Sub(cached.timestamp).Seconds()
	if timeDelta <= 0 {
		return 0, 0, 0, 0
	}

	// Calculate rates
	readBytesDelta := float64(current.readBytes - cached.readBytes)
	writeBytesDelta := float64(current.writeBytes - cached.writeBytes)
	readOpsDelta := float64(current.readOps - cached.readOps)
	writeOpsDelta := float64(current.writeOps - cached.writeOps)

	// Handle counter resets
	if readBytesDelta < 0 {
		readBytesDelta = 0
	}
	if writeBytesDelta < 0 {
		writeBytesDelta = 0
	}
	if readOpsDelta < 0 {
		readOpsDelta = 0
	}
	if writeOpsDelta < 0 {
		writeOpsDelta = 0
	}

	readRate = readBytesDelta / timeDelta
	writeRate = writeBytesDelta / timeDelta
	readIOPS = readOpsDelta / timeDelta
	writeIOPS = writeOpsDelta / timeDelta

	// Update cache
	c.UpdateDiskIO(device, current)

	return readRate, writeRate, readIOPS, writeIOPS
}

// CleanupProcessCache removes expired or inactive process entries
func (c *metricsCache) CleanupProcessCache(activePIDs []int32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create map of active PIDs
	active := make(map[int32]bool)
	for _, pid := range activePIDs {
		active[pid] = true
	}

	// Remove inactive or expired entries
	cutoff := time.Now().Add(-c.maxAge)
	for pid, snapshot := range c.processes {
		if !active[pid] || snapshot.timestamp.Before(cutoff) {
			delete(c.processes, pid)
		}
	}
}

// CleanupDiskIOCache removes expired disk I/O entries
func (c *metricsCache) CleanupDiskIOCache(activeDevices []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create map of active devices
	active := make(map[string]bool)
	for _, device := range activeDevices {
		active[device] = true
	}

	// Remove inactive or expired entries
	cutoff := time.Now().Add(-c.maxAge)
	for device, snapshot := range c.diskIO {
		if !active[device] || snapshot.timestamp.Before(cutoff) {
			delete(c.diskIO, device)
		}
	}
}

// Clear removes all cached data
func (c *metricsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processes = make(map[int32]*processMetricsSnapshot)
	c.diskIO = make(map[string]*diskIOSnapshot)
	c.networkIO = make(map[string]*networkIOSnapshot)
	c.system = nil
}

// CreateByteRateFromDelta creates a ByteRate value object from a delta calculation
func CreateByteRateFromDelta(bytesOld, bytesNew uint64, timeOld, timeNew time.Time) (valueobjects.ByteRate, error) {
	// Handle timestamp issues
	if !timeNew.After(timeOld) {
		return valueobjects.NewByteRate(0)
	}

	timeDelta := timeNew.Sub(timeOld).Seconds()
	if timeDelta <= 0 {
		return valueobjects.NewByteRate(0)
	}

	// Calculate delta, handling counter resets
	var bytesDelta uint64
	if bytesNew >= bytesOld {
		bytesDelta = bytesNew - bytesOld
	} else {
		// Counter reset, use new value as delta
		bytesDelta = bytesNew
	}

	rate := float64(bytesDelta) / timeDelta
	return valueobjects.NewByteRate(rate)
}

// CreateTimestamp creates a Timestamp value object for the current time
func CreateTimestamp() (*valueobjects.Timestamp, error) {
	return valueobjects.NewTimestamp(time.Now())
}

// cleanup performs a full cleanup of the metrics cache
func (c *metricsCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear all cache data
	c.processes = make(map[int32]*processMetricsSnapshot)
	c.diskIO = make(map[string]*diskIOSnapshot)
	c.networkIO = make(map[string]*networkIOSnapshot)
	c.system = nil
}