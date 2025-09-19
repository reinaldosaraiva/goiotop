//go:build darwin

package darwin

import (
	"context"
	"errors"
	"fmt"
	"syscall"
)

var (
	// ErrCgroupsNotSupported indicates that cgroups are not available on macOS
	ErrCgroupsNotSupported = errors.New("cgroups are not supported on macOS")

	// ErrLimitedContainerSupport indicates limited container support on macOS
	ErrLimitedContainerSupport = errors.New("limited container support on macOS")

	// ErrAlreadyCollecting indicates that continuous collection is already running
	ErrAlreadyCollecting = errors.New("continuous collection is already running")

	// ErrNotCollecting indicates that continuous collection is not running
	ErrNotCollecting = errors.New("continuous collection is not running")

	// ErrSandboxRestriction indicates operation blocked by sandbox
	ErrSandboxRestriction = errors.New("operation blocked by sandbox restriction")

	// ErrInsufficientPrivileges indicates insufficient privileges for operation
	ErrInsufficientPrivileges = errors.New("insufficient privileges")

	// ErrCGONotAvailable indicates CGO is not available
	ErrCGONotAvailable = errors.New("CGO is not available")

	// ErrProcessNotFound indicates the process was not found
	ErrProcessNotFound = errors.New("process not found")

	// ErrDeviceNotFound indicates the device was not found
	ErrDeviceNotFound = errors.New("device not found")
)

// CollectorError represents a collector-specific error with context
type CollectorError struct {
	Op      string // Operation that failed
	Source  string // Source component (e.g., "cgo", "sysctl", "process")
	Err     error  // Underlying error
	Context map[string]interface{} // Additional context
}

// Error implements the error interface
func (e *CollectorError) Error() string {
	if e.Context != nil && len(e.Context) > 0 {
		return fmt.Sprintf("%s failed in %s: %v (context: %v)", e.Op, e.Source, e.Err, e.Context)
	}
	return fmt.Sprintf("%s failed in %s: %v", e.Op, e.Source, e.Err)
}

// Unwrap implements the errors.Unwrap interface
func (e *CollectorError) Unwrap() error {
	return e.Err
}

// IsPermissionError checks if the error is a permission-related error
func IsPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if errors.Is(err, ErrInsufficientPrivileges) || errors.Is(err, ErrSandboxRestriction) {
		return true
	}

	// Check for system permission errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.EPERM || errno == syscall.EACCES
	}

	// Check for collector errors with permission context
	var collErr *CollectorError
	if errors.As(err, &collErr) {
		if collErr.Context != nil {
			if perm, ok := collErr.Context["permission_error"].(bool); ok && perm {
				return true
			}
		}
	}

	return false
}

// IsNotFoundError checks if the error indicates a resource was not found
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if errors.Is(err, ErrProcessNotFound) || errors.Is(err, ErrDeviceNotFound) {
		return true
	}

	// Check for system not found errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.ESRCH || errno == syscall.ENOENT
	}

	return false
}

// IsSandboxError checks if the error is sandbox-related
func IsSandboxError(err error) bool {
	return errors.Is(err, ErrSandboxRestriction)
}

// WrapCGOError wraps a CGO error with context
func WrapCGOError(op string, err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	return &CollectorError{
		Op:      op,
		Source:  "cgo",
		Err:     err,
		Context: context,
	}
}

// WrapSysctlError wraps a sysctl error with context
func WrapSysctlError(op string, err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	return &CollectorError{
		Op:      op,
		Source:  "sysctl",
		Err:     err,
		Context: context,
	}
}

// WrapProcessError wraps a process error with context
func WrapProcessError(op string, err error, pid int32) error {
	if err == nil {
		return nil
	}

	return &CollectorError{
		Op:     op,
		Source: "process",
		Err:    err,
		Context: map[string]interface{}{
			"pid": pid,
		},
	}
}

// AggregateErrors combines multiple errors into a single error
type AggregateError struct {
	Errors []error
}

// Error implements the error interface
func (e *AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors (%d): first error: %v", len(e.Errors), e.Errors[0])
}

// Add adds an error to the aggregate
func (e *AggregateError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (e *AggregateError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ErrorOrNil returns the aggregate error or nil if no errors
func (e *AggregateError) ErrorOrNil() error {
	if e.HasErrors() {
		return e
	}
	return nil
}

// ClassifyError classifies an error for appropriate handling
func ClassifyError(err error) string {
	if err == nil {
		return "none"
	}

	if IsPermissionError(err) {
		return "permission"
	}

	if IsNotFoundError(err) {
		return "not_found"
	}

	if IsSandboxError(err) {
		return "sandbox"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	return "unknown"
}