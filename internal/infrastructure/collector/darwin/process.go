//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// processParser handles process metrics collection for macOS
type processParser struct {
	mu sync.RWMutex

	// Cache for delta calculations
	cache map[int32]*processCache

	// Sandbox mode flag
	sandboxMode bool

	// Maximum concurrent process collections
	maxConcurrent int
}

// processCache stores previous process metrics for delta calculations
type processCache struct {
	lastBytesRead    int64
	lastBytesWritten int64
	lastReadOps      int64
	lastWriteOps     int64
	lastCPUTime      int64
	lastMeasurement  time.Time
}

// newProcessParser creates a new process parser
func newProcessParser(sandboxMode bool) *processParser {
	return &processParser{
		cache:         make(map[int32]*processCache),
		sandboxMode:   sandboxMode,
		maxConcurrent: 10,
	}
}

// collectProcessMetrics collects metrics for a specific process
func (p *processParser) collectProcessMetrics(ctx context.Context, pid valueobjects.ProcessID) (*entities.ProcessMetrics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pidInt := int32(pid.Value())

	// Try to get process info using CGO first
	cgoStats, cgoErr := GetProcessIOStats(pidInt)

	// Get process info from gopsutil
	proc, err := process.NewProcess(pidInt)
	if err != nil {
		if cgoErr != nil {
			return nil, fmt.Errorf("process not found: %w", err)
		}
		// Continue with CGO data only
	}

	// Get process name
	name := "unknown"
	if proc != nil {
		if n, err := proc.Name(); err == nil {
			name = n
		}
	}

	// Get or create cache entry
	cached, exists := p.cache[pidInt]
	if !exists {
		cached = &processCache{
			lastMeasurement: time.Now(),
		}
		p.cache[pidInt] = cached
	}

	now := time.Now()
	timeDelta := now.Sub(cached.lastMeasurement).Seconds()
	if timeDelta <= 0 {
		timeDelta = 1.0 // Prevent division by zero
	}

	// Calculate I/O rates
	var readRate, writeRate float64
	var ioReadOps, ioWriteOps int64

	if cgoStats != nil {
		// Calculate rates from CGO data
		if exists && timeDelta > 0 {
			readRate = float64(cgoStats.BytesRead-cached.lastBytesRead) / timeDelta
			writeRate = float64(cgoStats.BytesWritten-cached.lastBytesWritten) / timeDelta
			ioReadOps = cgoStats.ReadOps - cached.lastReadOps
			ioWriteOps = cgoStats.WriteOps - cached.lastWriteOps
		}

		// Update cache
		cached.lastBytesRead = cgoStats.BytesRead
		cached.lastBytesWritten = cgoStats.BytesWritten
		cached.lastReadOps = cgoStats.ReadOps
		cached.lastWriteOps = cgoStats.WriteOps
	} else if proc != nil && !p.sandboxMode {
		// Try to get I/O counters from gopsutil (may not work on macOS)
		if ioCounters, err := proc.IOCounters(); err == nil {
			if exists && timeDelta > 0 {
				readRate = float64(ioCounters.ReadBytes-uint64(cached.lastBytesRead)) / timeDelta
				writeRate = float64(ioCounters.WriteBytes-uint64(cached.lastBytesWritten)) / timeDelta
				ioReadOps = int64(ioCounters.ReadCount) - cached.lastReadOps
				ioWriteOps = int64(ioCounters.WriteCount) - cached.lastWriteOps
			}

			// Update cache
			cached.lastBytesRead = int64(ioCounters.ReadBytes)
			cached.lastBytesWritten = int64(ioCounters.WriteBytes)
			cached.lastReadOps = int64(ioCounters.ReadCount)
			cached.lastWriteOps = int64(ioCounters.WriteCount)
		}
	}

	// Ensure non-negative rates
	if readRate < 0 {
		readRate = 0
	}
	if writeRate < 0 {
		writeRate = 0
	}
	if ioReadOps < 0 {
		ioReadOps = 0
	}
	if ioWriteOps < 0 {
		ioWriteOps = 0
	}

	// Get CPU and memory usage
	var cpuPercent, memoryPercent float64
	var memoryRSS uint64

	if cgoStats != nil {
		// Calculate CPU percentage from CGO data
		if exists && timeDelta > 0 {
			currentCPUTime := cgoStats.CPUTimeUser + cgoStats.CPUTimeSystem
			if currentCPUTime > cached.lastCPUTime {
				cpuTimeDelta := float64(currentCPUTime-cached.lastCPUTime) / 1e9 // nanoseconds to seconds
				cpuPercent = (cpuTimeDelta / timeDelta) * 100
			}
		}
		cached.lastCPUTime = cgoStats.CPUTimeUser + cgoStats.CPUTimeSystem
		memoryRSS = uint64(cgoStats.MemoryRSS)
	}

	// Get additional metrics from gopsutil if available
	if proc != nil {
		if cpu, err := proc.CPUPercent(); err == nil && cpu > 0 {
			cpuPercent = cpu
		}

		if memInfo, err := proc.MemoryInfo(); err == nil {
			if memoryRSS == 0 {
				memoryRSS = memInfo.RSS
			}
			if memPercent, err := proc.MemoryPercent(); err == nil {
				memoryPercent = float64(memPercent)
			}
		}
	}

	// Update last measurement time
	cached.lastMeasurement = now

	// Create value objects
	readRateVO, err := valueobjects.NewByteRate(readRate)
	if err != nil {
		return nil, fmt.Errorf("invalid read rate: %w", err)
	}

	writeRateVO, err := valueobjects.NewByteRate(writeRate)
	if err != nil {
		return nil, fmt.Errorf("invalid write rate: %w", err)
	}

	cpuPercentVO, err := valueobjects.NewPercentage(cpuPercent)
	if err != nil {
		cpuPercentVO, _ = valueobjects.NewPercentage(0)
	}

	memoryPercentVO, err := valueobjects.NewPercentage(memoryPercent)
	if err != nil {
		memoryPercentVO, _ = valueobjects.NewPercentage(0)
	}

	// Create ProcessMetrics entity
	return entities.NewProcessMetrics(
		pid,
		name,
		cpuPercentVO,
		memoryPercentVO,
		memoryRSS,
		readRateVO,
		writeRateVO,
		ioReadOps,
		ioWriteOps,
		0, // Network read bytes (not implemented yet)
		0, // Network write bytes (not implemented yet)
	)
}

// collectAllProcessMetrics collects metrics for all processes
func (p *processParser) collectAllProcessMetrics(ctx context.Context) ([]*entities.ProcessMetrics, error) {
	// Get process list using CGO
	pids, err := GetProcessList()
	if err != nil {
		// Fall back to gopsutil
		processes, err := process.Processes()
		if err != nil {
			return nil, fmt.Errorf("failed to get process list: %w", err)
		}

		pids = make([]int32, len(processes))
		for i, proc := range processes {
			pids[i] = proc.Pid
		}
	}

	// Collect metrics for each process
	var (
		metrics []*entities.ProcessMetrics
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, p.maxConcurrent)
	)

	for _, pidInt := range pids {
		select {
		case <-ctx.Done():
			return metrics, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(pidInt int32) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			pid, err := valueobjects.NewProcessID(pidInt)
			if err != nil {
				return
			}

			metric, err := p.collectProcessMetrics(ctx, pid)
			if err != nil {
				// Skip processes we can't access
				return
			}

			mu.Lock()
			metrics = append(metrics, metric)
			mu.Unlock()
		}(pidInt)
	}

	wg.Wait()

	// Clean up old cache entries
	p.cleanupCache(pids)

	return metrics, nil
}

// cleanupCache removes cache entries for processes that no longer exist
func (p *processParser) cleanupCache(activePids []int32) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create a map of active PIDs for quick lookup
	active := make(map[int32]bool)
	for _, pid := range activePids {
		active[pid] = true
	}

	// Remove cache entries for inactive PIDs
	for pid := range p.cache {
		if !active[pid] {
			delete(p.cache, pid)
		}
	}

	// Also remove old entries that haven't been updated recently
	cutoff := time.Now().Add(-5 * time.Minute)
	for pid, cached := range p.cache {
		if cached.lastMeasurement.Before(cutoff) {
			delete(p.cache, pid)
		}
	}
}

// getActiveProcesses returns a list of active process IDs
func (p *processParser) getActiveProcesses(ctx context.Context) ([]valueobjects.ProcessID, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to get process PIDs: %w", err)
	}

	var processPIDs []valueobjects.ProcessID
	for _, pid := range pids {
		processID, err := valueobjects.NewProcessID(pid)
		if err != nil {
			continue // Skip invalid PIDs
		}
		processPIDs = append(processPIDs, processID)
	}

	return processPIDs, nil
}

// getProcessInfo returns detailed information about a specific process
func (p *processParser) getProcessInfo(ctx context.Context, pid valueobjects.ProcessID) (*services.ProcessInfo, error) {
	pidInt := int32(pid.Value())

	proc, err := process.NewProcess(pidInt)
	if err != nil {
		return nil, fmt.Errorf("process not found: %w", err)
	}

	// Get basic process information
	name, _ := proc.Name()
	exe, _ := proc.Exe()
	cmdline, _ := proc.Cmdline()
	ppidInt, _ := proc.Ppid()
	uids, _ := proc.Uids()
	gids, _ := proc.Gids()
	createTime, _ := proc.CreateTime()
	status, _ := proc.Status()
	numThreads, _ := proc.NumThreads()
	nice, _ := proc.Nice()
	environ, _ := proc.Environ()
	openFiles, _ := proc.OpenFiles()
	connections, _ := proc.Connections()

	// Convert parent PID
	var parentPID *valueobjects.ProcessID
	if ppidInt > 0 {
		if ppid, err := valueobjects.NewProcessID(ppidInt); err == nil {
			parentPID = &ppid
		}
	}

	// Determine process state
	state := entities.ProcessStateRunning
	switch status[0] {
	case "R":
		state = entities.ProcessStateRunning
	case "S":
		state = entities.ProcessStateSleeping
	case "T":
		state = entities.ProcessStateStopped
	case "Z":
		state = entities.ProcessStateZombie
	case "D":
		state = entities.ProcessStateWaiting
	}

	// Get UID and GID
	var uid, gid uint32
	if len(uids) > 0 {
		uid = uint32(uids[0])
	}
	if len(gids) > 0 {
		gid = uint32(gids[0])
	}

	// Parse environment into map
	envMap := make(map[string]string)
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Parse command line
	var cmdlineSlice []string
	if cmdline != "" {
		cmdlineSlice = strings.Fields(cmdline)
	}

	return &services.ProcessInfo{
		PID:         pid,
		Name:        name,
		ExecutePath: exe,
		CommandLine: cmdlineSlice,
		ParentPID:   parentPID,
		UserID:      uid,
		GroupID:     gid,
		StartTime:   time.Unix(0, createTime*int64(time.Millisecond)),
		State:       state,
		Threads:     int(numThreads),
		Priority:    0, // macOS doesn't expose priority the same way
		Nice:        int(nice),
		Environment: envMap,
		OpenFiles:   len(openFiles),
		Connections: len(connections),
	}, nil
}