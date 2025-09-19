package usecases

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/config"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
)

// ExportMetrics use case handles metric export to various formats
type ExportMetrics struct {
	repository repositories.MetricsRepository
	logger     *config.Logger
	config     *config.ExportConfig
}

// NewExportMetrics creates a new ExportMetrics use case
func NewExportMetrics(
	repository repositories.MetricsRepository,
	logger *config.Logger,
	cfg *config.ExportConfig,
) *ExportMetrics {
	return &ExportMetrics{
		repository: repository,
		logger:     logger,
		config:     cfg,
	}
}

// Execute executes the export operation based on the request
func (uc *ExportMetrics) Execute(ctx context.Context, request dto.ExportRequestDTO) (*dto.ExportResultDTO, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid export request")
		return nil, fmt.Errorf("invalid export request: %w", err)
	}

	startTime := time.Now()

	uc.logger.WithFields(map[string]interface{}{
		"format":      request.Format,
		"destination": request.Destination,
		"compress":    request.Options.Compress,
	}).Info("Starting metrics export")

	result := &dto.ExportResultDTO{
		Format:      request.Format,
		Destination: request.Destination,
		Path:        request.Path,
		Compressed:  request.Options.Compress,
		Metadata:    request.Metadata,
	}

	// Collect data to export
	data, err := uc.collectExportData(ctx, request)
	if err != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to collect export data")
		result.Success = false
		result.Errors = append(result.Errors, dto.NewError("DATA_COLLECTION_ERROR", err.Error()))
		return result, err
	}

	// Export based on format
	var exportErr error
	switch request.Format {
	case "json":
		exportErr = uc.exportJSON(ctx, data, request, result)
	case "csv":
		exportErr = uc.exportCSV(ctx, data, request, result)
	case "prometheus":
		exportErr = uc.exportPrometheus(ctx, data, request, result)
	default:
		exportErr = fmt.Errorf("unsupported export format: %s", request.Format)
	}

	if exportErr != nil {
		uc.logger.WithFields(map[string]interface{}{
			"error":  exportErr.Error(),
			"format": request.Format,
		}).Error("Export failed")
		result.Success = false
		result.Errors = append(result.Errors, dto.NewError("EXPORT_ERROR", exportErr.Error()))
		return result, exportErr
	}

	// Update result
	result.Success = true
	result.Duration = time.Since(startTime)

	// Log export operation
	config.LogExportOperation(uc.logger, request.Format, result.ItemsExported, result.BytesWritten, result.Duration, nil)

	return result, nil
}

// collectExportData collects data to be exported
func (uc *ExportMetrics) collectExportData(ctx context.Context, request dto.ExportRequestDTO) (*exportData, error) {
	data := &exportData{
		Timestamp: time.Now(),
		Metadata:  request.Metadata,
	}

	// Determine time range
	var startTime, endTime time.Time
	if !request.TimeRange.Start.IsZero() && !request.TimeRange.End.IsZero() {
		startTime = request.TimeRange.Start
		endTime = request.TimeRange.End
	} else {
		// Default to last hour
		endTime = time.Now()
		startTime = endTime.Add(-time.Hour)
	}

	// Collect system metrics if requested
	if request.Filters.IncludeSystem {
		systemFilter := repositories.SystemMetricsFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		systemMetrics, err := uc.repository.QuerySystemMetrics(ctx, systemFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to query system metrics: %w", err)
		}

		data.SystemMetrics = systemMetrics
	}

	// Collect process metrics if requested
	if request.Filters.IncludeProcesses {
		processFilter := repositories.ProcessMetricsFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		// Apply filters
		if request.Filters.MinCPU > 0 {
			processFilter.MinCPU = &request.Filters.MinCPU
		}
		if request.Filters.MinMemory > 0 {
			processFilter.MinMemory = &request.Filters.MinMemory
		}
		if len(request.Filters.ProcessNames) > 0 {
			// Apply process name filter
			// This would need additional implementation in the repository
		}

		processMetrics, err := uc.repository.QueryProcessMetrics(ctx, processFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to query process metrics: %w", err)
		}

		data.ProcessMetrics = processMetrics
	}

	return data, nil
}

// exportJSON exports data in JSON format
func (uc *ExportMetrics) exportJSON(ctx context.Context, data *exportData, request dto.ExportRequestDTO, result *dto.ExportResultDTO) error {
	// Prepare JSON data
	exportObj := map[string]interface{}{
		"export_info": map[string]interface{}{
			"timestamp":   data.Timestamp,
			"format":      "json",
			"compressed":  request.Options.Compress,
			"metadata":    data.Metadata,
			"time_range": map[string]interface{}{
				"start": request.TimeRange.Start,
				"end":   request.TimeRange.End,
			},
		},
	}

	// Add system metrics
	if len(data.SystemMetrics) > 0 {
		systemDTOs := make([]dto.SystemMetricsDTO, 0, len(data.SystemMetrics))
		for _, metrics := range data.SystemMetrics {
			systemDTOs = append(systemDTOs, *dto.FromSystemMetrics(metrics))
		}
		exportObj["system_metrics"] = systemDTOs
		result.ItemsExported += len(systemDTOs)
	}

	// Add process metrics
	if len(data.ProcessMetrics) > 0 {
		processDTOs := make([]dto.ProcessMetricsDTO, 0, len(data.ProcessMetrics))
		for _, metrics := range data.ProcessMetrics {
			processDTOs = append(processDTOs, *dto.FromProcessMetrics(metrics))
		}
		exportObj["process_metrics"] = processDTOs
		result.ItemsExported += len(processDTOs)
	}

	// Marshal to JSON
	var jsonData []byte
	var err error
	if request.Options.Pretty {
		jsonData, err = json.MarshalIndent(exportObj, "", "  ")
	} else {
		jsonData, err = json.Marshal(exportObj)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to destination
	return uc.writeToDestination(jsonData, request, result)
}

// exportCSV exports data in CSV format
func (uc *ExportMetrics) exportCSV(ctx context.Context, data *exportData, request dto.ExportRequestDTO, result *dto.ExportResultDTO) error {
	var buf bytes.Buffer

	// Use custom delimiter if specified
	delimiter := ','
	if request.Options.Delimiter != "" {
		delimiter = rune(request.Options.Delimiter[0])
	}

	writer := csv.NewWriter(&buf)
	writer.Comma = delimiter

	// Export system metrics
	if len(data.SystemMetrics) > 0 {
		// Write header
		if request.Options.IncludeHeader {
			header := []string{
				"timestamp", "hostname", "cpu_usage", "memory_usage", "disk_usage",
				"network_rx_bytes", "network_tx_bytes", "load_1", "load_5", "load_15",
				"uptime_seconds", "platform",
			}
			if err := writer.Write(header); err != nil {
				return fmt.Errorf("failed to write CSV header: %w", err)
			}
		}

		// Write data
		for _, metrics := range data.SystemMetrics {
			// Calculate average disk usage
			var diskUsage float64
			if len(metrics.Disk) > 0 {
				for _, disk := range metrics.Disk {
					diskUsage += disk.UsagePercent
				}
				diskUsage /= float64(len(metrics.Disk))
			}

			// Calculate total network traffic
			var networkRx, networkTx uint64
			for _, net := range metrics.Network {
				networkRx += net.BytesReceived
				networkTx += net.BytesSent
			}

			row := []string{
				uc.formatTimestamp(metrics.Timestamp, request.Options.DateFormat),
				metrics.Hostname,
				strconv.FormatFloat(metrics.CPU.UsagePercent, 'f', 2, 64),
				strconv.FormatFloat(metrics.Memory.UsagePercent, 'f', 2, 64),
				strconv.FormatFloat(diskUsage, 'f', 2, 64),
				strconv.FormatUint(networkRx, 10),
				strconv.FormatUint(networkTx, 10),
				strconv.FormatFloat(metrics.LoadAvg.Load1, 'f', 2, 64),
				strconv.FormatFloat(metrics.LoadAvg.Load5, 'f', 2, 64),
				strconv.FormatFloat(metrics.LoadAvg.Load15, 'f', 2, 64),
				strconv.FormatInt(metrics.Uptime, 10),
				metrics.Platform,
			}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
			result.ItemsExported++
		}
	}

	// Export process metrics to a separate section or file
	if len(data.ProcessMetrics) > 0 {
		// Add separator
		writer.Write([]string{})

		// Write process header
		if request.Options.IncludeHeader {
			header := []string{
				"timestamp", "pid", "name", "username", "state", "cpu_percent",
				"memory_percent", "memory_rss", "memory_vms", "io_read", "io_write",
				"num_threads", "command",
			}
			if err := writer.Write(header); err != nil {
				return fmt.Errorf("failed to write process CSV header: %w", err)
			}
		}

		// Write process data
		for _, proc := range data.ProcessMetrics {
			row := []string{
				uc.formatTimestamp(proc.CreateTime, request.Options.DateFormat),
				strconv.FormatInt(int64(proc.PID), 10),
				proc.Name,
				proc.Username,
				proc.State,
				strconv.FormatFloat(proc.CPUPercent, 'f', 2, 64),
				strconv.FormatFloat(proc.MemoryPercent, 'f', 2, 64),
				strconv.FormatUint(proc.MemoryRSS, 10),
				strconv.FormatUint(proc.MemoryVMS, 10),
				strconv.FormatUint(proc.IORead, 10),
				strconv.FormatUint(proc.IOWrite, 10),
				strconv.FormatInt(int64(proc.NumThreads), 10),
				proc.Command,
			}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write process CSV row: %w", err)
			}
			result.ItemsExported++
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	// Write to destination
	return uc.writeToDestination(buf.Bytes(), request, result)
}

// exportPrometheus exports data in Prometheus format
func (uc *ExportMetrics) exportPrometheus(ctx context.Context, data *exportData, request dto.ExportRequestDTO, result *dto.ExportResultDTO) error {
	var buf bytes.Buffer

	// Write system metrics in Prometheus format
	if len(data.SystemMetrics) > 0 {
		latest := data.SystemMetrics[len(data.SystemMetrics)-1] // Use latest metrics

		// CPU metrics
		fmt.Fprintf(&buf, "# HELP goiotop_cpu_usage_percent CPU usage percentage\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_cpu_usage_percent gauge\n")
		fmt.Fprintf(&buf, "goiotop_cpu_usage_percent{hostname=\"%s\"} %f\n", latest.Hostname, latest.CPU.UsagePercent)

		// Memory metrics
		fmt.Fprintf(&buf, "# HELP goiotop_memory_usage_percent Memory usage percentage\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_memory_usage_percent gauge\n")
		fmt.Fprintf(&buf, "goiotop_memory_usage_percent{hostname=\"%s\"} %f\n", latest.Hostname, latest.Memory.UsagePercent)

		fmt.Fprintf(&buf, "# HELP goiotop_memory_total_bytes Total memory in bytes\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_memory_total_bytes gauge\n")
		fmt.Fprintf(&buf, "goiotop_memory_total_bytes{hostname=\"%s\"} %d\n", latest.Hostname, latest.Memory.Total)

		fmt.Fprintf(&buf, "# HELP goiotop_memory_used_bytes Used memory in bytes\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_memory_used_bytes gauge\n")
		fmt.Fprintf(&buf, "goiotop_memory_used_bytes{hostname=\"%s\"} %d\n", latest.Hostname, latest.Memory.Used)

		// Disk metrics
		fmt.Fprintf(&buf, "# HELP goiotop_disk_usage_percent Disk usage percentage\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_disk_usage_percent gauge\n")
		for _, disk := range latest.Disk {
			fmt.Fprintf(&buf, "goiotop_disk_usage_percent{hostname=\"%s\",device=\"%s\",mount=\"%s\"} %f\n",
				latest.Hostname, disk.Device, disk.MountPoint, disk.UsagePercent)
		}

		// Network metrics
		fmt.Fprintf(&buf, "# HELP goiotop_network_bytes_received Network bytes received\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_network_bytes_received counter\n")
		for _, net := range latest.Network {
			fmt.Fprintf(&buf, "goiotop_network_bytes_received{hostname=\"%s\",interface=\"%s\"} %d\n",
				latest.Hostname, net.Interface, net.BytesReceived)
		}

		fmt.Fprintf(&buf, "# HELP goiotop_network_bytes_sent Network bytes sent\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_network_bytes_sent counter\n")
		for _, net := range latest.Network {
			fmt.Fprintf(&buf, "goiotop_network_bytes_sent{hostname=\"%s\",interface=\"%s\"} %d\n",
				latest.Hostname, net.Interface, net.BytesSent)
		}

		// Load average
		fmt.Fprintf(&buf, "# HELP goiotop_load_1 Load average 1 minute\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_load_1 gauge\n")
		fmt.Fprintf(&buf, "goiotop_load_1{hostname=\"%s\"} %f\n", latest.Hostname, latest.LoadAvg.Load1)

		fmt.Fprintf(&buf, "# HELP goiotop_load_5 Load average 5 minutes\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_load_5 gauge\n")
		fmt.Fprintf(&buf, "goiotop_load_5{hostname=\"%s\"} %f\n", latest.Hostname, latest.LoadAvg.Load5)

		fmt.Fprintf(&buf, "# HELP goiotop_load_15 Load average 15 minutes\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_load_15 gauge\n")
		fmt.Fprintf(&buf, "goiotop_load_15{hostname=\"%s\"} %f\n", latest.Hostname, latest.LoadAvg.Load15)

		// Uptime
		fmt.Fprintf(&buf, "# HELP goiotop_uptime_seconds System uptime in seconds\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_uptime_seconds counter\n")
		fmt.Fprintf(&buf, "goiotop_uptime_seconds{hostname=\"%s\"} %d\n", latest.Hostname, latest.Uptime)

		result.ItemsExported++
	}

	// Write process metrics in Prometheus format
	if len(data.ProcessMetrics) > 0 {
		// Process count by state
		stateCounts := make(map[string]int)
		for _, proc := range data.ProcessMetrics {
			stateCounts[proc.State]++
		}

		fmt.Fprintf(&buf, "# HELP goiotop_process_count Process count by state\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_process_count gauge\n")
		for state, count := range stateCounts {
			fmt.Fprintf(&buf, "goiotop_process_count{state=\"%s\"} %d\n", state, count)
			result.ItemsExported++
		}

		// Top processes by CPU
		fmt.Fprintf(&buf, "# HELP goiotop_process_cpu_percent Process CPU usage percentage\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_process_cpu_percent gauge\n")
		for _, proc := range data.ProcessMetrics {
			if proc.CPUPercent > 0 { // Only include processes with CPU usage
				fmt.Fprintf(&buf, "goiotop_process_cpu_percent{pid=\"%d\",name=\"%s\",user=\"%s\"} %f\n",
					proc.PID, proc.Name, proc.Username, proc.CPUPercent)
				result.ItemsExported++
			}
		}

		// Top processes by memory
		fmt.Fprintf(&buf, "# HELP goiotop_process_memory_percent Process memory usage percentage\n")
		fmt.Fprintf(&buf, "# TYPE goiotop_process_memory_percent gauge\n")
		for _, proc := range data.ProcessMetrics {
			if proc.MemoryPercent > 0 { // Only include processes with memory usage
				fmt.Fprintf(&buf, "goiotop_process_memory_percent{pid=\"%d\",name=\"%s\",user=\"%s\"} %f\n",
					proc.PID, proc.Name, proc.Username, proc.MemoryPercent)
			}
		}
	}

	// Write to destination
	return uc.writeToDestination(buf.Bytes(), request, result)
}

// writeToDestination writes data to the specified destination
func (uc *ExportMetrics) writeToDestination(data []byte, request dto.ExportRequestDTO, result *dto.ExportResultDTO) error {
	// Apply compression if requested
	if request.Options.Compress {
		compressed, err := uc.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		data = compressed
		result.Compressed = true
	}

	result.BytesWritten = int64(len(data))

	switch request.Destination {
	case "file":
		return uc.writeToFile(data, request.Path)
	case "http":
		return uc.writeToHTTP(data, request.Path)
	case "s3":
		return uc.writeToS3(data, request.Path)
	default:
		return fmt.Errorf("unsupported destination: %s", request.Destination)
	}
}

// writeToFile writes data to a file
func (uc *ExportMetrics) writeToFile(data []byte, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// writeToHTTP writes data via HTTP POST
func (uc *ExportMetrics) writeToHTTP(data []byte, url string) error {
	// Determine content type based on data
	contentType := "application/octet-stream"
	if json.Valid(data) {
		contentType = "application/json"
	} else if bytes.Contains(data, []byte("\n")) && bytes.Contains(data, []byte(",")) {
		contentType = "text/csv"
	} else if bytes.HasPrefix(data, []byte("# HELP")) || bytes.Contains(data, []byte("# TYPE")) {
		contentType = "text/plain; version=0.0.4" // Prometheus text format
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", strconv.Itoa(len(data)))
	req.Header.Set("User-Agent", "goiotop-exporter/1.0")

	// Add configurable headers if available in config
	if uc.config != nil && uc.config.HTTPHeaders != nil {
		for k, v := range uc.config.HTTPHeaders {
			req.Header.Set(k, v)
		}
	}

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("HTTP export failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// writeToS3 writes data to S3
func (uc *ExportMetrics) writeToS3(data []byte, path string) error {
	// For a full implementation, you would use AWS SDK v2 or similar
	// This is a stub implementation that can be extended

	// Parse path to extract bucket and key
	// Expected format: s3://bucket/path/to/object or bucket/path/to/object
	var bucket, key string
	if strings.HasPrefix(path, "s3://") {
		path = strings.TrimPrefix(path, "s3://")
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid S3 path: %s (expected bucket/key format)", path)
	}
	bucket = parts[0]
	key = parts[1]

	// Check if S3 client is available
	if uc.config != nil && uc.config.S3Endpoint != "" {
		// For testing or S3-compatible storage (MinIO, etc.)
		// You would configure the client here
		uc.logger.WithFields(map[string]interface{}{
			"bucket": bucket,
			"key":    key,
			"size":   len(data),
		}).Info("Would upload to S3")

		// Stub: simulate successful upload
		// In production, use AWS SDK:
		// client := s3.NewFromConfig(awsConfig)
		// _, err := client.PutObject(ctx, &s3.PutObjectInput{
		//     Bucket: &bucket,
		//     Key:    &key,
		//     Body:   bytes.NewReader(data),
		// })
		// return err
	}

	// Return error for now as AWS SDK is not included
	return fmt.Errorf("S3 export requires AWS SDK integration (bucket: %s, key: %s)", bucket, key)
}

// compressData compresses data using gzip
func (uc *ExportMetrics) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// formatTimestamp formats a timestamp according to the specified format
func (uc *ExportMetrics) formatTimestamp(t time.Time, format string) string {
	if format == "" {
		return t.Format(time.RFC3339)
	}

	// Support common format strings
	switch strings.ToLower(format) {
	case "iso8601":
		return t.Format(time.RFC3339)
	case "unix":
		return strconv.FormatInt(t.Unix(), 10)
	case "unixmilli":
		return strconv.FormatInt(t.UnixMilli(), 10)
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc822":
		return t.Format(time.RFC822)
	default:
		return t.Format(format)
	}
}

// StreamExport performs streaming export for large datasets
func (uc *ExportMetrics) StreamExport(ctx context.Context, request dto.ExportRequestDTO, writer io.Writer) error {
	// Validate request
	if err := request.Validate(); err != nil {
		return fmt.Errorf("invalid export request: %w", err)
	}

	// Determine time range
	var startTime, endTime time.Time
	if !request.TimeRange.Start.IsZero() && !request.TimeRange.End.IsZero() {
		startTime = request.TimeRange.Start
		endTime = request.TimeRange.End
	} else {
		endTime = time.Now()
		startTime = endTime.Add(-time.Hour)
	}

	batchSize := request.Options.BatchSize
	if batchSize <= 0 {
		batchSize = uc.config.BatchSize
	}

	// Setup writer based on format
	switch request.Format {
	case "json":
		return uc.streamJSON(ctx, writer, startTime, endTime, batchSize, request)
	case "csv":
		return uc.streamCSV(ctx, writer, startTime, endTime, batchSize, request)
	default:
		return fmt.Errorf("streaming not supported for format: %s", request.Format)
	}
}

// streamJSON streams data in JSON format
func (uc *ExportMetrics) streamJSON(ctx context.Context, writer io.Writer, startTime, endTime time.Time, batchSize int, request dto.ExportRequestDTO) error {
	encoder := json.NewEncoder(writer)

	// Write opening bracket
	if _, err := writer.Write([]byte("[\n")); err != nil {
		return err
	}

	first := true
	offset := 0

	for {
		// Query batch of metrics
		filter := repositories.SystemMetricsFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
			Limit:     batchSize,
			Offset:    offset,
		}

		metrics, err := uc.repository.QuerySystemMetrics(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to query metrics batch: %w", err)
		}

		if len(metrics) == 0 {
			break
		}

		// Write each metric
		for _, m := range metrics {
			if !first {
				if _, err := writer.Write([]byte(",\n")); err != nil {
					return err
				}
			}
			first = false

			dto := dto.FromSystemMetrics(m)
			if err := encoder.Encode(dto); err != nil {
				return fmt.Errorf("failed to encode metric: %w", err)
			}
		}

		offset += len(metrics)

		// Check if we've processed all metrics
		if len(metrics) < batchSize {
			break
		}

		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Write closing bracket
	if _, err := writer.Write([]byte("\n]")); err != nil {
		return err
	}

	return nil
}

// streamCSV streams data in CSV format
func (uc *ExportMetrics) streamCSV(ctx context.Context, writer io.Writer, startTime, endTime time.Time, batchSize int, request dto.ExportRequestDTO) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header if requested
	if request.Options.IncludeHeader {
		header := []string{
			"timestamp", "hostname", "cpu_usage", "memory_usage",
			"disk_usage", "load_1", "load_5", "load_15",
		}
		if err := csvWriter.Write(header); err != nil {
			return err
		}
	}

	offset := 0

	for {
		// Query batch of metrics
		filter := repositories.SystemMetricsFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
			Limit:     batchSize,
			Offset:    offset,
		}

		metrics, err := uc.repository.QuerySystemMetrics(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to query metrics batch: %w", err)
		}

		if len(metrics) == 0 {
			break
		}

		// Write each metric
		for _, m := range metrics {
			// Calculate average disk usage
			var diskUsage float64
			if len(m.Disk) > 0 {
				for _, disk := range m.Disk {
					diskUsage += disk.UsagePercent
				}
				diskUsage /= float64(len(m.Disk))
			}

			row := []string{
				uc.formatTimestamp(m.Timestamp, request.Options.DateFormat),
				m.Hostname,
				strconv.FormatFloat(m.CPU.UsagePercent, 'f', 2, 64),
				strconv.FormatFloat(m.Memory.UsagePercent, 'f', 2, 64),
				strconv.FormatFloat(diskUsage, 'f', 2, 64),
				strconv.FormatFloat(m.LoadAvg.Load1, 'f', 2, 64),
				strconv.FormatFloat(m.LoadAvg.Load5, 'f', 2, 64),
				strconv.FormatFloat(m.LoadAvg.Load15, 'f', 2, 64),
			}

			if err := csvWriter.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}

		// Flush periodically
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return err
		}

		offset += len(metrics)

		// Check if we've processed all metrics
		if len(metrics) < batchSize {
			break
		}

		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// exportData holds data to be exported
type exportData struct {
	SystemMetrics  []*entities.SystemMetrics
	ProcessMetrics []*entities.ProcessMetrics
	Timestamp      time.Time
	Metadata       map[string]string
}