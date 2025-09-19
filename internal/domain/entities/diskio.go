package entities

import (
	"errors"
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrInvalidDiskIO   = errors.New("invalid disk I/O metrics")
	ErrDeviceNameEmpty = errors.New("device name cannot be empty")
)

type DiskIO struct {
	id              string
	deviceName      string
	timestamp       valueobjects.Timestamp
	readRate        valueobjects.ByteRate
	writeRate       valueobjects.ByteRate
	readIOPS        float64
	writeIOPS       float64
	readLatency     time.Duration
	writeLatency    time.Duration
	queueDepth      int
	utilization     valueobjects.Percentage
	totalReadBytes  uint64
	totalWriteBytes uint64
	totalReadOps    uint64
	totalWriteOps   uint64
	mergedReads     uint64
	mergedWrites    uint64
	sectorSize      uint32
}

type IOPerformance string

const (
	IOPerformanceExcellent IOPerformance = "EXCELLENT"
	IOPerformanceGood      IOPerformance = "GOOD"
	IOPerformanceFair      IOPerformance = "FAIR"
	IOPerformancePoor      IOPerformance = "POOR"
	IOPerformanceCritical  IOPerformance = "CRITICAL"
)

func NewDiskIO(
	deviceName string,
	readRate, writeRate valueobjects.ByteRate,
) (*DiskIO, error) {
	if deviceName == "" {
		return nil, ErrDeviceNameEmpty
	}

	timestamp := valueobjects.NewTimestampNow()

	dio := &DiskIO{
		id:           generateDiskIOID(deviceName, *timestamp),
		deviceName:   deviceName,
		timestamp:    *timestamp,
		readRate:     readRate,
		writeRate:    writeRate,
		utilization:  valueobjects.NewPercentageZero(),
		sectorSize:   512,
	}

	if err := dio.validate(); err != nil {
		return nil, err
	}

	return dio, nil
}

func NewDiskIOWithIOPS(
	deviceName string,
	readRate, writeRate valueobjects.ByteRate,
	readIOPS, writeIOPS float64,
) (*DiskIO, error) {
	dio, err := NewDiskIO(deviceName, readRate, writeRate)
	if err != nil {
		return nil, err
	}

	dio.readIOPS = readIOPS
	dio.writeIOPS = writeIOPS

	return dio, nil
}

func generateDiskIOID(deviceName string, timestamp valueobjects.Timestamp) string {
	return fmt.Sprintf("dio_%s_%d", deviceName, timestamp.UnixMilli())
}

func (d *DiskIO) validate() error {
	if d.id == "" {
		return ErrInvalidDiskIO
	}
	if d.deviceName == "" {
		return ErrDeviceNameEmpty
	}
	return nil
}

func (d *DiskIO) ID() string {
	return d.id
}

func (d *DiskIO) DeviceName() string {
	return d.deviceName
}

func (d *DiskIO) Timestamp() valueobjects.Timestamp {
	return d.timestamp
}

func (d *DiskIO) ReadRate() valueobjects.ByteRate {
	return d.readRate
}

func (d *DiskIO) WriteRate() valueobjects.ByteRate {
	return d.writeRate
}

func (d *DiskIO) TotalRate() valueobjects.ByteRate {
	return d.readRate.Add(d.writeRate)
}

func (d *DiskIO) ReadIOPS() float64 {
	return d.readIOPS
}

func (d *DiskIO) WriteIOPS() float64 {
	return d.writeIOPS
}

func (d *DiskIO) TotalIOPS() float64 {
	return d.readIOPS + d.writeIOPS
}

func (d *DiskIO) ReadLatency() time.Duration {
	return d.readLatency
}

func (d *DiskIO) WriteLatency() time.Duration {
	return d.writeLatency
}

func (d *DiskIO) AverageLatency() time.Duration {
	if d.readLatency == 0 && d.writeLatency == 0 {
		return 0
	}
	if d.readLatency == 0 {
		return d.writeLatency
	}
	if d.writeLatency == 0 {
		return d.readLatency
	}
	return (d.readLatency + d.writeLatency) / 2
}

func (d *DiskIO) QueueDepth() int {
	return d.queueDepth
}

func (d *DiskIO) Utilization() valueobjects.Percentage {
	return d.utilization
}

func (d *DiskIO) TotalReadBytes() uint64 {
	return d.totalReadBytes
}

func (d *DiskIO) TotalWriteBytes() uint64 {
	return d.totalWriteBytes
}

func (d *DiskIO) TotalBytes() uint64 {
	return d.totalReadBytes + d.totalWriteBytes
}

func (d *DiskIO) SetIOPS(readIOPS, writeIOPS float64) {
	d.readIOPS = readIOPS
	d.writeIOPS = writeIOPS
}

func (d *DiskIO) SetLatency(readLatency, writeLatency time.Duration) {
	d.readLatency = readLatency
	d.writeLatency = writeLatency
}

func (d *DiskIO) SetQueueDepth(depth int) {
	d.queueDepth = depth
}

func (d *DiskIO) SetUtilization(utilization valueobjects.Percentage) {
	d.utilization = utilization
}

func (d *DiskIO) SetTotalStatistics(readBytes, writeBytes, readOps, writeOps uint64) {
	d.totalReadBytes = readBytes
	d.totalWriteBytes = writeBytes
	d.totalReadOps = readOps
	d.totalWriteOps = writeOps
}

func (d *DiskIO) SetMergedOperations(mergedReads, mergedWrites uint64) {
	d.mergedReads = mergedReads
	d.mergedWrites = mergedWrites
}

func (d *DiskIO) SetSectorSize(size uint32) {
	d.sectorSize = size
}

func (d *DiskIO) UpdateRates(readRate, writeRate valueobjects.ByteRate) {
	d.readRate = readRate
	d.writeRate = writeRate
	d.timestamp = *valueobjects.NewTimestampNow()
}

func (d *DiskIO) UpdateIOPS(readIOPS, writeIOPS float64) {
	d.readIOPS = readIOPS
	d.writeIOPS = writeIOPS
	d.timestamp = *valueobjects.NewTimestampNow()
}

func (d *DiskIO) CalculateEfficiency() float64 {
	if d.TotalIOPS() == 0 {
		return 0
	}

	totalMerged := float64(d.mergedReads + d.mergedWrites)
	totalOps := float64(d.totalReadOps + d.totalWriteOps)

	if totalOps == 0 {
		return 0
	}

	mergeRatio := totalMerged / totalOps

	averageOpSize := d.TotalRate().BytesPerSecond() / d.TotalIOPS()
	optimalOpSize := float64(128 * 1024)
	sizeEfficiency := averageOpSize / optimalOpSize
	if sizeEfficiency > 1 {
		sizeEfficiency = 1
	}

	efficiency := (mergeRatio*0.3 + sizeEfficiency*0.7) * 100
	if efficiency > 100 {
		efficiency = 100
	}

	return efficiency
}

func (d *DiskIO) GetPerformanceLevel() IOPerformance {
	if d.utilization.IsCritical() || d.AverageLatency() > 100*time.Millisecond {
		return IOPerformanceCritical
	}

	if d.utilization.IsWarning() || d.AverageLatency() > 50*time.Millisecond {
		return IOPerformancePoor
	}

	if d.utilization.IsHigh(50) || d.AverageLatency() > 20*time.Millisecond {
		return IOPerformanceFair
	}

	if d.AverageLatency() < 5*time.Millisecond && d.utilization.IsLow(30) {
		return IOPerformanceExcellent
	}

	return IOPerformanceGood
}

func (d *DiskIO) IsHighUtilization() bool {
	return d.utilization.IsHigh(80)
}

func (d *DiskIO) IsHighLatency() bool {
	return d.AverageLatency() > 50*time.Millisecond
}

func (d *DiskIO) IsHighThroughput() bool {
	const highThroughputThreshold = 100 * 1024 * 1024
	return d.TotalRate().BytesPerSecond() > highThroughputThreshold
}

func (d *DiskIO) IsSSD() bool {
	return d.readIOPS > 1000 || d.writeIOPS > 1000 ||
		   d.AverageLatency() < time.Millisecond
}

func (d *DiskIO) CalculateDelta(previous *DiskIO) (*DiskIODelta, error) {
	if previous == nil {
		return nil, errors.New("previous disk I/O cannot be nil")
	}

	if d.deviceName != previous.deviceName {
		return nil, errors.New("device names must match")
	}

	if !d.timestamp.After(previous.timestamp) {
		return nil, errors.New("current metrics must be newer than previous")
	}

	duration := d.timestamp.Sub(previous.timestamp)

	readRateDelta := d.readRate.BytesPerSecond() - previous.readRate.BytesPerSecond()
	writeRateDelta := d.writeRate.BytesPerSecond() - previous.writeRate.BytesPerSecond()

	return &DiskIODelta{
		DeviceName:        d.deviceName,
		Duration:          duration,
		ReadRateDelta:     readRateDelta,
		WriteRateDelta:    writeRateDelta,
		ReadIOPSDelta:     d.readIOPS - previous.readIOPS,
		WriteIOPSDelta:    d.writeIOPS - previous.writeIOPS,
		ReadLatencyDelta:  d.readLatency - previous.readLatency,
		WriteLatencyDelta: d.writeLatency - previous.writeLatency,
		UtilizationDelta:  d.utilization.Value() - previous.utilization.Value(),
	}, nil
}

func (d *DiskIO) GetRecommendation() string {
	perf := d.GetPerformanceLevel()

	switch perf {
	case IOPerformanceCritical:
		if d.IsHighLatency() {
			return "CRITICAL: Extremely high disk latency detected. Check for hardware issues or excessive load."
		}
		return "CRITICAL: Disk is at maximum capacity. Consider load distribution or hardware upgrade."

	case IOPerformancePoor:
		if d.queueDepth > 32 {
			return "POOR: High queue depth indicates I/O bottleneck. Consider optimizing applications or adding storage."
		}
		return "POOR: Disk performance degraded. Monitor for continued deterioration."

	case IOPerformanceFair:
		return "FAIR: Moderate disk load. Performance acceptable but could be optimized."

	case IOPerformanceGood:
		return "GOOD: Disk performing well within normal parameters."

	case IOPerformanceExcellent:
		return "EXCELLENT: Optimal disk performance with low latency and utilization."

	default:
		return "UNKNOWN: Unable to determine performance characteristics."
	}
}

func (d *DiskIO) Clone() *DiskIO {
	return &DiskIO{
		id:              d.id,
		deviceName:      d.deviceName,
		timestamp:       d.timestamp,
		readRate:        d.readRate,
		writeRate:       d.writeRate,
		readIOPS:        d.readIOPS,
		writeIOPS:       d.writeIOPS,
		readLatency:     d.readLatency,
		writeLatency:    d.writeLatency,
		queueDepth:      d.queueDepth,
		utilization:     d.utilization,
		totalReadBytes:  d.totalReadBytes,
		totalWriteBytes: d.totalWriteBytes,
		totalReadOps:    d.totalReadOps,
		totalWriteOps:   d.totalWriteOps,
		mergedReads:     d.mergedReads,
		mergedWrites:    d.mergedWrites,
		sectorSize:      d.sectorSize,
	}
}

func (d *DiskIO) String() string {
	return fmt.Sprintf("DiskIO[%s] Read:%s Write:%s IOPS:%.0f/%.0f Latency:%v/%v Util:%s Perf:%s",
		d.deviceName,
		d.readRate.HumanReadable(),
		d.writeRate.HumanReadable(),
		d.readIOPS,
		d.writeIOPS,
		d.readLatency,
		d.writeLatency,
		d.utilization.String(),
		d.GetPerformanceLevel(),
	)
}

type DiskIODelta struct {
	DeviceName        string
	Duration          time.Duration
	ReadRateDelta     float64
	WriteRateDelta    float64
	ReadIOPSDelta     float64
	WriteIOPSDelta    float64
	ReadLatencyDelta  time.Duration
	WriteLatencyDelta time.Duration
	UtilizationDelta  float64
}

func (d *DiskIODelta) IsImproving() bool {
	return d.ReadLatencyDelta < 0 && d.WriteLatencyDelta < 0 && d.UtilizationDelta < 0
}

func (d *DiskIODelta) IsDeteriorating() bool {
	return d.ReadLatencyDelta > 10*time.Millisecond ||
		   d.WriteLatencyDelta > 10*time.Millisecond ||
		   d.UtilizationDelta > 10
}

func (d *DiskIODelta) GetTrend() string {
	if d.IsImproving() {
		return "IMPROVING"
	}
	if d.IsDeteriorating() {
		return "DETERIORATING"
	}
	return "STABLE"
}