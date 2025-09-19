//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// sysctlInterface provides system metrics collection using sysctl and system calls
type sysctlInterface struct {
	// Cache for system info that doesn't change
	cpuCount int
	hostname string
	kernelVersion string
	bootTime time.Time

	// Metrics cache for rate calculations
	cache *metricsCache
}

// newSysctlInterface creates a new sysctl interface
func newSysctlInterface() *sysctlInterface {
	s := &sysctlInterface{
		cache: newMetricsCache(),
	}
	s.initializeStaticInfo()
	return s
}

// initializeStaticInfo initializes static system information
func (s *sysctlInterface) initializeStaticInfo() {
	// Get CPU count
	if info, err := GetSystemInfo(); err == nil {
		s.cpuCount = info.CPUCount
	} else if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		s.cpuCount = int(cpuInfo[0].Cores)
	}

	// Get hostname
	if info, err := host.Info(); err == nil {
		s.hostname = info.Hostname
		s.kernelVersion = info.KernelVersion
		s.bootTime = time.Unix(int64(info.BootTime), 0)
	}
}

// collectSystemMetrics collects system-wide metrics
func (s *sysctlInterface) collectSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error) {
	// Get system info from CGO
	sysInfo, err := GetSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	// Get CPU usage
	cpuPercent := 0.0
	if percentages, err := cpu.Percent(time.Millisecond*100, false); err == nil && len(percentages) > 0 {
		cpuPercent = percentages[0]
	}

	// Get memory statistics
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	// Get disk I/O statistics and calculate rates using cache
	var totalReadRate, totalWriteRate float64
	if diskStats, err := disk.IOCounters(); err == nil {
		now := time.Now()

		// Get cached system snapshot for rate calculations
		if s.cache.system != nil && now.Sub(s.cache.system.timestamp) > 0 {
			// Calculate rates from cached values
			var totalReadBytes, totalWriteBytes uint64
			for _, stat := range diskStats {
				totalReadBytes += stat.ReadBytes
				totalWriteBytes += stat.WriteBytes
			}

			timeDelta := now.Sub(s.cache.system.timestamp).Seconds()
			if timeDelta > 0 {
				// Calculate deltas
				readDelta := int64(totalReadBytes) - int64(s.cache.system.diskReadBytes)
				writeDelta := int64(totalWriteBytes) - int64(s.cache.system.diskWriteBytes)

				if readDelta > 0 {
					totalReadRate = float64(readDelta) / timeDelta
				}
				if writeDelta > 0 {
					totalWriteRate = float64(writeDelta) / timeDelta
				}
			}

			// Update cache
			s.cache.system = &systemMetricsSnapshot{
				diskReadBytes:  totalReadBytes,
				diskWriteBytes: totalWriteBytes,
				timestamp:      now,
			}
		} else {
			// Initialize cache
			var totalReadBytes, totalWriteBytes uint64
			for _, stat := range diskStats {
				totalReadBytes += stat.ReadBytes
				totalWriteBytes += stat.WriteBytes
			}

			s.cache.system = &systemMetricsSnapshot{
				diskReadBytes:  totalReadBytes,
				diskWriteBytes: totalWriteBytes,
				timestamp:      now,
			}
		}
	}

	// Create value objects
	cpuUsage, err := valueobjects.NewPercentage(cpuPercent)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU usage: %w", err)
	}

	memoryUsage, err := valueobjects.NewPercentage(vmStat.UsedPercent)
	if err != nil {
		return nil, fmt.Errorf("invalid memory usage: %w", err)
	}

	diskReadRate, err := valueobjects.NewByteRate(totalReadRate)
	if err != nil {
		diskReadRate, _ = valueobjects.NewByteRate(0)
	}

	diskWriteRate, err := valueobjects.NewByteRate(totalWriteRate)
	if err != nil {
		diskWriteRate, _ = valueobjects.NewByteRate(0)
	}

	// Create SystemMetrics entity
	return entities.NewSystemMetrics(
		uint32(s.cpuCount),
		cpuUsage,
		vmStat.Total,
		vmStat.Available,
		memoryUsage,
		diskReadRate,
		diskWriteRate,
		sysInfo.LoadAvg1,
		sysInfo.LoadAvg5,
		sysInfo.LoadAvg15,
	)
}


// initialize initializes the sysctl interface
func (s *sysctlInterface) initialize(ctx context.Context) error {
	// Re-initialize static info in case it failed during construction
	s.initializeStaticInfo()
	return nil
}

// getSystemInfo returns system information
func (s *sysctlInterface) getSystemInfo(ctx context.Context) (*services.SystemInfo, error) {
	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get swap info: %w", err)
	}

	var cpuModel string
	var cpuThreads int
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
		cpuThreads = int(cpuInfo[0].Cores)
	}

	// Get load averages
	loadAvg, err := GetSystemInfo()
	var loadAverage [3]float64
	if err == nil {
		loadAverage = [3]float64{loadAvg.LoadAvg1, loadAvg.LoadAvg5, loadAvg.LoadAvg15}
	}

	return &services.SystemInfo{
		Hostname:      info.Hostname,
		OS:            info.OS,
		Platform:      info.Platform,
		Architecture:  info.PlatformFamily,
		CPUModel:      cpuModel,
		CPUCores:      s.cpuCount,
		CPUThreads:    cpuThreads,
		TotalMemory:   vmStat.Total,
		TotalSwap:     swapStat.Total,
		BootTime:      time.Unix(int64(info.BootTime), 0),
		Uptime:        time.Since(time.Unix(int64(info.BootTime), 0)),
		LoadAverage:   loadAverage,
		KernelVersion: info.KernelVersion,
	}, nil
}