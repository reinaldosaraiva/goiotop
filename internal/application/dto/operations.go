package dto

import (
	"fmt"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// CollectionRequestDTO represents a metrics collection request
type CollectionRequestDTO struct {
	Mode        string                 `json:"mode" yaml:"mode"`               // "single", "continuous", "batch"
	Interval    time.Duration          `json:"interval" yaml:"interval"`
	Duration    time.Duration          `json:"duration" yaml:"duration"`
	Options     CollectionOptionsDTO   `json:"options" yaml:"options"`
	Filters     CollectionFiltersDTO   `json:"filters" yaml:"filters"`
	Metadata    map[string]string      `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// CollectionOptionsDTO represents collection options
type CollectionOptionsDTO struct {
	IncludeIdle        bool          `json:"include_idle" yaml:"include_idle"`
	IncludeKernel      bool          `json:"include_kernel" yaml:"include_kernel"`
	IncludeSystemProcs bool          `json:"include_system_procs" yaml:"include_system_procs"`
	ProcessLimit       int           `json:"process_limit" yaml:"process_limit"`
	SortBy             string        `json:"sort_by" yaml:"sort_by"`
	MinCPUThreshold    float64       `json:"min_cpu_threshold" yaml:"min_cpu_threshold"`
	MinMemThreshold    float64       `json:"min_mem_threshold" yaml:"min_mem_threshold"`
	SampleDuration     time.Duration `json:"sample_duration" yaml:"sample_duration"`
	EnablePressure     bool          `json:"enable_pressure" yaml:"enable_pressure"`
	EnableGPU          bool          `json:"enable_gpu" yaml:"enable_gpu"`
}

// CollectionFiltersDTO represents collection filters
type CollectionFiltersDTO struct {
	ProcessNames    []string      `json:"process_names,omitempty" yaml:"process_names,omitempty"`
	ProcessPatterns []string      `json:"process_patterns,omitempty" yaml:"process_patterns,omitempty"`
	ExcludeNames    []string      `json:"exclude_names,omitempty" yaml:"exclude_names,omitempty"`
	UserFilter      string        `json:"user_filter,omitempty" yaml:"user_filter,omitempty"`
	TimeRange       TimeRangeDTO  `json:"time_range,omitempty" yaml:"time_range,omitempty"`
}

// TimeRangeDTO represents a time range filter
type TimeRangeDTO struct {
	Start time.Time `json:"start" yaml:"start"`
	End   time.Time `json:"end" yaml:"end"`
}

// CollectionResultDTO represents the result of a collection operation
type CollectionResultDTO struct {
	Success       bool                  `json:"success" yaml:"success"`
	Timestamp     time.Time             `json:"timestamp" yaml:"timestamp"`
	Duration      time.Duration         `json:"duration" yaml:"duration"`
	SystemMetrics *SystemMetricsDTO     `json:"system_metrics,omitempty" yaml:"system_metrics,omitempty"`
	Processes     []ProcessMetricsDTO   `json:"processes,omitempty" yaml:"processes,omitempty"`
	ProcessCount  int                   `json:"process_count" yaml:"process_count"`
	Errors        []ErrorDTO            `json:"errors,omitempty" yaml:"errors,omitempty"`
	Statistics    CollectionStatsDTO    `json:"statistics" yaml:"statistics"`
	Metadata      map[string]string     `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// CollectionStatsDTO represents collection statistics
type CollectionStatsDTO struct {
	CollectionTime    time.Duration `json:"collection_time_ms" yaml:"collection_time_ms"`
	ProcessingTime    time.Duration `json:"processing_time_ms" yaml:"processing_time_ms"`
	ItemsCollected    int           `json:"items_collected" yaml:"items_collected"`
	ItemsFailed       int           `json:"items_failed" yaml:"items_failed"`
	MemoryUsed        uint64        `json:"memory_used_bytes" yaml:"memory_used_bytes"`
	CollectorVersion  string        `json:"collector_version" yaml:"collector_version"`
}

// DisplayOptionsDTO represents display options
type DisplayOptionsDTO struct {
	Mode            string           `json:"mode" yaml:"mode"`                         // "realtime", "historical", "summary"
	RefreshInterval time.Duration    `json:"refresh_interval" yaml:"refresh_interval"`
	MaxRows         int              `json:"max_rows" yaml:"max_rows"`
	ShowSystemStats bool             `json:"show_system_stats" yaml:"show_system_stats"`
	ShowProcesses   bool             `json:"show_processes" yaml:"show_processes"`
	ShowGraphs      bool             `json:"show_graphs" yaml:"show_graphs"`
	SortBy          string           `json:"sort_by" yaml:"sort_by"`
	SortOrder       string           `json:"sort_order" yaml:"sort_order"`             // "asc", "desc"
	Filters         DisplayFiltersDTO `json:"filters" yaml:"filters"`
	Pagination      PaginationDTO     `json:"pagination" yaml:"pagination"`
}

// DisplayFiltersDTO represents display filters
type DisplayFiltersDTO struct {
	MinCPU        float64       `json:"min_cpu,omitempty" yaml:"min_cpu,omitempty"`
	MinMemory     float64       `json:"min_memory,omitempty" yaml:"min_memory,omitempty"`
	ProcessFilter string        `json:"process_filter,omitempty" yaml:"process_filter,omitempty"`
	UserFilter    string        `json:"user_filter,omitempty" yaml:"user_filter,omitempty"`
	StateFilter   []string      `json:"state_filter,omitempty" yaml:"state_filter,omitempty"`
	TimeRange     TimeRangeDTO  `json:"time_range,omitempty" yaml:"time_range,omitempty"`
}

// PaginationDTO represents pagination options
type PaginationDTO struct {
	Page     int `json:"page" yaml:"page"`
	PageSize int `json:"page_size" yaml:"page_size"`
	Total    int `json:"total" yaml:"total"`
}

// DisplayResultDTO represents the result of a display operation
type DisplayResultDTO struct {
	Mode          string              `json:"mode" yaml:"mode"`
	Timestamp     time.Time           `json:"timestamp" yaml:"timestamp"`
	SystemSummary *SystemSummaryDTO   `json:"system_summary,omitempty" yaml:"system_summary,omitempty"`
	Processes     []ProcessMetricsDTO `json:"processes,omitempty" yaml:"processes,omitempty"`
	Graphs        []GraphDataDTO      `json:"graphs,omitempty" yaml:"graphs,omitempty"`
	Pagination    PaginationDTO       `json:"pagination" yaml:"pagination"`
	UpdatedAt     time.Time           `json:"updated_at" yaml:"updated_at"`
}

// SystemSummaryDTO represents a system summary
type SystemSummaryDTO struct {
	CPUUsage      float64           `json:"cpu_usage" yaml:"cpu_usage"`
	MemoryUsage   float64           `json:"memory_usage" yaml:"memory_usage"`
	DiskUsage     float64           `json:"disk_usage" yaml:"disk_usage"`
	NetworkRX     float64           `json:"network_rx_mbps" yaml:"network_rx_mbps"`
	NetworkTX     float64           `json:"network_tx_mbps" yaml:"network_tx_mbps"`
	ProcessCount  int               `json:"process_count" yaml:"process_count"`
	ThreadCount   int               `json:"thread_count" yaml:"thread_count"`
	LoadAverage   LoadAvgDTO        `json:"load_average" yaml:"load_average"`
	Uptime        time.Duration     `json:"uptime" yaml:"uptime"`
	TopProcesses  []ProcessSummaryDTO `json:"top_processes,omitempty" yaml:"top_processes,omitempty"`
}

// ProcessSummaryDTO represents a process summary
type ProcessSummaryDTO struct {
	PID          int32   `json:"pid" yaml:"pid"`
	Name         string  `json:"name" yaml:"name"`
	CPUPercent   float64 `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent" yaml:"memory_percent"`
	State        string  `json:"state" yaml:"state"`
}

// GraphDataDTO represents graph data
type GraphDataDTO struct {
	Name      string      `json:"name" yaml:"name"`
	Type      string      `json:"type" yaml:"type"` // "line", "bar", "pie"
	Labels    []string    `json:"labels" yaml:"labels"`
	Values    []float64   `json:"values" yaml:"values"`
	Unit      string      `json:"unit" yaml:"unit"`
	Timestamp time.Time   `json:"timestamp" yaml:"timestamp"`
}

// ExportRequestDTO represents an export request
type ExportRequestDTO struct {
	Format      string            `json:"format" yaml:"format"`           // "json", "csv", "prometheus"
	Destination string            `json:"destination" yaml:"destination"` // "file", "http", "s3"
	Path        string            `json:"path,omitempty" yaml:"path,omitempty"`
	TimeRange   TimeRangeDTO      `json:"time_range,omitempty" yaml:"time_range,omitempty"`
	Filters     ExportFiltersDTO  `json:"filters,omitempty" yaml:"filters,omitempty"`
	Options     ExportOptionsDTO  `json:"options" yaml:"options"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ExportFiltersDTO represents export filters
type ExportFiltersDTO struct {
	IncludeSystem    bool     `json:"include_system" yaml:"include_system"`
	IncludeProcesses bool     `json:"include_processes" yaml:"include_processes"`
	ProcessNames     []string `json:"process_names,omitempty" yaml:"process_names,omitempty"`
	MinCPU           float64  `json:"min_cpu,omitempty" yaml:"min_cpu,omitempty"`
	MinMemory        float64  `json:"min_memory,omitempty" yaml:"min_memory,omitempty"`
}

// ExportOptionsDTO represents export options
type ExportOptionsDTO struct {
	Compress      bool   `json:"compress" yaml:"compress"`
	BatchSize     int    `json:"batch_size" yaml:"batch_size"`
	IncludeHeader bool   `json:"include_header" yaml:"include_header"`
	DateFormat    string `json:"date_format" yaml:"date_format"`
	Delimiter     string `json:"delimiter,omitempty" yaml:"delimiter,omitempty"` // for CSV
	Pretty        bool   `json:"pretty" yaml:"pretty"`                           // for JSON
}

// ExportResultDTO represents the result of an export operation
type ExportResultDTO struct {
	Success      bool              `json:"success" yaml:"success"`
	Format       string            `json:"format" yaml:"format"`
	Destination  string            `json:"destination" yaml:"destination"`
	Path         string            `json:"path,omitempty" yaml:"path,omitempty"`
	ItemsExported int              `json:"items_exported" yaml:"items_exported"`
	BytesWritten int64             `json:"bytes_written" yaml:"bytes_written"`
	Duration     time.Duration     `json:"duration" yaml:"duration"`
	Compressed   bool              `json:"compressed" yaml:"compressed"`
	Errors       []ErrorDTO        `json:"errors,omitempty" yaml:"errors,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ErrorDTO represents an error
type ErrorDTO struct {
	Code      string    `json:"code" yaml:"code"`
	Message   string    `json:"message" yaml:"message"`
	Details   string    `json:"details,omitempty" yaml:"details,omitempty"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	Component string    `json:"component,omitempty" yaml:"component,omitempty"`
}

// ToCollectionOptions converts DTO to domain CollectionOptions
func (dto *CollectionOptionsDTO) ToCollectionOptions() *services.CollectionOptions {
	if dto == nil {
		return services.DefaultCollectionOptions()
	}

	return &services.CollectionOptions{
		IncludeIdle:        dto.IncludeIdle,
		IncludeKernel:      dto.IncludeKernel,
		IncludeSystemProcs: dto.IncludeSystemProcs,
		ProcessLimit:       dto.ProcessLimit,
		SortBy:             dto.SortBy,
		MinCPUThreshold:    dto.MinCPUThreshold,
		MinMemThreshold:    dto.MinMemThreshold,
		SampleDuration:     dto.SampleDuration,
		EnablePressure:     dto.EnablePressure,
		EnableGPU:          dto.EnableGPU,
	}
}

// FromCollectionOptions converts domain CollectionOptions to DTO
func FromCollectionOptions(opts *services.CollectionOptions) *CollectionOptionsDTO {
	if opts == nil {
		return nil
	}

	return &CollectionOptionsDTO{
		IncludeIdle:        opts.IncludeIdle,
		IncludeKernel:      opts.IncludeKernel,
		IncludeSystemProcs: opts.IncludeSystemProcs,
		ProcessLimit:       opts.ProcessLimit,
		SortBy:             opts.SortBy,
		MinCPUThreshold:    opts.MinCPUThreshold,
		MinMemThreshold:    opts.MinMemThreshold,
		SampleDuration:     opts.SampleDuration,
		EnablePressure:     opts.EnablePressure,
		EnableGPU:          opts.EnableGPU,
	}
}

// Validate validates the CollectionRequestDTO
func (dto *CollectionRequestDTO) Validate() error {
	validModes := []string{"single", "continuous", "batch"}
	found := false
	for _, mode := range validModes {
		if dto.Mode == mode {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid collection mode: %s", dto.Mode)
	}

	if dto.Mode == "continuous" && dto.Interval <= 0 {
		return fmt.Errorf("interval is required for continuous mode")
	}

	if dto.Options.ProcessLimit <= 0 {
		dto.Options.ProcessLimit = 100 // default
	}

	return nil
}

// Validate validates the DisplayOptionsDTO
func (dto *DisplayOptionsDTO) Validate() error {
	validModes := []string{"realtime", "historical", "summary"}
	found := false
	for _, mode := range validModes {
		if dto.Mode == mode {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid display mode: %s", dto.Mode)
	}

	validSortOrders := []string{"asc", "desc"}
	found = false
	for _, order := range validSortOrders {
		if dto.SortOrder == order {
			found = true
			break
		}
	}
	if !found {
		dto.SortOrder = "desc" // default
	}

	if dto.MaxRows <= 0 {
		dto.MaxRows = 20 // default
	}

	if dto.Pagination.PageSize <= 0 {
		dto.Pagination.PageSize = 20 // default
	}

	return nil
}

// Validate validates the ExportRequestDTO
func (dto *ExportRequestDTO) Validate() error {
	validFormats := []string{"json", "csv", "prometheus"}
	found := false
	for _, format := range validFormats {
		if dto.Format == format {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid export format: %s", dto.Format)
	}

	validDestinations := []string{"file", "http", "s3"}
	found = false
	for _, dest := range validDestinations {
		if dto.Destination == dest {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid export destination: %s", dto.Destination)
	}

	if dto.Destination == "file" && dto.Path == "" {
		return fmt.Errorf("path is required for file destination")
	}

	if dto.Options.BatchSize <= 0 {
		dto.Options.BatchSize = 100 // default
	}

	return nil
}

// NewError creates a new ErrorDTO
func NewError(code, message string) ErrorDTO {
	return ErrorDTO{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewDetailedError creates a new ErrorDTO with details
func NewDetailedError(code, message, details, component string) ErrorDTO {
	return ErrorDTO{
		Code:      code,
		Message:   message,
		Details:   details,
		Component: component,
		Timestamp: time.Now(),
	}
}

// ProcessDisplayDTO represents process information formatted for display
type ProcessDisplayDTO struct {
	PID            int32   `json:"pid" yaml:"pid"`
	TID            int32   `json:"tid,omitempty" yaml:"tid,omitempty"`
	Name           string  `json:"name" yaml:"name"`
	User           string  `json:"user" yaml:"user"`
	CPUPercent     float64 `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent" yaml:"memory_percent"`
	MemoryRSS      uint64  `json:"memory_rss" yaml:"memory_rss"`
	DiskReadRate   float64 `json:"disk_read_rate" yaml:"disk_read_rate"`   // MB/s
	DiskWriteRate  float64 `json:"disk_write_rate" yaml:"disk_write_rate"` // MB/s
	NetRXRate      float64 `json:"net_rx_rate" yaml:"net_rx_rate"`         // MB/s
	NetTXRate      float64 `json:"net_tx_rate" yaml:"net_tx_rate"`         // MB/s
	State          string  `json:"state" yaml:"state"`
	Priority       int32   `json:"priority" yaml:"priority"`
	Nice           int32   `json:"nice" yaml:"nice"`
	ThreadCount    int32   `json:"thread_count" yaml:"thread_count"`
	OpenFiles      int32   `json:"open_files" yaml:"open_files"`
	Command        string  `json:"command" yaml:"command"`
	StartTime      time.Time `json:"start_time" yaml:"start_time"`
	RunTime        time.Duration `json:"run_time" yaml:"run_time"`
}

// DisplaySummaryDTO represents system summary formatted for display
type DisplaySummaryDTO struct {
	TotalDiskRead    float64 `json:"total_disk_read" yaml:"total_disk_read"`      // MB/s
	TotalDiskWrite   float64 `json:"total_disk_write" yaml:"total_disk_write"`    // MB/s
	TotalNetRX       float64 `json:"total_net_rx" yaml:"total_net_rx"`            // MB/s
	TotalNetTX       float64 `json:"total_net_tx" yaml:"total_net_tx"`            // MB/s
	AvgCPUPercent    float64 `json:"avg_cpu_percent" yaml:"avg_cpu_percent"`
	AvgMemoryPercent float64 `json:"avg_memory_percent" yaml:"avg_memory_percent"`
	ActiveProcesses  int     `json:"active_processes" yaml:"active_processes"`
	TotalProcesses   int     `json:"total_processes" yaml:"total_processes"`
	IOWaitPercent    float64 `json:"iowait_percent" yaml:"iowait_percent"`
	SwapPercent      float64 `json:"swap_percent" yaml:"swap_percent"`
	LoadAvg1         float64 `json:"load_avg_1" yaml:"load_avg_1"`
	LoadAvg5         float64 `json:"load_avg_5" yaml:"load_avg_5"`
	LoadAvg15        float64 `json:"load_avg_15" yaml:"load_avg_15"`
	Uptime           time.Duration `json:"uptime" yaml:"uptime"`
}

// FromProcessMetricsDTO converts ProcessMetricsDTO to ProcessDisplayDTO
func FromProcessMetricsDTO(pm *ProcessMetricsDTO) *ProcessDisplayDTO {
	if pm == nil {
		return nil
	}

	return &ProcessDisplayDTO{
		PID:           pm.PID,
		TID:           pm.TID,
		Name:          pm.Name,
		User:          pm.User,
		CPUPercent:    pm.CPU.UsagePercent,
		MemoryPercent: pm.Memory.UsagePercent,
		MemoryRSS:     pm.Memory.RSS,
		DiskReadRate:  pm.IO.Disk.ReadRate / (1024 * 1024),  // Convert to MB/s
		DiskWriteRate: pm.IO.Disk.WriteRate / (1024 * 1024), // Convert to MB/s
		NetRXRate:     pm.IO.Network.RecvRate / (1024 * 1024), // Convert to MB/s
		NetTXRate:     pm.IO.Network.SendRate / (1024 * 1024), // Convert to MB/s
		State:         pm.State,
		Priority:      pm.Priority,
		Nice:          pm.Nice,
		ThreadCount:   pm.ThreadCount,
		OpenFiles:     pm.OpenFiles,
		Command:       pm.Command,
		StartTime:     time.Unix(pm.CreateTime, 0),
		RunTime:       time.Duration(time.Now().Unix() - pm.CreateTime) * time.Second,
	}
}

// FromSystemMetricsDTO converts SystemMetricsDTO to DisplaySummaryDTO
func FromSystemMetricsDTO(sm *SystemMetricsDTO) *DisplaySummaryDTO {
	if sm == nil {
		return nil
	}

	var totalDiskRead, totalDiskWrite float64
	if sm.IO != nil && sm.IO.Disk != nil {
		totalDiskRead = sm.IO.Disk.ReadRate / (1024 * 1024)   // Convert to MB/s
		totalDiskWrite = sm.IO.Disk.WriteRate / (1024 * 1024) // Convert to MB/s
	}

	var totalNetRX, totalNetTX float64
	if sm.IO != nil && sm.IO.Network != nil {
		totalNetRX = sm.IO.Network.RecvRate / (1024 * 1024)  // Convert to MB/s
		totalNetTX = sm.IO.Network.SendRate / (1024 * 1024)  // Convert to MB/s
	}

	var avgCPU, ioWait float64
	if sm.CPU != nil {
		avgCPU = sm.CPU.UsagePercent
		ioWait = sm.CPU.IOWaitPercent
	}

	var avgMemory, swapPercent float64
	if sm.Memory != nil {
		avgMemory = sm.Memory.UsagePercent
		swapPercent = sm.Memory.SwapPercent
	}

	var load1, load5, load15 float64
	if sm.Load != nil {
		load1 = sm.Load.Load1
		load5 = sm.Load.Load5
		load15 = sm.Load.Load15
	}

	return &DisplaySummaryDTO{
		TotalDiskRead:    totalDiskRead,
		TotalDiskWrite:   totalDiskWrite,
		TotalNetRX:       totalNetRX,
		TotalNetTX:       totalNetTX,
		AvgCPUPercent:    avgCPU,
		AvgMemoryPercent: avgMemory,
		ActiveProcesses:  sm.ProcessCount,
		TotalProcesses:   sm.ProcessCount,
		IOWaitPercent:    ioWait,
		SwapPercent:      swapPercent,
		LoadAvg1:         load1,
		LoadAvg5:         load5,
		LoadAvg15:        load15,
		Uptime:           time.Duration(sm.UptimeSeconds) * time.Second,
	}
}

// Validate validates ProcessDisplayDTO
func (dto *ProcessDisplayDTO) Validate() error {
	if dto.PID <= 0 {
		return fmt.Errorf("invalid PID: %d", dto.PID)
	}
	if dto.Name == "" {
		return fmt.Errorf("process name cannot be empty")
	}
	return nil
}

// Validate validates DisplaySummaryDTO
func (dto *DisplaySummaryDTO) Validate() error {
	if dto.TotalProcesses < 0 {
		return fmt.Errorf("invalid total processes: %d", dto.TotalProcesses)
	}
	if dto.ActiveProcesses < 0 {
		return fmt.Errorf("invalid active processes: %d", dto.ActiveProcesses)
	}
	return nil
}