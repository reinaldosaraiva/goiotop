package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrMetricsNotFound    = errors.New("metrics not found")
	ErrInvalidTimeRange   = errors.New("invalid time range")
	ErrRepositoryError    = errors.New("repository error")
	ErrDuplicateMetrics   = errors.New("duplicate metrics")
	ErrStorageUnavailable = errors.New("storage unavailable")
)

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func NewTimeRange(start, end time.Time) (*TimeRange, error) {
	if start.After(end) {
		return nil, ErrInvalidTimeRange
	}
	return &TimeRange{
		Start: start,
		End:   end,
	}, nil
}

func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

func (tr TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}

type MetricsFilter struct {
	TimeRange   *TimeRange
	ProcessIDs  []valueobjects.ProcessID
	DeviceNames []string
	Limit       int
	Offset      int
}

type AggregationType string

const (
	AggregationAverage AggregationType = "average"
	AggregationMax     AggregationType = "max"
	AggregationMin     AggregationType = "min"
	AggregationSum     AggregationType = "sum"
	AggregationCount   AggregationType = "count"
)

type AggregationInterval string

const (
	IntervalSecond  AggregationInterval = "1s"
	IntervalMinute  AggregationInterval = "1m"
	IntervalFiveMin AggregationInterval = "5m"
	IntervalHour    AggregationInterval = "1h"
	IntervalDay     AggregationInterval = "1d"
)

type AggregationRequest struct {
	Type     AggregationType
	Interval AggregationInterval
	Filter   MetricsFilter
}

type SystemMetricsAggregation struct {
	TimeRange       TimeRange
	Count           int
	AvgCPUUsage     float64
	MaxCPUUsage     float64
	MinCPUUsage     float64
	AvgMemoryUsage  float64
	MaxMemoryUsage  float64
	MinMemoryUsage  float64
	AvgDiskReadRate float64
	MaxDiskReadRate float64
	AvgDiskWriteRate float64
	MaxDiskWriteRate float64
}

type ProcessMetricsAggregation struct {
	ProcessID       valueobjects.ProcessID
	ProcessName     string
	TimeRange       TimeRange
	Count           int
	AvgCPUUsage     float64
	MaxCPUUsage     float64
	AvgMemoryUsage  float64
	MaxMemoryUsage  float64
	TotalDiskRead   uint64
	TotalDiskWrite  uint64
}

type MetricsRepository interface {
	SaveSystemMetrics(ctx context.Context, metrics *entities.SystemMetrics) error

	GetSystemMetrics(ctx context.Context, id string) (*entities.SystemMetrics, error)

	GetLatestSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error)

	QuerySystemMetrics(ctx context.Context, filter MetricsFilter) ([]*entities.SystemMetrics, error)

	DeleteSystemMetrics(ctx context.Context, id string) error

	SaveProcessMetrics(ctx context.Context, metrics *entities.ProcessMetrics) error

	GetProcessMetrics(ctx context.Context, pid valueobjects.ProcessID, timestamp time.Time) (*entities.ProcessMetrics, error)

	GetLatestProcessMetrics(ctx context.Context, pid valueobjects.ProcessID) (*entities.ProcessMetrics, error)

	QueryProcessMetrics(ctx context.Context, filter MetricsFilter) ([]*entities.ProcessMetrics, error)

	DeleteProcessMetrics(ctx context.Context, pid valueobjects.ProcessID, timestamp time.Time) error

	SaveCPUPressure(ctx context.Context, pressure *entities.CPUPressure) error

	GetCPUPressure(ctx context.Context, id string) (*entities.CPUPressure, error)

	GetLatestCPUPressure(ctx context.Context) (*entities.CPUPressure, error)

	QueryCPUPressure(ctx context.Context, filter MetricsFilter) ([]*entities.CPUPressure, error)

	DeleteCPUPressure(ctx context.Context, id string) error

	SaveDiskIO(ctx context.Context, diskIO *entities.DiskIO) error

	GetDiskIO(ctx context.Context, deviceName string, timestamp time.Time) (*entities.DiskIO, error)

	GetLatestDiskIO(ctx context.Context, deviceName string) (*entities.DiskIO, error)

	QueryDiskIO(ctx context.Context, filter MetricsFilter) ([]*entities.DiskIO, error)

	DeleteDiskIO(ctx context.Context, deviceName string, timestamp time.Time) error

	AggregateSystemMetrics(ctx context.Context, request AggregationRequest) (*SystemMetricsAggregation, error)

	AggregateProcessMetrics(ctx context.Context, pid valueobjects.ProcessID, request AggregationRequest) (*ProcessMetricsAggregation, error)

	GetTopProcessesByCPU(ctx context.Context, limit int, timeRange *TimeRange) ([]*entities.ProcessMetrics, error)

	GetTopProcessesByMemory(ctx context.Context, limit int, timeRange *TimeRange) ([]*entities.ProcessMetrics, error)

	GetTopProcessesByDiskIO(ctx context.Context, limit int, timeRange *TimeRange) ([]*entities.ProcessMetrics, error)

	GetHistoricalSystemMetrics(ctx context.Context, duration time.Duration) ([]*entities.SystemMetrics, error)

	GetHistoricalProcessMetrics(ctx context.Context, pid valueobjects.ProcessID, duration time.Duration) ([]*entities.ProcessMetrics, error)

	CountSystemMetrics(ctx context.Context, filter MetricsFilter) (int64, error)

	CountProcessMetrics(ctx context.Context, filter MetricsFilter) (int64, error)

	PurgeOldMetrics(ctx context.Context, before time.Time) error

	GetStorageStats(ctx context.Context) (*StorageStats, error)

	BeginTransaction(ctx context.Context) (Transaction, error)

	Ping(ctx context.Context) error

	Close() error
}

type StorageStats struct {
	SystemMetricsCount   int64
	ProcessMetricsCount  int64
	CPUPressureCount     int64
	DiskIOCount          int64
	TotalSize            int64
	OldestTimestamp      time.Time
	NewestTimestamp      time.Time
	CompressionRatio     float64
	AverageQueryTime     time.Duration
}

type Transaction interface {
	Commit() error
	Rollback() error
	SaveSystemMetrics(metrics *entities.SystemMetrics) error
	SaveProcessMetrics(metrics *entities.ProcessMetrics) error
	SaveCPUPressure(pressure *entities.CPUPressure) error
	SaveDiskIO(diskIO *entities.DiskIO) error
}

type MetricsSummary struct {
	TimeRange            TimeRange
	SystemMetricsCount   int
	ProcessMetricsCount  int
	UniqueProcessCount   int
	AvgCPUUsage          float64
	AvgMemoryUsage       float64
	PeakCPUUsage         float64
	PeakMemoryUsage      float64
	TotalDiskRead        uint64
	TotalDiskWrite       uint64
	TopCPUProcess        *entities.ProcessMetrics
	TopMemoryProcess     *entities.ProcessMetrics
	TopDiskIOProcess     *entities.ProcessMetrics
}

type QueryOptions struct {
	SortBy    string
	SortOrder string
	GroupBy   string
	Having    string
}