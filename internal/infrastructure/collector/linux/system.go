//go:build linux

package linux

import (
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// systemParser handles system-wide metrics collection
type systemParser struct {
	cache *metricsCache
}

// newSystemParser creates a new system parser
func newSystemParser() *systemParser {
	return &systemParser{
		cache: newMetricsCache(),
	}
}

// initialize initializes the system parser
func (s *systemParser) initialize() error {
	// Pre-warm CPU info
	_, err := cpu.Percent(0, false)
	return err
}

// collectSystemMetrics collects system-wide metrics
func (s *systemParser) collectSystemMetrics() (*entities.SystemMetrics, error) {
	// Get CPU usage percentage
	cpuPercents, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var cpuUsage float64
	if len(cpuPercents) > 0 {
		cpuUsage = cpuPercents[0]
	}

	// Get memory information
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		// Load average may not be available on all systems
		loadAvg = &load.AvgStat{}
	}

	// Get total disk I/O rates
	totalReadRate, totalWriteRate, err := s.getSystemDiskIOTotals()
	if err != nil {
		// Don't fail if disk I/O is unavailable
		totalReadRate = 0
		totalWriteRate = 0
	}

	// Get process count
	processCount, err := s.getProcessCount()
	if err != nil {
		processCount = 0
	}

	// Create value objects
	cpuPercentage, err := valueobjects.NewPercentage(cpuUsage)
	if err != nil {
		cpuPercentage = valueobjects.NewPercentageZero()
	}

	memoryPercentage, err := valueobjects.NewPercentage(memInfo.UsedPercent)
	if err != nil {
		memoryPercentage = valueobjects.NewPercentageZero()
	}

	diskReadRate, err := valueobjects.NewByteRate(totalReadRate)
	if err != nil {
		diskReadRate = valueobjects.NewByteRateZero()
	}

	diskWriteRate, err := valueobjects.NewByteRate(totalWriteRate)
	if err != nil {
		diskWriteRate = valueobjects.NewByteRateZero()
	}

	// Create SystemMetrics entity with proper constructor
	systemMetrics, err := entities.NewSystemMetrics(
		cpuPercentage,
		memoryPercentage,
		diskReadRate,
		diskWriteRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create system metrics: %w", err)
	}

	// Set additional fields using setters
	systemMetrics.SetLoadAverage(loadAvg.Load1, loadAvg.Load5, loadAvg.Load15)
	threadCount := s.getThreadCount()
	systemMetrics.SetProcessCount(int(processCount), int(threadCount))

	return systemMetrics, nil
}

// getSystemDiskIOTotals calculates total disk I/O rates for all disks
func (s *systemParser) getSystemDiskIOTotals() (uint64, uint64, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var totalReadBytes, totalWriteBytes uint64
	for device, counter := range ioCounters {
		// Skip non-physical devices
		if s.shouldSkipDevice(device) {
			continue
		}

		totalReadBytes += counter.ReadBytes
		totalWriteBytes += counter.WriteBytes
	}

	// Calculate rates using cache
	readRate := s.cache.calculateSystemIORate(true, totalReadBytes)
	writeRate := s.cache.calculateSystemIORate(false, totalWriteBytes)

	return readRate, writeRate, nil
}

// getProcessCount returns the total number of processes
func (s *systemParser) getProcessCount() (uint32, error) {
	pids, err := process.Pids()
	if err != nil {
		return 0, err
	}
	return uint32(len(pids)), nil
}

// getThreadCount returns the total number of threads
func (s *systemParser) getThreadCount() uint32 {
	// This is an approximation - sum all threads from all processes
	pids, err := process.Pids()
	if err != nil {
		return 0
	}

	var totalThreads uint32
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		threads, err := p.NumThreads()
		if err != nil {
			continue
		}

		totalThreads += uint32(threads)
	}

	return totalThreads
}

// collectCPUInfo collects detailed CPU information
func (s *systemParser) collectCPUInfo() (*entities.CPUInfo, error) {
	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	if len(cpuInfo) == 0 {
		return nil, fmt.Errorf("no CPU info available")
	}

	// Get CPU counts
	physicalCount, err := cpu.Counts(false)
	if err != nil {
		physicalCount = 1
	}

	logicalCount, err := cpu.Counts(true)
	if err != nil {
		logicalCount = 1
	}

	// Get CPU times
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU times: %w", err)
	}

	var totalTime cpu.TimesStat
	if len(cpuTimes) > 0 {
		totalTime = cpuTimes[0]
	}

	// Get per-CPU usage
	perCPU, err := cpu.Percent(0, true)
	if err != nil {
		perCPU = []float64{}
	}

	// Create CPUInfo entity
	return entities.NewCPUInfo(
		cpuInfo[0].ModelName,
		cpuInfo[0].VendorID,
		uint32(physicalCount),
		uint32(logicalCount),
		cpuInfo[0].Mhz,
		totalTime.User,
		totalTime.System,
		totalTime.Idle,
		totalTime.Iowait,
		totalTime.Irq,
		totalTime.Softirq,
		totalTime.Steal,
		perCPU,
	), nil
}

// collectMemoryInfo collects detailed memory information
func (s *systemParser) collectMemoryInfo() (*entities.MemoryInfo, error) {
	// Get virtual memory stats
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	// Get swap memory stats
	swapStat, err := mem.SwapMemory()
	if err != nil {
		// Swap may not be available
		swapStat = &mem.SwapMemoryStat{}
	}

	// Create MemoryInfo entity
	return entities.NewMemoryInfo(
		vmStat.Total,
		vmStat.Available,
		vmStat.Used,
		vmStat.Free,
		vmStat.Buffers,
		vmStat.Cached,
		swapStat.Total,
		swapStat.Used,
		swapStat.Free,
		vmStat.Shared,
		vmStat.Active,
		vmStat.Inactive,
		vmStat.Slab,
	), nil
}

// shouldSkipDevice determines if a device should be skipped
func (s *systemParser) shouldSkipDevice(device string) bool {
	// Reuse logic from diskStatsParser
	parser := newDiskStatsParser()
	return parser.shouldSkipDevice(device)
}

// getSystemInfo returns detailed system information
func (s *systemParser) getSystemInfo() (*services.SystemInfo, error) {
	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	var cpuModel string
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	cpuCores, err := cpu.Counts(false)
	if err != nil {
		cpuCores = 1
	}

	cpuThreads, err := cpu.Counts(true)
	if err != nil {
		cpuThreads = 1
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		swapInfo = &mem.SwapMemoryStat{}
	}

	loadAvg, err := load.Avg()
	if err != nil {
		loadAvg = &load.AvgStat{}
	}

	return &services.SystemInfo{
		Hostname:      info.Hostname,
		OS:            info.OS,
		Platform:      info.Platform,
		Architecture:  info.KernelArch,
		CPUModel:      cpuModel,
		CPUCores:      cpuCores,
		CPUThreads:    cpuThreads,
		TotalMemory:   memInfo.Total,
		TotalSwap:     swapInfo.Total,
		BootTime:      time.Unix(int64(info.BootTime), 0),
		Uptime:        time.Duration(info.Uptime) * time.Second,
		LoadAverage:   [3]float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
		KernelVersion: info.KernelVersion,
	}, nil
}