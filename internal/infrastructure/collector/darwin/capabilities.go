//go:build darwin

package darwin

import (
	"os"
	"runtime"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// collectorCapabilities holds the detected capabilities for the macOS collector
type collectorCapabilities struct {
	hasCGO           bool
	hasSysctl        bool
	hasProcessIO     bool
	hasDocker        bool
	hasSandbox       bool
	hasFullAccess    bool
	maxConcurrent    int
}

// detectCapabilities detects available capabilities at runtime
func detectCapabilities() *collectorCapabilities {
	caps := &collectorCapabilities{
		maxConcurrent: runtime.NumCPU(),
	}

	// Check CGO availability by trying to call a CGO function
	caps.hasCGO = checkCGOAvailability()

	// Check sysctl access
	caps.hasSysctl = checkSysctlAccess()

	// Check process I/O access
	caps.hasProcessIO = checkProcessIOAccess()

	// Check Docker availability
	caps.hasDocker = checkDockerAvailability()

	// Check sandbox restrictions
	caps.hasSandbox = CheckSandboxRestrictions()

	// Determine if we have full access
	caps.hasFullAccess = caps.hasCGO && caps.hasSysctl && caps.hasProcessIO && !caps.hasSandbox

	return caps
}

// checkCGOAvailability checks if CGO is available and working
func checkCGOAvailability() bool {
	// Try to get system info using CGO
	_, err := GetSystemInfo()
	return err == nil
}

// checkSysctlAccess checks if we can access sysctl
func checkSysctlAccess() bool {
	// Try to get system info which uses sysctl internally
	info, err := GetSystemInfo()
	if err != nil {
		return false
	}

	// Check if we got valid data
	return info.CPUCount > 0 && info.MemoryTotal > 0
}

// checkProcessIOAccess checks if we can access process I/O statistics
func checkProcessIOAccess() bool {
	// Try to get I/O stats for our own process
	pid := int32(os.Getpid())
	_, err := GetProcessIOStats(pid)
	return err == nil
}

// checkDockerAvailability checks if Docker is available
func checkDockerAvailability() bool {
	// Simple check for Docker socket
	// This could be enhanced to actually try connecting to Docker
	_, err := os.Stat("/var/run/docker.sock")
	return err == nil
}

// ToServiceCapabilities converts internal capabilities to service capabilities
func (c *collectorCapabilities) ToServiceCapabilities() services.CollectorCapabilities {
	return services.CollectorCapabilities{
		SupportsProcessIO:        c.hasProcessIO,
		SupportsCgroups:          false, // macOS doesn't have cgroups
		SupportsContainers:       c.hasDocker,
		SupportsPressureMetrics:  false, // No native PSI on macOS
		SupportsNetworkIO:        true,  // Always available via gopsutil
		SupportsDiskIO:           true,  // Always available via gopsutil
		MaxConcurrentCollections: c.maxConcurrent,
	}
}

// GetCapabilityReport returns a detailed capability report
func (c *collectorCapabilities) GetCapabilityReport() map[string]interface{} {
	return map[string]interface{}{
		"platform":          "darwin",
		"cgo_available":     c.hasCGO,
		"sysctl_access":     c.hasSysctl,
		"process_io_access": c.hasProcessIO,
		"docker_available":  c.hasDocker,
		"sandboxed":         c.hasSandbox,
		"full_access":       c.hasFullAccess,
		"max_concurrent":    c.maxConcurrent,
		"features": map[string]bool{
			"system_metrics":   c.hasSysctl,
			"process_metrics":  c.hasProcessIO,
			"disk_io":          true,
			"network_io":       true,
			"cpu_pressure":     c.hasSysctl,
			"memory_pressure":  c.hasSysctl,
			"io_pressure":      false,
			"cgroups":          false,
			"containers":       c.hasDocker,
		},
	}
}

// RequiresElevation checks if elevation would provide more capabilities
func (c *collectorCapabilities) RequiresElevation() bool {
	// If we're sandboxed or don't have full process I/O access,
	// elevation might help
	return c.hasSandbox || !c.hasProcessIO
}

// GetLimitations returns a list of current limitations
func (c *collectorCapabilities) GetLimitations() []string {
	var limitations []string

	if !c.hasCGO {
		limitations = append(limitations, "CGO not available - limited system call access")
	}

	if !c.hasSysctl {
		limitations = append(limitations, "sysctl access limited - reduced system metrics")
	}

	if !c.hasProcessIO {
		limitations = append(limitations, "process I/O access limited - per-process I/O stats unavailable")
	}

	if c.hasSandbox {
		limitations = append(limitations, "running in sandbox - some features restricted")
	}

	if !c.hasDocker {
		limitations = append(limitations, "Docker not available - container metrics unavailable")
	}

	// macOS-specific limitations
	limitations = append(limitations, "cgroups not supported on macOS")
	limitations = append(limitations, "PSI metrics not available - using approximations")

	return limitations
}