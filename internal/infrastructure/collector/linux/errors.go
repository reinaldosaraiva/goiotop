//go:build linux

package linux

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Common errors
var (
	ErrPSINotSupported   = errors.New("PSI (Pressure Stall Information) not supported on this kernel")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrProcessNotFound   = errors.New("process not found")
	ErrDeviceNotFound    = errors.New("device not found")
	ErrParsingFailed     = errors.New("parsing failed")
	ErrCollectorNotReady = errors.New("collector not ready")
	ErrInvalidMetric     = errors.New("invalid metric")
	ErrCacheExpired      = errors.New("cache expired")
)

// ErrorType represents the type of error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypePermission
	ErrorTypeNotFound
	ErrorTypeParsing
	ErrorTypeNotSupported
	ErrorTypeTimeout
	ErrorTypeIO
	ErrorTypeValidation
)

// CollectorError represents a collector-specific error
type CollectorError struct {
	Type    ErrorType
	Op      string
	Path    string
	Err     error
	Message string
}

// Error implements the error interface
func (e *CollectorError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s: %s: %v", e.Op, e.Path, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
}

// Unwrap returns the underlying error
func (e *CollectorError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *CollectorError) Is(target error) bool {
	if target == nil {
		return false
	}

	switch target {
	case ErrPermissionDenied:
		return e.Type == ErrorTypePermission
	case ErrProcessNotFound, ErrDeviceNotFound:
		return e.Type == ErrorTypeNotFound
	case ErrParsingFailed:
		return e.Type == ErrorTypeParsing
	case ErrPSINotSupported:
		return e.Type == ErrorTypeNotSupported
	}

	return errors.Is(e.Err, target)
}

// newCollectorError creates a new CollectorError
func newCollectorError(errType ErrorType, op, path string, err error) *CollectorError {
	return &CollectorError{
		Type:    errType,
		Op:      op,
		Path:    path,
		Err:     err,
		Message: getErrorMessage(errType),
	}
}

// getErrorMessage returns a message for the error type
func getErrorMessage(errType ErrorType) string {
	switch errType {
	case ErrorTypePermission:
		return "insufficient permissions"
	case ErrorTypeNotFound:
		return "resource not found"
	case ErrorTypeParsing:
		return "failed to parse data"
	case ErrorTypeNotSupported:
		return "feature not supported"
	case ErrorTypeTimeout:
		return "operation timed out"
	case ErrorTypeIO:
		return "I/O error"
	case ErrorTypeValidation:
		return "validation failed"
	default:
		return "unknown error"
	}
}

// classifyError determines the error type from a system error
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// Check for permission errors
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) || os.IsPermission(err) {
		return ErrorTypePermission
	}

	// Check for not found errors
	if errors.Is(err, syscall.ENOENT) || os.IsNotExist(err) {
		return ErrorTypeNotFound
	}

	// Check for I/O errors
	if errors.Is(err, syscall.EIO) {
		return ErrorTypeIO
	}

	// Check for timeout errors
	if errors.Is(err, syscall.ETIMEDOUT) {
		return ErrorTypeTimeout
	}

	return ErrorTypeUnknown
}

// wrapError wraps an error with collector context
func wrapError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	errType := classifyError(err)
	return newCollectorError(errType, op, path, err)
}

// ErrorAggregator collects multiple errors
type ErrorAggregator struct {
	errors []error
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: make([]error, 0),
	}
}

// Add adds an error to the aggregator
func (a *ErrorAggregator) Add(err error) {
	if err != nil {
		a.errors = append(a.errors, err)
	}
}

// AddWithContext adds an error with context
func (a *ErrorAggregator) AddWithContext(op, path string, err error) {
	if err != nil {
		a.errors = append(a.errors, wrapError(op, path, err))
	}
}

// HasErrors returns true if there are any errors
func (a *ErrorAggregator) HasErrors() bool {
	return len(a.errors) > 0
}

// Count returns the number of errors
func (a *ErrorAggregator) Count() int {
	return len(a.errors)
}

// Errors returns all collected errors
func (a *ErrorAggregator) Errors() []error {
	return a.errors
}

// Error returns a combined error message
func (a *ErrorAggregator) Error() string {
	if len(a.errors) == 0 {
		return ""
	}

	if len(a.errors) == 1 {
		return a.errors[0].Error()
	}

	return fmt.Sprintf("multiple errors occurred (%d errors)", len(a.errors))
}

// FirstError returns the first error or nil
func (a *ErrorAggregator) FirstError() error {
	if len(a.errors) > 0 {
		return a.errors[0]
	}
	return nil
}

// FilterByType returns errors of a specific type
func (a *ErrorAggregator) FilterByType(errType ErrorType) []error {
	var filtered []error
	for _, err := range a.errors {
		if collErr, ok := err.(*CollectorError); ok && collErr.Type == errType {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// isRecoverable determines if an error is recoverable
func isRecoverable(err error) bool {
	if err == nil {
		return true
	}

	// Permission errors are not recoverable
	if errors.Is(err, ErrPermissionDenied) {
		return false
	}

	// Not supported errors are not recoverable
	if errors.Is(err, ErrPSINotSupported) {
		return false
	}

	// Most other errors are potentially recoverable
	return true
}

// shouldRetry determines if an operation should be retried
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Temporary errors should be retried
	if errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.EAGAIN) {
		return true
	}

	// Process not found might be temporary (process just exited)
	if errors.Is(err, ErrProcessNotFound) {
		return false // Don't retry for missing processes
	}

	return false
}

// formatErrorForLogging formats an error for logging
func formatErrorForLogging(err error) string {
	if err == nil {
		return ""
	}

	if collErr, ok := err.(*CollectorError); ok {
		return fmt.Sprintf("[%s] %s", getErrorTypeName(collErr.Type), collErr.Error())
	}

	return err.Error()
}

// getErrorTypeName returns a string name for the error type
func getErrorTypeName(errType ErrorType) string {
	switch errType {
	case ErrorTypePermission:
		return "PERMISSION"
	case ErrorTypeNotFound:
		return "NOT_FOUND"
	case ErrorTypeParsing:
		return "PARSING"
	case ErrorTypeNotSupported:
		return "NOT_SUPPORTED"
	case ErrorTypeTimeout:
		return "TIMEOUT"
	case ErrorTypeIO:
		return "IO"
	case ErrorTypeValidation:
		return "VALIDATION"
	default:
		return "UNKNOWN"
	}
}