//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// diskIOParser handles disk I/O metrics collection for macOS
type diskIOParser struct {
	mu sync.RWMutex

	// Cache for delta calculations
	cache map[string]*diskIOCache

	// Last collection time for rate calculations
	lastCollectionTime time.Time
}

// diskIOCache stores previous disk I/O metrics for delta calculations
type diskIOCache struct {
	readBytes     uint64
	writeBytes    uint64
	readCount     uint64
	writeCount    uint64
	readTime      uint64
	writeTime     uint64
	ioTime        uint64  // Add IoTime tracking
	lastTimestamp time.Time
}

// newDiskIOParser creates a new disk I/O parser
func newDiskIOParser() *diskIOParser {
	return &diskIOParser{
		cache: make(map[string]*diskIOCache),
	}
}

// collectDiskIO collects disk I/O statistics for a specific device
func (d *diskIOParser) collectDiskIO(ctx context.Context, device string) (*entities.DiskIO, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get disk I/O counters from gopsutil
	ioCounters, err := disk.IOCounters(device)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	counter, exists := ioCounters[device]
	if !exists {
		// Try without the device filter to get all and find the right one
		allCounters, err := disk.IOCounters()
		if err != nil {
			return nil, fmt.Errorf("failed to get all disk I/O counters: %w", err)
		}

		// Look for the device in various forms
		for name, c := range allCounters {
			if name == device || strings.HasSuffix(name, device) || strings.HasPrefix(device, name) {
				counter = c
				exists = true
				break
			}
		}

		if !exists {
			return nil, fmt.Errorf("device %s not found", device)
		}
	}

	// Calculate rates
	readRate, writeRate, readIOPS, writeIOPS := d.calculateRates(device, &counter)

	// Create value objects
	readRateVO, err := valueobjects.NewByteRate(readRate)
	if err != nil {
		return nil, fmt.Errorf("invalid read rate: %w", err)
	}

	writeRateVO, err := valueobjects.NewByteRate(writeRate)
	if err != nil {
		return nil, fmt.Errorf("invalid write rate: %w", err)
	}

	// Create DiskIO entity
	return entities.NewDiskIO(
		counter.Name,
		readRateVO,
		writeRateVO,
		readIOPS,
		writeIOPS,
		counter.ReadBytes,
		counter.WriteBytes,
		d.calculateUtilization(&counter),
	)
}

// collectAllDiskIO collects disk I/O statistics for all devices
func (d *diskIOParser) collectAllDiskIO(ctx context.Context) ([]*entities.DiskIO, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get all disk I/O counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var metrics []*entities.DiskIO

	for device, counter := range ioCounters {
		// Skip virtual devices and system devices
		if d.shouldSkipDevice(device) {
			continue
		}

		// Calculate rates
		readRate, writeRate, readIOPS, writeIOPS := d.calculateRates(device, &counter)

		// Create value objects
		readRateVO, err := valueobjects.NewByteRate(readRate)
		if err != nil {
			readRateVO, _ = valueobjects.NewByteRate(0)
		}

		writeRateVO, err := valueobjects.NewByteRate(writeRate)
		if err != nil {
			writeRateVO, _ = valueobjects.NewByteRate(0)
		}

		// Create DiskIO entity
		diskIO, err := entities.NewDiskIO(
			counter.Name,
			readRateVO,
			writeRateVO,
			readIOPS,
			writeIOPS,
			counter.ReadBytes,
			counter.WriteBytes,
			d.calculateUtilization(&counter),
		)

		if err == nil {
			metrics = append(metrics, diskIO)
		}
	}

	// Clean up old cache entries
	d.cleanupCache(ioCounters)

	return metrics, nil
}

// collectIOPressure estimates I/O pressure using disk statistics
func (d *diskIOParser) collectIOPressure(ctx context.Context) (*entities.IOPressure, error) {
	// Create a new ioPressureProxy and use it to collect pressure
	proxy := &ioPressureProxy{
		lastDiskStats: make(map[string]diskStats),
	}
	return proxy.collectIOPressure(ctx)
}

// getDiskDevices returns a list of disk devices
func (d *diskIOParser) getDiskDevices(ctx context.Context) ([]string, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk devices: %w", err)
	}

	var devices []string
	for device := range ioCounters {
		if !d.shouldSkipDevice(device) {
			devices = append(devices, device)
		}
	}

	return devices, nil
}

// calculateRates calculates read/write rates and IOPS from counters
func (d *diskIOParser) calculateRates(device string, counter *disk.IOCountersStat) (readRate, writeRate, readIOPS, writeIOPS float64) {
	cached, exists := d.cache[device]
	if !exists {
		// First time seeing this device
		d.cache[device] = &diskIOCache{
			readBytes:     counter.ReadBytes,
			writeBytes:    counter.WriteBytes,
			readCount:     counter.ReadCount,
			writeCount:    counter.WriteCount,
			readTime:      counter.ReadTime,
			writeTime:     counter.WriteTime,
			ioTime:        counter.IoTime,
			lastTimestamp: time.Now(),
		}
		return 0, 0, 0, 0
	}

	// Calculate time delta
	now := time.Now()
	timeDelta := now.Sub(cached.lastTimestamp).Seconds()
	if timeDelta <= 0 {
		return 0, 0, 0, 0
	}

	// Calculate rates
	readRate = float64(counter.ReadBytes-cached.readBytes) / timeDelta
	writeRate = float64(counter.WriteBytes-cached.writeBytes) / timeDelta
	readIOPS = float64(counter.ReadCount-cached.readCount) / timeDelta
	writeIOPS = float64(counter.WriteCount-cached.writeCount) / timeDelta

	// Ensure non-negative values
	if readRate < 0 {
		readRate = 0
	}
	if writeRate < 0 {
		writeRate = 0
	}
	if readIOPS < 0 {
		readIOPS = 0
	}
	if writeIOPS < 0 {
		writeIOPS = 0
	}

	// Update cache
	cached.readBytes = counter.ReadBytes
	cached.writeBytes = counter.WriteBytes
	cached.readCount = counter.ReadCount
	cached.writeCount = counter.WriteCount
	cached.readTime = counter.ReadTime
	cached.writeTime = counter.WriteTime
	cached.ioTime = counter.IoTime
	cached.lastTimestamp = now

	return readRate, writeRate, readIOPS, writeIOPS
}

// calculateUtilization calculates disk utilization percentage
func (d *diskIOParser) calculateUtilization(counter *disk.IOCountersStat) float64 {
	// macOS doesn't provide direct disk utilization
	// We estimate based on I/O time if available
	if counter.IoTime == 0 {
		return 0
	}

	// This is a simplified calculation
	// Real utilization would need kernel-level statistics
	cached, exists := d.cache[counter.Name]
	if !exists {
		return 0
	}

	timeDelta := time.Since(cached.lastTimestamp).Milliseconds()
	if timeDelta <= 0 {
		return 0
	}

	// Calculate IoTime delta
	ioTimeDelta := int64(counter.IoTime) - int64(cached.ioTime)
	if ioTimeDelta < 0 {
		ioTimeDelta = 0
	}

	// Calculate utilization as percentage of time spent doing I/O
	utilization := float64(ioTimeDelta) / float64(timeDelta) * 100

	if utilization > 100 {
		utilization = 100
	}
	if utilization < 0 {
		utilization = 0
	}

	return utilization
}

// shouldSkipDevice determines if a device should be skipped
func (d *diskIOParser) shouldSkipDevice(device string) bool {
	// Skip system and virtual devices
	skipPrefixes := []string{
		"loop",
		"ram",
		"devfs",
		"map",
		"com.apple",
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(device, prefix) {
			return true
		}
	}

	// Skip if device name contains certain patterns
	skipPatterns := []string{
		"autofs",
		"devfs",
		"fdesc",
		"kernfs",
		"nullfs",
		"procfs",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(device, pattern) {
			return true
		}
	}

	return false
}

// cleanupCache removes old cache entries
func (d *diskIOParser) cleanupCache(currentDevices map[string]disk.IOCountersStat) {
	// Remove cache entries for devices that no longer exist
	for device := range d.cache {
		if _, exists := currentDevices[device]; !exists {
			delete(d.cache, device)
		}
	}

	// Remove old entries that haven't been updated recently
	cutoff := time.Now().Add(-5 * time.Minute)
	for device, cached := range d.cache {
		if cached.lastTimestamp.Before(cutoff) {
			delete(d.cache, device)
		}
	}
}

// networkIOCache for rate calculations
var networkIOCache = struct {
	sync.RWMutex
	cache map[string]*networkIOSnapshot
}{
	cache: make(map[string]*networkIOSnapshot),
}

// collectNetworkIOGopsutil collects network I/O statistics using gopsutil
func collectNetworkIOGopsutil(ctx context.Context, iface string) (*entities.NetworkIO, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network I/O counters: %w", err)
	}

	for _, counter := range counters {
		if counter.Name == iface {
			// Calculate rates using cache
			networkIOCache.Lock()
			defer networkIOCache.Unlock()

			now := time.Now()
			cached, exists := networkIOCache.cache[iface]

			var rxRate, txRate float64
			if exists && now.Sub(cached.timestamp) > 0 {
				timeDelta := now.Sub(cached.timestamp).Seconds()
				// Calculate deltas
				rxDelta := int64(counter.BytesRecv) - int64(cached.bytesRecv)
				txDelta := int64(counter.BytesSent) - int64(cached.bytesSent)

				if rxDelta > 0 {
					rxRate = float64(rxDelta) / timeDelta
				}
				if txDelta > 0 {
					txRate = float64(txDelta) / timeDelta
				}
			}

			// Update cache
			networkIOCache.cache[iface] = &networkIOSnapshot{
				interface_:  iface,
				bytesRecv:   counter.BytesRecv,
				bytesSent:   counter.BytesSent,
				packetsRecv: counter.PacketsRecv,
				packetsSent: counter.PacketsSent,
				timestamp:   now,
			}

			rxRateVO, _ := valueobjects.NewByteRate(rxRate)
			txRateVO, _ := valueobjects.NewByteRate(txRate)

			return entities.NewNetworkIO(
				counter.Name,
				rxRateVO,
				txRateVO,
				counter.PacketsRecv,
				counter.PacketsSent,
				counter.Errin,
				counter.Errout,
				counter.Dropin,
				counter.Dropout,
			)
		}
	}

	return nil, fmt.Errorf("interface %s not found", iface)
}

// collectAllNetworkIOGopsutil collects network I/O statistics for all interfaces
func collectAllNetworkIOGopsutil(ctx context.Context) ([]*entities.NetworkIO, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network I/O counters: %w", err)
	}

	var metrics []*entities.NetworkIO

	for _, counter := range counters {
		// Skip loopback and inactive interfaces
		if counter.Name == "lo0" || counter.BytesRecv == 0 && counter.BytesSent == 0 {
			continue
		}

		// For simplicity, we're not calculating rates here
		// In production, you'd want to cache and calculate deltas
		rxRate, _ := valueobjects.NewByteRate(float64(counter.BytesRecv))
		txRate, _ := valueobjects.NewByteRate(float64(counter.BytesSent))

		networkIO, err := entities.NewNetworkIO(
			counter.Name,
			rxRate,
			txRate,
			counter.PacketsRecv,
			counter.PacketsSent,
			counter.Errin,
			counter.Errout,
			counter.Dropin,
			counter.Dropout,
		)

		if err == nil {
			metrics = append(metrics, networkIO)
		}
	}

	return metrics, nil
}