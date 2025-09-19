package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/config"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/application/usecases"
	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/alerting"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/export"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/optimization"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/storage"
	"github.com/sirupsen/logrus"
)

// MetricsOrchestrator coordinates between use cases and manages application workflow
type MetricsOrchestrator struct {
	collector   services.MetricsCollector
	repository  repositories.MetricsRepository
	config      *config.Config
	logger      *config.Logger

	// Use cases
	collectUseCase  *usecases.CollectMetrics
	displayUseCase  *usecases.DisplayMetrics
	exportUseCase   *usecases.ExportMetrics
	alertingUseCase *usecases.AlertingUseCase

	// Advanced features
	prometheusExporter *export.PrometheusExporter
	alertingEngine     *alerting.AlertingEngine
	poolManager        *optimization.MetricsPoolManager
	cache              *optimization.MultiLevelCache

	// State management
	mu               sync.RWMutex
	running          bool
	continuousCancel context.CancelFunc
	healthStatus     dto.HealthCheckDTO
	lastCollection   time.Time
	collectionStats  CollectionStats
}

// CollectionStats tracks collection statistics
type CollectionStats struct {
	TotalCollections  int64
	SuccessfulCollections int64
	FailedCollections int64
	TotalDuration    time.Duration
	LastError        error
	LastErrorTime    time.Time
}

// NewMetricsOrchestrator creates a new MetricsOrchestrator
func NewMetricsOrchestrator(
	collector services.MetricsCollector,
	repository repositories.MetricsRepository,
	cfg *config.Config,
	logger *config.Logger,
) *MetricsOrchestrator {
	return &MetricsOrchestrator{
		collector:  collector,
		repository: repository,
		config:     cfg,
		logger:     logger,
		healthStatus: dto.HealthCheckDTO{
			Status:    "healthy",
			Timestamp: time.Now(),
		},
	}
}

// Initialize initializes the orchestrator and all components
func (o *MetricsOrchestrator) Initialize(ctx context.Context) error {
	o.logger.Info("Initializing MetricsOrchestrator")

	// Initialize collector
	if err := o.collector.Initialize(ctx); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to initialize collector")
		return fmt.Errorf("failed to initialize collector: %w", err)
	}

	// Initialize repository if needed
	if o.repository == nil && o.config.Historical.Enabled {
		// Create memory repository with advanced features
		repoConfig := storage.MemoryRepositoryConfig{
			SystemMetricsCapacity:  o.config.Historical.SystemMetricsCapacity,
			ProcessMetricsCapacity: o.config.Historical.ProcessMetricsCapacity,
			CPUPressureCapacity:    o.config.Historical.CPUPressureCapacity,
			DiskIOCapacity:         o.config.Historical.DiskIOCapacity,
			RetentionPeriod:        o.config.Historical.RetentionPeriod,
			CleanupInterval:        o.config.Historical.CompactionInterval,
		}
		o.repository = storage.NewMemoryRepository(repoConfig)
	}

	// Check repository connectivity with Ping instead of Initialize
	if err := o.repository.Ping(ctx); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to ping repository")
		return fmt.Errorf("failed to ping repository: %w", err)
	}

	// Initialize performance optimization
	if o.config.Optimization.EnablePooling {
		o.poolManager = optimization.NewMetricsPoolManager()
		o.logger.Info("Object pooling enabled")
	}

	if o.config.Optimization.EnableCaching {
		cacheConfig := optimization.CacheConfig{
			L1Size:          o.config.Optimization.CacheL1Size,
			L2Size:          o.config.Optimization.CacheL2Size,
			L1TTL:           o.config.Optimization.CacheL1TTL,
			L2TTL:           o.config.Optimization.CacheL2TTL,
			WarmupEnabled:   o.config.Optimization.CacheWarmupEnabled,
			CleanupInterval: o.config.Optimization.CacheCleanupInterval,
			MaxMemoryBytes:  int64(o.config.Optimization.MaxMemoryMB) * 1024 * 1024,
		}
		logrusLogger := logrus.StandardLogger()
		var err error
		o.cache, err = optimization.NewMultiLevelCache(cacheConfig, logrusLogger)
		if err != nil {
			o.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to initialize cache")
		}
	}

	// Initialize Prometheus exporter
	if o.config.Export.Prometheus.Enabled {
		prometheusConfig := export.PrometheusConfig{
			Enabled:          o.config.Export.Prometheus.Enabled,
			Port:             o.config.Export.Prometheus.Port,
			Path:             o.config.Export.Prometheus.Path,
			UpdateInterval:   o.config.Collection.Interval,
			IncludeProcesses: o.config.Collection.EnableProcess,
			ProcessLimit:     o.config.Collection.CollectorOptions.ProcessLimit,
		}
		logrusLogger := logrus.StandardLogger()
		o.prometheusExporter = export.NewPrometheusExporter(prometheusConfig, logrusLogger)
		if err := o.prometheusExporter.Start(ctx); err != nil {
			o.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to start Prometheus exporter")
		}
	}

	// Initialize alerting engine
	if o.config.Alerting.Enabled {
		alertingConfig := alerting.AlertingConfig{
			Enabled:            o.config.Alerting.Enabled,
			EvaluationInterval: o.config.Alerting.EvaluationInterval,
			RateLimitWindow:    o.config.Alerting.RateLimitWindow,
			MaxHistorySize:     o.config.Alerting.MaxHistorySize,
			DefaultSeverity:    alerting.AlertSeverity(o.config.Alerting.DefaultSeverity),
		}

		notificationConfig := alerting.NotificationConfig{
			Enabled:           o.config.Alerting.Notifications.Enabled,
			RateLimitPerAlert: o.config.Alerting.Notifications.RateLimitPerAlert,
			RetryAttempts:     o.config.Alerting.Notifications.RetryAttempts,
			RetryDelay:        o.config.Alerting.Notifications.RetryDelay,
		}

		// Convert channel configs
		for _, ch := range o.config.Alerting.Notifications.Channels {
			notificationConfig.Channels = append(notificationConfig.Channels, alerting.ChannelConfig{
				Name:    ch.Name,
				Type:    ch.Type,
				Enabled: ch.Enabled,
				Config:  ch.Config,
			})
		}

		logrusLogger := logrus.StandardLogger()
		notificationManager := alerting.NewNotificationManager(notificationConfig, logrusLogger)
		o.alertingEngine = alerting.NewAlertingEngine(alertingConfig, notificationManager, logrusLogger)

		// Load alert rules from config
		for _, ruleConfig := range o.config.Alerting.Rules {
			rule := &alerting.AlertRule{
				ID:          ruleConfig.ID,
				Name:        ruleConfig.Name,
				Description: ruleConfig.Description,
				MetricType:  ruleConfig.MetricType,
				Threshold:   ruleConfig.Threshold,
				Operator:    ruleConfig.Operator,
				Duration:    ruleConfig.Duration,
				Severity:    alerting.AlertSeverity(ruleConfig.Severity),
				Labels:      ruleConfig.Labels,
				Annotations: ruleConfig.Annotations,
				Channels:    ruleConfig.Channels,
				Enabled:     ruleConfig.Enabled,
			}
			if err := o.alertingEngine.AddRule(rule); err != nil {
				o.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
					"rule":  rule.ID,
				}).Error("Failed to add alert rule")
			}
		}

		if err := o.alertingEngine.Start(ctx); err != nil {
			o.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to start alerting engine")
		}

		o.alertingUseCase = usecases.NewAlertingUseCase(o.alertingEngine, logrusLogger)
	}

	// Create use cases
	o.collectUseCase = usecases.NewCollectMetrics(
		o.collector,
		o.repository,
		o.logger,
		&o.config.Collection,
	)

	o.displayUseCase = usecases.NewDisplayMetrics(
		o.repository,
		o.logger,
		&o.config.Display,
	)

	o.exportUseCase = usecases.NewExportMetrics(
		o.repository,
		o.logger,
		&o.config.Export,
	)

	// Perform initial health check
	if err := o.performHealthCheck(ctx); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Initial health check failed")
	}

	// Start health check routine
	go o.healthCheckRoutine(ctx)

	// Start cleanup routine if configured
	if o.config.Repository.RetentionDays > 0 {
		go o.cleanupRoutine(ctx)
	}

	o.logger.Info("MetricsOrchestrator initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the orchestrator
func (o *MetricsOrchestrator) Shutdown(ctx context.Context) error {
	o.logger.Info("Shutting down MetricsOrchestrator")

	// Stop continuous collection if running
	if err := o.StopContinuousCollection(ctx); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Error stopping continuous collection")
	}

	// Stop Prometheus exporter
	if o.prometheusExporter != nil {
		if err := o.prometheusExporter.Stop(ctx); err != nil {
			o.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Error stopping Prometheus exporter")
		}
	}

	// Stop alerting engine
	if o.alertingEngine != nil {
		if err := o.alertingEngine.Stop(ctx); err != nil {
			o.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Error stopping alerting engine")
		}
	}

	// Cleanup collector
	if err := o.collector.Cleanup(); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Error cleaning up collector")
	}

	// Close repository connections (no ctx needed)
	if err := o.repository.Close(); err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Error closing repository")
	}

	// Clear cache
	if o.cache != nil {
		o.cache.Clear()
	}

	// Close logger
	if err := o.logger.Close(); err != nil {
		return fmt.Errorf("error closing logger: %w", err)
	}

	o.logger.Info("MetricsOrchestrator shut down successfully")
	return nil
}

// CollectMetrics performs metric collection
func (o *MetricsOrchestrator) CollectMetrics(ctx context.Context, request dto.CollectionRequestDTO) (*dto.CollectionResultDTO, error) {
	o.mu.Lock()
	o.collectionStats.TotalCollections++
	o.mu.Unlock()

	startTime := time.Now()

	// Validate capabilities
	if err := o.collectUseCase.ValidateCollectionCapabilities(request); err != nil {
		o.mu.Lock()
		o.collectionStats.FailedCollections++
		o.collectionStats.LastError = err
		o.collectionStats.LastErrorTime = time.Now()
		o.mu.Unlock()
		return nil, err
	}

	// Execute collection
	result, err := o.collectUseCase.Execute(ctx, request)

	o.mu.Lock()
	defer o.mu.Unlock()

	o.collectionStats.TotalDuration += time.Since(startTime)
	o.lastCollection = time.Now()

	if err != nil {
		o.collectionStats.FailedCollections++
		o.collectionStats.LastError = err
		o.collectionStats.LastErrorTime = time.Now()
		return nil, err
	}

	if result.Success {
		o.collectionStats.SuccessfulCollections++
	} else {
		o.collectionStats.FailedCollections++
	}

	return result, nil
}

// DisplayMetrics displays metrics based on options
func (o *MetricsOrchestrator) DisplayMetrics(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	return o.displayUseCase.Execute(ctx, options)
}

// ExportMetrics exports metrics based on request
func (o *MetricsOrchestrator) ExportMetrics(ctx context.Context, request dto.ExportRequestDTO) (*dto.ExportResultDTO, error) {
	return o.exportUseCase.Execute(ctx, request)
}

// StartContinuousCollection starts continuous metric collection
func (o *MetricsOrchestrator) StartContinuousCollection(ctx context.Context, interval time.Duration) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return fmt.Errorf("continuous collection already running")
	}

	// Create cancellable context
	collectionCtx, cancel := context.WithCancel(ctx)
	o.continuousCancel = cancel

	// Start collection
	callback := func(result dto.CollectionResultDTO) {
		// Update statistics
		o.mu.Lock()
		o.lastCollection = time.Now()
		if result.Success {
			o.collectionStats.SuccessfulCollections++
		} else {
			o.collectionStats.FailedCollections++
			if len(result.Errors) > 0 {
				o.collectionStats.LastError = fmt.Errorf(result.Errors[0].Message)
				o.collectionStats.LastErrorTime = time.Now()
			}
		}
		o.mu.Unlock()

		// Log collection result
		if result.Success {
			o.logger.WithFields(map[string]interface{}{
				"items_collected": result.Statistics.ItemsCollected,
				"duration_ms":     result.Duration.Milliseconds(),
			}).Debug("Continuous collection cycle completed")
		} else {
			o.logger.WithFields(map[string]interface{}{
				"errors": len(result.Errors),
			}).Warn("Continuous collection cycle failed")
		}

		// Trigger export if enabled
		if o.config.Export.Enabled {
			go o.autoExport(context.Background())
		}
	}

	if err := o.collectUseCase.StartContinuousCollection(collectionCtx, interval, callback); err != nil {
		cancel()
		return err
	}

	o.running = true
	o.logger.Info("Continuous collection started")

	return nil
}

// StopContinuousCollection stops continuous metric collection
func (o *MetricsOrchestrator) StopContinuousCollection(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return nil
	}

	if o.continuousCancel != nil {
		o.continuousCancel()
	}

	if err := o.collectUseCase.StopContinuousCollection(ctx); err != nil {
		return err
	}

	o.running = false
	o.logger.Info("Continuous collection stopped")

	return nil
}

// GetHealthStatus returns the current health status
func (o *MetricsOrchestrator) GetHealthStatus(ctx context.Context) dto.HealthCheckDTO {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.healthStatus
}

// GetSystemInfo returns system information
func (o *MetricsOrchestrator) GetSystemInfo(ctx context.Context) (*dto.SystemInfoDTO, error) {
	info, err := o.collector.GetSystemInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	capabilities := o.collector.GetCapabilities()

	// Convert SystemInfo struct to DTO
	systemInfo := &dto.SystemInfoDTO{
		Hostname:        info.Hostname,
		Platform:        info.Platform,
		PlatformVersion: info.OS, // Using OS field for platform version
		Architecture:    info.Architecture,
		CPUModel:        info.CPUModel,
		CPUCores:        info.CPUCores,
		CPUThreads:      info.CPUThreads,
		TotalMemory:     info.TotalMemory,
		BootTime:        info.BootTime,
		Uptime:          info.Uptime,
		CollectorInfo: dto.CollectorInfoDTO{
			Version: "1.0.0",
			Backend: "native",
		},
	}

	// Add capabilities from struct
	capList := make([]string, 0)
	if capabilities.SupportsSystemMetrics {
		capList = append(capList, "system")
	}
	if capabilities.SupportsProcessMetrics {
		capList = append(capList, "process")
	}
	if capabilities.SupportsCPUPressure {
		capList = append(capList, "pressure")
	}
	if capabilities.SupportsDiskIO {
		capList = append(capList, "disk")
	}
	if capabilities.SupportsNetworkIO {
		capList = append(capList, "network")
	}
	systemInfo.CollectorInfo.Capabilities = capList

	return systemInfo, nil
}

// GetCollectorCapabilities returns collector capabilities
func (o *MetricsOrchestrator) GetCollectorCapabilities(ctx context.Context) (*dto.CollectorCapabilitiesDTO, error) {
	caps := o.collector.GetCapabilities()

	// Convert struct to DTO
	capabilities := &dto.CollectorCapabilitiesDTO{
		SupportsCPU:        caps.SupportsSystemMetrics, // CPU is part of system metrics
		SupportsMemory:     caps.SupportsSystemMetrics, // Memory is part of system metrics
		SupportsDisk:       caps.SupportsDiskIO,
		SupportsNetwork:    caps.SupportsNetworkIO,
		SupportsProcess:    caps.SupportsProcessMetrics,
		SupportsGPU:        false, // Check PlatformSpecific if available
		SupportsPressure:   caps.SupportsCPUPressure,
		SupportsContinuous: caps.SupportsSystemMetrics && caps.SupportsProcessMetrics,
	}

	// Check platform-specific for GPU and other features
	if caps.PlatformSpecific != nil {
		if gpu, ok := caps.PlatformSpecific["gpu"]; ok {
			capabilities.SupportsGPU = gpu
		}
		if temp, ok := caps.PlatformSpecific["temperature"]; ok {
			capabilities.SupportsTemperature = temp
		}
	}

	// Platform-specific capabilities
	capabilities.PlatformSpecific = caps.PlatformSpecific

	return capabilities, nil
}

// UpdateConfiguration updates the orchestrator configuration
func (o *MetricsOrchestrator) UpdateConfiguration(newConfig *config.Config) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.logger.Info("Updating configuration")

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Stop continuous collection if interval changed
	if o.running && newConfig.Collection.Interval != o.config.Collection.Interval {
		if err := o.StopContinuousCollection(context.Background()); err != nil {
			return fmt.Errorf("failed to stop continuous collection: %w", err)
		}

		// Restart with new interval
		defer func() {
			if err := o.StartContinuousCollection(context.Background(), newConfig.Collection.Interval); err != nil {
				o.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("Failed to restart continuous collection")
			}
		}()
	}

	// Update configuration
	o.config = newConfig

	// Recreate use cases with new config
	o.collectUseCase = usecases.NewCollectMetrics(
		o.collector,
		o.repository,
		o.logger,
		&newConfig.Collection,
	)

	o.displayUseCase = usecases.NewDisplayMetrics(
		o.repository,
		o.logger,
		&newConfig.Display,
	)

	o.exportUseCase = usecases.NewExportMetrics(
		o.repository,
		o.logger,
		&newConfig.Export,
	)

	o.logger.Info("Configuration updated successfully")
	return nil
}

// performHealthCheck performs a health check
func (o *MetricsOrchestrator) performHealthCheck(ctx context.Context) error {
	components := make([]dto.ComponentHealthDTO, 0)

	// Check collector health
	collectorHealth := dto.ComponentHealthDTO{
		Name:      "collector",
		Status:    "healthy",
		LastCheck: time.Now(),
		Details:   make(map[string]string),
	}

	// Collector doesn't have HealthCheck, use IsCollecting() as proxy
	if o.collector.IsCollecting() {
		collectorHealth.Details["collecting"] = "true"
	}
	if err := o.collector.GetLastError(); err != nil {
		collectorHealth.Status = "degraded"
		collectorHealth.Message = err.Error()
	}

	components = append(components, collectorHealth)

	// Check repository health
	repoHealth := dto.ComponentHealthDTO{
		Name:      "repository",
		Status:    "healthy",
		LastCheck: time.Now(),
		Details:   make(map[string]string),
	}

	// Use Ping for repository health check
	if err := o.repository.Ping(ctx); err != nil {
		repoHealth.Status = "unhealthy"
		repoHealth.Message = err.Error()
	}

	components = append(components, repoHealth)

	// Calculate overall status
	overallStatus := "healthy"
	issues := make([]string, 0)

	for _, comp := range components {
		if comp.Status == "unhealthy" {
			overallStatus = "unhealthy"
			issues = append(issues, fmt.Sprintf("%s: %s", comp.Name, comp.Message))
		} else if comp.Status == "degraded" {
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
			issues = append(issues, fmt.Sprintf("%s: %s", comp.Name, comp.Message))
		}
	}

	// Get statistics
	o.mu.RLock()
	stats := o.collectionStats
	lastCollection := o.lastCollection
	o.mu.RUnlock()

	// Create health metrics
	var errorRate float64
	if stats.TotalCollections > 0 {
		errorRate = float64(stats.FailedCollections) / float64(stats.TotalCollections) * 100
	}

	avgDuration := time.Duration(0)
	if stats.SuccessfulCollections > 0 {
		avgDuration = stats.TotalDuration / time.Duration(stats.SuccessfulCollections)
	}

	healthMetrics := dto.HealthMetricsDTO{
		CollectionsTotal:   stats.TotalCollections,
		CollectionsFailed:  stats.FailedCollections,
		CollectionDuration: avgDuration,
		ErrorRate:          errorRate,
		LastCollectionTime: lastCollection,
	}

	// Update health status
	o.mu.Lock()
	o.healthStatus = dto.HealthCheckDTO{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Components: components,
		Metrics:    healthMetrics,
		Issues:     issues,
		LastCheck:  time.Now(),
		NextCheck:  time.Now().Add(o.config.App.HealthCheckPeriod),
	}
	o.mu.Unlock()

	// Log health check result
	config.LogHealthCheck(o.logger, "orchestrator", overallStatus == "healthy", map[string]interface{}{
		"status":      overallStatus,
		"components":  len(components),
		"error_rate":  errorRate,
		"issues":      len(issues),
	})

	if overallStatus != "healthy" {
		return fmt.Errorf("health check failed: %s", overallStatus)
	}

	return nil
}

// healthCheckRoutine performs periodic health checks
func (o *MetricsOrchestrator) healthCheckRoutine(ctx context.Context) {
	ticker := time.NewTicker(o.config.App.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := o.performHealthCheck(ctx); err != nil {
				o.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Warn("Health check failed")
			}
		}
	}
}

// cleanupRoutine performs periodic cleanup of old data
func (o *MetricsOrchestrator) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(o.config.Repository.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.logger.Debug("Running cleanup routine")

			cutoff := time.Now().AddDate(0, 0, -o.config.Repository.RetentionDays)
			// Use PurgeOldMetrics instead of DeleteOldMetrics
			if err := o.repository.PurgeOldMetrics(ctx, cutoff); err != nil {
				o.logger.WithFields(map[string]interface{}{
					"error":  err.Error(),
					"cutoff": cutoff,
				}).Error("Cleanup failed")
			} else {
				o.logger.WithFields(map[string]interface{}{
					"cutoff": cutoff,
				}).Info("Cleanup completed")
			}
		}
	}
}

// autoExport performs automatic export based on configuration
func (o *MetricsOrchestrator) autoExport(ctx context.Context) {
	if !o.config.Export.Enabled {
		return
	}

	request := dto.ExportRequestDTO{
		Format:      o.config.Export.Format,
		Destination: o.config.Export.Destination,
		Path:        fmt.Sprintf("export_%s.%s", time.Now().Format("20060102_150405"), o.config.Export.Format),
		TimeRange: dto.TimeRangeDTO{
			Start: time.Now().Add(-o.config.Export.Interval),
			End:   time.Now(),
		},
		Filters: dto.ExportFiltersDTO{
			IncludeSystem:    true,
			IncludeProcesses: true,
		},
		Options: dto.ExportOptionsDTO{
			Compress:   o.config.Export.Compression,
			BatchSize:  o.config.Export.BatchSize,
			DateFormat: o.config.Display.DateFormat,
		},
	}

	result, err := o.exportUseCase.Execute(ctx, request)
	if err != nil {
		o.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Auto-export failed")
	} else if result.Success {
		o.logger.WithFields(map[string]interface{}{
			"path":  result.Path,
			"items": result.ItemsExported,
			"bytes": result.BytesWritten,
		}).Info("Auto-export completed")
	}
}

// GetCollectionStats returns collection statistics
func (o *MetricsOrchestrator) GetCollectionStats() CollectionStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.collectionStats
}