package entities

import (
	"errors"
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrInvalidCPUPressure = errors.New("invalid CPU pressure metrics")
)

type CPUPressure struct {
	id               string
	timestamp        valueobjects.Timestamp
	some10s          valueobjects.Percentage
	some60s          valueobjects.Percentage
	some300s         valueobjects.Percentage
	full10s          valueobjects.Percentage
	full60s          valueobjects.Percentage
	full300s         valueobjects.Percentage
	totalSomeTime    time.Duration
	totalFullTime    time.Duration
	stallEvents      uint64
	averageStallTime time.Duration
}

type PressureSeverity string

const (
	SeverityNone     PressureSeverity = "NONE"
	SeverityLow      PressureSeverity = "LOW"
	SeverityMedium   PressureSeverity = "MEDIUM"
	SeverityHigh     PressureSeverity = "HIGH"
	SeverityCritical PressureSeverity = "CRITICAL"
)

func severityRank(s PressureSeverity) int {
	switch s {
	case SeverityNone:
		return 0
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return 0
	}
}

func NewCPUPressure(
	some10s, some60s, some300s valueobjects.Percentage,
) (*CPUPressure, error) {
	timestamp := valueobjects.NewTimestampNow()

	cp := &CPUPressure{
		id:            generateCPUPressureID(*timestamp),
		timestamp:     *timestamp,
		some10s:       some10s,
		some60s:       some60s,
		some300s:      some300s,
		full10s:       valueobjects.NewPercentageZero(),
		full60s:       valueobjects.NewPercentageZero(),
		full300s:      valueobjects.NewPercentageZero(),
		totalSomeTime: 0,
		totalFullTime: 0,
		stallEvents:   0,
	}

	if err := cp.validate(); err != nil {
		return nil, err
	}

	return cp, nil
}

func NewCPUPressureWithFull(
	some10s, some60s, some300s valueobjects.Percentage,
	full10s, full60s, full300s valueobjects.Percentage,
) (*CPUPressure, error) {
	timestamp := valueobjects.NewTimestampNow()

	cp := &CPUPressure{
		id:            generateCPUPressureID(*timestamp),
		timestamp:     *timestamp,
		some10s:       some10s,
		some60s:       some60s,
		some300s:      some300s,
		full10s:       full10s,
		full60s:       full60s,
		full300s:      full300s,
		totalSomeTime: 0,
		totalFullTime: 0,
		stallEvents:   0,
	}

	if err := cp.validate(); err != nil {
		return nil, err
	}

	return cp, nil
}

func generateCPUPressureID(timestamp valueobjects.Timestamp) string {
	return fmt.Sprintf("cp_%d", timestamp.UnixMilli())
}

func (cp *CPUPressure) validate() error {
	if cp.id == "" {
		return ErrInvalidCPUPressure
	}
	return nil
}

func (cp *CPUPressure) ID() string {
	return cp.id
}

func (cp *CPUPressure) Timestamp() valueobjects.Timestamp {
	return cp.timestamp
}

func (cp *CPUPressure) Some10s() valueobjects.Percentage {
	return cp.some10s
}

func (cp *CPUPressure) Some60s() valueobjects.Percentage {
	return cp.some60s
}

func (cp *CPUPressure) Some300s() valueobjects.Percentage {
	return cp.some300s
}

func (cp *CPUPressure) Full10s() valueobjects.Percentage {
	return cp.full10s
}

func (cp *CPUPressure) Full60s() valueobjects.Percentage {
	return cp.full60s
}

func (cp *CPUPressure) Full300s() valueobjects.Percentage {
	return cp.full300s
}

func (cp *CPUPressure) TotalSomeTime() time.Duration {
	return cp.totalSomeTime
}

func (cp *CPUPressure) TotalFullTime() time.Duration {
	return cp.totalFullTime
}

func (cp *CPUPressure) StallEvents() uint64 {
	return cp.stallEvents
}

func (cp *CPUPressure) AverageStallTime() time.Duration {
	return cp.averageStallTime
}

func (cp *CPUPressure) SetStallStatistics(totalSome, totalFull time.Duration, events uint64) {
	cp.totalSomeTime = totalSome
	cp.totalFullTime = totalFull
	cp.stallEvents = events
	if events > 0 {
		cp.averageStallTime = time.Duration(int64(totalSome) / int64(events))
	}
}

func (cp *CPUPressure) GetSomeSeverity() PressureSeverity {
	return cp.calculateSeverity(cp.some10s, cp.some60s, cp.some300s)
}

func (cp *CPUPressure) GetFullSeverity() PressureSeverity {
	return cp.calculateSeverity(cp.full10s, cp.full60s, cp.full300s)
}

func (cp *CPUPressure) GetOverallSeverity() PressureSeverity {
	someSeverity := cp.GetSomeSeverity()
	fullSeverity := cp.GetFullSeverity()

	if severityRank(fullSeverity) > severityRank(someSeverity) {
		return fullSeverity
	}
	return someSeverity
}

func (cp *CPUPressure) calculateSeverity(p10s, p60s, p300s valueobjects.Percentage) PressureSeverity {
	if p10s.Value() >= 80 || p60s.Value() >= 70 || p300s.Value() >= 60 {
		return SeverityCritical
	}
	if p10s.Value() >= 60 || p60s.Value() >= 50 || p300s.Value() >= 40 {
		return SeverityHigh
	}
	if p10s.Value() >= 40 || p60s.Value() >= 30 || p300s.Value() >= 25 {
		return SeverityMedium
	}
	if p10s.Value() >= 20 || p60s.Value() >= 15 || p300s.Value() >= 10 {
		return SeverityLow
	}
	return SeverityNone
}

func (cp *CPUPressure) IsUnderPressure() bool {
	return severityRank(cp.GetOverallSeverity()) >= severityRank(SeverityMedium)
}

func (cp *CPUPressure) IsCritical() bool {
	return cp.GetOverallSeverity() == SeverityCritical
}

func (cp *CPUPressure) GetTrend() string {
	if cp.some10s.GreaterThan(cp.some60s) && cp.some60s.GreaterThan(cp.some300s) {
		return "INCREASING"
	}
	if cp.some10s.LessThan(cp.some60s) && cp.some60s.LessThan(cp.some300s) {
		return "DECREASING"
	}
	return "STABLE"
}

func (cp *CPUPressure) CalculatePressureScore() float64 {
	someScore := (cp.some10s.Value()*0.5 + cp.some60s.Value()*0.3 + cp.some300s.Value()*0.2) * 0.6
	fullScore := (cp.full10s.Value()*0.5 + cp.full60s.Value()*0.3 + cp.full300s.Value()*0.2) * 0.4
	return someScore + fullScore
}

func (cp *CPUPressure) CalculateDelta(previous *CPUPressure) (*CPUPressureDelta, error) {
	if previous == nil {
		return nil, errors.New("previous pressure cannot be nil")
	}

	if !cp.timestamp.After(previous.timestamp) {
		return nil, errors.New("current pressure must be newer than previous")
	}

	duration := cp.timestamp.Sub(previous.timestamp)

	return &CPUPressureDelta{
		Duration:       duration,
		Some10sDelta:   cp.some10s.Value() - previous.some10s.Value(),
		Some60sDelta:   cp.some60s.Value() - previous.some60s.Value(),
		Some300sDelta:  cp.some300s.Value() - previous.some300s.Value(),
		Full10sDelta:   cp.full10s.Value() - previous.full10s.Value(),
		Full60sDelta:   cp.full60s.Value() - previous.full60s.Value(),
		Full300sDelta:  cp.full300s.Value() - previous.full300s.Value(),
		StallTimeDelta: cp.totalSomeTime - previous.totalSomeTime,
		EventsDelta:    int64(cp.stallEvents) - int64(previous.stallEvents),
	}, nil
}

func (cp *CPUPressure) GetRecommendation() string {
	severity := cp.GetOverallSeverity()
	trend := cp.GetTrend()

	switch severity {
	case SeverityCritical:
		if trend == "INCREASING" {
			return "URGENT: System is critically overloaded and deteriorating. Immediate intervention required."
		}
		return "CRITICAL: System is under severe CPU pressure. Consider reducing workload or adding resources."

	case SeverityHigh:
		if trend == "INCREASING" {
			return "WARNING: High CPU pressure increasing. Monitor closely and prepare to scale."
		}
		return "HIGH: Significant CPU pressure detected. Consider optimizing workloads."

	case SeverityMedium:
		if trend == "INCREASING" {
			return "CAUTION: Moderate CPU pressure trending upward. Review resource allocation."
		}
		return "MODERATE: Some CPU pressure present. System functioning but could benefit from optimization."

	case SeverityLow:
		return "LOW: Minor CPU pressure detected. System performing adequately."

	default:
		return "NORMAL: No significant CPU pressure. System resources are sufficient."
	}
}

func (cp *CPUPressure) Clone() *CPUPressure {
	return &CPUPressure{
		id:               cp.id,
		timestamp:        cp.timestamp,
		some10s:          cp.some10s,
		some60s:          cp.some60s,
		some300s:         cp.some300s,
		full10s:          cp.full10s,
		full60s:          cp.full60s,
		full300s:         cp.full300s,
		totalSomeTime:    cp.totalSomeTime,
		totalFullTime:    cp.totalFullTime,
		stallEvents:      cp.stallEvents,
		averageStallTime: cp.averageStallTime,
	}
}

func (cp *CPUPressure) String() string {
	return fmt.Sprintf("CPUPressure[Some(10s:%.1f%% 60s:%.1f%% 300s:%.1f%%) Full(10s:%.1f%% 60s:%.1f%% 300s:%.1f%%) Severity:%s Trend:%s]",
		cp.some10s.Value(),
		cp.some60s.Value(),
		cp.some300s.Value(),
		cp.full10s.Value(),
		cp.full60s.Value(),
		cp.full300s.Value(),
		cp.GetOverallSeverity(),
		cp.GetTrend(),
	)
}

type CPUPressureDelta struct {
	Duration       time.Duration
	Some10sDelta   float64
	Some60sDelta   float64
	Some300sDelta  float64
	Full10sDelta   float64
	Full60sDelta   float64
	Full300sDelta  float64
	StallTimeDelta time.Duration
	EventsDelta    int64
}

func (d *CPUPressureDelta) IsImproving() bool {
	return d.Some10sDelta < 0 && d.Some60sDelta < 0 && d.Some300sDelta < 0
}

func (d *CPUPressureDelta) IsDeteriorating() bool {
	return d.Some10sDelta > 5 || d.Some60sDelta > 5 || d.Some300sDelta > 5
}

func (d *CPUPressureDelta) GetTrend() string {
	if d.IsImproving() {
		return "IMPROVING"
	}
	if d.IsDeteriorating() {
		return "DETERIORATING"
	}
	return "STABLE"
}