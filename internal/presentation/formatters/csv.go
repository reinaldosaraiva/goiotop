package formatters

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// CSVFormatter formats DTOs as CSV output
type CSVFormatter struct {
	delimiter      rune
	includeHeaders bool
	escapeQuotes   bool
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(options ...CSVOption) *CSVFormatter {
	f := &CSVFormatter{
		delimiter:      ',',
		includeHeaders: true,
		escapeQuotes:   true,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

// CSVOption is a configuration option for CSV formatter
type CSVOption func(*CSVFormatter)

// WithDelimiter sets the CSV delimiter
func WithDelimiter(delimiter rune) CSVOption {
	return func(f *CSVFormatter) {
		f.delimiter = delimiter
	}
}

// WithHeaders enables or disables CSV headers
func WithHeaders(include bool) CSVOption {
	return func(f *CSVFormatter) {
		f.includeHeaders = include
	}
}

// WithEscapeQuotes enables or disables quote escaping
func WithEscapeQuotes(escape bool) CSVOption {
	return func(f *CSVFormatter) {
		f.escapeQuotes = escape
	}
}

// FormatProcessMetrics formats process metrics as CSV
func (f *CSVFormatter) FormatProcessMetrics(metrics []dto.ProcessMetricDTO, writer io.Writer) error {
	w := csv.NewWriter(writer)
	w.Comma = f.delimiter
	defer w.Flush()

	// Write headers
	if f.includeHeaders {
		headers := []string{
			"timestamp", "pid", "name", "command", "user",
			"cpu_percent", "memory_percent", "memory_rss", "memory_vms",
			"disk_read_bytes", "disk_write_bytes", "disk_read_rate", "disk_write_rate",
			"network_rx_bytes", "network_tx_bytes", "network_rx_rate", "network_tx_rate",
			"io_wait_time", "threads", "status",
		}
		if err := w.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Write data rows
	for _, m := range metrics {
		row := []string{
			m.Timestamp.Format(time.RFC3339),
			strconv.Itoa(int(m.PID)),
			m.Name,
			m.Command,
			m.User,
			formatFloat(m.CPUPercent, 2),
			formatFloat(m.MemoryPercent, 2),
			strconv.FormatUint(m.MemoryRSS, 10),
			strconv.FormatUint(m.MemoryVMS, 10),
			strconv.FormatUint(m.DiskReadBytes, 10),
			strconv.FormatUint(m.DiskWriteBytes, 10),
			formatFloat(m.DiskReadRate, 2),
			formatFloat(m.DiskWriteRate, 2),
			strconv.FormatUint(m.NetworkRxBytes, 10),
			strconv.FormatUint(m.NetworkTxBytes, 10),
			formatFloat(m.NetworkRxRate, 2),
			formatFloat(m.NetworkTxRate, 2),
			formatFloat(m.IOWaitTime, 2),
			strconv.Itoa(int(m.Threads)),
			m.Status,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// FormatDiskMetrics formats disk metrics as CSV
func (f *CSVFormatter) FormatDiskMetrics(metrics []dto.DiskMetricDTO, writer io.Writer) error {
	w := csv.NewWriter(writer)
	w.Comma = f.delimiter
	defer w.Flush()

	// Write headers
	if f.includeHeaders {
		headers := []string{
			"timestamp", "device", "mount_point", "filesystem",
			"reads_completed", "writes_completed", "read_bytes", "write_bytes",
			"read_time_ms", "write_time_ms", "io_time_ms", "queue_size",
			"utilization_percent", "read_rate", "write_rate",
		}
		if err := w.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Write data rows
	for _, m := range metrics {
		row := []string{
			m.Timestamp.Format(time.RFC3339),
			m.Device,
			m.MountPoint,
			m.Filesystem,
			strconv.FormatUint(m.ReadsCompleted, 10),
			strconv.FormatUint(m.WritesCompleted, 10),
			strconv.FormatUint(m.ReadBytes, 10),
			strconv.FormatUint(m.WriteBytes, 10),
			strconv.FormatUint(m.ReadTimeMs, 10),
			strconv.FormatUint(m.WriteTimeMs, 10),
			strconv.FormatUint(m.IOTimeMs, 10),
			strconv.FormatUint(m.QueueSize, 10),
			formatFloat(m.UtilizationPercent, 2),
			formatFloat(m.ReadRate, 2),
			formatFloat(m.WriteRate, 2),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// FormatNetworkMetrics formats network metrics as CSV
func (f *CSVFormatter) FormatNetworkMetrics(metrics []dto.NetworkMetricDTO, writer io.Writer) error {
	w := csv.NewWriter(writer)
	w.Comma = f.delimiter
	defer w.Flush()

	// Write headers
	if f.includeHeaders {
		headers := []string{
			"timestamp", "interface", "rx_bytes", "tx_bytes",
			"rx_packets", "tx_packets", "rx_errors", "tx_errors",
			"rx_drops", "tx_drops", "rx_rate", "tx_rate",
		}
		if err := w.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Write data rows
	for _, m := range metrics {
		row := []string{
			m.Timestamp.Format(time.RFC3339),
			m.Interface,
			strconv.FormatUint(m.RxBytes, 10),
			strconv.FormatUint(m.TxBytes, 10),
			strconv.FormatUint(m.RxPackets, 10),
			strconv.FormatUint(m.TxPackets, 10),
			strconv.FormatUint(m.RxErrors, 10),
			strconv.FormatUint(m.TxErrors, 10),
			strconv.FormatUint(m.RxDrops, 10),
			strconv.FormatUint(m.TxDrops, 10),
			formatFloat(m.RxRate, 2),
			formatFloat(m.TxRate, 2),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// FormatCollectionResult formats a collection result as CSV
func (f *CSVFormatter) FormatCollectionResult(result dto.CollectionResultDTO, writer io.Writer) error {
	w := csv.NewWriter(writer)
	w.Comma = f.delimiter
	defer w.Flush()

	// Write a combined CSV with all metrics
	if f.includeHeaders {
		headers := []string{
			"timestamp", "metric_type", "name", "value1", "value2", "value3", "value4",
		}
		if err := w.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Process metrics
	for _, m := range result.ProcessMetrics {
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			"process",
			m.Name,
			formatFloat(m.CPUPercent, 2),
			formatFloat(m.MemoryPercent, 2),
			formatFloat(m.DiskReadRate, 2),
			formatFloat(m.DiskWriteRate, 2),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Disk metrics
	for _, m := range result.DiskMetrics {
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			"disk",
			m.Device,
			formatFloat(m.ReadRate, 2),
			formatFloat(m.WriteRate, 2),
			formatFloat(m.UtilizationPercent, 2),
			"",
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Network metrics
	for _, m := range result.NetworkMetrics {
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			"network",
			m.Interface,
			formatFloat(m.RxRate, 2),
			formatFloat(m.TxRate, 2),
			"",
			"",
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// FormatDisplayResult formats a display result as CSV
func (f *CSVFormatter) FormatDisplayResult(result dto.DisplayResultDTO, writer io.Writer) error {
	w := csv.NewWriter(writer)
	w.Comma = f.delimiter
	defer w.Flush()

	// Write headers
	if f.includeHeaders {
		headers := []string{
			"timestamp", "pid", "name", "user",
			"cpu_percent", "memory_percent",
			"disk_read_rate", "disk_write_rate",
			"network_rx_rate", "network_tx_rate",
			"total_io_rate",
		}
		if err := w.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Write process data
	for _, p := range result.Processes {
		totalIO := p.DiskReadRate + p.DiskWriteRate + p.NetworkRxRate + p.NetworkTxRate
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			strconv.Itoa(int(p.PID)),
			p.Name,
			p.User,
			formatFloat(p.CPUPercent, 2),
			formatFloat(p.MemoryPercent, 2),
			formatFloat(p.DiskReadRate, 2),
			formatFloat(p.DiskWriteRate, 2),
			formatFloat(p.NetworkRxRate, 2),
			formatFloat(p.NetworkTxRate, 2),
			formatFloat(totalIO, 2),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// formatFloat formats a float value with specified precision
func formatFloat(value float64, precision int) string {
	return strconv.FormatFloat(value, 'f', precision, 64)
}