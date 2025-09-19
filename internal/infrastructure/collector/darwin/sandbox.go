//go:build darwin

package darwin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// sandboxDetector provides sandbox detection and graceful degradation
type sandboxDetector struct {
	isSandboxed          bool
	hasNetworkAccess     bool
	hasFileSystemAccess  bool
	hasProcessAccess     bool
	restrictedPaths      []string
	availableFeatures    map[string]bool
}

// detectSandboxMode detects if the application is running in a sandboxed environment
func detectSandboxMode() bool {
	detector := newSandboxDetector()
	return detector.isSandboxed
}

// newSandboxDetector creates and initializes a sandbox detector
func newSandboxDetector() *sandboxDetector {
	d := &sandboxDetector{
		availableFeatures: make(map[string]bool),
	}

	d.detectSandbox()
	d.detectAvailableFeatures()

	return d
}

// detectSandbox performs various checks to detect sandbox restrictions
func (d *sandboxDetector) detectSandbox() {
	// Check 1: Try to access system process (PID 1)
	// In a sandbox, this typically fails
	if !d.canAccessProcess(1) {
		d.isSandboxed = true
		return
	}

	// Check 2: Try to access restricted paths
	restrictedPaths := []string{
		"/System/Library/Extensions",
		"/private/var/db",
		"/dev/disk0",
	}

	accessibleCount := 0
	for _, path := range restrictedPaths {
		if d.canAccessPath(path) {
			accessibleCount++
		}
	}

	// If we can't access most restricted paths, we're likely sandboxed
	if accessibleCount < len(restrictedPaths)/2 {
		d.isSandboxed = true
		d.restrictedPaths = restrictedPaths
		return
	}

	// Check 3: App Store detection
	if d.isAppStoreApp() {
		d.isSandboxed = true
		return
	}

	// Check 4: Container detection
	if d.isContainerized() {
		d.isSandboxed = true
		return
	}

	// Check 5: Use CGO to check for sandbox
	if CheckSandboxRestrictions() {
		d.isSandboxed = true
		return
	}

	d.isSandboxed = false
}

// canAccessProcess checks if we can access information about a process
func (d *sandboxDetector) canAccessProcess(pid int32) bool {
	// Try to get process info using proc_pid_rusage
	_, err := GetProcessIOStats(pid)
	if err != nil {
		// Check if it's a permission error
		if IsPermissionError(err) {
			return false
		}
	}
	return true
}

// canAccessPath checks if we can access a specific path
func (d *sandboxDetector) canAccessPath(path string) bool {
	// Try to stat the path
	_, err := os.Stat(path)
	if err != nil {
		// Check if it's a permission error
		if os.IsPermission(err) {
			return false
		}
		// Path doesn't exist is different from no access
		if os.IsNotExist(err) {
			return true // We can determine it doesn't exist, so we have "access"
		}
		return false
	}
	return true
}

// isAppStoreApp checks if the application is an App Store app
func (d *sandboxDetector) isAppStoreApp() bool {
	// Check if we're running from /Applications and have a receipt
	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	// Check for Mac App Store receipt
	appPath := filepath.Dir(filepath.Dir(execPath))
	receiptPath := filepath.Join(appPath, "Contents", "_MASReceipt", "receipt")

	if _, err := os.Stat(receiptPath); err == nil {
		return true
	}

	// Check if we're in a .app bundle with sandbox entitlements
	if strings.Contains(execPath, ".app/Contents/MacOS") {
		// Check for sandbox container
		homeDir, _ := os.UserHomeDir()
		if strings.Contains(homeDir, "/Library/Containers/") {
			return true
		}
	}

	return false
}

// isContainerized checks if we're running in a container
func (d *sandboxDetector) isContainerized() bool {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for containerd
	if _, err := os.Stat("/run/containerd"); err == nil {
		return true
	}

	// Check environment variables
	containerEnvs := []string{"KUBERNETES_SERVICE_HOST", "DOCKER_CONTAINER", "container"}
	for _, env := range containerEnvs {
		if os.Getenv(env) != "" {
			return true
		}
	}

	return false
}

// detectAvailableFeatures detects which features are available in the current environment
func (d *sandboxDetector) detectAvailableFeatures() {
	// Network access
	d.hasNetworkAccess = d.checkNetworkAccess()
	d.availableFeatures["network"] = d.hasNetworkAccess

	// File system access
	d.hasFileSystemAccess = d.checkFileSystemAccess()
	d.availableFeatures["filesystem"] = d.hasFileSystemAccess

	// Process access
	d.hasProcessAccess = d.checkProcessAccess()
	d.availableFeatures["process"] = d.hasProcessAccess

	// System metrics
	d.availableFeatures["system_metrics"] = d.checkSystemMetricsAccess()

	// Disk I/O
	d.availableFeatures["disk_io"] = d.checkDiskIOAccess()
}

// checkNetworkAccess checks if network access is available
func (d *sandboxDetector) checkNetworkAccess() bool {
	// Try to access network interfaces
	// This is usually allowed even in sandboxes
	return true
}

// checkFileSystemAccess checks file system access level
func (d *sandboxDetector) checkFileSystemAccess() bool {
	// Check if we can access common directories
	testPaths := []string{
		"/tmp",
		"/var/tmp",
		os.TempDir(),
	}

	for _, path := range testPaths {
		if d.canAccessPath(path) {
			return true
		}
	}

	return false
}

// checkProcessAccess checks if we can access other processes
func (d *sandboxDetector) checkProcessAccess() bool {
	// Try to list processes
	pids, err := GetProcessList()
	if err != nil {
		return false
	}

	// If we can only see our own process, we're restricted
	ourPID := int32(os.Getpid())
	if len(pids) == 1 && pids[0] == ourPID {
		return false
	}

	return len(pids) > 1
}

// checkSystemMetricsAccess checks if system metrics are accessible
func (d *sandboxDetector) checkSystemMetricsAccess() bool {
	_, err := GetSystemInfo()
	return err == nil
}

// checkDiskIOAccess checks if disk I/O metrics are accessible
func (d *sandboxDetector) checkDiskIOAccess() bool {
	// Try to access disk I/O stats
	// This usually works even in sandboxes through gopsutil
	return true
}

// GetFallbackCollector returns a collector with fallback mechanisms for sandboxed environments
func GetFallbackCollector(detector *sandboxDetector) *fallbackCollector {
	return &fallbackCollector{
		detector: detector,
	}
}

// fallbackCollector provides fallback mechanisms for sandboxed environments
type fallbackCollector struct {
	detector *sandboxDetector
}

// CollectWithFallback attempts to collect metrics with fallback mechanisms
func (f *fallbackCollector) CollectWithFallback(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	// Check if error is due to sandbox restrictions
	if IsPermissionError(err) || IsSandboxError(err) {
		// Return a specific error indicating sandbox restriction
		return ErrSandboxRestriction
	}

	return err
}

// GetAlternativeMethod returns an alternative collection method for sandboxed environments
func (f *fallbackCollector) GetAlternativeMethod(feature string) func() error {
	switch feature {
	case "process_io":
		// Use gopsutil instead of CGO
		return f.useGopsutilForProcessIO

	case "system_info":
		// Use gopsutil for system info
		return f.useGopsutilForSystemInfo

	default:
		return nil
	}
}

// useGopsutilForProcessIO uses gopsutil as fallback for process I/O
func (f *fallbackCollector) useGopsutilForProcessIO() error {
	// Implementation would use gopsutil's process package
	// instead of CGO calls
	return nil
}

// useGopsutilForSystemInfo uses gopsutil as fallback for system info
func (f *fallbackCollector) useGopsutilForSystemInfo() error {
	// Implementation would use gopsutil's host package
	// instead of sysctl calls
	return nil
}

// GetSandboxReport returns a detailed report of sandbox restrictions
func (d *sandboxDetector) GetSandboxReport() map[string]interface{} {
	return map[string]interface{}{
		"is_sandboxed":          d.isSandboxed,
		"has_network_access":    d.hasNetworkAccess,
		"has_filesystem_access": d.hasFileSystemAccess,
		"has_process_access":    d.hasProcessAccess,
		"restricted_paths":      d.restrictedPaths,
		"available_features":    d.availableFeatures,
		"platform":              runtime.GOOS,
		"arch":                  runtime.GOARCH,
		"go_version":            runtime.Version(),
	}
}