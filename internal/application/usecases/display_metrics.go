package usecases

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/config"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
)

// DisplayMetrics use case handles metric presentation and formatting
type DisplayMetrics struct {
	repository repositories.MetricsRepository
	logger     *config.Logger
	config     *config.DisplayConfig
}

// NewDisplayMetrics creates a new DisplayMetrics use case
func NewDisplayMetrics(
	repository repositories.MetricsRepository,
	logger *config.Logger,
	cfg *config.DisplayConfig,
) *DisplayMetrics {
	return &DisplayMetrics{
		repository: repository,
		logger:     logger,
		config:     cfg,
	}
}

// Execute executes the display operation based on the options
func (uc *DisplayMetrics) Execute(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	// Validate options
	if err := options.Validate(); err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid display options")
		return nil, fmt.Errorf("invalid display options: %w", err)
	}

	startTime := time.Now()

	uc.logger.WithFields(map[string]interface{}{
		"mode":      options.Mode,
		"max_rows":  options.MaxRows,
		"sort_by":   options.SortBy,
		"sort_order": options.SortOrder,
	}).Debug("Executing display metrics")

	var result *dto.DisplayResultDTO
	var err error

	switch options.Mode {
	case "realtime":
		result, err = uc.displayRealtime(ctx, options)
	case "historical":
		result, err = uc.displayHistorical(ctx, options)
	case "summary":
		result, err = uc.displaySummary(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported display mode: %s", options.Mode)
	}

	if err != nil {
		return nil, err
	}

	// Log display operation
	config.LogDisplayOperation(uc.logger, options.Mode, len(result.Processes), time.Since(startTime))

	return result, nil
}

// displayRealtime displays real-time metrics
func (uc *DisplayMetrics) displayRealtime(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	result := &dto.DisplayResultDTO{
		Mode:      "realtime",
		Timestamp: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Get latest system metrics
	if options.ShowSystemStats {
		systemFilter := repositories.MetricsFilter{
			Limit: 1,
		}

		systemMetrics, err := uc.repository.QuerySystemMetrics(ctx, systemFilter)
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to query system metrics")
			return nil, fmt.Errorf("failed to query system metrics: %w", err)
		}

		if len(systemMetrics) > 0 {
			result.SystemSummary = uc.createSystemSummary(systemMetrics[0])
		}
	}

	// Get process metrics
	if options.ShowProcesses {
		processes, err := uc.getTopProcesses(ctx, options)
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to get top processes")
			return nil, fmt.Errorf("failed to get top processes: %w", err)
		}

		result.Processes = processes

		// Apply pagination
		if options.Pagination.PageSize > 0 {
			result.Processes, result.Pagination = uc.paginateProcesses(processes, options.Pagination)
		}
	}

	// Generate graphs if requested
	if options.ShowGraphs {
		graphs, err := uc.generateGraphs(ctx, time.Minute) // Last minute
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Failed to generate graphs")
			// Continue without graphs
		} else {
			result.Graphs = graphs
		}
	}

	return result, nil
}

// displayHistorical displays historical metrics
func (uc *DisplayMetrics) displayHistorical(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	result := &dto.DisplayResultDTO{
		Mode:      "historical",
		Timestamp: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Determine time range
	var startTime, endTime time.Time
	if !options.Filters.TimeRange.Start.IsZero() && !options.Filters.TimeRange.End.IsZero() {
		startTime = options.Filters.TimeRange.Start
		endTime = options.Filters.TimeRange.End
	} else {
		// Default to last hour
		endTime = time.Now()
		startTime = endTime.Add(-time.Hour)
	}

	// Query historical system metrics
	if options.ShowSystemStats {
		timeRange, _ := repositories.NewTimeRange(startTime, endTime)
		systemFilter := repositories.MetricsFilter{
			TimeRange: timeRange,
			Limit:     100, // Limit for performance
		}

		systemMetrics, err := uc.repository.QuerySystemMetrics(ctx, systemFilter)
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to query historical system metrics")
			return nil, fmt.Errorf("failed to query historical system metrics: %w", err)
		}

		if len(systemMetrics) > 0 {
			// Use the latest for summary
			result.SystemSummary = uc.createSystemSummary(systemMetrics[0])

			// Generate historical graphs
			if options.ShowGraphs {
				graphs := uc.generateHistoricalGraphs(systemMetrics)
				result.Graphs = append(result.Graphs, graphs...)
			}
		}
	}

	// Query historical process metrics
	if options.ShowProcesses {
		timeRange, _ := repositories.NewTimeRange(startTime, endTime)
		processFilter := repositories.MetricsFilter{
			TimeRange: timeRange,
		}

		// Note: MinCPU/MinMemory/Username filtering will need to be done after retrieval
		// as MetricsFilter doesn't support these fields

		processes, err := uc.repository.QueryProcessMetrics(ctx, processFilter)
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to query historical process metrics")
			return nil, fmt.Errorf("failed to query historical process metrics: %w", err)
		}

		// Apply post-query filters (MinCPU, MinMemory, UserFilter)
		filteredProcesses := make([]*entities.ProcessMetrics, 0)
		for _, proc := range processes {
			if options.Filters.MinCPU > 0 && proc.CPUPercent < options.Filters.MinCPU {
				continue
			}
			if options.Filters.MinMemory > 0 && proc.MemoryPercent < options.Filters.MinMemory {
				continue
			}
			if options.Filters.UserFilter != "" && proc.Username != options.Filters.UserFilter {
				continue
			}
			filteredProcesses = append(filteredProcesses, proc)
		}

		// Convert and sort processes
		result.Processes = uc.convertAndSortProcesses(filteredProcesses, options.SortBy, options.SortOrder)

		// Apply pagination
		if options.Pagination.PageSize > 0 {
			result.Processes, result.Pagination = uc.paginateProcesses(result.Processes, options.Pagination)
		}
	}

	return result, nil
}

// displaySummary displays a summary of metrics
func (uc *DisplayMetrics) displaySummary(ctx context.Context, options dto.DisplayOptionsDTO) (*dto.DisplayResultDTO, error) {
	result := &dto.DisplayResultDTO{
		Mode:      "summary",
		Timestamp: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Get aggregated system metrics
	endTime := time.Now()
	startTime := endTime.Add(-uc.config.GraphHistory)

	timeRange, _ := repositories.NewTimeRange(startTime, endTime)
	aggregationRequest := repositories.AggregationRequest{
		Type:     repositories.AggregationAverage,
		Interval: repositories.IntervalFiveMin,
		Filter: repositories.MetricsFilter{
			TimeRange: timeRange,
		},
	}

	aggregatedSystem, err := uc.repository.AggregateSystemMetrics(ctx, aggregationRequest)
	if err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get aggregated system metrics")
		return nil, fmt.Errorf("failed to get aggregated system metrics: %w", err)
	}

	// Create summary from aggregated data
	if aggregatedSystem != nil {
		summary := &dto.SystemSummaryDTO{
			CPUUsage:     aggregatedSystem.AvgCPUUsage,
			MemoryUsage:  aggregatedSystem.AvgMemoryUsage,
			DiskUsage:    0, // Not available in aggregation
			NetworkRX:    0, // Not available in aggregation
			NetworkTX:    0, // Not available in aggregation
			ProcessCount: 0,
			ThreadCount:  0,
		}

		// Get latest system metrics for current values
		latestFilter := repositories.MetricsFilter{
			Limit: 1,
		}
		latestMetrics, err := uc.repository.QuerySystemMetrics(ctx, latestFilter)
		if err == nil && len(latestMetrics) > 0 {
			summary.LoadAverage = dto.LoadAvgDTO{
				Load1:  latestMetrics[0].LoadAvg.Load1,
				Load5:  latestMetrics[0].LoadAvg.Load5,
				Load15: latestMetrics[0].LoadAvg.Load15,
			}
			summary.Uptime = time.Duration(latestMetrics[0].Uptime) * time.Second
		}

		// Get top processes
		topProcesses, err := uc.repository.GetTopProcessesByCPU(ctx, 5, nil)
		if err == nil {
			summary.TopProcesses = make([]dto.ProcessSummaryDTO, 0, len(topProcesses))
			for _, proc := range topProcesses {
				summary.TopProcesses = append(summary.TopProcesses, dto.ProcessSummaryDTO{
					PID:           proc.PID,
					Name:          proc.Name,
					CPUPercent:    proc.CPUPercent,
					MemoryPercent: proc.MemoryPercent,
					State:         proc.State,
				})
			}
		}

		// Count processes and threads
		processCount, err := uc.repository.CountProcessMetrics(ctx, repositories.MetricsFilter{})
		if err == nil {
			summary.ProcessCount = processCount
		}

		result.SystemSummary = summary
	}

	// Generate summary graphs
	if options.ShowGraphs {
		graphs, err := uc.generateSummaryGraphs(ctx, startTime, endTime)
		if err != nil {
			uc.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Failed to generate summary graphs")
		} else {
			result.Graphs = graphs
		}
	}

	return result, nil
}

// getTopProcesses retrieves top processes based on display options
func (uc *DisplayMetrics) getTopProcesses(ctx context.Context, options dto.DisplayOptionsDTO) ([]dto.ProcessMetricsDTO, error) {
	limit := options.MaxRows
	if limit <= 0 {
		limit = uc.config.MaxRows
	}

	var processes []*entities.ProcessMetrics
	var err error

	switch options.SortBy {
	case "cpu":
		processes, err = uc.repository.GetTopProcessesByCPU(ctx, limit, nil)
	case "memory":
		processes, err = uc.repository.GetTopProcessesByMemory(ctx, limit, nil)
	default:
		// Default to CPU
		processes, err = uc.repository.GetTopProcessesByCPU(ctx, limit, nil)
	}

	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := make([]*entities.ProcessMetrics, 0)
	for _, proc := range processes {
		if uc.shouldIncludeProcessForDisplay(proc, options.Filters) {
			filtered = append(filtered, proc)
		}
	}

	// Convert to DTOs
	dtos := make([]dto.ProcessMetricsDTO, 0, len(filtered))
	for _, proc := range filtered {
		dtos = append(dtos, *dto.FromProcessMetrics(proc))
	}

	return dtos, nil
}

// shouldIncludeProcessForDisplay checks if a process should be displayed
func (uc *DisplayMetrics) shouldIncludeProcessForDisplay(proc *entities.ProcessMetrics, filters dto.DisplayFiltersDTO) bool {
	if filters.MinCPU > 0 && proc.CPUPercent < filters.MinCPU {
		return false
	}

	if filters.MinMemory > 0 && proc.MemoryPercent < filters.MinMemory {
		return false
	}

	if filters.UserFilter != "" && proc.Username != filters.UserFilter {
		return false
	}

	if filters.ProcessFilter != "" && proc.Name != filters.ProcessFilter {
		return false
	}

	if len(filters.StateFilter) > 0 {
		found := false
		for _, state := range filters.StateFilter {
			if proc.State == state {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// convertAndSortProcesses converts and sorts process metrics
func (uc *DisplayMetrics) convertAndSortProcesses(processes []*entities.ProcessMetrics, sortBy, sortOrder string) []dto.ProcessMetricsDTO {
	// Convert to DTOs
	dtos := make([]dto.ProcessMetricsDTO, 0, len(processes))
	for _, proc := range processes {
		dtos = append(dtos, *dto.FromProcessMetrics(proc))
	}

	// Sort
	sort.Slice(dtos, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "cpu":
			less = dtos[i].CPUPercent < dtos[j].CPUPercent
		case "memory":
			less = dtos[i].MemoryPercent < dtos[j].MemoryPercent
		case "pid":
			less = dtos[i].PID < dtos[j].PID
		case "name":
			less = dtos[i].Name < dtos[j].Name
		default:
			less = dtos[i].CPUPercent < dtos[j].CPUPercent
		}

		if sortOrder == "asc" {
			return less
		}
		return !less
	})

	return dtos
}

// paginateProcesses applies pagination to process list
func (uc *DisplayMetrics) paginateProcesses(processes []dto.ProcessMetricsDTO, pagination dto.PaginationDTO) ([]dto.ProcessMetricsDTO, dto.PaginationDTO) {
	total := len(processes)
	pagination.Total = total

	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	start := (pagination.Page - 1) * pagination.PageSize
	if start >= total {
		return []dto.ProcessMetricsDTO{}, pagination
	}

	end := start + pagination.PageSize
	if end > total {
		end = total
	}

	return processes[start:end], pagination
}

// createSystemSummary creates a system summary from system metrics
func (uc *DisplayMetrics) createSystemSummary(metrics *entities.SystemMetrics) *dto.SystemSummaryDTO {
	summary := &dto.SystemSummaryDTO{
		CPUUsage:    metrics.CPU.UsagePercent,
		MemoryUsage: metrics.Memory.UsagePercent,
		LoadAverage: dto.LoadAvgDTO{
			Load1:  metrics.LoadAvg.Load1,
			Load5:  metrics.LoadAvg.Load5,
			Load15: metrics.LoadAvg.Load15,
		},
		Uptime: time.Duration(metrics.Uptime) * time.Second,
	}

	// Calculate average disk usage
	if len(metrics.Disk) > 0 {
		var total float64
		for _, disk := range metrics.Disk {
			total += disk.UsagePercent
		}
		summary.DiskUsage = total / float64(len(metrics.Disk))
	}

	// Calculate network rates (simplified)
	for _, net := range metrics.Network {
		summary.NetworkRX += float64(net.BytesReceived) / 1024 / 1024 / 8
		summary.NetworkTX += float64(net.BytesSent) / 1024 / 1024 / 8
	}

	return summary
}

// generateGraphs generates graph data for display
func (uc *DisplayMetrics) generateGraphs(ctx context.Context, duration time.Duration) ([]dto.GraphDataDTO, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	timeRange, _ := repositories.NewTimeRange(startTime, endTime)
	filter := repositories.MetricsFilter{
		TimeRange: timeRange,
		Limit:     60, // One point per second for a minute
	}

	metrics, err := uc.repository.QuerySystemMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	graphs := make([]dto.GraphDataDTO, 0)

	// CPU usage graph
	cpuGraph := dto.GraphDataDTO{
		Name:      "CPU Usage",
		Type:      "line",
		Unit:      "%",
		Timestamp: time.Now(),
		Labels:    make([]string, 0, len(metrics)),
		Values:    make([]float64, 0, len(metrics)),
	}

	// Memory usage graph
	memGraph := dto.GraphDataDTO{
		Name:      "Memory Usage",
		Type:      "line",
		Unit:      "%",
		Timestamp: time.Now(),
		Labels:    make([]string, 0, len(metrics)),
		Values:    make([]float64, 0, len(metrics)),
	}

	for _, m := range metrics {
		label := m.Timestamp.Format("15:04:05")
		cpuGraph.Labels = append(cpuGraph.Labels, label)
		cpuGraph.Values = append(cpuGraph.Values, m.CPU.UsagePercent)
		memGraph.Labels = append(memGraph.Labels, label)
		memGraph.Values = append(memGraph.Values, m.Memory.UsagePercent)
	}

	graphs = append(graphs, cpuGraph, memGraph)

	return graphs, nil
}

// generateHistoricalGraphs generates graphs from historical data
func (uc *DisplayMetrics) generateHistoricalGraphs(metrics []*entities.SystemMetrics) []dto.GraphDataDTO {
	graphs := make([]dto.GraphDataDTO, 0)

	// CPU usage over time
	cpuGraph := dto.GraphDataDTO{
		Name:      "CPU Usage History",
		Type:      "line",
		Unit:      "%",
		Timestamp: time.Now(),
		Labels:    make([]string, 0, len(metrics)),
		Values:    make([]float64, 0, len(metrics)),
	}

	// Memory usage over time
	memGraph := dto.GraphDataDTO{
		Name:      "Memory Usage History",
		Type:      "line",
		Unit:      "%",
		Timestamp: time.Now(),
		Labels:    make([]string, 0, len(metrics)),
		Values:    make([]float64, 0, len(metrics)),
	}

	for _, m := range metrics {
		label := m.Timestamp.Format("15:04")
		cpuGraph.Labels = append(cpuGraph.Labels, label)
		cpuGraph.Values = append(cpuGraph.Values, m.CPU.UsagePercent)
		memGraph.Labels = append(memGraph.Labels, label)
		memGraph.Values = append(memGraph.Values, m.Memory.UsagePercent)
	}

	graphs = append(graphs, cpuGraph, memGraph)

	return graphs
}

// generateSummaryGraphs generates summary graphs
func (uc *DisplayMetrics) generateSummaryGraphs(ctx context.Context, startTime, endTime time.Time) ([]dto.GraphDataDTO, error) {
	// Get aggregated metrics
	timeRange, _ := repositories.NewTimeRange(startTime, endTime)
	aggregationRequest := repositories.AggregationRequest{
		Type:     repositories.AggregationAverage,
		Interval: repositories.IntervalFiveMin,
		Filter: repositories.MetricsFilter{
			TimeRange: timeRange,
		},
	}

	aggregatedMetrics, err := uc.repository.AggregateSystemMetrics(ctx, aggregationRequest)
	if err != nil {
		return nil, err
	}

	graphs := make([]dto.GraphDataDTO, 0)

	// Resource usage distribution (pie chart)
	if len(aggregatedMetrics) > 0 {
		latest := aggregatedMetrics[0]

		resourceGraph := dto.GraphDataDTO{
			Name:      "Resource Usage Distribution",
			Type:      "pie",
			Unit:      "%",
			Timestamp: time.Now(),
			Labels:    []string{"CPU", "Memory", "Disk"},
			Values:    make([]float64, 3),
		}

		resourceGraph.Values[0] = latest.CPU.UsagePercent
		resourceGraph.Values[1] = latest.Memory.UsagePercent

		// Calculate average disk usage
		if len(latest.Disk) > 0 {
			var total float64
			for _, disk := range latest.Disk {
				total += disk.UsagePercent
			}
			resourceGraph.Values[2] = total / float64(len(latest.Disk))
		}

		graphs = append(graphs, resourceGraph)
	}

	// Top processes by CPU (bar chart)
	topProcesses, err := uc.repository.GetTopProcessesByCPU(ctx, 5, nil)
	if err == nil && len(topProcesses) > 0 {
		topCPUGraph := dto.GraphDataDTO{
			Name:      "Top Processes by CPU",
			Type:      "bar",
			Unit:      "%",
			Timestamp: time.Now(),
			Labels:    make([]string, 0, len(topProcesses)),
			Values:    make([]float64, 0, len(topProcesses)),
		}

		for _, proc := range topProcesses {
			topCPUGraph.Labels = append(topCPUGraph.Labels, proc.Name)
			topCPUGraph.Values = append(topCPUGraph.Values, proc.CPUPercent)
		}

		graphs = append(graphs, topCPUGraph)
	}

	return graphs, nil
}