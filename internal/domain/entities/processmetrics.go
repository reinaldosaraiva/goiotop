package entities

import (
	"errors"
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrInvalidProcessMetrics = errors.New("invalid process metrics")
	ErrProcessNameEmpty     = errors.New("process name cannot be empty")
)

type ProcessMetrics struct {
	processID      valueobjects.ProcessID
	name           string
	timestamp      valueobjects.Timestamp
	cpuUsage       valueobjects.Percentage
	memoryUsage    valueobjects.Percentage
	diskReadRate   valueobjects.ByteRate
	diskWriteRate  valueobjects.ByteRate
	memoryRSS      uint64
	memoryVMS      uint64
	threads        int
	state          ProcessState
	parentPID      *valueobjects.ProcessID
	uid            uint32
	gid            uint32
	priority       int
	nice           int
}

type ProcessState string

const (
	ProcessStateRunning      ProcessState = "R"
	ProcessStateSleeping     ProcessState = "S"
	ProcessStateWaiting      ProcessState = "D"
	ProcessStateStopped      ProcessState = "T"
	ProcessStateZombie       ProcessState = "Z"
	ProcessStateIdle         ProcessState = "I"
	ProcessStateUnknown      ProcessState = "?"
)

func NewProcessMetrics(
	pid valueobjects.ProcessID,
	name string,
	cpuUsage valueobjects.Percentage,
	memoryUsage valueobjects.Percentage,
	diskReadRate valueobjects.ByteRate,
	diskWriteRate valueobjects.ByteRate,
) (*ProcessMetrics, error) {
	if name == "" {
		return nil, ErrProcessNameEmpty
	}

	timestamp := valueobjects.NewTimestampNow()

	pm := &ProcessMetrics{
		processID:     pid,
		name:          name,
		timestamp:     *timestamp,
		cpuUsage:      cpuUsage,
		memoryUsage:   memoryUsage,
		diskReadRate:  diskReadRate,
		diskWriteRate: diskWriteRate,
		state:         ProcessStateUnknown,
		threads:       1,
	}

	if err := pm.validate(); err != nil {
		return nil, err
	}

	return pm, nil
}

func (pm *ProcessMetrics) validate() error {
	if pm.name == "" {
		return ErrProcessNameEmpty
	}
	if !pm.processID.IsValid() {
		return ErrInvalidProcessMetrics
	}
	return nil
}

func (pm *ProcessMetrics) ProcessID() valueobjects.ProcessID {
	return pm.processID
}

func (pm *ProcessMetrics) Name() string {
	return pm.name
}

func (pm *ProcessMetrics) Timestamp() valueobjects.Timestamp {
	return pm.timestamp
}

func (pm *ProcessMetrics) CPUUsage() valueobjects.Percentage {
	return pm.cpuUsage
}

func (pm *ProcessMetrics) MemoryUsage() valueobjects.Percentage {
	return pm.memoryUsage
}

func (pm *ProcessMetrics) DiskReadRate() valueobjects.ByteRate {
	return pm.diskReadRate
}

func (pm *ProcessMetrics) DiskWriteRate() valueobjects.ByteRate {
	return pm.diskWriteRate
}

func (pm *ProcessMetrics) TotalDiskIO() valueobjects.ByteRate {
	return pm.diskReadRate.Add(pm.diskWriteRate)
}

func (pm *ProcessMetrics) MemoryRSS() uint64 {
	return pm.memoryRSS
}

func (pm *ProcessMetrics) MemoryVMS() uint64 {
	return pm.memoryVMS
}

func (pm *ProcessMetrics) Threads() int {
	return pm.threads
}

func (pm *ProcessMetrics) State() ProcessState {
	return pm.state
}

func (pm *ProcessMetrics) ParentPID() *valueobjects.ProcessID {
	return pm.parentPID
}

func (pm *ProcessMetrics) UID() uint32 {
	return pm.uid
}

func (pm *ProcessMetrics) GID() uint32 {
	return pm.gid
}

func (pm *ProcessMetrics) Priority() int {
	return pm.priority
}

func (pm *ProcessMetrics) Nice() int {
	return pm.nice
}

func (pm *ProcessMetrics) SetMemoryDetails(rss, vms uint64) {
	pm.memoryRSS = rss
	pm.memoryVMS = vms
}

func (pm *ProcessMetrics) SetThreads(count int) {
	pm.threads = count
}

func (pm *ProcessMetrics) SetState(state ProcessState) {
	pm.state = state
}

func (pm *ProcessMetrics) SetParentPID(ppid *valueobjects.ProcessID) {
	pm.parentPID = ppid
}

func (pm *ProcessMetrics) SetOwnership(uid, gid uint32) {
	pm.uid = uid
	pm.gid = gid
}

func (pm *ProcessMetrics) SetScheduling(priority, nice int) {
	pm.priority = priority
	pm.nice = nice
}

func (pm *ProcessMetrics) UpdateCPUUsage(usage valueobjects.Percentage) {
	pm.cpuUsage = usage
	pm.timestamp = *valueobjects.NewTimestampNow()
}

func (pm *ProcessMetrics) UpdateMemoryUsage(usage valueobjects.Percentage) {
	pm.memoryUsage = usage
	pm.timestamp = *valueobjects.NewTimestampNow()
}

func (pm *ProcessMetrics) UpdateDiskIO(readRate, writeRate valueobjects.ByteRate) {
	pm.diskReadRate = readRate
	pm.diskWriteRate = writeRate
	pm.timestamp = *valueobjects.NewTimestampNow()
}

func (pm *ProcessMetrics) IsSystemProcess() bool {
	return pm.processID.IsSystemProcess()
}

func (pm *ProcessMetrics) IsKernelThread() bool {
	return pm.processID.IsKernelThread()
}

func (pm *ProcessMetrics) IsRunning() bool {
	return pm.state == ProcessStateRunning
}

func (pm *ProcessMetrics) IsSleeping() bool {
	return pm.state == ProcessStateSleeping
}

func (pm *ProcessMetrics) IsZombie() bool {
	return pm.state == ProcessStateZombie
}

func (pm *ProcessMetrics) IsStopped() bool {
	return pm.state == ProcessStateStopped
}

func (pm *ProcessMetrics) IsHighCPU() bool {
	return pm.cpuUsage.IsHigh(50)
}

func (pm *ProcessMetrics) IsHighMemory() bool {
	return pm.memoryUsage.IsHigh(30)
}

func (pm *ProcessMetrics) IsHighIO() bool {
	const highIOThreshold = 10 * 1024 * 1024
	totalIO := pm.TotalDiskIO()
	return totalIO.BytesPerSecond() > highIOThreshold
}

func (pm *ProcessMetrics) GetResourceIntensity() string {
	if pm.cpuUsage.IsCritical() || pm.memoryUsage.IsCritical() {
		return "CRITICAL"
	}
	if pm.IsHighCPU() || pm.IsHighMemory() || pm.IsHighIO() {
		return "HIGH"
	}
	if pm.cpuUsage.IsWarning() || pm.memoryUsage.IsWarning() {
		return "MEDIUM"
	}
	return "LOW"
}

func (pm *ProcessMetrics) CalculateResourceScore() float64 {
	cpuScore := pm.cpuUsage.Value() * 0.4
	memScore := pm.memoryUsage.Value() * 0.4

	maxIO := 100.0 * 1024 * 1024
	ioScore := (pm.TotalDiskIO().BytesPerSecond() / maxIO) * 100 * 0.2
	if ioScore > 20 {
		ioScore = 20
	}

	return cpuScore + memScore + ioScore
}

func (pm *ProcessMetrics) CompareTo(other *ProcessMetrics) int {
	if other == nil {
		return 1
	}

	thisScore := pm.CalculateResourceScore()
	otherScore := other.CalculateResourceScore()

	if thisScore > otherScore {
		return 1
	} else if thisScore < otherScore {
		return -1
	}
	return 0
}

func (pm *ProcessMetrics) CalculateDelta(previous *ProcessMetrics) (*ProcessMetricsDelta, error) {
	if previous == nil {
		return nil, errors.New("previous metrics cannot be nil")
	}

	if !pm.processID.Equal(previous.processID) {
		return nil, errors.New("process IDs must match")
	}

	if !pm.timestamp.After(previous.timestamp) {
		return nil, errors.New("current metrics must be newer than previous")
	}

	duration := pm.timestamp.Sub(previous.timestamp)

	cpuDelta := pm.cpuUsage.Value() - previous.cpuUsage.Value()
	memoryDelta := pm.memoryUsage.Value() - previous.memoryUsage.Value()

	diskReadDelta := pm.diskReadRate.BytesPerSecond() - previous.diskReadRate.BytesPerSecond()
	diskWriteDelta := pm.diskWriteRate.BytesPerSecond() - previous.diskWriteRate.BytesPerSecond()

	return &ProcessMetricsDelta{
		ProcessID:      pm.processID,
		ProcessName:    pm.name,
		Duration:       duration,
		CPUDelta:       cpuDelta,
		MemoryDelta:    memoryDelta,
		DiskReadDelta:  diskReadDelta,
		DiskWriteDelta: diskWriteDelta,
	}, nil
}

func (pm *ProcessMetrics) Clone() *ProcessMetrics {
	cloned := &ProcessMetrics{
		processID:     pm.processID,
		name:          pm.name,
		timestamp:     pm.timestamp,
		cpuUsage:      pm.cpuUsage,
		memoryUsage:   pm.memoryUsage,
		diskReadRate:  pm.diskReadRate,
		diskWriteRate: pm.diskWriteRate,
		memoryRSS:     pm.memoryRSS,
		memoryVMS:     pm.memoryVMS,
		threads:       pm.threads,
		state:         pm.state,
		uid:           pm.uid,
		gid:           pm.gid,
		priority:      pm.priority,
		nice:          pm.nice,
	}

	if pm.parentPID != nil {
		ppid := *pm.parentPID
		cloned.parentPID = &ppid
	}

	return cloned
}

func (pm *ProcessMetrics) String() string {
	return fmt.Sprintf("Process[%s:%s] CPU:%.1f%% MEM:%.1f%% IO:%s State:%s",
		pm.processID.String(),
		pm.name,
		pm.cpuUsage.Value(),
		pm.memoryUsage.Value(),
		pm.TotalDiskIO().HumanReadable(),
		pm.state,
	)
}

type ProcessMetricsDelta struct {
	ProcessID      valueobjects.ProcessID
	ProcessName    string
	Duration       time.Duration
	CPUDelta       float64
	MemoryDelta    float64
	DiskReadDelta  float64
	DiskWriteDelta float64
}

func (d *ProcessMetricsDelta) IsImproving() bool {
	return d.CPUDelta < 0 && d.MemoryDelta < 0
}

func (d *ProcessMetricsDelta) IsDeteriorating() bool {
	return d.CPUDelta > 5 || d.MemoryDelta > 5
}

func (d *ProcessMetricsDelta) GetTrend() string {
	if d.IsImproving() {
		return "IMPROVING"
	}
	if d.IsDeteriorating() {
		return "DETERIORATING"
	}
	return "STABLE"
}