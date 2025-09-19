package dto

import (
	"fmt"
	"time"
)

// ConfigDTO represents the application configuration for external interfaces
type ConfigDTO struct {
	App        AppConfigDTO        `json:"app" yaml:"app"`
	Collection CollectionConfigDTO `json:"collection" yaml:"collection"`
	Display    DisplayConfigDTO    `json:"display" yaml:"display"`
	Export     ExportConfigDTO     `json:"export" yaml:"export"`
	Logging    LoggingConfigDTO    `json:"logging" yaml:"logging"`
	Repository RepositoryConfigDTO `json:"repository" yaml:"repository"`
	Version    string              `json:"version" yaml:"version"`
}

// AppConfigDTO contains general application settings
type AppConfigDTO struct {
	Name              string        `json:"name" yaml:"name"`
	Version           string        `json:"version" yaml:"version"`
	Mode              string        `json:"mode" yaml:"mode"`
	ShutdownTimeout   time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
	HealthCheckPeriod time.Duration `json:"health_check_period" yaml:"health_check_period"`
}

// CollectionConfigDTO contains collection settings
type CollectionConfigDTO struct {
	Interval      time.Duration        `json:"interval" yaml:"interval"`
	Timeout       time.Duration        `json:"timeout" yaml:"timeout"`
	BatchSize     int                  `json:"batch_size" yaml:"batch_size"`
	MaxRetries    int                  `json:"max_retries" yaml:"max_retries"`
	RetryDelay    time.Duration        `json:"retry_delay" yaml:"retry_delay"`
	EnabledMetrics EnabledMetricsDTO   `json:"enabled_metrics" yaml:"enabled_metrics"`
	Options       CollectionOptionsDTO `json:"options" yaml:"options"`
}

// EnabledMetricsDTO indicates which metrics are enabled
type EnabledMetricsDTO struct {
	CPU     bool `json:"cpu" yaml:"cpu"`
	Memory  bool `json:"memory" yaml:"memory"`
	Disk    bool `json:"disk" yaml:"disk"`
	Network bool `json:"network" yaml:"network"`
	Process bool `json:"process" yaml:"process"`
	GPU     bool `json:"gpu" yaml:"gpu"`
}

// DisplayConfigDTO contains display settings
type DisplayConfigDTO struct {
	RefreshRate      time.Duration `json:"refresh_rate" yaml:"refresh_rate"`
	MaxRows          int           `json:"max_rows" yaml:"max_rows"`
	ShowSystemStats  bool          `json:"show_system_stats" yaml:"show_system_stats"`
	ShowProcessList  bool          `json:"show_process_list" yaml:"show_process_list"`
	ShowGraphs       bool          `json:"show_graphs" yaml:"show_graphs"`
	GraphHistory     time.Duration `json:"graph_history" yaml:"graph_history"`
	ColorScheme      string        `json:"color_scheme" yaml:"color_scheme"`
	DateFormat       string        `json:"date_format" yaml:"date_format"`
	TimeFormat       string        `json:"time_format" yaml:"time_format"`
	EnablePagination bool          `json:"enable_pagination" yaml:"enable_pagination"`
	PageSize         int           `json:"page_size" yaml:"page_size"`
}

// ExportConfigDTO contains export settings
type ExportConfigDTO struct {
	Enabled     bool             `json:"enabled" yaml:"enabled"`
	Format      string           `json:"format" yaml:"format"`
	Destination string           `json:"destination" yaml:"destination"`
	Interval    time.Duration    `json:"interval" yaml:"interval"`
	BufferSize  int              `json:"buffer_size" yaml:"buffer_size"`
	Compression bool             `json:"compression" yaml:"compression"`
	BatchSize   int              `json:"batch_size" yaml:"batch_size"`
}

// LoggingConfigDTO contains logging settings
type LoggingConfigDTO struct {
	Level            string            `json:"level" yaml:"level"`
	Format           string            `json:"format" yaml:"format"`
	Output           string            `json:"output" yaml:"output"`
	File             string            `json:"file" yaml:"file"`
	MaxSize          int               `json:"max_size" yaml:"max_size"`
	MaxAge           int               `json:"max_age" yaml:"max_age"`
	MaxBackups       int               `json:"max_backups" yaml:"max_backups"`
	Compress         bool              `json:"compress" yaml:"compress"`
	EnableCaller     bool              `json:"enable_caller" yaml:"enable_caller"`
	EnableStacktrace bool              `json:"enable_stacktrace" yaml:"enable_stacktrace"`
	Fields           map[string]string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// RepositoryConfigDTO contains repository settings
type RepositoryConfigDTO struct {
	Type            string        `json:"type" yaml:"type"`
	ConnectionURL   string        `json:"connection_url,omitempty" yaml:"connection_url,omitempty"`
	MaxConnections  int           `json:"max_connections" yaml:"max_connections"`
	RetentionDays   int           `json:"retention_days" yaml:"retention_days"`
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
}

// SystemInfoDTO represents system information
type SystemInfoDTO struct {
	Hostname        string                 `json:"hostname" yaml:"hostname"`
	Platform        string                 `json:"platform" yaml:"platform"`
	PlatformVersion string                 `json:"platform_version" yaml:"platform_version"`
	Architecture    string                 `json:"architecture" yaml:"architecture"`
	CPUModel        string                 `json:"cpu_model" yaml:"cpu_model"`
	CPUCores        int                    `json:"cpu_cores" yaml:"cpu_cores"`
	CPUThreads      int                    `json:"cpu_threads" yaml:"cpu_threads"`
	TotalMemory     uint64                 `json:"total_memory_bytes" yaml:"total_memory_bytes"`
	TotalDisk       uint64                 `json:"total_disk_bytes" yaml:"total_disk_bytes"`
	BootTime        time.Time              `json:"boot_time" yaml:"boot_time"`
	Uptime          time.Duration          `json:"uptime" yaml:"uptime"`
	Timezone        string                 `json:"timezone" yaml:"timezone"`
	CollectorInfo   CollectorInfoDTO       `json:"collector_info" yaml:"collector_info"`
}

// CollectorInfoDTO represents collector information
type CollectorInfoDTO struct {
	Version      string   `json:"version" yaml:"version"`
	Backend      string   `json:"backend" yaml:"backend"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// CollectorCapabilitiesDTO represents collector capabilities
type CollectorCapabilitiesDTO struct {
	SupportsCPU           bool                 `json:"supports_cpu" yaml:"supports_cpu"`
	SupportsMemory        bool                 `json:"supports_memory" yaml:"supports_memory"`
	SupportsDisk          bool                 `json:"supports_disk" yaml:"supports_disk"`
	SupportsNetwork       bool                 `json:"supports_network" yaml:"supports_network"`
	SupportsProcess       bool                 `json:"supports_process" yaml:"supports_process"`
	SupportsGPU           bool                 `json:"supports_gpu" yaml:"supports_gpu"`
	SupportsPressure      bool                 `json:"supports_pressure" yaml:"supports_pressure"`
	SupportsTemperature   bool                 `json:"supports_temperature" yaml:"supports_temperature"`
	SupportsContinuous    bool                 `json:"supports_continuous" yaml:"supports_continuous"`
	MaxProcessLimit       int                  `json:"max_process_limit" yaml:"max_process_limit"`
	MinSampleDuration     time.Duration        `json:"min_sample_duration" yaml:"min_sample_duration"`
	PlatformSpecific      map[string]bool      `json:"platform_specific" yaml:"platform_specific"`
}

// HealthCheckDTO represents a health check result
type HealthCheckDTO struct {
	Status       string                 `json:"status" yaml:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp    time.Time              `json:"timestamp" yaml:"timestamp"`
	Components   []ComponentHealthDTO   `json:"components" yaml:"components"`
	Metrics      HealthMetricsDTO       `json:"metrics" yaml:"metrics"`
	Issues       []string               `json:"issues,omitempty" yaml:"issues,omitempty"`
	LastCheck    time.Time              `json:"last_check" yaml:"last_check"`
	NextCheck    time.Time              `json:"next_check" yaml:"next_check"`
}

// ComponentHealthDTO represents individual component health
type ComponentHealthDTO struct {
	Name      string            `json:"name" yaml:"name"`
	Status    string            `json:"status" yaml:"status"`
	Message   string            `json:"message,omitempty" yaml:"message,omitempty"`
	Details   map[string]string `json:"details,omitempty" yaml:"details,omitempty"`
	LastCheck time.Time         `json:"last_check" yaml:"last_check"`
}

// HealthMetricsDTO represents health-related metrics
type HealthMetricsDTO struct {
	Uptime              time.Duration `json:"uptime" yaml:"uptime"`
	CollectionsTotal    int64         `json:"collections_total" yaml:"collections_total"`
	CollectionsFailed   int64         `json:"collections_failed" yaml:"collections_failed"`
	CollectionDuration  time.Duration `json:"avg_collection_duration" yaml:"avg_collection_duration"`
	MemoryUsage         uint64        `json:"memory_usage_bytes" yaml:"memory_usage_bytes"`
	GoroutineCount      int           `json:"goroutine_count" yaml:"goroutine_count"`
	LastCollectionTime  time.Time     `json:"last_collection_time" yaml:"last_collection_time"`
	ErrorRate           float64       `json:"error_rate_percent" yaml:"error_rate_percent"`
}

// StatusResponseDTO represents a status response
type StatusResponseDTO struct {
	Status    string           `json:"status" yaml:"status"`
	Version   string           `json:"version" yaml:"version"`
	Timestamp time.Time        `json:"timestamp" yaml:"timestamp"`
	System    *SystemInfoDTO   `json:"system,omitempty" yaml:"system,omitempty"`
	Health    *HealthCheckDTO  `json:"health,omitempty" yaml:"health,omitempty"`
	Config    *ConfigSummaryDTO `json:"config,omitempty" yaml:"config,omitempty"`
}

// ConfigSummaryDTO represents a configuration summary
type ConfigSummaryDTO struct {
	Mode               string        `json:"mode" yaml:"mode"`
	CollectionInterval time.Duration `json:"collection_interval" yaml:"collection_interval"`
	ExportEnabled      bool          `json:"export_enabled" yaml:"export_enabled"`
	ExportFormat       string        `json:"export_format,omitempty" yaml:"export_format,omitempty"`
	RepositoryType     string        `json:"repository_type" yaml:"repository_type"`
	LogLevel           string        `json:"log_level" yaml:"log_level"`
}

// CapabilityCheckDTO represents a capability check request/response
type CapabilityCheckDTO struct {
	Capability string `json:"capability" yaml:"capability"`
	Available  bool   `json:"available" yaml:"available"`
	Reason     string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// Validate validates the SystemInfoDTO
func (dto *SystemInfoDTO) Validate() error {
	if dto.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if dto.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if dto.CPUCores <= 0 {
		return fmt.Errorf("invalid CPU cores: %d", dto.CPUCores)
	}
	if dto.TotalMemory == 0 {
		return fmt.Errorf("invalid total memory: %d", dto.TotalMemory)
	}
	return nil
}

// Validate validates the HealthCheckDTO
func (dto *HealthCheckDTO) Validate() error {
	validStatuses := []string{"healthy", "degraded", "unhealthy"}
	found := false
	for _, status := range validStatuses {
		if dto.Status == status {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid health status: %s", dto.Status)
	}

	if dto.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	for _, component := range dto.Components {
		if component.Name == "" {
			return fmt.Errorf("component name is required")
		}
		found = false
		for _, status := range validStatuses {
			if component.Status == status {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid component status for %s: %s", component.Name, component.Status)
		}
	}

	return nil
}

// IsHealthy returns true if the health check indicates a healthy system
func (dto *HealthCheckDTO) IsHealthy() bool {
	return dto.Status == "healthy"
}

// IsDegraded returns true if the health check indicates a degraded system
func (dto *HealthCheckDTO) IsDegraded() bool {
	return dto.Status == "degraded"
}

// IsUnhealthy returns true if the health check indicates an unhealthy system
func (dto *HealthCheckDTO) IsUnhealthy() bool {
	return dto.Status == "unhealthy"
}

// GetUnhealthyComponents returns a list of unhealthy components
func (dto *HealthCheckDTO) GetUnhealthyComponents() []ComponentHealthDTO {
	var unhealthy []ComponentHealthDTO
	for _, component := range dto.Components {
		if component.Status != "healthy" {
			unhealthy = append(unhealthy, component)
		}
	}
	return unhealthy
}

// HasCapability checks if a specific capability is available
func (dto *CollectorCapabilitiesDTO) HasCapability(capability string) bool {
	switch capability {
	case "cpu":
		return dto.SupportsCPU
	case "memory":
		return dto.SupportsMemory
	case "disk":
		return dto.SupportsDisk
	case "network":
		return dto.SupportsNetwork
	case "process":
		return dto.SupportsProcess
	case "gpu":
		return dto.SupportsGPU
	case "pressure":
		return dto.SupportsPressure
	case "temperature":
		return dto.SupportsTemperature
	case "continuous":
		return dto.SupportsContinuous
	default:
		if dto.PlatformSpecific != nil {
			return dto.PlatformSpecific[capability]
		}
		return false
	}
}

// GetEnabledCapabilities returns a list of enabled capabilities
func (dto *CollectorCapabilitiesDTO) GetEnabledCapabilities() []string {
	var capabilities []string

	if dto.SupportsCPU {
		capabilities = append(capabilities, "cpu")
	}
	if dto.SupportsMemory {
		capabilities = append(capabilities, "memory")
	}
	if dto.SupportsDisk {
		capabilities = append(capabilities, "disk")
	}
	if dto.SupportsNetwork {
		capabilities = append(capabilities, "network")
	}
	if dto.SupportsProcess {
		capabilities = append(capabilities, "process")
	}
	if dto.SupportsGPU {
		capabilities = append(capabilities, "gpu")
	}
	if dto.SupportsPressure {
		capabilities = append(capabilities, "pressure")
	}
	if dto.SupportsTemperature {
		capabilities = append(capabilities, "temperature")
	}
	if dto.SupportsContinuous {
		capabilities = append(capabilities, "continuous")
	}

	// Add platform-specific capabilities
	if dto.PlatformSpecific != nil {
		for cap, enabled := range dto.PlatformSpecific {
			if enabled {
				capabilities = append(capabilities, cap)
			}
		}
	}

	return capabilities
}