//go:build linux

package linux

import (
	"sync"
	"time"
)

// cacheEntry represents a cached metric value
type cacheEntry struct {
	value     uint64
	timestamp time.Time
}

// processIOCache caches process I/O metrics for rate calculation
type processIOCache struct {
	readBytes  uint64
	writeBytes uint64
	timestamp  time.Time
}

// diskIOCache caches disk I/O metrics for rate calculation
type diskIOCache struct {
	readBytes       uint64
	writeBytes      uint64
	readsCompleted  uint64
	writesCompleted uint64
	ioTime          uint64
	timeInQueue     uint64
	timestamp       time.Time
}

// metricsCache provides caching and delta calculation for metrics
type metricsCache struct {
	mu sync.RWMutex

	// Process I/O caches (key: PID)
	processIO map[int]*processIOCache

	// Disk I/O caches (key: device name)
	diskIO map[string]*diskIOCache

	// System-wide I/O cache
	systemReadBytes  uint64
	systemWriteBytes uint64
	systemTimestamp  time.Time

	// Configuration
	maxAge         time.Duration
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	latencyHistory []time.Duration
	maxHistory     int
}

// newMetricsCache creates a new metrics cache
func newMetricsCache() *metricsCache {
	c := &metricsCache{
		processIO:      make(map[int]*processIOCache),
		diskIO:         make(map[string]*diskIOCache),
		maxAge:         5 * time.Minute,
		stopCleanup:    make(chan struct{}),
		latencyHistory: make([]time.Duration, 0, 100),
		maxHistory:     100,
	}

	// Start cleanup goroutine
	c.startCleanup()

	return c
}

// calculateIORate calculates I/O rate for a process
func (c *metricsCache) calculateIORate(pid int, isRead bool, currentBytes uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	cache, exists := c.processIO[pid]
	if !exists {
		// First time seeing this process
		c.processIO[pid] = &processIOCache{
			readBytes:  currentBytes,
			writeBytes: currentBytes,
			timestamp:  time.Now(),
		}
		return 0
	}

	// Calculate time delta
	now := time.Now()
	timeDelta := now.Sub(cache.timestamp).Seconds()
	if timeDelta <= 0 {
		return 0
	}

	var previousBytes uint64
	var rate uint64

	if isRead {
		previousBytes = cache.readBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		cache.readBytes = currentBytes
	} else {
		previousBytes = cache.writeBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		cache.writeBytes = currentBytes
	}

	cache.timestamp = now
	return rate
}

// calculateDiskIORate calculates I/O rate for a disk
func (c *metricsCache) calculateDiskIORate(device string, isRead bool, currentBytes uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	cache, exists := c.diskIO[device]
	if !exists {
		// First time seeing this device
		c.diskIO[device] = &diskIOCache{
			readBytes:  currentBytes,
			writeBytes: currentBytes,
			timestamp:  time.Now(),
		}
		return 0
	}

	// Calculate time delta
	now := time.Now()
	timeDelta := now.Sub(cache.timestamp).Seconds()
	if timeDelta <= 0 {
		return 0
	}

	var previousBytes uint64
	var rate uint64

	if isRead {
		previousBytes = cache.readBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		cache.readBytes = currentBytes
	} else {
		previousBytes = cache.writeBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		cache.writeBytes = currentBytes
	}

	cache.timestamp = now
	return rate
}

// calculateDiskIOPS calculates IOPS for a disk
func (c *metricsCache) calculateDiskIOPS(device string, isRead bool, currentOps uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	cache, exists := c.diskIO[device]
	if !exists {
		// First time seeing this device
		c.diskIO[device] = &diskIOCache{
			readsCompleted:  currentOps,
			writesCompleted: currentOps,
			timestamp:       time.Now(),
		}
		return 0
	}

	// Calculate time delta
	now := time.Now()
	timeDelta := now.Sub(cache.timestamp).Seconds()
	if timeDelta <= 0 {
		return 0
	}

	var previousOps uint64
	var iops uint64

	if isRead {
		previousOps = cache.readsCompleted
		if currentOps >= previousOps {
			opsDelta := currentOps - previousOps
			iops = uint64(float64(opsDelta) / timeDelta)
		}
		cache.readsCompleted = currentOps
	} else {
		previousOps = cache.writesCompleted
		if currentOps >= previousOps {
			opsDelta := currentOps - previousOps
			iops = uint64(float64(opsDelta) / timeDelta)
		}
		cache.writesCompleted = currentOps
	}

	cache.timestamp = now
	return iops
}

// calculateSystemIORate calculates system-wide I/O rate
func (c *metricsCache) calculateSystemIORate(isRead bool, currentBytes uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if this is the first measurement
	if c.systemTimestamp.IsZero() {
		c.systemReadBytes = currentBytes
		c.systemWriteBytes = currentBytes
		c.systemTimestamp = time.Now()
		return 0
	}

	// Calculate time delta
	now := time.Now()
	timeDelta := now.Sub(c.systemTimestamp).Seconds()
	if timeDelta <= 0 {
		return 0
	}

	var previousBytes uint64
	var rate uint64

	if isRead {
		previousBytes = c.systemReadBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		c.systemReadBytes = currentBytes
	} else {
		previousBytes = c.systemWriteBytes
		if currentBytes >= previousBytes {
			byteDelta := currentBytes - previousBytes
			rate = uint64(float64(byteDelta) / timeDelta)
		}
		c.systemWriteBytes = currentBytes
	}

	c.systemTimestamp = now
	return rate
}

// reset clears all cached data
func (c *metricsCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processIO = make(map[int]*processIOCache)
	c.diskIO = make(map[string]*diskIOCache)
	c.systemReadBytes = 0
	c.systemWriteBytes = 0
	c.systemTimestamp = time.Time{}
	c.latencyHistory = make([]time.Duration, 0, c.maxHistory)
}

// cleanup removes stale entries from cache
func (c *metricsCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Clean up process I/O cache
	for pid, cache := range c.processIO {
		if now.Sub(cache.timestamp) > c.maxAge {
			delete(c.processIO, pid)
		}
	}

	// Clean up disk I/O cache
	for device, cache := range c.diskIO {
		if now.Sub(cache.timestamp) > c.maxAge {
			delete(c.diskIO, device)
		}
	}
}

// startCleanup starts the cleanup goroutine
func (c *metricsCache) startCleanup() {
	c.cleanupTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanup()
			case <-c.stopCleanup:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// stop stops the cleanup goroutine
func (c *metricsCache) stop() {
	close(c.stopCleanup)
}

// recordLatency records a collection latency
func (c *metricsCache) recordLatency(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.latencyHistory = append(c.latencyHistory, latency)
	if len(c.latencyHistory) > c.maxHistory {
		c.latencyHistory = c.latencyHistory[1:]
	}
}

// getAverageLatency returns the average collection latency
func (c *metricsCache) getAverageLatency() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.latencyHistory) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range c.latencyHistory {
		total += latency
	}

	return total / time.Duration(len(c.latencyHistory))
}

// isExpired checks if a cache entry is expired
func (c *metricsCache) isExpired(timestamp time.Time) bool {
	return time.Since(timestamp) > c.maxAge
}

// removeProcess removes a process from the cache
func (c *metricsCache) removeProcess(pid int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.processIO, pid)
}

// removeDevice removes a device from the cache
func (c *metricsCache) removeDevice(device string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.diskIO, device)
}

// getProcessCacheSize returns the number of cached processes
func (c *metricsCache) getProcessCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.processIO)
}

// getDiskCacheSize returns the number of cached disks
func (c *metricsCache) getDiskCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.diskIO)
}

// getDiskStat gets a specific disk statistic from cache
func (c *metricsCache) getDiskStat(device string, stat string) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cache, exists := c.diskIO[device]
	if !exists {
		return 0
	}

	switch stat {
	case "io_time":
		return cache.ioTime
	case "time_in_queue":
		return cache.timeInQueue
	default:
		return 0
	}
}

// updateDiskStat updates a specific disk statistic in cache
func (c *metricsCache) updateDiskStat(device string, stat string, value uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cache, exists := c.diskIO[device]
	if !exists {
		cache = &diskIOCache{
			timestamp: time.Now(),
		}
		c.diskIO[device] = cache
	}

	switch stat {
	case "io_time":
		cache.ioTime = value
	case "time_in_queue":
		cache.timeInQueue = value
	}
	cache.timestamp = time.Now()
}

// getDiskStatTimestamp gets the timestamp for disk statistics
func (c *metricsCache) getDiskStatTimestamp(device string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cache, exists := c.diskIO[device]
	if !exists {
		return time.Time{}
	}
	return cache.timestamp
}