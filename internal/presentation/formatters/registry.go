package formatters

import (
	"fmt"
	"io"
	"strings"
)

// Formatter is the common interface for all formatters
type Formatter interface {
	Format(data interface{}, writer io.Writer) error
}

// FormatterType represents the type of formatter
type FormatterType string

const (
	// JSONFormat represents JSON output format
	JSONFormat FormatterType = "json"
	// CSVFormat represents CSV output format
	CSVFormat FormatterType = "csv"
	// TextFormat represents plain text output format
	TextFormat FormatterType = "text"
	// PrometheusFormat represents Prometheus metrics format
	PrometheusFormat FormatterType = "prometheus"
)

// Registry manages formatter instances
type Registry struct {
	formatters map[FormatterType]Formatter
	defaults   map[FormatterType]interface{}
}

// NewRegistry creates a new formatter registry
func NewRegistry() *Registry {
	r := &Registry{
		formatters: make(map[FormatterType]Formatter),
		defaults:   make(map[FormatterType]interface{}),
	}

	// Register default formatters
	r.RegisterDefaults()

	return r
}

// RegisterDefaults registers the default formatters
func (r *Registry) RegisterDefaults() {
	// Register JSON formatter
	r.Register(JSONFormat, NewJSONFormatter(WithIndent(true)))

	// Register CSV formatter
	r.Register(CSVFormat, NewCSVFormatter(WithHeaders(true)))

	// Register Text formatter
	r.Register(TextFormat, NewTextFormatter(
		WithTableHeaders(true),
		WithColor(false), // Disabled by default
	))
}

// Register registers a formatter for a specific format type
func (r *Registry) Register(formatType FormatterType, formatter Formatter) {
	r.formatters[formatType] = formatter
}

// Get retrieves a formatter by type
func (r *Registry) Get(formatType FormatterType) (Formatter, error) {
	formatter, exists := r.formatters[formatType]
	if !exists {
		return nil, fmt.Errorf("formatter not found for type: %s", formatType)
	}
	return formatter, nil
}

// GetOrCreate gets an existing formatter or creates a new one with options
func (r *Registry) GetOrCreate(formatType FormatterType, options ...interface{}) (Formatter, error) {
	// Try to get existing formatter first
	if formatter, exists := r.formatters[formatType]; exists {
		return formatter, nil
	}

	// Create new formatter based on type
	var formatter Formatter
	switch formatType {
	case JSONFormat:
		jsonOpts := []JSONOption{}
		for _, opt := range options {
			if jsonOpt, ok := opt.(JSONOption); ok {
				jsonOpts = append(jsonOpts, jsonOpt)
			}
		}
		formatter = NewJSONFormatter(jsonOpts...)

	case CSVFormat:
		csvOpts := []CSVOption{}
		for _, opt := range options {
			if csvOpt, ok := opt.(CSVOption); ok {
				csvOpts = append(csvOpts, csvOpt)
			}
		}
		formatter = NewCSVFormatter(csvOpts...)

	case TextFormat:
		textOpts := []TextOption{}
		for _, opt := range options {
			if textOpt, ok := opt.(TextOption); ok {
				textOpts = append(textOpts, textOpt)
			}
		}
		formatter = NewTextFormatter(textOpts...)

	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatType)
	}

	// Register the new formatter
	r.Register(formatType, formatter)
	return formatter, nil
}

// ParseFormat parses a format string and returns the corresponding FormatterType
func ParseFormat(format string) (FormatterType, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))

	switch normalized {
	case "json":
		return JSONFormat, nil
	case "csv":
		return CSVFormat, nil
	case "text", "plain", "txt":
		return TextFormat, nil
	case "prometheus", "prom":
		return PrometheusFormat, nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

// IsValidFormat checks if a format string is valid
func IsValidFormat(format string) bool {
	_, err := ParseFormat(format)
	return err == nil
}

// GetSupportedFormats returns a list of supported formats
func GetSupportedFormats() []string {
	return []string{
		string(JSONFormat),
		string(CSVFormat),
		string(TextFormat),
		string(PrometheusFormat),
	}
}

// FormatterFactory creates formatters with specific configurations
type FormatterFactory struct {
	registry *Registry
}

// NewFormatterFactory creates a new formatter factory
func NewFormatterFactory() *FormatterFactory {
	return &FormatterFactory{
		registry: NewRegistry(),
	}
}

// CreateFormatter creates a formatter based on format type and configuration
func (f *FormatterFactory) CreateFormatter(formatType FormatterType, config map[string]interface{}) (Formatter, error) {
	switch formatType {
	case JSONFormat:
		return f.createJSONFormatter(config), nil
	case CSVFormat:
		return f.createCSVFormatter(config), nil
	case TextFormat:
		return f.createTextFormatter(config), nil
	default:
		return nil, fmt.Errorf("unsupported formatter type: %s", formatType)
	}
}

// createJSONFormatter creates a JSON formatter with configuration
func (f *FormatterFactory) createJSONFormatter(config map[string]interface{}) *JSONFormatter {
	opts := []JSONOption{}

	if indent, ok := config["indent"].(bool); ok {
		opts = append(opts, WithIndent(indent))
	}
	if streaming, ok := config["streaming"].(bool); ok {
		opts = append(opts, WithStreaming(streaming))
	}
	if compact, ok := config["compact"].(bool); ok {
		opts = append(opts, WithCompact(compact))
	}

	return NewJSONFormatter(opts...)
}

// createCSVFormatter creates a CSV formatter with configuration
func (f *FormatterFactory) createCSVFormatter(config map[string]interface{}) *CSVFormatter {
	opts := []CSVOption{}

	if delimiter, ok := config["delimiter"].(string); ok && len(delimiter) > 0 {
		opts = append(opts, WithDelimiter(rune(delimiter[0])))
	}
	if headers, ok := config["headers"].(bool); ok {
		opts = append(opts, WithHeaders(headers))
	}
	if escape, ok := config["escape"].(bool); ok {
		opts = append(opts, WithEscapeQuotes(escape))
	}

	return NewCSVFormatter(opts...)
}

// createTextFormatter creates a text formatter with configuration
func (f *FormatterFactory) createTextFormatter(config map[string]interface{}) *TextFormatter {
	opts := []TextOption{}

	if color, ok := config["color"].(bool); ok {
		opts = append(opts, WithColor(color))
	}
	if headers, ok := config["headers"].(bool); ok {
		opts = append(opts, WithTableHeaders(headers))
	}
	if width, ok := config["columnWidth"].(int); ok {
		opts = append(opts, WithColumnWidth(width))
	}
	if progress, ok := config["progress"].(bool); ok {
		opts = append(opts, WithProgressBars(progress))
	}
	if graphs, ok := config["graphs"].(bool); ok {
		opts = append(opts, WithGraphs(graphs))
	}

	return NewTextFormatter(opts...)
}