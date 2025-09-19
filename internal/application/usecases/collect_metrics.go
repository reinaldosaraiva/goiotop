package usecases

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/config"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// CollectMetrics use case orchestrates metric collection
type CollectMetrics struct {
	collector  services.MetricsCollector
	repository repositories.MetricsRepository
	logger     *config.Logger
	config     *config.CollectionConfig
}

// NewCollectMetrics creates a new CollectMetrics use case
func NewCollectMetrics(
	collector services.MetricsCollector,
	repository repositories.MetricsRepository,
	logger *config.Logger,
	cfg *config.CollectionConfig,
) *CollectMetrics {
	return &CollectMetrics{
		collector:  collector,
		repository: repository,
		logger:     logger,
		config:     cfg,
	}
}

// Execute executes the collection based on the request
func (uc *CollectMetrics) Execute(ctx context.Context, request dto.CollectionRequestDTO) (*dto.CollectionResultDTO, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid collection request")
		return nil, fmt.Errorf("invalid collection request: %w", err)
	}

	uc.logger.WithFields(map[string]interface{}{
		"mode":     request.Mode,
		"interval": request.Interval,
		"duration": request.Duration,
	}).Info("Starting metrics collection")

	switch request.Mode {
	case "single":
		return uc.executeSingleCollection(ctx, request)
	case "continuous":
		return uc.executeContinuousCollection(ctx, request)
	case "batch":
		return uc.executeBatchCollection(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported collection mode: %s", request.Mode)
	}
}

// executeSingleCollection performs a single collection
func (uc *CollectMetrics) executeSingleCollection(ctx context.Context, request dto.CollectionRequestDTO) (*dto.CollectionResultDTO, error) {
	startTime := time.Now()
	result := &dto.CollectionResultDTO{
		Timestamp: startTime,
		Metadata:  request.Metadata,
		Statistics: dto.CollectionStatsDTO{
			CollectorVersion: "1.0.0",
		},
	}

	// Convert options
	opts := request.Options.ToCollectionOptions()

	// Collect system metrics
	systemMetrics, err := uc.collectSystemMetrics(ctx, opts)
	if err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to collect system metrics")
		result.Errors = append(result.Errors, dto.NewError("SYSTEM_COLLECTION_ERROR", err.Error()))
	} else {
		result.SystemMetrics = dto.FromSystemMetrics(systemMetrics)
		result.Statistics.ItemsCollected++

		// Save to repository
		if err := uc.repository.SaveSystemMetrics(ctx, systemMetrics); err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to save system metrics")
		}
	}

	// Collect process metrics
	processes, err := uc.collectProcessMetrics(ctx, opts, request.Filters)
	if err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to collect process metrics")
		result.Errors = append(result.Errors, dto.NewError("PROCESS_COLLECTION_ERROR", err.Error()))
	} else {
		result.Processes = make([]dto.ProcessMetricsDTO, 0, len(processes))
		for _, proc := range processes {
			result.Processes = append(result.Processes, *dto.FromProcessMetrics(proc))
		}
		result.ProcessCount = len(processes)
		result.Statistics.ItemsCollected += len(processes)

		// Save to repository - SaveProcessMetrics takes single item, not slice
		for _, proc := range processes {
			if err := uc.repository.SaveProcessMetrics(ctx, proc); err != nil {
				uc.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
			}).Error("Failed to save process metrics")
			}
		}
	}

	// Update statistics
	result.Duration = time.Since(startTime)
	result.Statistics.CollectionTime = result.Duration
	result.Statistics.ProcessingTime = time.Millisecond * 10 // Placeholder
	result.Success = len(result.Errors) == 0
	result.Statistics.ItemsFailed = len(result.Errors)

	// Log collection stats
	config.LogCollectionStats(uc.logger, result.Duration, result.Statistics.ItemsCollected, len(result.Errors))

	return result, nil
}

// executeContinuousCollection performs continuous collection
func (uc *CollectMetrics) executeContinuousCollection(ctx context.Context, request dto.CollectionRequestDTO) (*dto.CollectionResultDTO, error) {
	if request.Interval <= 0 {
		request.Interval = uc.config.Interval
	}

	if request.Duration <= 0 {
		return nil, fmt.Errorf("duration is required for continuous collection")
	}

	startTime := time.Now()
	endTime := startTime.Add(request.Duration)
	ticker := time.NewTicker(request.Interval)
	defer ticker.Stop()

	var (
		mu               sync.Mutex
		totalCollected   int
		totalErrors      int
		lastSystemMetrics *dto.SystemMetricsDTO
		allProcesses     []dto.ProcessMetricsDTO
	)

	// Collection callback
	callback := func(system *entities.SystemMetrics, processes []*entities.ProcessMetrics, err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			totalErrors++
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Collection callback error")
			return
		}

		if system != nil {
			lastSystemMetrics = dto.FromSystemMetrics(system)
			totalCollected++

			// Save to repository
			if saveErr := uc.repository.SaveSystemMetrics(ctx, system); saveErr != nil {
				uc.logger.WithFields(map[string]interface{}{
					"error": saveErr.Error(),
				}).Error("Failed to save system metrics in callback")
			}
		}

		if len(processes) > 0 {
			for _, proc := range processes {
				allProcesses = append(allProcesses, *dto.FromProcessMetrics(proc))
			}
			totalCollected += len(processes)

			// Save to repository - SaveProcessMetrics takes single item, not slice
			for _, proc := range processes {
				if saveErr := uc.repository.SaveProcessMetrics(ctx, proc); saveErr != nil {
					uc.logger.WithFields(map[string]interface{}{
						"error": saveErr.Error(),
					}).Error("Failed to save process metrics in callback")
				}
			}
		}
	}

	// Start continuous collection with proper synchronization
	opts := request.Options.ToCollectionOptions()
	stopChan := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopChan:
				return
			case <-ticker.C:
				if time.Now().After(endTime) {
					return
				}

				// Collect metrics
				system, err := uc.collector.CollectSystemMetrics(ctx)
				if err != nil {
					callback(nil, nil, err)
					continue
				}

				processes, err := uc.collector.CollectAllProcessMetrics(ctx)
				if err != nil {
					callback(system, nil, err)
					continue
				}

				callback(system, processes, nil)
			}
		}
	}()

	// Wait for duration
	select {
	case <-ctx.Done():
		close(stopChan)
		wg.Wait() // Wait for goroutine to exit
		return nil, ctx.Err()
	case <-time.After(request.Duration):
		close(stopChan)
		wg.Wait() // Wait for goroutine to exit
	}

	// Build result
	result := &dto.CollectionResultDTO{
		Success:       totalErrors == 0,
		Timestamp:     startTime,
		Duration:      time.Since(startTime),
		SystemMetrics: lastSystemMetrics,
		Processes:     allProcesses,
		ProcessCount:  len(allProcesses),
		Statistics: dto.CollectionStatsDTO{
			CollectionTime:   time.Since(startTime),
			ItemsCollected:   totalCollected,
			ItemsFailed:      totalErrors,
			CollectorVersion: "1.0.0",
		},
		Metadata: request.Metadata,
	}

	// Log collection stats
	config.LogCollectionStats(uc.logger, result.Duration, totalCollected, totalErrors)

	return result, nil
}

// executeBatchCollection performs batch collection
func (uc *CollectMetrics) executeBatchCollection(ctx context.Context, request dto.CollectionRequestDTO) (*dto.CollectionResultDTO, error) {
	startTime := time.Now()
	batchSize := request.Options.ProcessLimit
	if batchSize <= 0 {
		batchSize = uc.config.BatchSize
	}

	result := &dto.CollectionResultDTO{
		Timestamp: startTime,
		Metadata:  request.Metadata,
		Statistics: dto.CollectionStatsDTO{
			CollectorVersion: "1.0.0",
		},
	}

	opts := request.Options.ToCollectionOptions()

	// CollectAll takes options by value, not pointer
	allMetrics, err := uc.collector.CollectAll(ctx, *opts)
	if err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to collect all metrics")
		result.Errors = append(result.Errors, dto.NewError("BATCH_COLLECTION_ERROR", err.Error()))
		result.Success = false
		return result, err
	}

	// Convert system metrics
	if allMetrics.SystemMetrics != nil {
		result.SystemMetrics = dto.FromSystemMetrics(allMetrics.SystemMetrics)
		result.Statistics.ItemsCollected++

		// Save to repository
		if err := uc.repository.SaveSystemMetrics(ctx, allMetrics.SystemMetrics); err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to save system metrics")
		}
	}

	// Process in batches
	if len(allMetrics.ProcessMetrics) > 0 {
		for i := 0; i < len(allMetrics.ProcessMetrics); i += batchSize {
			end := i + batchSize
			if end > len(allMetrics.ProcessMetrics) {
				end = len(allMetrics.ProcessMetrics)
			}

			batch := allMetrics.ProcessMetrics[i:end]

			// Convert batch
			for _, proc := range batch {
				result.Processes = append(result.Processes, *dto.FromProcessMetrics(proc))
			}

			// Save batch to repository - SaveProcessMetrics takes single item, not slice
			for _, proc := range batch {
				if err := uc.repository.SaveProcessMetrics(ctx, proc); err != nil {
					uc.logger.WithFields(map[string]interface{}{
						"error":      err.Error(),
						"batch_size": len(batch),
					}).Error("Failed to save process metrics batch")
					result.Errors = append(result.Errors, dto.NewError("BATCH_SAVE_ERROR", err.Error()))
				}
			}
		}

		result.ProcessCount = len(allMetrics.ProcessMetrics)
		result.Statistics.ItemsCollected += len(allMetrics.ProcessMetrics)
	}

	// Update statistics
	result.Duration = time.Since(startTime)
	result.Statistics.CollectionTime = result.Duration
	result.Success = len(result.Errors) == 0
	result.Statistics.ItemsFailed = len(result.Errors)

	// Log collection stats
	config.LogCollectionStats(uc.logger, result.Duration, result.Statistics.ItemsCollected, len(result.Errors))

	return result, nil
}

// collectSystemMetrics collects system metrics
func (uc *CollectMetrics) collectSystemMetrics(ctx context.Context, opts *services.CollectionOptions) (*entities.SystemMetrics, error) {
	// CollectSystemMetrics doesn't take options parameter in domain interface
	return uc.collector.CollectSystemMetrics(ctx)
}

// collectProcessMetrics collects process metrics with filtering
func (uc *CollectMetrics) collectProcessMetrics(ctx context.Context, opts *services.CollectionOptions, filters dto.CollectionFiltersDTO) ([]*entities.ProcessMetrics, error) {
	// CollectAllProcessMetrics is the correct method for getting all processes
	processes, err := uc.collector.CollectAllProcessMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := make([]*entities.ProcessMetrics, 0)
	for _, proc := range processes {
		if uc.shouldIncludeProcess(proc, filters) {
			filtered = append(filtered, proc)
		}
	}

	return filtered, nil
}

// shouldIncludeProcess checks if a process should be included based on filters
func (uc *CollectMetrics) shouldIncludeProcess(proc *entities.ProcessMetrics, filters dto.CollectionFiltersDTO) bool {
	// Check process name filters
	if len(filters.ProcessNames) > 0 {
		found := false
		for _, name := range filters.ProcessNames {
			if proc.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check process patterns (regex or glob patterns)
	if len(filters.ProcessPatterns) > 0 {
		matched := false
		for _, pattern := range filters.ProcessPatterns {
			// Try to compile as regex
			if re, err := regexp.Compile(pattern); err == nil {
				if re.MatchString(proc.Name) || re.MatchString(proc.CmdLine) {
					matched = true
					break
				}
			} else {
				// Fall back to simple glob-like matching
				// Convert glob pattern to regex (simplified)
				globPattern := regexp.QuoteMeta(pattern)
				globPattern = regexp.MustCompile(`\\\*`).ReplaceAllString(globPattern, ".*")
				globPattern = regexp.MustCompile(`\\\?`).ReplaceAllString(globPattern, ".")
				if re, err := regexp.Compile("^" + globPattern + "$"); err == nil {
					if re.MatchString(proc.Name) {
						matched = true
						break
					}
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude names
	for _, exclude := range filters.ExcludeNames {
		if proc.Name == exclude {
			return false
		}
	}

	// Check user filter
	if filters.UserFilter != "" && proc.Username != filters.UserFilter {
		return false
	}

	return true
}

// StartContinuousCollection starts continuous metric collection
func (uc *CollectMetrics) StartContinuousCollection(ctx context.Context, interval time.Duration, callback func(dto.CollectionResultDTO)) error {
	if interval <= 0 {
		interval = uc.config.Interval
	}

	opts := services.CollectionOptions{
		Mode:                  services.CollectionModeRealTime,
		Interval:              interval,
		IncludeSystemMetrics:  true,
		IncludeProcessMetrics: true,
		IncludeCPUPressure:    uc.config.CollectorOptions.EnablePressure,
		IncludeDiskIO:         true,
		MaxProcesses:          uc.config.CollectorOptions.ProcessLimit,
		CollectKernelThreads:  uc.config.CollectorOptions.IncludeKernel,
		ProcessFilter: services.ProcessFilter{
			MinCPUThreshold: uc.config.CollectorOptions.MinCPUThreshold,
			MinMemThreshold: uc.config.CollectorOptions.MinMemThreshold,
		},
	}

	// Create callback wrapper matching CollectionCallback signature
	collectorCallback := func(result *services.CollectionResult) {
		dtoResult := dto.CollectionResultDTO{
			Timestamp: result.Timestamp,
		}

		if result.HasErrors() {
			for _, err := range result.Errors {
				dtoResult.Errors = append(dtoResult.Errors, dto.NewError("COLLECTION_ERROR", err.Error()))
			}
			dtoResult.Success = false
		} else {
			dtoResult.Success = true

			if result.SystemMetrics != nil {
				dtoResult.SystemMetrics = dto.FromSystemMetrics(result.SystemMetrics)
				dtoResult.Statistics.ItemsCollected++
			}

			if len(result.ProcessMetrics) > 0 {
				dtoResult.Processes = make([]dto.ProcessMetricsDTO, 0, len(result.ProcessMetrics))
				for _, proc := range result.ProcessMetrics {
					dtoResult.Processes = append(dtoResult.Processes, *dto.FromProcessMetrics(proc))
				}
				dtoResult.ProcessCount = len(result.ProcessMetrics)
				dtoResult.Statistics.ItemsCollected += len(result.ProcessMetrics)
			}
		}

		callback(dtoResult)
	}

	// Start continuous collection with correct signature
	return uc.collector.StartContinuousCollection(ctx, opts, collectorCallback)
}

// StopContinuousCollection stops continuous metric collection
func (uc *CollectMetrics) StopContinuousCollection(ctx context.Context) error {
	return uc.collector.StopContinuousCollection(ctx)
}

// ValidateCollectionCapabilities validates if the requested collection is possible
func (uc *CollectMetrics) ValidateCollectionCapabilities(request dto.CollectionRequestDTO) error {
	// GetCapabilities returns a struct, not a map
	capabilities := uc.collector.GetCapabilities()

	// Check if continuous collection is supported - use struct field
	if request.Mode == "continuous" {
		// Continuous is supported if collector supports system and process metrics
		if !capabilities.SupportsSystemMetrics || !capabilities.SupportsProcessMetrics {
			return errors.New("continuous collection is not supported")
		}
	}

	// Check if GPU collection is requested and supported
	if request.Options.EnableGPU {
		// GPU support would be in PlatformSpecific map if available
		if gpuSupported, ok := capabilities.PlatformSpecific["gpu"]; !ok || !gpuSupported {
			return errors.New("GPU metrics collection is not supported")
		}
	}

	// Check if pressure metrics are requested and supported
	if request.Options.EnablePressure && !capabilities.SupportsCPUPressure {
		return errors.New("pressure metrics collection is not supported")
	}

	return nil
}