//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
	"github.com/shirou/gopsutil/v4/disk"
)

const (
	diskStatsPath = "/proc/diskstats"
	sysBlockPath  = "/sys/block"
	sectorSize    = 512 // Standard sector size in bytes
)

// diskStatsParser handles disk I/O statistics collection
type diskStatsParser struct {
	diskStatsPath string
	sysBlockPath  string
	cache         *metricsCache
}

// newDiskStatsParser creates a new disk stats parser
func newDiskStatsParser() *diskStatsParser {
	return &diskStatsParser{
		diskStatsPath: diskStatsPath,
		sysBlockPath:  sysBlockPath,
		cache:         newMetricsCache(),
	}
}

// collectDiskIO collects I/O metrics for a specific disk
func (d *diskStatsParser) collectDiskIO(device string) (*entities.DiskIO, error) {
	// Use gopsutil for basic stats
	ioCounters, err := disk.IOCounters(device)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	counter, exists := ioCounters[device]
	if !exists {
		return nil, fmt.Errorf("device %s not found", device)
	}

	// Calculate byte rates using cache
	readRate := d.cache.calculateDiskIORate(device, true, counter.ReadBytes)
	writeRate := d.cache.calculateDiskIORate(device, false, counter.WriteBytes)

	// Create value objects
	readByteRate, err := valueobjects.NewByteRate(readRate)
	if err != nil {
		readByteRate = valueobjects.NewByteRateZero()
	}

	writeByteRate, err := valueobjects.NewByteRate(writeRate)
	if err != nil {
		writeByteRate = valueobjects.NewByteRateZero()
	}

	// Create basic DiskIO entity
	diskIO, err := entities.NewDiskIO(device, readByteRate, writeByteRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk I/O metrics: %w", err)
	}

	// Get additional metrics from /proc/diskstats
	stats, err := d.readDiskStats(device)
	if err == nil {
		// Calculate IOPS
		readIOPS := d.cache.calculateDiskIOPS(device, true, stats.readsCompleted)
		writeIOPS := d.cache.calculateDiskIOPS(device, false, stats.writesCompleted)

		// Calculate delta-based utilization and queue depth
		utilizationPct, avgQueueDepth := d.calculateDeltaMetrics(device, stats)

		// Set additional metrics
		diskIO.SetIOPS(readIOPS, writeIOPS)
		diskIO.SetQueueDepth(int(avgQueueDepth))

		// Create utilization percentage
		utilization, err := valueobjects.NewPercentage(utilizationPct)
		if err != nil {
			utilization = valueobjects.NewPercentageZero()
		}
		diskIO.SetUtilization(utilization)

		// Set latency if available
		if counter.ReadTime > 0 && stats.readsCompleted > 0 {
			avgReadLatency := time.Duration(counter.ReadTime/stats.readsCompleted) * time.Millisecond
			avgWriteLatency := time.Duration(0)
			if counter.WriteTime > 0 && stats.writesCompleted > 0 {
				avgWriteLatency = time.Duration(counter.WriteTime/stats.writesCompleted) * time.Millisecond
			}
			diskIO.SetLatency(avgReadLatency, avgWriteLatency)
		}

		// Set total statistics
		diskIO.SetTotalStatistics(counter.ReadBytes, counter.WriteBytes, stats.readsCompleted, stats.writesCompleted)
		diskIO.SetMergedOperations(stats.readsMerged, stats.writesMerged)
	}

	return diskIO, nil
}

// collectAllDiskIO collects I/O metrics for all disks
func (d *diskStatsParser) collectAllDiskIO() ([]*entities.DiskIO, error) {
	// Get all disk I/O counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var diskList []*entities.DiskIO

	for device, counter := range ioCounters {
		// Skip non-physical devices
		if d.shouldSkipDevice(device) {
			continue
		}

		// Calculate byte rates using cache
		readRate := d.cache.calculateDiskIORate(device, true, counter.ReadBytes)
		writeRate := d.cache.calculateDiskIORate(device, false, counter.WriteBytes)

		// Create value objects
		readByteRate, err := valueobjects.NewByteRate(readRate)
		if err != nil {
			readByteRate = valueobjects.NewByteRateZero()
		}
		writeByteRate, err := valueobjects.NewByteRate(writeRate)
		if err != nil {
			writeByteRate = valueobjects.NewByteRateZero()
		}

		// Create basic DiskIO entity
		diskIO, err := entities.NewDiskIO(device, readByteRate, writeByteRate)
		if err != nil {
			continue // Skip this device on error
		}

		// Get additional metrics from /proc/diskstats
		stats, err := d.readDiskStats(device)
		if err == nil {
			// Calculate IOPS
			readIOPS := d.cache.calculateDiskIOPS(device, true, stats.readsCompleted)
			writeIOPS := d.cache.calculateDiskIOPS(device, false, stats.writesCompleted)

			// Calculate delta-based utilization and queue depth
			utilizationPct, avgQueueDepth := d.calculateDeltaMetrics(device, stats)

			// Set additional metrics
			diskIO.SetIOPS(readIOPS, writeIOPS)
			diskIO.SetQueueDepth(int(avgQueueDepth))

			// Create utilization percentage
			utilization, err := valueobjects.NewPercentage(utilizationPct)
			if err != nil {
				utilization = valueobjects.NewPercentageZero()
			}
			diskIO.SetUtilization(utilization)

			// Set latency if available
			if counter.ReadTime > 0 && stats.readsCompleted > 0 {
				avgReadLatency := time.Duration(counter.ReadTime/stats.readsCompleted) * time.Millisecond
				avgWriteLatency := time.Duration(0)
				if counter.WriteTime > 0 && stats.writesCompleted > 0 {
					avgWriteLatency = time.Duration(counter.WriteTime/stats.writesCompleted) * time.Millisecond
				}
				diskIO.SetLatency(avgReadLatency, avgWriteLatency)
			}

			// Set total statistics
			diskIO.SetTotalStatistics(counter.ReadBytes, counter.WriteBytes, stats.readsCompleted, stats.writesCompleted)
			diskIO.SetMergedOperations(stats.readsMerged, stats.writesMerged)
		}

		diskList = append(diskList, diskIO)
	}

	return diskList, nil
}

// diskStats holds parsed /proc/diskstats data
type diskStats struct {
	majorNumber      uint64
	minorNumber      uint64
	deviceName       string
	readsCompleted   uint64
	readsMerged      uint64
	sectorsRead      uint64
	timeReading      uint64
	writesCompleted  uint64
	writesMerged     uint64
	sectorsWritten   uint64
	timeWriting      uint64
	iOsInProgress    uint64
	ioTime           uint64
	timeInQueue      uint64
	discardsCompleted uint64
	discardsMerged   uint64
	sectorsDiscarded uint64
	timeDiscarding   uint64
}

// readDiskStats reads and parses /proc/diskstats for a specific device
func (d *diskStatsParser) readDiskStats(device string) (*diskStats, error) {
	file, err := os.Open(d.diskStatsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", d.diskStatsPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Format: major minor name ...stats...
		if len(fields) < 14 {
			continue
		}

		if fields[2] != device {
			continue
		}

		stats := &diskStats{
			deviceName: device,
		}

		// Parse all fields
		stats.majorNumber, _ = strconv.ParseUint(fields[0], 10, 64)
		stats.minorNumber, _ = strconv.ParseUint(fields[1], 10, 64)
		stats.readsCompleted, _ = strconv.ParseUint(fields[3], 10, 64)
		stats.readsMerged, _ = strconv.ParseUint(fields[4], 10, 64)
		stats.sectorsRead, _ = strconv.ParseUint(fields[5], 10, 64)
		stats.timeReading, _ = strconv.ParseUint(fields[6], 10, 64)
		stats.writesCompleted, _ = strconv.ParseUint(fields[7], 10, 64)
		stats.writesMerged, _ = strconv.ParseUint(fields[8], 10, 64)
		stats.sectorsWritten, _ = strconv.ParseUint(fields[9], 10, 64)
		stats.timeWriting, _ = strconv.ParseUint(fields[10], 10, 64)
		stats.iOsInProgress, _ = strconv.ParseUint(fields[11], 10, 64)
		stats.ioTime, _ = strconv.ParseUint(fields[12], 10, 64)
		stats.timeInQueue, _ = strconv.ParseUint(fields[13], 10, 64)

		// Extended fields (kernel 4.18+)
		if len(fields) >= 18 {
			stats.discardsCompleted, _ = strconv.ParseUint(fields[14], 10, 64)
			stats.discardsMerged, _ = strconv.ParseUint(fields[15], 10, 64)
			stats.sectorsDiscarded, _ = strconv.ParseUint(fields[16], 10, 64)
			stats.timeDiscarding, _ = strconv.ParseUint(fields[17], 10, 64)
		}

		return stats, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", d.diskStatsPath, err)
	}

	return nil, fmt.Errorf("device %s not found in diskstats", device)
}

// shouldSkipDevice determines if a device should be skipped
func (d *diskStatsParser) shouldSkipDevice(device string) bool {
	// Skip loop devices
	if strings.HasPrefix(device, "loop") {
		return true
	}

	// Skip ram disks
	if strings.HasPrefix(device, "ram") {
		return true
	}

	// Skip device mapper devices (optional, might want to include these)
	if strings.HasPrefix(device, "dm-") {
		// Check if it's a real mapped device
		sysPath := filepath.Join(d.sysBlockPath, device)
		if _, err := os.Stat(sysPath); err != nil {
			return true
		}
	}

	// Skip partitions (e.g., sda1, sda2) - only report whole disks
	// This is optional depending on requirements
	if d.isPartition(device) {
		return true
	}

	return false
}

// isPartition checks if a device is a partition
func (d *diskStatsParser) isPartition(device string) bool {
	// Check if the device name ends with a number (common for partitions)
	if len(device) > 0 {
		lastChar := device[len(device)-1]
		if lastChar >= '0' && lastChar <= '9' {
			// Check if parent device exists
			parentDevice := device[:len(device)-1]
			parentPath := filepath.Join(d.sysBlockPath, parentDevice)
			if _, err := os.Stat(parentPath); err == nil {
				return true
			}

			// For nvme devices (nvme0n1p1)
			if strings.Contains(device, "p") {
				parts := strings.Split(device, "p")
				if len(parts) == 2 {
					parentPath := filepath.Join(d.sysBlockPath, parts[0])
					if _, err := os.Stat(parentPath); err == nil {
						return true
					}
				}
			}
		}
	}
	return false
}

// getDeviceSize gets the size of a device in bytes
func (d *diskStatsParser) getDeviceSize(device string) (uint64, error) {
	sizePath := filepath.Join(d.sysBlockPath, device, "size")
	data, err := os.ReadFile(sizePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read device size: %w", err)
	}

	sizeStr := strings.TrimSpace(string(data))
	sectors, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse device size: %w", err)
	}

	return sectors * sectorSize, nil
}

// getQueueDepth gets the queue depth for a device
func (d *diskStatsParser) getQueueDepth(device string) (uint32, error) {
	queuePath := filepath.Join(d.sysBlockPath, device, "queue", "nr_requests")
	data, err := os.ReadFile(queuePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read queue depth: %w", err)
	}

	depthStr := strings.TrimSpace(string(data))
	depth, err := strconv.ParseUint(depthStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse queue depth: %w", err)
	}

	return uint32(depth), nil
}

// getScheduler gets the I/O scheduler for a device
func (d *diskStatsParser) getScheduler(device string) (string, error) {
	schedulerPath := filepath.Join(d.sysBlockPath, device, "queue", "scheduler")
	data, err := os.ReadFile(schedulerPath)
	if err != nil {
		return "", fmt.Errorf("failed to read scheduler: %w", err)
	}

	scheduler := strings.TrimSpace(string(data))
	// The active scheduler is in brackets [mq-deadline]
	if strings.Contains(scheduler, "[") && strings.Contains(scheduler, "]") {
		start := strings.Index(scheduler, "[")
		end := strings.Index(scheduler, "]")
		if start < end {
			return scheduler[start+1 : end], nil
		}
	}

	return scheduler, nil
}

// calculateDeltaMetrics calculates utilization and queue depth based on deltas
func (d *diskStatsParser) calculateDeltaMetrics(device string, stats *diskStats) (float64, float64) {
	// Get previous stats from cache
	prevIOTime := d.cache.getDiskStat(device, "io_time")
	prevTimeInQueue := d.cache.getDiskStat(device, "time_in_queue")
	prevTimestamp := d.cache.getDiskStatTimestamp(device)

	now := time.Now()

	// Update cache with current values
	d.cache.updateDiskStat(device, "io_time", stats.ioTime)
	d.cache.updateDiskStat(device, "time_in_queue", stats.timeInQueue)

	// If no previous data, return zeros
	if prevIOTime == 0 || prevTimestamp.IsZero() {
		return 0.0, 0.0
	}

	// Calculate time interval in milliseconds
	intervalMs := float64(now.Sub(prevTimestamp).Milliseconds())
	if intervalMs <= 0 {
		return 0.0, 0.0
	}

	// Calculate deltas
	deltaIOTime := float64(stats.ioTime - prevIOTime)
	deltaTimeInQueue := float64(stats.timeInQueue - prevTimeInQueue)

	// Calculate utilization percentage (ioTime is in milliseconds)
	utilizationPct := (deltaIOTime / intervalMs) * 100.0
	if utilizationPct > 100.0 {
		utilizationPct = 100.0
	}
	if utilizationPct < 0.0 {
		utilizationPct = 0.0
	}

	// Calculate average queue depth (timeInQueue is in milliseconds)
	avgQueueDepth := deltaTimeInQueue / intervalMs
	if avgQueueDepth < 0 {
		avgQueueDepth = 0
	}

	return utilizationPct, avgQueueDepth
}

// getAvailableDevices returns list of available disk devices
func (d *diskStatsParser) getAvailableDevices() ([]string, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var devices []string
	for device := range ioCounters {
		if !d.shouldSkipDevice(device) {
			devices = append(devices, device)
		}
	}

	return devices, nil
}