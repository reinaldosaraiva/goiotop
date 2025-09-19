package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Collection   CollectionConfig   `mapstructure:"collection"`
	Display      DisplayConfig      `mapstructure:"display"`
	Export       ExportConfig       `mapstructure:"export"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Repository   RepositoryConfig   `mapstructure:"repository"`
	Alerting     AlertingConfig     `mapstructure:"alerting"`
	Historical   HistoricalConfig   `mapstructure:"historical"`
	Optimization OptimizationConfig `mapstructure:"optimization"`
}

// AppConfig contains general application settings
type AppConfig struct {
	Name              string        `mapstructure:"name"`
	Version           string        `mapstructure:"version"`
	Mode              string        `mapstructure:"mode"` // "production", "development", "test"
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
}

// CollectionConfig contains metrics collection settings
type CollectionConfig struct {
	Interval         time.Duration   `mapstructure:"interval"`
	Timeout          time.Duration   `mapstructure:"timeout"`
	BatchSize        int             `mapstructure:"batch_size"`
	MaxRetries       int             `mapstructure:"max_retries"`
	RetryDelay       time.Duration   `mapstructure:"retry_delay"`
	EnableCPU        bool            `mapstructure:"enable_cpu"`
	EnableMemory     bool            `mapstructure:"enable_memory"`
	EnableDisk       bool            `mapstructure:"enable_disk"`
	EnableNetwork    bool            `mapstructure:"enable_network"`
	EnableProcess    bool            `mapstructure:"enable_process"`
	EnableGPU        bool            `mapstructure:"enable_gpu"`
	ProcessFilter    ProcessFilter   `mapstructure:"process_filter"`
	CollectorOptions CollectorConfig `mapstructure:"collector_options"`
}

// ProcessFilter defines process filtering criteria
type ProcessFilter struct {
	MinCPUPercent    float64  `mapstructure:"min_cpu_percent"`
	MinMemoryPercent float64  `mapstructure:"min_memory_percent"`
	ExcludeKernel    bool     `mapstructure:"exclude_kernel"`
	IncludePatterns  []string `mapstructure:"include_patterns"`
	ExcludePatterns  []string `mapstructure:"exclude_patterns"`
	TopN             int      `mapstructure:"top_n"`
}

// CollectorConfig maps to domain CollectionOptions
type CollectorConfig struct {
	IncludeIdle        bool          `mapstructure:"include_idle"`
	IncludeKernel      bool          `mapstructure:"include_kernel"`
	IncludeSystemProcs bool          `mapstructure:"include_system_procs"`
	ProcessLimit       int           `mapstructure:"process_limit"`
	SortBy             string        `mapstructure:"sort_by"`
	MinCPUThreshold    float64       `mapstructure:"min_cpu_threshold"`
	MinMemThreshold    float64       `mapstructure:"min_mem_threshold"`
	SampleDuration     time.Duration `mapstructure:"sample_duration"`
	EnablePressure     bool          `mapstructure:"enable_pressure"`
	EnableGPU          bool          `mapstructure:"enable_gpu"`
}

// DisplayConfig contains display and UI settings
type DisplayConfig struct {
	RefreshRate      time.Duration `mapstructure:"refresh_rate"`
	MaxRows          int           `mapstructure:"max_rows"`
	ShowSystemStats  bool          `mapstructure:"show_system_stats"`
	ShowProcessList  bool          `mapstructure:"show_process_list"`
	ShowGraphs       bool          `mapstructure:"show_graphs"`
	GraphHistory     time.Duration `mapstructure:"graph_history"`
	ColorScheme      string        `mapstructure:"color_scheme"`
	DateFormat       string        `mapstructure:"date_format"`
	TimeFormat       string        `mapstructure:"time_format"`
	EnablePagination bool          `mapstructure:"enable_pagination"`
	PageSize         int           `mapstructure:"page_size"`
}

// ExportConfig contains export settings
type ExportConfig struct {
	Enabled     bool              `mapstructure:"enabled"`
	Format      string            `mapstructure:"format"` // "json", "csv", "prometheus"
	Destination string            `mapstructure:"destination"`
	Interval    time.Duration     `mapstructure:"interval"`
	BufferSize  int               `mapstructure:"buffer_size"`
	Compression bool              `mapstructure:"compression"`
	BatchSize   int               `mapstructure:"batch_size"`
	Prometheus  PrometheusConfig  `mapstructure:"prometheus"`
	FileExport  FileExportConfig  `mapstructure:"file"`
	HTTPHeaders map[string]string `mapstructure:"http_headers"`
	S3Endpoint  string            `mapstructure:"s3_endpoint"`
}

// PrometheusConfig contains Prometheus exporter settings
type PrometheusConfig struct {
	Enabled    bool              `mapstructure:"enabled"`
	Port       int               `mapstructure:"port"`
	Path       string            `mapstructure:"path"`
	Namespace  string            `mapstructure:"namespace"`
	Subsystem  string            `mapstructure:"subsystem"`
	Labels     map[string]string `mapstructure:"labels"`
}

// FileExportConfig contains file export settings
type FileExportConfig struct {
	Directory    string        `mapstructure:"directory"`
	FilePattern  string        `mapstructure:"file_pattern"`
	MaxFileSize  int64         `mapstructure:"max_file_size"`
	MaxFiles     int           `mapstructure:"max_files"`
	RotateDaily  bool          `mapstructure:"rotate_daily"`
	Compress     bool          `mapstructure:"compress"`
	BufferSize   int           `mapstructure:"buffer_size"`
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level           string          `mapstructure:"level"`
	Format          string          `mapstructure:"format"` // "text", "json"
	Output          string          `mapstructure:"output"` // "stdout", "stderr", "file"
	File            string          `mapstructure:"file"`
	MaxSize         int             `mapstructure:"max_size"` // megabytes
	MaxAge          int             `mapstructure:"max_age"`  // days
	MaxBackups      int             `mapstructure:"max_backups"`
	Compress        bool            `mapstructure:"compress"`
	EnableCaller    bool            `mapstructure:"enable_caller"`
	EnableStacktrace bool           `mapstructure:"enable_stacktrace"`
	Fields          map[string]any  `mapstructure:"fields"`
}

// RepositoryConfig contains repository settings
type RepositoryConfig struct {
	Type            string        `mapstructure:"type"` // "memory", "sqlite", "postgres"
	ConnectionURL   string        `mapstructure:"connection_url"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	RetentionDays   int           `mapstructure:"retention_days"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// AlertingConfig contains alerting system configuration
type AlertingConfig struct {
	Enabled            bool                     `mapstructure:"enabled"`
	EvaluationInterval time.Duration            `mapstructure:"evaluation_interval"`
	RateLimitWindow    time.Duration            `mapstructure:"rate_limit_window"`
	MaxHistorySize     int                      `mapstructure:"max_history_size"`
	DefaultSeverity    string                   `mapstructure:"default_severity"`
	Rules              []AlertRuleConfig        `mapstructure:"rules"`
	Notifications      NotificationConfig       `mapstructure:"notifications"`
}

// AlertRuleConfig represents an alert rule configuration
type AlertRuleConfig struct {
	ID          string            `mapstructure:"id"`
	Name        string            `mapstructure:"name"`
	Description string            `mapstructure:"description"`
	MetricType  string            `mapstructure:"metric_type"`
	Threshold   float64           `mapstructure:"threshold"`
	Operator    string            `mapstructure:"operator"`
	Duration    time.Duration     `mapstructure:"duration"`
	Severity    string            `mapstructure:"severity"`
	Labels      map[string]string `mapstructure:"labels"`
	Annotations map[string]string `mapstructure:"annotations"`
	Channels    []string          `mapstructure:"channels"`
	Enabled     bool              `mapstructure:"enabled"`
}

// NotificationConfig contains notification configuration
type NotificationConfig struct {
	Enabled           bool            `mapstructure:"enabled"`
	RateLimitPerAlert int             `mapstructure:"rate_limit_per_alert"`
	RetryAttempts     int             `mapstructure:"retry_attempts"`
	RetryDelay        time.Duration   `mapstructure:"retry_delay"`
	Channels          []ChannelConfig `mapstructure:"channels"`
}

// ChannelConfig contains notification channel configuration
type ChannelConfig struct {
	Name    string                 `mapstructure:"name"`
	Type    string                 `mapstructure:"type"`
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

// HistoricalConfig contains historical metrics configuration
type HistoricalConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	SystemMetricsCapacity  int           `mapstructure:"system_metrics_capacity"`
	ProcessMetricsCapacity int           `mapstructure:"process_metrics_capacity"`
	CPUPressureCapacity    int           `mapstructure:"cpu_pressure_capacity"`
	DiskIOCapacity         int           `mapstructure:"disk_io_capacity"`
	RetentionPeriod        time.Duration `mapstructure:"retention_period"`
	AggregationInterval    time.Duration `mapstructure:"aggregation_interval"`
	CompactionInterval     time.Duration `mapstructure:"compaction_interval"`
}

// OptimizationConfig contains performance optimization configuration
type OptimizationConfig struct {
	EnablePooling         bool          `mapstructure:"enable_pooling"`
	EnableCaching         bool          `mapstructure:"enable_caching"`
	CacheL1Size           int           `mapstructure:"cache_l1_size"`
	CacheL2Size           int           `mapstructure:"cache_l2_size"`
	CacheL1TTL            time.Duration `mapstructure:"cache_l1_ttl"`
	CacheL2TTL            time.Duration `mapstructure:"cache_l2_ttl"`
	CacheWarmupEnabled    bool          `mapstructure:"cache_warmup_enabled"`
	CacheCleanupInterval  time.Duration `mapstructure:"cache_cleanup_interval"`
	PoolProcessMetricsSize int          `mapstructure:"pool_process_metrics_size"`
	PoolBufferSize        int           `mapstructure:"pool_buffer_size"`
	MaxMemoryMB           int           `mapstructure:"max_memory_mb"`
}

// Load loads configuration from file and environment
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/goiotop")
		v.AddConfigPath("$HOME/.goiotop")
	}

	// Enable environment variables
	v.SetEnvPrefix("GOIOTOP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults and env vars
	}

	// Unmarshal configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadWithWatcher loads config and watches for changes
func LoadWithWatcher(configPath string, onChange func(*Config)) (*Config, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	v := viper.New()

	// Configure watcher with the same file/path as Load
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/goiotop")
		v.AddConfigPath("$HOME/.goiotop")
	}

	// Read config before watching
	if err := v.ReadInConfig(); err != nil {
		// If config file not found, still return the loaded config
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return cfg, fmt.Errorf("error reading config for watcher: %w", err)
		}
	}

	v.WatchConfig()
	// Use the correct callback signature with fsnotify.Event
	v.OnConfigChange(func(e fsnotify.Event) {
		newCfg, err := Load(configPath)
		if err == nil && onChange != nil {
			onChange(newCfg)
		}
	})

	return cfg, nil
}

// setDefaults sets default values for all configuration options
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "goiotop")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.mode", "production")
	v.SetDefault("app.shutdown_timeout", 10*time.Second)
	v.SetDefault("app.health_check_period", 30*time.Second)

	// Collection defaults
	v.SetDefault("collection.interval", 2*time.Second)
	v.SetDefault("collection.timeout", 5*time.Second)
	v.SetDefault("collection.batch_size", 100)
	v.SetDefault("collection.max_retries", 3)
	v.SetDefault("collection.retry_delay", time.Second)
	v.SetDefault("collection.enable_cpu", true)
	v.SetDefault("collection.enable_memory", true)
	v.SetDefault("collection.enable_disk", true)
	v.SetDefault("collection.enable_network", true)
	v.SetDefault("collection.enable_process", true)
	v.SetDefault("collection.enable_gpu", false)

	// Process filter defaults
	v.SetDefault("collection.process_filter.min_cpu_percent", 0.0)
	v.SetDefault("collection.process_filter.min_memory_percent", 0.0)
	v.SetDefault("collection.process_filter.exclude_kernel", false)
	v.SetDefault("collection.process_filter.top_n", 20)

	// Collector options defaults (matching domain defaults)
	v.SetDefault("collection.collector_options.include_idle", true)
	v.SetDefault("collection.collector_options.include_kernel", true)
	v.SetDefault("collection.collector_options.include_system_procs", false)
	v.SetDefault("collection.collector_options.process_limit", 100)
	v.SetDefault("collection.collector_options.sort_by", "cpu")
	v.SetDefault("collection.collector_options.min_cpu_threshold", 0.0)
	v.SetDefault("collection.collector_options.min_mem_threshold", 0.0)
	v.SetDefault("collection.collector_options.sample_duration", time.Second)
	v.SetDefault("collection.collector_options.enable_pressure", true)
	v.SetDefault("collection.collector_options.enable_gpu", false)

	// Display defaults
	v.SetDefault("display.refresh_rate", 2*time.Second)
	v.SetDefault("display.max_rows", 20)
	v.SetDefault("display.show_system_stats", true)
	v.SetDefault("display.show_process_list", true)
	v.SetDefault("display.show_graphs", true)
	v.SetDefault("display.graph_history", 60*time.Second)
	v.SetDefault("display.color_scheme", "default")
	v.SetDefault("display.date_format", "2006-01-02")
	v.SetDefault("display.time_format", "15:04:05")
	v.SetDefault("display.enable_pagination", true)
	v.SetDefault("display.page_size", 20)

	// Export defaults
	v.SetDefault("export.enabled", false)
	v.SetDefault("export.format", "json")
	v.SetDefault("export.destination", "file")
	v.SetDefault("export.interval", 60*time.Second)
	v.SetDefault("export.buffer_size", 1000)
	v.SetDefault("export.compression", false)
	v.SetDefault("export.batch_size", 100)

	// Prometheus defaults
	v.SetDefault("export.prometheus.enabled", false)
	v.SetDefault("export.prometheus.port", 9090)
	v.SetDefault("export.prometheus.path", "/metrics")
	v.SetDefault("export.prometheus.namespace", "goiotop")
	v.SetDefault("export.prometheus.subsystem", "")

	// File export defaults
	v.SetDefault("export.file.directory", "./data")
	v.SetDefault("export.file.file_pattern", "metrics_%s.json")
	v.SetDefault("export.file.max_file_size", 100*1024*1024) // 100MB
	v.SetDefault("export.file.max_files", 10)
	v.SetDefault("export.file.rotate_daily", true)
	v.SetDefault("export.file.compress", false)
	v.SetDefault("export.file.buffer_size", 4096)
	v.SetDefault("export.file.flush_timeout", 5*time.Second)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.file", "./logs/goiotop.log")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_age", 30)
	v.SetDefault("logging.max_backups", 10)
	v.SetDefault("logging.compress", true)
	v.SetDefault("logging.enable_caller", false)
	v.SetDefault("logging.enable_stacktrace", false)

	// Repository defaults
	v.SetDefault("repository.type", "memory")
	v.SetDefault("repository.max_connections", 10)
	v.SetDefault("repository.max_idle_conns", 5)
	v.SetDefault("repository.conn_max_lifetime", 30*time.Minute)
	v.SetDefault("repository.conn_max_idle_time", 10*time.Minute)
	v.SetDefault("repository.retention_days", 7)
	v.SetDefault("repository.cleanup_interval", 24*time.Hour)

	// Alerting defaults
	v.SetDefault("alerting.enabled", true)
	v.SetDefault("alerting.evaluation_interval", 30*time.Second)
	v.SetDefault("alerting.rate_limit_window", 5*time.Minute)
	v.SetDefault("alerting.max_history_size", 1000)
	v.SetDefault("alerting.default_severity", "warning")
	v.SetDefault("alerting.notifications.enabled", true)
	v.SetDefault("alerting.notifications.rate_limit_per_alert", 5)
	v.SetDefault("alerting.notifications.retry_attempts", 3)
	v.SetDefault("alerting.notifications.retry_delay", 5*time.Second)

	// Historical defaults
	v.SetDefault("historical.enabled", true)
	v.SetDefault("historical.system_metrics_capacity", 10000)
	v.SetDefault("historical.process_metrics_capacity", 50000)
	v.SetDefault("historical.cpu_pressure_capacity", 10000)
	v.SetDefault("historical.disk_io_capacity", 10000)
	v.SetDefault("historical.retention_period", 24*time.Hour)
	v.SetDefault("historical.aggregation_interval", 5*time.Minute)
	v.SetDefault("historical.compaction_interval", 1*time.Hour)

	// Optimization defaults
	v.SetDefault("optimization.enable_pooling", true)
	v.SetDefault("optimization.enable_caching", true)
	v.SetDefault("optimization.cache_l1_size", 100)
	v.SetDefault("optimization.cache_l2_size", 1000)
	v.SetDefault("optimization.cache_l1_ttl", 10*time.Second)
	v.SetDefault("optimization.cache_l2_ttl", 60*time.Second)
	v.SetDefault("optimization.cache_warmup_enabled", true)
	v.SetDefault("optimization.cache_cleanup_interval", 5*time.Minute)
	v.SetDefault("optimization.pool_process_metrics_size", 100)
	v.SetDefault("optimization.pool_buffer_size", 4096)
	v.SetDefault("optimization.max_memory_mb", 100)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate app config
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.App.ShutdownTimeout <= 0 {
		return fmt.Errorf("app.shutdown_timeout must be positive")
	}

	// Validate collection config
	if c.Collection.Interval <= 0 {
		return fmt.Errorf("collection.interval must be positive")
	}
	if c.Collection.Timeout <= 0 {
		return fmt.Errorf("collection.timeout must be positive")
	}
	if c.Collection.BatchSize <= 0 {
		return fmt.Errorf("collection.batch_size must be positive")
	}
	if c.Collection.CollectorOptions.ProcessLimit <= 0 {
		return fmt.Errorf("collection.collector_options.process_limit must be positive")
	}

	// Validate display config
	if c.Display.RefreshRate <= 0 {
		return fmt.Errorf("display.refresh_rate must be positive")
	}
	if c.Display.MaxRows <= 0 {
		return fmt.Errorf("display.max_rows must be positive")
	}

	// Validate export config
	if c.Export.Enabled {
		validFormats := []string{"json", "csv", "prometheus"}
		found := false
		for _, format := range validFormats {
			if c.Export.Format == format {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("export.format must be one of: %v", validFormats)
		}

		if c.Export.Prometheus.Enabled && c.Export.Prometheus.Port <= 0 {
			return fmt.Errorf("export.prometheus.port must be positive")
		}
	}

	// Validate logging config
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	found := false
	for _, level := range validLevels {
		if strings.ToLower(c.Logging.Level) == level {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("logging.level must be one of: %v", validLevels)
	}

	validFormats := []string{"text", "json"}
	found = false
	for _, format := range validFormats {
		if c.Logging.Format == format {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("logging.format must be one of: %v", validFormats)
	}

	// Validate repository config
	validTypes := []string{"memory", "sqlite", "postgres"}
	found = false
	for _, t := range validTypes {
		if c.Repository.Type == t {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("repository.type must be one of: %v", validTypes)
	}

	return nil
}

// MergeWithDefaults merges configuration with default values
func (c *Config) MergeWithDefaults() *Config {
	// This is handled by viper's SetDefault mechanism
	return c
}

// SaveToFile saves the configuration to a file
func (c *Config) SaveToFile(path string) error {
	v := viper.New()

	// Convert config struct to viper
	if err := v.MergeConfigMap(c.toMap()); err != nil {
		return fmt.Errorf("error converting config to map: %w", err)
	}

	// Ensure directory exists using os.MkdirAll
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	// Write config file
	return v.WriteConfigAs(path)
}

// toMap converts the config struct to a map for viper
func (c *Config) toMap() map[string]interface{} {
	// This would normally use a library like mapstructure
	// For simplicity, we'll create a basic implementation
	return map[string]interface{}{
		"app":          c.App,
		"collection":   c.Collection,
		"display":      c.Display,
		"export":       c.Export,
		"logging":      c.Logging,
		"repository":   c.Repository,
		"alerting":     c.Alerting,
		"historical":   c.Historical,
		"optimization": c.Optimization,
	}
}

// GetCollectorOptions converts config to domain CollectionOptions
func (c *Config) GetCollectorOptions() map[string]interface{} {
	return map[string]interface{}{
		"includeIdle":        c.Collection.CollectorOptions.IncludeIdle,
		"includeKernel":      c.Collection.CollectorOptions.IncludeKernel,
		"includeSystemProcs": c.Collection.CollectorOptions.IncludeSystemProcs,
		"processLimit":       c.Collection.CollectorOptions.ProcessLimit,
		"sortBy":             c.Collection.CollectorOptions.SortBy,
		"minCPUThreshold":    c.Collection.CollectorOptions.MinCPUThreshold,
		"minMemThreshold":    c.Collection.CollectorOptions.MinMemThreshold,
		"sampleDuration":     c.Collection.CollectorOptions.SampleDuration,
		"enablePressure":     c.Collection.CollectorOptions.EnablePressure,
		"enableGPU":          c.Collection.CollectorOptions.EnableGPU,
	}
}