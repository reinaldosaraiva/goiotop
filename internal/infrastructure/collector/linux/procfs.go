//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	procPath = "/proc"
)

// procfsParser handles parsing of /proc filesystem for process metrics
type procfsParser struct {
	procPath string
	cache    *metricsCache
}

// newProcfsParser creates a new procfs parser
func newProcfsParser() *procfsParser {
	return &procfsParser{
		procPath: procPath,
		cache:    newMetricsCache(),
	}
}

// collectProcessMetrics collects metrics for a specific process
func (p *procfsParser) collectProcessMetrics(pid int) (*entities.ProcessMetrics, error) {
	// Use gopsutil for process information
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to get process %d: %w", pid, err)
	}

	// Get process name
	name, err := proc.Name()
	if err != nil {
		name = "unknown"
	}

	// Get CPU usage percentage
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		cpuPercent = 0.0
	}

	// Get memory info
	memInfo, err := proc.MemoryInfo()
	var memoryRSS, memoryVMS uint64
	if err == nil && memInfo != nil {
		memoryRSS = memInfo.RSS
		memoryVMS = memInfo.VMS
	}

	// Get total system memory for percentage calculation
	totalMem, err := mem.VirtualMemory()
	var memoryPct float64
	if err == nil && totalMem != nil && totalMem.Total > 0 {
		memoryPct = (float64(memoryRSS) / float64(totalMem.Total)) * 100.0
	}

	// Get I/O stats from /proc/[pid]/io
	ioStats, err := p.readProcessIO(pid)
	if err != nil {
		// May not have permission, use zero values
		ioStats = &processIOStats{}
	}

	// Calculate rates using cache
	readRate := p.cache.calculateIORate(pid, true, ioStats.readBytes)
	writeRate := p.cache.calculateIORate(pid, false, ioStats.writeBytes)

	// Create value objects
	processID, err := valueobjects.NewProcessID(uint32(pid))
	if err != nil {
		return nil, fmt.Errorf("invalid process ID: %w", err)
	}

	cpuUsage, err := valueobjects.NewPercentage(cpuPercent)
	if err != nil {
		cpuUsage = valueobjects.NewPercentageZero()
	}

	memoryPercentage, err := valueobjects.NewPercentage(memoryPct)
	if err != nil {
		memoryPercentage = valueobjects.NewPercentageZero()
	}

	readByteRate, err := valueobjects.NewByteRate(readRate)
	if err != nil {
		readByteRate = valueobjects.NewByteRateZero()
	}

	writeByteRate, err := valueobjects.NewByteRate(writeRate)
	if err != nil {
		writeByteRate = valueobjects.NewByteRateZero()
	}

	// Get thread count
	threadCount, _ := proc.NumThreads()

	// Get process state
	status, _ := proc.Status()
	state := p.mapProcessStateToEntity(status)

	// Create ProcessMetrics entity with proper constructor
	processMetrics, err := entities.NewProcessMetrics(
		processID,
		name,
		cpuUsage,
		memoryPercentage,
		readByteRate,
		writeByteRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create process metrics: %w", err)
	}

	// Set additional details using setters
	processMetrics.SetMemoryDetails(memoryRSS, memoryVMS)
	processMetrics.SetThreads(threadCount)
	processMetrics.SetState(state)

	// Set parent PID if available
	ppid, err := proc.Ppid()
	if err == nil && ppid > 0 {
		parentPID, err := valueobjects.NewProcessID(uint32(ppid))
		if err == nil {
			processMetrics.SetParentPID(&parentPID)
		}
	}

	// Set ownership
	uids, err := proc.Uids()
	if err == nil && len(uids) > 0 {
		gids, err := proc.Gids()
		if err == nil && len(gids) > 0 {
			processMetrics.SetOwnership(uint32(uids[0]), uint32(gids[0]))
		}
	}

	return processMetrics, nil
}

// collectAllProcessMetricsWithStats collects metrics for all processes with stats
func (p *procfsParser) collectAllProcessMetricsWithStats(filter *services.ProcessFilter) ([]*entities.ProcessMetrics, int, error) {
	// List all PIDs
	pids, err := process.Pids()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list processes: %w", err)
	}

	var processList []*entities.ProcessMetrics
	skippedCount := 0

	for _, pid := range pids {
		// Skip kernel threads if not requested
		if filter != nil && !filter.CollectKernelThreads && p.isKernelThread(int(pid)) {
			skippedCount++
			continue
		}

		// Skip PID 0 (kernel)
		if pid == 0 {
			skippedCount++
			continue
		}

		metrics, err := p.collectProcessMetrics(int(pid))
		if err != nil {
			// Skip processes we can't read (permission denied, process died, etc.)
			skippedCount++
			continue
		}

		// Apply filter if provided
		if filter != nil {
			if !p.matchesFilter(metrics, filter) {
				skippedCount++
				continue
			}
		}

		// Apply max processes limit if set
		if filter != nil && filter.MaxProcesses > 0 && len(processList) >= filter.MaxProcesses {
			skippedCount++
			continue
		}

		processList = append(processList, metrics)
	}

	return processList, skippedCount, nil
}

// processIOStats holds I/O statistics for a process
type processIOStats struct {
	readBytes    uint64
	writeBytes   uint64
	readOps      uint64
	writeOps     uint64
	readChars    uint64
	writeChars   uint64
	cancelledIOs uint64
}

// readProcessIO reads I/O statistics from /proc/[pid]/io
func (p *procfsParser) readProcessIO(pid int) (*processIOStats, error) {
	ioPath := filepath.Join(p.procPath, strconv.Itoa(pid), "io")
	file, err := os.Open(ioPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, ErrPermissionDenied
		}
		return nil, fmt.Errorf("failed to open %s: %w", ioPath, err)
	}
	defer file.Close()

	stats := &processIOStats{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		switch parts[0] {
		case "rchar:":
			stats.readChars = value
		case "wchar:":
			stats.writeChars = value
		case "syscr:":
			stats.readOps = value
		case "syscw:":
			stats.writeOps = value
		case "read_bytes:":
			stats.readBytes = value
		case "write_bytes:":
			stats.writeBytes = value
		case "cancelled_write_bytes:":
			stats.cancelledIOs = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ioPath, err)
	}

	return stats, nil
}

// readProcessStat reads process statistics from /proc/[pid]/stat
func (p *procfsParser) readProcessStat(pid int) (*processStat, error) {
	statPath := filepath.Join(p.procPath, strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(statPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", statPath, err)
	}

	return p.parseStat(string(data))
}

// processStat holds parsed /proc/[pid]/stat data
type processStat struct {
	pid        int
	comm       string
	state      rune
	ppid       int
	pgrp       int
	session    int
	ttyNr      int
	tpgid      int
	flags      uint
	minflt     uint64
	cminflt    uint64
	majflt     uint64
	cmajflt    uint64
	utime      uint64
	stime      uint64
	cutime     int64
	cstime     int64
	priority   int64
	nice       int64
	numThreads int64
	vsize      uint64
	rss        int64
}

// parseStat parses /proc/[pid]/stat content
func (p *procfsParser) parseStat(stat string) (*processStat, error) {
	// Find the last ) to handle process names with spaces and parentheses
	lastParen := strings.LastIndex(stat, ")")
	if lastParen == -1 {
		return nil, fmt.Errorf("invalid stat format")
	}

	// Extract comm (process name) between first ( and last )
	firstParen := strings.Index(stat, "(")
	if firstParen == -1 {
		return nil, fmt.Errorf("invalid stat format")
	}

	ps := &processStat{}
	ps.comm = stat[firstParen+1 : lastParen]

	// Parse PID before the first (
	pidStr := stat[:firstParen-1]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PID: %w", err)
	}
	ps.pid = pid

	// Parse fields after the last )
	fields := strings.Fields(stat[lastParen+2:])
	if len(fields) < 20 {
		return nil, fmt.Errorf("insufficient stat fields")
	}

	// Parse state
	if len(fields[0]) > 0 {
		ps.state = rune(fields[0][0])
	}

	// Parse numeric fields
	ps.ppid, _ = strconv.Atoi(fields[1])
	ps.pgrp, _ = strconv.Atoi(fields[2])
	ps.session, _ = strconv.Atoi(fields[3])
	ps.ttyNr, _ = strconv.Atoi(fields[4])
	ps.tpgid, _ = strconv.Atoi(fields[5])

	flags, _ := strconv.ParseUint(fields[6], 10, 32)
	ps.flags = uint(flags)

	ps.minflt, _ = strconv.ParseUint(fields[7], 10, 64)
	ps.cminflt, _ = strconv.ParseUint(fields[8], 10, 64)
	ps.majflt, _ = strconv.ParseUint(fields[9], 10, 64)
	ps.cmajflt, _ = strconv.ParseUint(fields[10], 10, 64)
	ps.utime, _ = strconv.ParseUint(fields[11], 10, 64)
	ps.stime, _ = strconv.ParseUint(fields[12], 10, 64)
	ps.cutime, _ = strconv.ParseInt(fields[13], 10, 64)
	ps.cstime, _ = strconv.ParseInt(fields[14], 10, 64)
	ps.priority, _ = strconv.ParseInt(fields[15], 10, 64)
	ps.nice, _ = strconv.ParseInt(fields[16], 10, 64)
	ps.numThreads, _ = strconv.ParseInt(fields[17], 10, 64)

	if len(fields) > 20 {
		ps.vsize, _ = strconv.ParseUint(fields[20], 10, 64)
	}
	if len(fields) > 21 {
		ps.rss, _ = strconv.ParseInt(fields[21], 10, 64)
	}

	return ps, nil
}

// matchesFilter checks if a process matches the given filter
func (p *procfsParser) matchesFilter(metrics *entities.ProcessMetrics, filter *services.ProcessFilter) bool {
	// Check include patterns
	if len(filter.IncludePatterns) > 0 {
		matched := false
		for _, pattern := range filter.IncludePatterns {
			if strings.Contains(metrics.Name(), pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range filter.ExcludePatterns {
		if strings.Contains(metrics.Name(), pattern) {
			return false
		}
	}

	// Check CPU threshold
	if filter.MinCPUThreshold > 0 && metrics.CPUUsage().Value() < filter.MinCPUThreshold {
		return false
	}

	// Check memory threshold
	if filter.MinMemThreshold > 0 && metrics.MemoryUsage().Value() < filter.MinMemThreshold {
		return false
	}

	// Check user filter
	if len(filter.UserFilter) > 0 {
		matched := false
		for _, uid := range filter.UserFilter {
			if metrics.UID() == uid {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// getTopByCPU returns top N processes by CPU usage
func (p *procfsParser) getTopByCPU(processes []*entities.ProcessMetrics, n int) []*entities.ProcessMetrics {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUUsage().Value() > processes[j].CPUUsage().Value()
	})

	if n > len(processes) {
		n = len(processes)
	}

	return processes[:n]
}

// getTopByMemory returns top N processes by memory usage
func (p *procfsParser) getTopByMemory(processes []*entities.ProcessMetrics, n int) []*entities.ProcessMetrics {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].MemoryUsage().Value() > processes[j].MemoryUsage().Value()
	})

	if n > len(processes) {
		n = len(processes)
	}

	return processes[:n]
}

// getTopByDiskIO returns top N processes by disk I/O
func (p *procfsParser) getTopByDiskIO(processes []*entities.ProcessMetrics, n int) []*entities.ProcessMetrics {
	sort.Slice(processes, func(i, j int) bool {
		totalI := processes[i].DiskReadRate().BytesPerSecond() + processes[i].DiskWriteRate().BytesPerSecond()
		totalJ := processes[j].DiskReadRate().BytesPerSecond() + processes[j].DiskWriteRate().BytesPerSecond()
		return totalI > totalJ
	})

	if n > len(processes) {
		n = len(processes)
	}

	return processes[:n]
}

// mapProcessStateToEntity maps process state strings to ProcessState entity
func (p *procfsParser) mapProcessStateToEntity(status []string) entities.ProcessState {
	if len(status) == 0 {
		return entities.ProcessStateUnknown
	}

	switch status[0] {
	case "R":
		return entities.ProcessStateRunning
	case "S":
		return entities.ProcessStateSleeping
	case "D":
		return entities.ProcessStateWaiting
	case "Z":
		return entities.ProcessStateZombie
	case "T", "t":
		return entities.ProcessStateStopped
	case "I":
		return entities.ProcessStateIdle
	default:
		return entities.ProcessStateUnknown
	}
}

// isKernelThread checks if a process is a kernel thread
func (p *procfsParser) isKernelThread(pid int) bool {
	// Kernel threads have no /proc/[pid]/exe
	exePath := filepath.Join(p.procPath, strconv.Itoa(pid), "exe")
	_, err := os.Readlink(exePath)
	return err != nil && os.IsNotExist(err)
}

// hasPermission checks if we have permission to read process info
func (p *procfsParser) hasPermission(pid int) bool {
	ioPath := filepath.Join(p.procPath, strconv.Itoa(pid), "io")
	_, err := os.Stat(ioPath)
	if err != nil {
		return !os.IsPermission(err)
	}
	return true
}

// getProcessOwner gets the owner UID of a process
func (p *procfsParser) getProcessOwner(pid int) (uint32, error) {
	statPath := filepath.Join(p.procPath, strconv.Itoa(pid))
	info, err := os.Stat(statPath)
	if err != nil {
		return 0, err
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Uid, nil
	}

	return 0, fmt.Errorf("failed to get process owner")
}

// getActiveProcesses returns list of active process IDs
func (p *procfsParser) getActiveProcesses() ([]valueobjects.ProcessID, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	var activeProcesses []valueobjects.ProcessID
	for _, pid := range pids {
		if pid > 0 {
			processID, err := valueobjects.NewProcessID(uint32(pid))
			if err == nil {
				activeProcesses = append(activeProcesses, processID)
			}
		}
	}

	return activeProcesses, nil
}

// getProcessInfo returns detailed process information
func (p *procfsParser) getProcessInfo(pid int) (*services.ProcessInfo, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to get process %d: %w", pid, err)
	}

	processID, err := valueobjects.NewProcessID(uint32(pid))
	if err != nil {
		return nil, fmt.Errorf("invalid process ID: %w", err)
	}

	name, _ := proc.Name()
	exePath, _ := proc.Exe()
	cmdLine, _ := proc.CmdlineSlice()
	ppid, _ := proc.Ppid()
	uids, _ := proc.Uids()
	gids, _ := proc.Gids()
	createTime, _ := proc.CreateTime()
	status, _ := proc.Status()
	threadCount, _ := proc.NumThreads()
	nice, _ := proc.Nice()
	openFiles, _ := proc.OpenFiles()
	connections, _ := proc.Connections()

	var parentPID *valueobjects.ProcessID
	if ppid > 0 {
		pid, err := valueobjects.NewProcessID(uint32(ppid))
		if err == nil {
			parentPID = &pid
		}
	}

	var uid, gid uint32
	if len(uids) > 0 {
		uid = uint32(uids[0])
	}
	if len(gids) > 0 {
		gid = uint32(gids[0])
	}

	state := p.mapProcessStateToEntity(status)

	return &services.ProcessInfo{
		PID:         processID,
		Name:        name,
		ExecutePath: exePath,
		CommandLine: cmdLine,
		ParentPID:   parentPID,
		UserID:      uid,
		GroupID:     gid,
		StartTime:   time.Unix(0, createTime*1000000),
		State:       state,
		Threads:     int(threadCount),
		Priority:    0, // Would need to read from /proc/[pid]/stat
		Nice:        int(nice),
		Environment: make(map[string]string), // Would need to read from /proc/[pid]/environ
		OpenFiles:   len(openFiles),
		Connections: len(connections),
	}, nil
}