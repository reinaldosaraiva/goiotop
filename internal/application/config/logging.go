package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
	config LoggingConfig
	file   *os.File
}

// InitializeLogging initializes the logging system based on configuration
func InitializeLogging(cfg *LoggingConfig) (*Logger, error) {
	logger := &Logger{
		Logger: logrus.New(),
		config: *cfg,
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}
	logger.SetLevel(level)

	// Set formatter
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   time.RFC3339Nano,
			DisableTimestamp:  false,
			DisableHTMLEscape: true,
			DataKey:           "fields",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
			CallerPrettyfier: callerPrettyfier,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat:  time.RFC3339,
			FullTimestamp:    true,
			DisableColors:    false,
			DisableTimestamp: false,
			DisableSorting:   false,
			ForceColors:      true,
			PadLevelText:     true,
			QuoteEmptyFields: true,
			CallerPrettyfier: callerPrettyfier,
		})
	default:
		logger.SetFormatter(&CustomFormatter{
			TimestampFormat: time.RFC3339,
			EnableColors:    true,
			EnableCaller:    cfg.EnableCaller,
		})
	}

	// Set output
	output, err := getOutput(cfg)
	if err != nil {
		return nil, fmt.Errorf("error setting up log output: %w", err)
	}
	logger.SetOutput(output)
	if file, ok := output.(*os.File); ok && file != os.Stdout && file != os.Stderr {
		logger.file = file
	}

	// Set report caller
	logger.SetReportCaller(cfg.EnableCaller)

	// Add default fields using a hook or store them
	if cfg.Fields != nil && len(cfg.Fields) > 0 {
		// Add a hook that adds default fields to all log entries
		logger.AddHook(&DefaultFieldsHook{
			Fields: cfg.Fields,
		})
	}

	// Add hooks
	if cfg.EnableStacktrace {
		logger.AddHook(&StacktraceHook{
			MinLevel: logrus.ErrorLevel,
		})
	}

	logger.AddHook(&ContextHook{})
	logger.AddHook(&PerformanceHook{})

	return logger, nil
}

// WithFields returns a logrus.Entry with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithContext returns a new logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract request ID and other context values
	fields := make(logrus.Fields)

	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		fields["correlation_id"] = correlationID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}

	if len(fields) > 0 {
		return &Logger{
			Logger: l.Logger.WithFields(fields).Logger,
			config: l.config,
			file:   l.file,
		}
	}

	return l
}

// Close closes the log file if it was opened
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// CustomFormatter implements a custom log formatter
type CustomFormatter struct {
	TimestampFormat string
	EnableColors    bool
	EnableCaller    bool
}

// Format formats a log entry
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var sb strings.Builder

	// Timestamp
	timestamp := entry.Time.Format(f.TimestampFormat)
	sb.WriteString(timestamp)
	sb.WriteString(" ")

	// Level
	level := strings.ToUpper(entry.Level.String())
	if f.EnableColors {
		switch entry.Level {
		case logrus.TraceLevel:
			level = "\033[37m" + level + "\033[0m"
		case logrus.DebugLevel:
			level = "\033[36m" + level + "\033[0m"
		case logrus.InfoLevel:
			level = "\033[32m" + level + "\033[0m"
		case logrus.WarnLevel:
			level = "\033[33m" + level + "\033[0m"
		case logrus.ErrorLevel:
			level = "\033[31m" + level + "\033[0m"
		case logrus.FatalLevel, logrus.PanicLevel:
			level = "\033[35m" + level + "\033[0m"
		}
	}
	sb.WriteString(fmt.Sprintf("[%5s]", level))
	sb.WriteString(" ")

	// Caller
	if f.EnableCaller && entry.HasCaller() {
		caller := fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
		sb.WriteString(fmt.Sprintf("[%s]", caller))
		sb.WriteString(" ")
	}

	// Message
	sb.WriteString(entry.Message)

	// Fields
	if len(entry.Data) > 0 {
		sb.WriteString(" ")
		first := true
		for key, value := range entry.Data {
			if !first {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", key, value))
			first = false
		}
	}

	sb.WriteString("\n")
	return []byte(sb.String()), nil
}

// DefaultFieldsHook adds default fields to all log entries
type DefaultFieldsHook struct {
	Fields map[string]any
}

// Levels returns the log levels this hook applies to
func (h *DefaultFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire adds default fields to the log entry
func (h *DefaultFieldsHook) Fire(entry *logrus.Entry) error {
	for k, v := range h.Fields {
		// Don't overwrite existing fields
		if _, exists := entry.Data[k]; !exists {
			entry.Data[k] = v
		}
	}
	return nil
}

// ContextHook adds context information to log entries
type ContextHook struct{}

// Levels returns the log levels this hook applies to
func (h *ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire adds context information to the log entry
func (h *ContextHook) Fire(entry *logrus.Entry) error {
	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		entry.Data["hostname"] = hostname
	}

	// Add process info
	entry.Data["pid"] = os.Getpid()

	// Add goroutine info
	entry.Data["goroutines"] = runtime.NumGoroutine()

	return nil
}

// PerformanceHook tracks performance metrics for log operations
type PerformanceHook struct {
	slowThreshold time.Duration
}

// Levels returns the log levels this hook applies to
func (h *PerformanceHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Fire tracks performance metrics
func (h *PerformanceHook) Fire(entry *logrus.Entry) error {
	// Add memory stats for error and above
	if entry.Level <= logrus.ErrorLevel {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		entry.Data["memory_alloc_mb"] = m.Alloc / 1024 / 1024
		entry.Data["memory_sys_mb"] = m.Sys / 1024 / 1024
		entry.Data["num_gc"] = m.NumGC
	}

	return nil
}

// StacktraceHook adds stack traces to error logs
type StacktraceHook struct {
	MinLevel logrus.Level
}

// Levels returns the log levels this hook applies to
func (h *StacktraceHook) Levels() []logrus.Level {
	levels := []logrus.Level{}
	for _, level := range logrus.AllLevels {
		if level <= h.MinLevel {
			levels = append(levels, level)
		}
	}
	return levels
}

// Fire adds stack trace to the log entry
func (h *StacktraceHook) Fire(entry *logrus.Entry) error {
	// Get stack trace
	stack := make([]byte, 4096)
	n := runtime.Stack(stack, false)
	entry.Data["stacktrace"] = string(stack[:n])
	return nil
}

// getOutput returns the appropriate output writer based on configuration
func getOutput(cfg *LoggingConfig) (io.Writer, error) {
	switch cfg.Output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error creating log directory %s: %w", dir, err)
		}

		// Use lumberjack for log rotation
		if cfg.MaxSize > 0 || cfg.MaxAge > 0 || cfg.MaxBackups > 0 {
			return &lumberjack.Logger{
				Filename:   cfg.File,
				MaxSize:    cfg.MaxSize,    // megabytes
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,      // days
				Compress:   cfg.Compress,
				LocalTime:  true,
			}, nil
		}

		// Fallback to simple file without rotation
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("error opening log file %s: %w", cfg.File, err)
		}
		return file, nil
	default:
		return os.Stdout, nil
	}
}

// callerPrettyfier formats the caller information
func callerPrettyfier(f *runtime.Frame) (function string, file string) {
	function = filepath.Base(f.Function)
	file = fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
	return
}

// LogCollectionStats logs collection statistics
func LogCollectionStats(logger *Logger, duration time.Duration, collected int, errors int) {
	fields := logrus.Fields{
		"duration_ms": duration.Milliseconds(),
		"collected":   collected,
		"errors":      errors,
		"rate":        float64(collected) / duration.Seconds(),
	}

	if errors > 0 {
		logger.WithFields(fields).Warn("Collection completed with errors")
	} else {
		logger.WithFields(fields).Info("Collection completed successfully")
	}
}

// LogExportOperation logs export operation details
func LogExportOperation(logger *Logger, format string, count int, size int64, duration time.Duration, err error) {
	fields := logrus.Fields{
		"format":      format,
		"count":       count,
		"size_bytes":  size,
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		fields["error"] = err.Error()
		logger.WithFields(fields).Error("Export operation failed")
	} else {
		logger.WithFields(fields).Info("Export operation completed")
	}
}

// LogDisplayOperation logs display operation details
func LogDisplayOperation(logger *Logger, mode string, count int, duration time.Duration) {
	fields := logrus.Fields{
		"mode":        mode,
		"count":       count,
		"duration_ms": duration.Milliseconds(),
	}
	logger.WithFields(fields).Debug("Display operation completed")
}

// LogHealthCheck logs health check results
func LogHealthCheck(logger *Logger, component string, healthy bool, details map[string]interface{}) {
	fields := logrus.Fields{
		"component": component,
		"healthy":   healthy,
	}

	for k, v := range details {
		fields[k] = v
	}

	if healthy {
		logger.WithFields(fields).Debug("Health check passed")
	} else {
		logger.WithFields(fields).Warn("Health check failed")
	}
}

// NewNopLogger creates a logger that discards all output
func NewNopLogger() *Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &Logger{Logger: logger}
}