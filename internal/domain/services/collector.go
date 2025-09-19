package services

import (
	"context"
	"errors"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

var (
	ErrCollectorNotInitialized = errors.New("collector not initialized")
	ErrCollectionFailed        = errors.New("metrics collection failed")
	ErrPlatformNotSupported    = errors.New("platform not supported")
	ErrInsufficientPrivileges = errors.New("insufficient privileges for metric collection")
	ErrCollectorBusy           = errors.New("collector is busy")
)

type CollectionMode string

const (
	CollectionModeRealTime CollectionMode = "realtime"
	CollectionModeBatch    CollectionMode = "batch"
	CollectionModeOnDemand CollectionMode = "ondemand"
)

type CollectionOptions struct {
	Mode              CollectionMode
	Interval          time.Duration
	IncludeSystemMetrics bool
	IncludeProcessMetrics bool
	IncludeCPUPressure   bool
	IncludeDiskIO        bool
	ProcessFilter        ProcessFilter
	DiskFilter           []string
	MaxProcesses         int
	CollectKernelThreads bool
}

type ProcessFilter struct {
	IncludePatterns []string
	ExcludePatterns []string
	MinCPUThreshold float64
	MinMemThreshold float64
	UserFilter      []uint32
}

func DefaultCollectionOptions() CollectionOptions {
	return CollectionOptions{
		Mode:                  CollectionModeRealTime,
		Interval:              time.Second,
		IncludeSystemMetrics:  true,
		IncludeProcessMetrics: true,
		IncludeCPUPressure:    true,
		IncludeDiskIO:         true,
		MaxProcesses:          0,
		CollectKernelThreads:  false,
	}
}

type CollectionResult struct {
	Timestamp        time.Time
	SystemMetrics    *entities.SystemMetrics
	ProcessMetrics   []*entities.ProcessMetrics
	CPUPressure      *entities.CPUPressure
	DiskIO           []*entities.DiskIO
	Errors           []error
	CollectionTime   time.Duration
	ProcessCount     int
	SkippedProcesses int
}

func (cr *CollectionResult) HasErrors() bool {
	return len(cr.Errors) > 0
}

func (cr *CollectionResult) IsComplete() bool {
	return !cr.HasErrors() && cr.SystemMetrics != nil
}

type MetricsCollector interface {
	Initialize(ctx context.Context) error

	CollectSystemMetrics(ctx context.Context) (*entities.SystemMetrics, error)

	CollectProcessMetrics(ctx context.Context, pid valueobjects.ProcessID) (*entities.ProcessMetrics, error)

	CollectAllProcessMetrics(ctx context.Context) ([]*entities.ProcessMetrics, error)

	CollectCPUPressure(ctx context.Context) (*entities.CPUPressure, error)

	CollectDiskIO(ctx context.Context, deviceName string) (*entities.DiskIO, error)

	CollectAllDiskIO(ctx context.Context) ([]*entities.DiskIO, error)

	CollectAll(ctx context.Context, options CollectionOptions) (*CollectionResult, error)

	GetActiveProcesses(ctx context.Context) ([]valueobjects.ProcessID, error)

	GetProcessInfo(ctx context.Context, pid valueobjects.ProcessID) (*ProcessInfo, error)

	GetSystemInfo(ctx context.Context) (*SystemInfo, error)

	GetDiskDevices(ctx context.Context) ([]string, error)

	StartContinuousCollection(ctx context.Context, options CollectionOptions, callback CollectionCallback) error

	StopContinuousCollection(ctx context.Context) error

	IsCollecting() bool

	GetCapabilities() CollectorCapabilities

	GetLastError() error

	Cleanup() error
}

type CollectionCallback func(result *CollectionResult)

type ProcessInfo struct {
	PID         valueobjects.ProcessID
	Name        string
	ExecutePath string
	CommandLine []string
	ParentPID   *valueobjects.ProcessID
	UserID      uint32
	GroupID     uint32
	StartTime   time.Time
	State       entities.ProcessState
	Threads     int
	Priority    int
	Nice        int
	Environment map[string]string
	OpenFiles   int
	Connections int
}

type SystemInfo struct {
	Hostname       string
	OS             string
	Platform       string
	Architecture   string
	CPUModel       string
	CPUCores       int
	CPUThreads     int
	TotalMemory    uint64
	TotalSwap      uint64
	BootTime       time.Time
	Uptime         time.Duration
	LoadAverage    [3]float64
	KernelVersion  string
}

type CollectorCapabilities struct {
	SupportsSystemMetrics  bool
	SupportsProcessMetrics bool
	SupportsCPUPressure    bool
	SupportsDiskIO         bool
	SupportsNetworkIO      bool
	SupportsMemoryPressure bool
	SupportsIOPressure     bool
	SupportsCGroups        bool
	SupportsContainers     bool
	RequiresRoot           bool
	PlatformSpecific       map[string]bool
}

type MetricsSnapshot struct {
	Timestamp      time.Time
	System         *entities.SystemMetrics
	TopCPUProcess  *entities.ProcessMetrics
	TopMemProcess  *entities.ProcessMetrics
	TopIOProcess   *entities.ProcessMetrics
	CPUPressure    *entities.CPUPressure
	DiskIOSummary  *DiskIOSummary
	ProcessSummary *ProcessSummary
}

type DiskIOSummary struct {
	TotalReadRate  valueobjects.ByteRate
	TotalWriteRate valueobjects.ByteRate
	TotalIOPS      float64
	DeviceCount    int
	BusiestDevice  string
	HighLatency    bool
}

type ProcessSummary struct {
	TotalProcesses   int
	RunningProcesses int
	SleepingProcesses int
	StoppedProcesses int
	ZombieProcesses  int
	TotalCPUUsage    float64
	TotalMemoryUsage float64
	TotalThreads     int
}

type CollectionStatistics struct {
	TotalCollections      uint64
	SuccessfulCollections uint64
	FailedCollections     uint64
	AverageCollectionTime time.Duration
	LastCollectionTime    time.Time
	LastCollectionDuration time.Duration
	ErrorCounts           map[string]uint64
	ProcessingRate        float64
}

type CollectorFactory interface {
	CreateCollector(platform string) (MetricsCollector, error)
	GetSupportedPlatforms() []string
	AutoDetectPlatform() string
}

type CollectorRegistry interface {
	Register(name string, collector MetricsCollector) error
	Get(name string) (MetricsCollector, error)
	List() []string
	Unregister(name string) error
}

type CollectorHealthCheck struct {
	Status           string
	LastCheckTime    time.Time
	IsHealthy        bool
	ErrorMessage     string
	SystemAccessible bool
	ProcessAccessible bool
	DiskAccessible   bool
	Recommendations  []string
}