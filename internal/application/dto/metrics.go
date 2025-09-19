package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// SystemMetricsDTO represents system metrics for external interfaces
type SystemMetricsDTO struct {
	Timestamp time.Time           `json:"timestamp" yaml:"timestamp"`
	Hostname  string              `json:"hostname" yaml:"hostname"`
	CPU       CPUMetricsDTO       `json:"cpu" yaml:"cpu"`
	Memory    MemoryMetricsDTO    `json:"memory" yaml:"memory"`
	Disk      []DiskMetricsDTO    `json:"disk" yaml:"disk"`
	Network   []NetworkMetricsDTO `json:"network" yaml:"network"`
	LoadAvg   LoadAvgDTO          `json:"load_avg" yaml:"load_avg"`
	Uptime    int64               `json:"uptime_seconds" yaml:"uptime_seconds"`
	Platform  string              `json:"platform" yaml:"platform"`
	Metadata  map[string]string   `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// CPUMetricsDTO represents CPU metrics
type CPUMetricsDTO struct {
	UsagePercent   float64              `json:"usage_percent" yaml:"usage_percent"`
	UserPercent    float64              `json:"user_percent" yaml:"user_percent"`
	SystemPercent  float64              `json:"system_percent" yaml:"system_percent"`
	IdlePercent    float64              `json:"idle_percent" yaml:"idle_percent"`
	IowaitPercent  float64              `json:"iowait_percent" yaml:"iowait_percent"`
	StealPercent   float64              `json:"steal_percent" yaml:"steal_percent"`
	CoreCount      int                  `json:"core_count" yaml:"core_count"`
	ThreadCount    int                  `json:"thread_count" yaml:"thread_count"`
	Frequency      float64              `json:"frequency_mhz" yaml:"frequency_mhz"`
	Temperature    float64              `json:"temperature_celsius,omitempty" yaml:"temperature_celsius,omitempty"`
	PerCore        []PerCoreMetricsDTO  `json:"per_core,omitempty" yaml:"per_core,omitempty"`
	Pressure       *CPUPressureDTO      `json:"pressure,omitempty" yaml:"pressure,omitempty"`
}

// PerCoreMetricsDTO represents per-core CPU metrics
type PerCoreMetricsDTO struct {
	Core          int     `json:"core" yaml:"core"`
	UsagePercent  float64 `json:"usage_percent" yaml:"usage_percent"`
	UserPercent   float64 `json:"user_percent" yaml:"user_percent"`
	SystemPercent float64 `json:"system_percent" yaml:"system_percent"`
	IdlePercent   float64 `json:"idle_percent" yaml:"idle_percent"`
	Temperature   float64 `json:"temperature_celsius,omitempty" yaml:"temperature_celsius,omitempty"`
}

// MemoryMetricsDTO represents memory metrics
type MemoryMetricsDTO struct {
	Total        uint64  `json:"total_bytes" yaml:"total_bytes"`
	Available    uint64  `json:"available_bytes" yaml:"available_bytes"`
	Used         uint64  `json:"used_bytes" yaml:"used_bytes"`
	Free         uint64  `json:"free_bytes" yaml:"free_bytes"`
	UsagePercent float64 `json:"usage_percent" yaml:"usage_percent"`
	Cached       uint64  `json:"cached_bytes" yaml:"cached_bytes"`
	Buffers      uint64  `json:"buffers_bytes" yaml:"buffers_bytes"`
	SwapTotal    uint64  `json:"swap_total_bytes" yaml:"swap_total_bytes"`
	SwapUsed     uint64  `json:"swap_used_bytes" yaml:"swap_used_bytes"`
	SwapFree     uint64  `json:"swap_free_bytes" yaml:"swap_free_bytes"`
	SwapPercent  float64 `json:"swap_percent" yaml:"swap_percent"`
	Pressure     float64 `json:"pressure,omitempty" yaml:"pressure,omitempty"`
}

// DiskMetricsDTO represents disk metrics
type DiskMetricsDTO struct {
	Device       string  `json:"device" yaml:"device"`
	MountPoint   string  `json:"mount_point" yaml:"mount_point"`
	FSType       string  `json:"fs_type" yaml:"fs_type"`
	Total        uint64  `json:"total_bytes" yaml:"total_bytes"`
	Used         uint64  `json:"used_bytes" yaml:"used_bytes"`
	Free         uint64  `json:"free_bytes" yaml:"free_bytes"`
	UsagePercent float64 `json:"usage_percent" yaml:"usage_percent"`
	InodesTotal  uint64  `json:"inodes_total" yaml:"inodes_total"`
	InodesUsed   uint64  `json:"inodes_used" yaml:"inodes_used"`
	InodesFree   uint64  `json:"inodes_free" yaml:"inodes_free"`
	InodesUsedPercent float64 `json:"inodes_used_percent" yaml:"inodes_used_percent"`
	IO           *DiskIODTO `json:"io,omitempty" yaml:"io,omitempty"`
}

// NetworkMetricsDTO represents network interface metrics
type NetworkMetricsDTO struct {
	Interface     string    `json:"interface" yaml:"interface"`
	BytesSent     uint64    `json:"bytes_sent" yaml:"bytes_sent"`
	BytesReceived uint64    `json:"bytes_received" yaml:"bytes_received"`
	PacketsSent   uint64    `json:"packets_sent" yaml:"packets_sent"`
	PacketsReceived uint64  `json:"packets_received" yaml:"packets_received"`
	ErrorsIn      uint64    `json:"errors_in" yaml:"errors_in"`
	ErrorsOut     uint64    `json:"errors_out" yaml:"errors_out"`
	DropsIn       uint64    `json:"drops_in" yaml:"drops_in"`
	DropsOut      uint64    `json:"drops_out" yaml:"drops_out"`
	Speed         int64     `json:"speed_mbps,omitempty" yaml:"speed_mbps,omitempty"`
	Addresses     []string  `json:"addresses,omitempty" yaml:"addresses,omitempty"`
}

// LoadAvgDTO represents system load averages
type LoadAvgDTO struct {
	Load1  float64 `json:"load_1" yaml:"load_1"`
	Load5  float64 `json:"load_5" yaml:"load_5"`
	Load15 float64 `json:"load_15" yaml:"load_15"`
}

// ProcessMetricsDTO represents process metrics for external interfaces
type ProcessMetricsDTO struct {
	PID          int32     `json:"pid" yaml:"pid"`
	Name         string    `json:"name" yaml:"name"`
	Username     string    `json:"username" yaml:"username"`
	State        string    `json:"state" yaml:"state"`
	CPUPercent   float64   `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryPercent float64  `json:"memory_percent" yaml:"memory_percent"`
	MemoryRSS    uint64    `json:"memory_rss_bytes" yaml:"memory_rss_bytes"`
	MemoryVMS    uint64    `json:"memory_vms_bytes" yaml:"memory_vms_bytes"`
	IORead       uint64    `json:"io_read_bytes" yaml:"io_read_bytes"`
	IOWrite      uint64    `json:"io_write_bytes" yaml:"io_write_bytes"`
	IOReadRate   float64   `json:"io_read_rate_bps" yaml:"io_read_rate_bps"`
	IOWriteRate  float64   `json:"io_write_rate_bps" yaml:"io_write_rate_bps"`
	NumThreads   int32     `json:"num_threads" yaml:"num_threads"`
	NumFDs       int32     `json:"num_fds,omitempty" yaml:"num_fds,omitempty"`
	CreateTime   time.Time `json:"create_time" yaml:"create_time"`
	CPUTime      float64   `json:"cpu_time_seconds" yaml:"cpu_time_seconds"`
	Command      string    `json:"command" yaml:"command"`
	CommandArgs  []string  `json:"command_args,omitempty" yaml:"command_args,omitempty"`
	Parent       int32     `json:"parent_pid" yaml:"parent_pid"`
	Children     []int32   `json:"children_pids,omitempty" yaml:"children_pids,omitempty"`
	Nice         int32     `json:"nice" yaml:"nice"`
	Priority     int32     `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// CPUPressureDTO represents CPU pressure metrics
type CPUPressureDTO struct {
	Some10  float64 `json:"some_10" yaml:"some_10"`
	Some60  float64 `json:"some_60" yaml:"some_60"`
	Some300 float64 `json:"some_300" yaml:"some_300"`
	Total   uint64  `json:"total" yaml:"total"`
}

// DiskIODTO represents disk I/O metrics
type DiskIODTO struct {
	Device           string  `json:"device" yaml:"device"`
	ReadBytes        uint64  `json:"read_bytes" yaml:"read_bytes"`
	WriteBytes       uint64  `json:"write_bytes" yaml:"write_bytes"`
	ReadCount        uint64  `json:"read_count" yaml:"read_count"`
	WriteCount       uint64  `json:"write_count" yaml:"write_count"`
	ReadTime         uint64  `json:"read_time_ms" yaml:"read_time_ms"`
	WriteTime        uint64  `json:"write_time_ms" yaml:"write_time_ms"`
	IOTime           uint64  `json:"io_time_ms" yaml:"io_time_ms"`
	WeightedIOTime   uint64  `json:"weighted_io_time_ms" yaml:"weighted_io_time_ms"`
	IopsInProgress   uint64  `json:"iops_in_progress" yaml:"iops_in_progress"`
	ReadBytesRate    float64 `json:"read_bytes_rate" yaml:"read_bytes_rate"`
	WriteBytesRate   float64 `json:"write_bytes_rate" yaml:"write_bytes_rate"`
	ReadOpsRate      float64 `json:"read_ops_rate" yaml:"read_ops_rate"`
	WriteOpsRate     float64 `json:"write_ops_rate" yaml:"write_ops_rate"`
	AvgQueueSize     float64 `json:"avg_queue_size" yaml:"avg_queue_size"`
	AvgReadLatency   float64 `json:"avg_read_latency_ms" yaml:"avg_read_latency_ms"`
	AvgWriteLatency  float64 `json:"avg_write_latency_ms" yaml:"avg_write_latency_ms"`
	Utilization      float64 `json:"utilization_percent" yaml:"utilization_percent"`
}

// FromSystemMetrics converts domain entity to DTO
func FromSystemMetrics(metrics *entities.SystemMetrics) *SystemMetricsDTO {
	if metrics == nil {
		return nil
	}

	// Get load averages
	load1, load5, load15 := metrics.LoadAverage()
	// Get process counts
	active, total := metrics.ProcessCount()

	dto := &SystemMetricsDTO{
		Timestamp: metrics.Timestamp().ToTime(),
		Hostname:  "", // Hostname is not part of the domain entity
		CPU: CPUMetricsDTO{
			UsagePercent:   metrics.CPUUsage().Value(),
			UserPercent:    0, // Not available in simplified entity
			SystemPercent:  0, // Not available in simplified entity
			IdlePercent:    100 - metrics.CPUUsage().Value(),
			IowaitPercent:  0, // Not available in simplified entity
			StealPercent:   0, // Not available in simplified entity
			CoreCount:      0, // Not available in simplified entity
			ThreadCount:    0, // Not available in simplified entity
			Frequency:      0, // Not available in simplified entity
			Temperature:    0, // Not available in simplified entity
		},
		Memory: MemoryMetricsDTO{
			Total:        0, // Not available in simplified entity
			Available:    0, // Not available in simplified entity
			Used:         0, // Not available in simplified entity
			Free:         0, // Not available in simplified entity
			UsagePercent: metrics.MemoryUsage().Value(),
			Cached:       0, // Not available in simplified entity
			Buffers:      0, // Not available in simplified entity
			SwapTotal:    0, // Not available in simplified entity
			SwapUsed:     0, // Not available in simplified entity
			SwapFree:     0, // Not available in simplified entity
			SwapPercent:  0, // Not available in simplified entity
			Pressure:     0, // Not available in simplified entity
		},
		Disk: []DiskMetricsDTO{{
			Device:       "aggregate",
			MountPoint:   "/",
			FSType:       "unknown",
			Total:        0,
			Used:         0,
			Free:         0,
			UsagePercent: 0,
			IO: &DiskIODTO{
				Device:         "aggregate",
				ReadBytesRate:  metrics.DiskReadRate().BytesPerSecond(),
				WriteBytesRate: metrics.DiskWriteRate().BytesPerSecond(),
			},
		}},
		Network: []NetworkMetricsDTO{{
			Interface:     "aggregate",
			BytesReceived: uint64(metrics.NetworkInRate().BytesPerSecond()),
			BytesSent:     uint64(metrics.NetworkOutRate().BytesPerSecond()),
		}},
		LoadAvg: LoadAvgDTO{
			Load1:  load1,
			Load5:  load5,
			Load15: load15,
		},
		Uptime:   0, // Not available in simplified entity
		Platform: "", // Not available in simplified entity
		Metadata: map[string]string{
			"active_processes": fmt.Sprintf("%d", active),
			"total_processes":  fmt.Sprintf("%d", total),
		},
	}

	return dto
}

// ToSystemMetrics converts DTO to domain entity
// Note: The domain entity uses value objects and has a simplified structure
// Many fields from the DTO cannot be mapped directly
func (dto *SystemMetricsDTO) ToSystemMetrics() (*entities.SystemMetrics, error) {
	if dto == nil {
		return nil, nil
	}

	// Create value objects
	cpuUsage, err := valueobjects.NewPercentage(dto.CPU.UsagePercent)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU usage: %w", err)
	}

	memUsage, err := valueobjects.NewPercentage(dto.Memory.UsagePercent)
	if err != nil {
		return nil, fmt.Errorf("invalid memory usage: %w", err)
	}

	// Extract disk rates from the first disk IO if available
	var diskReadRate, diskWriteRate valueobjects.ByteRate
	if len(dto.Disk) > 0 && dto.Disk[0].IO != nil {
		diskReadRate = valueobjects.NewByteRate(dto.Disk[0].IO.ReadBytesRate)
		diskWriteRate = valueobjects.NewByteRate(dto.Disk[0].IO.WriteBytesRate)
	} else {
		diskReadRate = valueobjects.NewByteRateZero()
		diskWriteRate = valueobjects.NewByteRateZero()
	}

	// Extract network rates
	var networkInRate, networkOutRate valueobjects.ByteRate
	if len(dto.Network) > 0 {
		// Convert from bytes transferred to bytes per second (this is simplified)
		networkInRate = valueobjects.NewByteRate(float64(dto.Network[0].BytesReceived))
		networkOutRate = valueobjects.NewByteRate(float64(dto.Network[0].BytesSent))
	} else {
		networkInRate = valueobjects.NewByteRateZero()
		networkOutRate = valueobjects.NewByteRateZero()
	}

	// Create the system metrics entity
	metrics, err := entities.NewSystemMetricsWithNetwork(
		*cpuUsage,
		*memUsage,
		diskReadRate,
		diskWriteRate,
		networkInRate,
		networkOutRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create system metrics: %w", err)
	}

	// Set load averages
	metrics.SetLoadAverage(dto.LoadAvg.Load1, dto.LoadAvg.Load5, dto.LoadAvg.Load15)

	// Set process count from metadata if available
	if dto.Metadata != nil {
		var active, total int
		if v, ok := dto.Metadata["active_processes"]; ok {
			fmt.Sscanf(v, "%d", &active)
		}
		if v, ok := dto.Metadata["total_processes"]; ok {
			fmt.Sscanf(v, "%d", &total)
		}
		metrics.SetProcessCount(active, total)
	}

	return metrics, nil
}

// FromProcessMetrics converts domain entity to DTO
func FromProcessMetrics(metrics *entities.ProcessMetrics) *ProcessMetricsDTO {
	if metrics == nil {
		return nil
	}

	// Convert parent PID if present
	var parentPID int32
	if parent := metrics.ParentPID(); parent != nil {
		parentPID = parent.ToInt32()
	}

	return &ProcessMetricsDTO{
		PID:           metrics.ProcessID().ToInt32(),
		Name:          metrics.Name(),
		Username:      "", // Not available in the entity, would need to be looked up from UID
		State:         string(metrics.State()),
		CPUPercent:    metrics.CPUUsage().Value(),
		MemoryPercent: metrics.MemoryUsage().Value(),
		MemoryRSS:     metrics.MemoryRSS(),
		MemoryVMS:     metrics.MemoryVMS(),
		IORead:        uint64(metrics.DiskReadRate().BytesPerSecond()),
		IOWrite:       uint64(metrics.DiskWriteRate().BytesPerSecond()),
		IOReadRate:    metrics.DiskReadRate().BytesPerSecond(),
		IOWriteRate:   metrics.DiskWriteRate().BytesPerSecond(),
		NumThreads:    int32(metrics.Threads()),
		NumFDs:        0, // Not available in simplified entity
		CreateTime:    metrics.Timestamp().ToTime(),
		CPUTime:       0, // Not available in simplified entity
		Command:       "", // Not available in simplified entity
		CommandArgs:   []string{}, // Not available in simplified entity
		Parent:        parentPID,
		Children:      []int32{}, // Not available in simplified entity
		Nice:          int32(metrics.Nice()),
		Priority:      int32(metrics.Priority()),
	}
}

// ToProcessMetrics converts DTO to domain entity
func (dto *ProcessMetricsDTO) ToProcessMetrics() (*entities.ProcessMetrics, error) {
	if dto == nil {
		return nil, nil
	}

	// Create value objects
	pid := valueobjects.NewProcessID(dto.PID)

	cpuUsage, err := valueobjects.NewPercentage(dto.CPUPercent)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU usage: %w", err)
	}

	memUsage, err := valueobjects.NewPercentage(dto.MemoryPercent)
	if err != nil {
		return nil, fmt.Errorf("invalid memory usage: %w", err)
	}

	diskReadRate := valueobjects.NewByteRate(dto.IOReadRate)
	diskWriteRate := valueobjects.NewByteRate(dto.IOWriteRate)

	// Create the process metrics entity
	metrics, err := entities.NewProcessMetrics(
		*pid,
		dto.Name,
		*cpuUsage,
		*memUsage,
		diskReadRate,
		diskWriteRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create process metrics: %w", err)
	}

	// Set additional fields through methods if available
	metrics.SetMemoryRSS(dto.MemoryRSS)
	metrics.SetMemoryVMS(dto.MemoryVMS)
	metrics.SetThreads(int(dto.NumThreads))
	metrics.SetState(entities.ProcessState(dto.State))

	if dto.Parent > 0 {
		parentPID := valueobjects.NewProcessID(dto.Parent)
		metrics.SetParentPID(parentPID)
	}

	metrics.SetUID(uint32(dto.PID)) // This is simplified, should be actual UID
	metrics.SetGID(0) // This is simplified
	metrics.SetPriority(int(dto.Priority))
	metrics.SetNice(int(dto.Nice))

	return metrics, nil
}

// FromCPUPressure converts domain entity to DTO
func FromCPUPressure(pressure *entities.CPUPressure) *CPUPressureDTO {
	if pressure == nil {
		return nil
	}

	return &CPUPressureDTO{
		Some10:  pressure.Some(10).Value(),
		Some60:  pressure.Some(60).Value(),
		Some300: pressure.Some(300).Value(),
		Total:   pressure.Total(),
	}
}

// ToCPUPressure converts DTO to domain entity
func (dto *CPUPressureDTO) ToCPUPressure() (*entities.CPUPressure, error) {
	if dto == nil {
		return nil, nil
	}

	// Create CPU pressure entity using factory method if available
	// Otherwise use the struct directly with value objects
	pressure := entities.NewCPUPressure(
		valueobjects.NewTimestampNow(),
	)

	// Set the values if setters are available
	pressure.SetSome(10, dto.Some10)
	pressure.SetSome(60, dto.Some60)
	pressure.SetSome(300, dto.Some300)
	pressure.SetTotal(dto.Total)

	return pressure, nil
}

// FromDiskIO converts domain entity to DTO
func FromDiskIO(io *entities.DiskIO) *DiskIODTO {
	if io == nil {
		return nil
	}

	return &DiskIODTO{
		Device:          io.DeviceName(),
		ReadBytes:       io.ReadBytes(),
		WriteBytes:      io.WriteBytes(),
		ReadCount:       io.ReadOperations(),
		WriteCount:      io.WriteOperations(),
		ReadTime:        uint64(io.ReadTime().Milliseconds()),
		WriteTime:       uint64(io.WriteTime().Milliseconds()),
		IOTime:          uint64(io.TotalTime().Milliseconds()),
		WeightedIOTime:  0, // Not available in domain entity
		IopsInProgress:  0, // Not available in domain entity
		ReadBytesRate:   io.ReadRate().BytesPerSecond(),
		WriteBytesRate:  io.WriteRate().BytesPerSecond(),
		ReadOpsRate:     float64(io.ReadOperations()) / io.TotalTime().Seconds(),
		WriteOpsRate:    float64(io.WriteOperations()) / io.TotalTime().Seconds(),
		AvgQueueSize:    0, // Not available in domain entity
		AvgReadLatency:  io.AverageReadLatency().Seconds() * 1000, // Convert to ms
		AvgWriteLatency: io.AverageWriteLatency().Seconds() * 1000, // Convert to ms
		Utilization:     io.Utilization().Value(),
	}
}

// ToDiskIO converts DTO to domain entity
func (dto *DiskIODTO) ToDiskIO() (*entities.DiskIO, error) {
	if dto == nil {
		return nil, nil
	}

	// Create value objects
	timestamp := valueobjects.NewTimestampNow()
	readRate := valueobjects.NewByteRate(dto.ReadBytesRate)
	writeRate := valueobjects.NewByteRate(dto.WriteBytesRate)

	utilization, err := valueobjects.NewPercentage(dto.Utilization)
	if err != nil {
		// If utilization is invalid, use zero
		utilization, _ = valueobjects.NewPercentage(0)
	}

	// Create the DiskIO entity
	diskIO := entities.NewDiskIO(
		dto.Device,
		*timestamp,
		dto.ReadBytes,
		dto.WriteBytes,
		dto.ReadCount,
		dto.WriteCount,
		readRate,
		writeRate,
		*utilization,
	)

	// Set additional fields if setters are available
	diskIO.SetReadTime(time.Duration(dto.ReadTime) * time.Millisecond)
	diskIO.SetWriteTime(time.Duration(dto.WriteTime) * time.Millisecond)
	diskIO.SetAverageLatencies(
		time.Duration(dto.AvgReadLatency)*time.Millisecond,
		time.Duration(dto.AvgWriteLatency)*time.Millisecond,
	)

	return diskIO, nil
}

// Validate validates the SystemMetricsDTO
func (dto *SystemMetricsDTO) Validate() error {
	if dto.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if dto.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if dto.CPU.CoreCount <= 0 {
		return fmt.Errorf("invalid CPU core count: %d", dto.CPU.CoreCount)
	}
	if dto.Memory.Total == 0 {
		return fmt.Errorf("invalid memory total: %d", dto.Memory.Total)
	}
	return nil
}

// Validate validates the ProcessMetricsDTO
func (dto *ProcessMetricsDTO) Validate() error {
	if dto.PID <= 0 {
		return fmt.Errorf("invalid PID: %d", dto.PID)
	}
	if dto.Name == "" {
		return fmt.Errorf("process name is required")
	}
	if dto.CPUPercent < 0 {
		return fmt.Errorf("invalid CPU percent: %f", dto.CPUPercent)
	}
	if dto.MemoryPercent < 0 {
		return fmt.Errorf("invalid memory percent: %f", dto.MemoryPercent)
	}
	return nil
}

// ToJSON converts DTO to JSON
func (dto *SystemMetricsDTO) ToJSON() ([]byte, error) {
	return json.MarshalIndent(dto, "", "  ")
}

// FromJSON creates DTO from JSON
func SystemMetricsFromJSON(data []byte) (*SystemMetricsDTO, error) {
	var dto SystemMetricsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

// ToJSON converts DTO to JSON
func (dto *ProcessMetricsDTO) ToJSON() ([]byte, error) {
	return json.MarshalIndent(dto, "", "  ")
}

// FromJSON creates DTO from JSON
func ProcessMetricsFromJSON(data []byte) (*ProcessMetricsDTO, error) {
	var dto ProcessMetricsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}

// ProcessMetricsList represents a list of process metrics
type ProcessMetricsList struct {
	Processes []ProcessMetricsDTO `json:"processes" yaml:"processes"`
	Total     int                 `json:"total" yaml:"total"`
	Timestamp time.Time           `json:"timestamp" yaml:"timestamp"`
}

// SystemMetricsList represents a list of system metrics
type SystemMetricsList struct {
	Metrics   []SystemMetricsDTO `json:"metrics" yaml:"metrics"`
	Total     int                `json:"total" yaml:"total"`
	StartTime time.Time          `json:"start_time" yaml:"start_time"`
	EndTime   time.Time          `json:"end_time" yaml:"end_time"`
}