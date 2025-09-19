package formatters

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// JSONFormatter formats DTOs as JSON output
type JSONFormatter struct {
	indent     bool
	streaming  bool
	compact    bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(options ...JSONOption) *JSONFormatter {
	f := &JSONFormatter{
		indent: true,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

// JSONOption is a configuration option for JSON formatter
type JSONOption func(*JSONFormatter)

// WithIndent enables or disables indentation
func WithIndent(indent bool) JSONOption {
	return func(f *JSONFormatter) {
		f.indent = indent
	}
}

// WithStreaming enables streaming mode for large datasets
func WithStreaming(streaming bool) JSONOption {
	return func(f *JSONFormatter) {
		f.streaming = streaming
	}
}

// WithCompact enables compact output
func WithCompact(compact bool) JSONOption {
	return func(f *JSONFormatter) {
		f.compact = compact
	}
}

// Format implements the Formatter interface
func (f *JSONFormatter) Format(data interface{}, writer io.Writer) error {
	encoder := json.NewEncoder(writer)

	if f.indent && !f.compact {
		encoder.SetIndent("", "  ")
	}

	if f.streaming {
		return f.formatStreaming(data, encoder)
	}

	return encoder.Encode(data)
}

// FormatCollectionResult formats a collection result as JSON
func (f *JSONFormatter) FormatCollectionResult(result dto.CollectionResultDTO, writer io.Writer) error {
	return f.Format(result, writer)
}

// FormatDisplayResult formats a display result as JSON
func (f *JSONFormatter) FormatDisplayResult(result dto.DisplayResultDTO, writer io.Writer) error {
	return f.Format(result, writer)
}

// FormatExportResult formats an export result as JSON
func (f *JSONFormatter) FormatExportResult(result dto.ExportResultDTO, writer io.Writer) error {
	return f.Format(result, writer)
}

// FormatSystemInfo formats system info as JSON
func (f *JSONFormatter) FormatSystemInfo(info dto.SystemInfoDTO, writer io.Writer) error {
	return f.Format(info, writer)
}

// FormatHealthCheck formats health check as JSON
func (f *JSONFormatter) FormatHealthCheck(health dto.HealthCheckDTO, writer io.Writer) error {
	return f.Format(health, writer)
}

// FormatCapabilities formats capabilities as JSON
func (f *JSONFormatter) FormatCapabilities(caps dto.CollectorCapabilitiesDTO, writer io.Writer) error {
	return f.Format(caps, writer)
}

// FormatProcessMetrics formats process metrics as JSON
func (f *JSONFormatter) FormatProcessMetrics(metrics []dto.ProcessMetricDTO, writer io.Writer) error {
	if f.streaming {
		return f.formatStreamingArray(metrics, writer)
	}
	return f.Format(metrics, writer)
}

// FormatDiskMetrics formats disk metrics as JSON
func (f *JSONFormatter) FormatDiskMetrics(metrics []dto.DiskMetricDTO, writer io.Writer) error {
	if f.streaming {
		return f.formatStreamingArray(metrics, writer)
	}
	return f.Format(metrics, writer)
}

// FormatNetworkMetrics formats network metrics as JSON
func (f *JSONFormatter) FormatNetworkMetrics(metrics []dto.NetworkMetricDTO, writer io.Writer) error {
	if f.streaming {
		return f.formatStreamingArray(metrics, writer)
	}
	return f.Format(metrics, writer)
}

// formatStreaming handles streaming JSON output for large datasets
func (f *JSONFormatter) formatStreaming(data interface{}, encoder *json.Encoder) error {
	// For streaming, we output each item separately
	// This is useful for continuous monitoring or large datasets

	switch v := data.(type) {
	case []dto.ProcessMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return err
			}
		}
		return nil

	case []dto.DiskMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return err
			}
		}
		return nil

	case []dto.NetworkMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return err
			}
		}
		return nil

	default:
		// For non-array types, just encode normally
		return encoder.Encode(data)
	}
}

// formatStreamingArray formats arrays in streaming mode with newline-delimited JSON
func (f *JSONFormatter) formatStreamingArray(data interface{}, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	// No indentation for streaming
	encoder.SetIndent("", "")

	switch v := data.(type) {
	case []dto.ProcessMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return fmt.Errorf("failed to encode process metric: %w", err)
			}
		}
	case []dto.DiskMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return fmt.Errorf("failed to encode disk metric: %w", err)
			}
		}
	case []dto.NetworkMetricDTO:
		for _, item := range v {
			if err := encoder.Encode(item); err != nil {
				return fmt.Errorf("failed to encode network metric: %w", err)
			}
		}
	default:
		return fmt.Errorf("unsupported type for streaming: %T", data)
	}

	return nil
}

// ValidateJSON validates if the data can be marshaled to JSON
func (f *JSONFormatter) ValidateJSON(data interface{}) error {
	_, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}
	return nil
}