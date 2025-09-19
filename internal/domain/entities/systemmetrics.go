package entities

import (
	"errors"
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrInvalidSystemMetrics = errors.New("invalid system metrics")
)

type SystemMetrics struct {
	id             string
	timestamp      valueobjects.Timestamp
	cpuUsage       valueobjects.Percentage
	memoryUsage    valueobjects.Percentage
	diskReadRate   valueobjects.ByteRate
	diskWriteRate  valueobjects.ByteRate
	networkInRate  valueobjects.ByteRate
	networkOutRate valueobjects.ByteRate
	loadAverage1   float64
	loadAverage5   float64
	loadAverage15  float64
	activeProcesses int
	totalProcesses  int
}

func NewSystemMetrics(
	cpuUsage valueobjects.Percentage,
	memoryUsage valueobjects.Percentage,
	diskReadRate valueobjects.ByteRate,
	diskWriteRate valueobjects.ByteRate,
) (*SystemMetrics, error) {
	timestamp := valueobjects.NewTimestampNow()

	sm := &SystemMetrics{
		id:            generateSystemMetricsID(*timestamp),
		timestamp:     *timestamp,
		cpuUsage:      cpuUsage,
		memoryUsage:   memoryUsage,
		diskReadRate:  diskReadRate,
		diskWriteRate: diskWriteRate,
		networkInRate: valueobjects.NewByteRateZero(),
		networkOutRate: valueobjects.NewByteRateZero(),
	}

	if err := sm.validate(); err != nil {
		return nil, err
	}

	return sm, nil
}

func NewSystemMetricsWithNetwork(
	cpuUsage valueobjects.Percentage,
	memoryUsage valueobjects.Percentage,
	diskReadRate valueobjects.ByteRate,
	diskWriteRate valueobjects.ByteRate,
	networkInRate valueobjects.ByteRate,
	networkOutRate valueobjects.ByteRate,
) (*SystemMetrics, error) {
	timestamp := valueobjects.NewTimestampNow()

	sm := &SystemMetrics{
		id:             generateSystemMetricsID(*timestamp),
		timestamp:      *timestamp,
		cpuUsage:       cpuUsage,
		memoryUsage:    memoryUsage,
		diskReadRate:   diskReadRate,
		diskWriteRate:  diskWriteRate,
		networkInRate:  networkInRate,
		networkOutRate: networkOutRate,
	}

	if err := sm.validate(); err != nil {
		return nil, err
	}

	return sm, nil
}

func generateSystemMetricsID(timestamp valueobjects.Timestamp) string {
	return fmt.Sprintf("sm_%d", timestamp.UnixMilli())
}

func (sm *SystemMetrics) validate() error {
	if sm.id == "" {
		return ErrInvalidSystemMetrics
	}
	return nil
}

func (sm *SystemMetrics) ID() string {
	return sm.id
}

func (sm *SystemMetrics) Timestamp() valueobjects.Timestamp {
	return sm.timestamp
}

func (sm *SystemMetrics) CPUUsage() valueobjects.Percentage {
	return sm.cpuUsage
}

func (sm *SystemMetrics) MemoryUsage() valueobjects.Percentage {
	return sm.memoryUsage
}

func (sm *SystemMetrics) DiskReadRate() valueobjects.ByteRate {
	return sm.diskReadRate
}

func (sm *SystemMetrics) DiskWriteRate() valueobjects.ByteRate {
	return sm.diskWriteRate
}

func (sm *SystemMetrics) NetworkInRate() valueobjects.ByteRate {
	return sm.networkInRate
}

func (sm *SystemMetrics) NetworkOutRate() valueobjects.ByteRate {
	return sm.networkOutRate
}

func (sm *SystemMetrics) TotalDiskIO() valueobjects.ByteRate {
	return sm.diskReadRate.Add(sm.diskWriteRate)
}

func (sm *SystemMetrics) TotalNetworkIO() valueobjects.ByteRate {
	return sm.networkInRate.Add(sm.networkOutRate)
}

func (sm *SystemMetrics) LoadAverage() (float64, float64, float64) {
	return sm.loadAverage1, sm.loadAverage5, sm.loadAverage15
}

func (sm *SystemMetrics) ProcessCount() (int, int) {
	return sm.activeProcesses, sm.totalProcesses
}

func (sm *SystemMetrics) SetLoadAverage(avg1, avg5, avg15 float64) {
	sm.loadAverage1 = avg1
	sm.loadAverage5 = avg5
	sm.loadAverage15 = avg15
}

func (sm *SystemMetrics) SetProcessCount(active, total int) {
	sm.activeProcesses = active
	sm.totalProcesses = total
}

func (sm *SystemMetrics) UpdateCPUUsage(usage valueobjects.Percentage) {
	sm.cpuUsage = usage
	sm.timestamp = *valueobjects.NewTimestampNow()
}

func (sm *SystemMetrics) UpdateMemoryUsage(usage valueobjects.Percentage) {
	sm.memoryUsage = usage
	sm.timestamp = *valueobjects.NewTimestampNow()
}

func (sm *SystemMetrics) UpdateDiskIO(readRate, writeRate valueobjects.ByteRate) {
	sm.diskReadRate = readRate
	sm.diskWriteRate = writeRate
	sm.timestamp = *valueobjects.NewTimestampNow()
}

func (sm *SystemMetrics) UpdateNetworkIO(inRate, outRate valueobjects.ByteRate) {
	sm.networkInRate = inRate
	sm.networkOutRate = outRate
	sm.timestamp = *valueobjects.NewTimestampNow()
}

func (sm *SystemMetrics) CalculateDelta(previous *SystemMetrics) (*SystemMetricsDelta, error) {
	if previous == nil {
		return nil, errors.New("previous metrics cannot be nil")
	}

	if !sm.timestamp.After(previous.timestamp) {
		return nil, errors.New("current metrics must be newer than previous")
	}

	duration := sm.timestamp.Sub(previous.timestamp)

	cpuDelta := sm.cpuUsage.Value() - previous.cpuUsage.Value()
	memoryDelta := sm.memoryUsage.Value() - previous.memoryUsage.Value()

	diskReadDelta := sm.diskReadRate.BytesPerSecond() - previous.diskReadRate.BytesPerSecond()
	diskWriteDelta := sm.diskWriteRate.BytesPerSecond() - previous.diskWriteRate.BytesPerSecond()

	networkInDelta := sm.networkInRate.BytesPerSecond() - previous.networkInRate.BytesPerSecond()
	networkOutDelta := sm.networkOutRate.BytesPerSecond() - previous.networkOutRate.BytesPerSecond()

	return &SystemMetricsDelta{
		Duration:        duration,
		CPUDelta:        cpuDelta,
		MemoryDelta:     memoryDelta,
		DiskReadDelta:   diskReadDelta,
		DiskWriteDelta:  diskWriteDelta,
		NetworkInDelta:  networkInDelta,
		NetworkOutDelta: networkOutDelta,
	}, nil
}

func (sm *SystemMetrics) IsHighLoad() bool {
	return sm.cpuUsage.IsCritical() || sm.memoryUsage.IsCritical()
}

func (sm *SystemMetrics) IsUnderPressure() bool {
	return sm.cpuUsage.IsWarning() || sm.memoryUsage.IsWarning()
}

func (sm *SystemMetrics) GetPressureLevel() string {
	if sm.cpuUsage.IsCritical() || sm.memoryUsage.IsCritical() {
		return "CRITICAL"
	}
	if sm.cpuUsage.IsWarning() || sm.memoryUsage.IsWarning() {
		return "WARNING"
	}
	return "NORMAL"
}

func (sm *SystemMetrics) Clone() *SystemMetrics {
	return &SystemMetrics{
		id:             sm.id,
		timestamp:      sm.timestamp,
		cpuUsage:       sm.cpuUsage,
		memoryUsage:    sm.memoryUsage,
		diskReadRate:   sm.diskReadRate,
		diskWriteRate:  sm.diskWriteRate,
		networkInRate:  sm.networkInRate,
		networkOutRate: sm.networkOutRate,
		loadAverage1:   sm.loadAverage1,
		loadAverage5:   sm.loadAverage5,
		loadAverage15:  sm.loadAverage15,
		activeProcesses: sm.activeProcesses,
		totalProcesses:  sm.totalProcesses,
	}
}

type SystemMetricsDelta struct {
	Duration        time.Duration
	CPUDelta        float64
	MemoryDelta     float64
	DiskReadDelta   float64
	DiskWriteDelta  float64
	NetworkInDelta  float64
	NetworkOutDelta float64
}

func (d *SystemMetricsDelta) IsImproving() bool {
	return d.CPUDelta < 0 && d.MemoryDelta < 0
}

func (d *SystemMetricsDelta) IsDeteriorating() bool {
	return d.CPUDelta > 10 || d.MemoryDelta > 10
}