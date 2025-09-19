//go:build linux

package linux

import (
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// detectSystemCapabilities detects available features on the Linux system
func detectSystemCapabilities() services.CollectorCapabilities {
	caps := services.CollectorCapabilities{
		SupportsSystemMetrics:  true, // Always supported on Linux
		SupportsProcessMetrics: true, // Always supported on Linux
		SupportsCPUPressure:    checkPSISupport(),
		SupportsDiskIO:         checkDiskIOSupport(),
		SupportsNetworkIO:      false, // Not implemented yet
		SupportsMemoryPressure: checkPSISupport(),
		SupportsIOPressure:     checkPSISupport(),
		SupportsCGroups:        checkCgroupSupport(),
		SupportsContainers:     checkContainerSupport(),
		RequiresRoot:           !checkProcessReadPermissions() || !checkProcessIOSupport(),
		PlatformSpecific:       make(map[string]bool),
	}

	// Add platform-specific information
	caps.PlatformSpecific["platform"] = true
	caps.PlatformSpecific["kernel_version"] = true
	caps.PlatformSpecific["go_version"] = true
	caps.PlatformSpecific["has_root"] = checkRootPrivileges()
	caps.PlatformSpecific["can_read_all_processes"] = checkProcessReadPermissions()
	caps.PlatformSpecific["supports_process_io"] = checkProcessIOSupport()
	caps.PlatformSpecific["cgroup_v1"] = detectCgroupVersion() == 1
	caps.PlatformSpecific["cgroup_v2"] = detectCgroupVersion() == 2

	// Store actual values as well (using a different map would be better, but keeping simple)
	// These are just indicators for now
	caps.PlatformSpecific["is_container"] = checkContainerSupport()
	caps.PlatformSpecific["has_psi"] = checkPSISupport()

	return caps
}

// checkPSISupport checks if PSI is available
func checkPSISupport() bool {
	// PSI requires kernel 4.20+
	_, err := os.Stat("/proc/pressure/cpu")
	return err == nil
}

// checkProcessIOSupport checks if per-process I/O accounting is available
func checkProcessIOSupport() bool {
	// Check if we can read /proc/self/io
	_, err := os.Stat("/proc/self/io")
	return err == nil
}

// checkDiskIOSupport checks if disk I/O statistics are available
func checkDiskIOSupport() bool {
	// Check if /proc/diskstats exists
	_, err := os.Stat("/proc/diskstats")
	return err == nil
}

// checkContainerSupport checks for container runtime indicators
func checkContainerSupport() bool {
	// Basic check for container environments
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// Check cgroup path for container indicators
	cgroupData, err := os.ReadFile("/proc/self/cgroup")
	if err == nil {
		cgroupStr := string(cgroupData)
		if strings.Contains(cgroupStr, "docker") ||
			strings.Contains(cgroupStr, "kubepods") ||
			strings.Contains(cgroupStr, "containerd") {
			return true
		}
	}

	return false
}

// checkRootPrivileges checks if running as root
func checkRootPrivileges() bool {
	return os.Geteuid() == 0
}

// checkProcessReadPermissions checks if we can read all process information
func checkProcessReadPermissions() bool {
	// If we're root, we can read everything
	if checkRootPrivileges() {
		return true
	}

	// Check if we have CAP_SYS_PTRACE capability
	// This is a simplified check - full capability checking requires additional libraries
	return false
}

// checkCgroupSupport checks if cgroups are available
func checkCgroupSupport() bool {
	// Check for cgroup v1
	if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
		return true
	}

	// Check for cgroup v2
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		return true
	}

	return false
}

// detectCgroupVersion detects the cgroup version
func detectCgroupVersion() int {
	// Check for cgroup v2 unified hierarchy
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		return 2
	}

	// Check for cgroup v1
	if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
		return 1
	}

	return 0
}

// getKernelVersion gets the kernel version string
func getKernelVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return "unknown"
	}

	// Convert to string
	release := make([]byte, 0, len(uname.Release))
	for _, b := range uname.Release {
		if b == 0 {
			break
		}
		release = append(release, byte(b))
	}

	return string(release)
}

// buildFeaturesList builds a list of available features
func buildFeaturesList(caps services.CollectorCapabilities) []string {
	var features []string

	if caps.SupportsSystemMetrics {
		features = append(features, "System Metrics")
	}
	if caps.SupportsProcessMetrics {
		features = append(features, "Process Metrics")
	}
	if caps.SupportsCPUPressure {
		features = append(features, "CPU Pressure (PSI)")
	}
	if caps.SupportsMemoryPressure {
		features = append(features, "Memory Pressure (PSI)")
	}
	if caps.SupportsIOPressure {
		features = append(features, "I/O Pressure (PSI)")
	}
	if caps.SupportsDiskIO {
		features = append(features, "Disk I/O Statistics")
	}
	if caps.SupportsContainers {
		features = append(features, "Container Metrics")
	}
	if caps.SupportsCGroups {
		features = append(features, "Cgroups")
	}
	if !caps.RequiresRoot {
		features = append(features, "No Root Required")
	}

	return features
}

// buildLimitationsList builds a list of limitations
func buildLimitationsList(caps services.CollectorCapabilities) []string {
	var limitations []string

	if caps.RequiresRoot && !checkRootPrivileges() {
		limitations = append(limitations, "Limited process visibility (not running as root)")
	}
	if !caps.SupportsCPUPressure {
		limitations = append(limitations, "PSI not available (kernel < 4.20)")
	}
	if caps.RequiresRoot {
		limitations = append(limitations, "Some features require root privileges")
	}
	if !caps.SupportsNetworkIO {
		limitations = append(limitations, "Network I/O monitoring not implemented")
	}

	return limitations
}

// checkKernelVersion checks if kernel version meets minimum requirements
func checkKernelVersion(major, minor int) bool {
	version := getKernelVersion()
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	// Parse major version
	var kernelMajor, kernelMinor int
	if _, err := strings.Cut(parts[0], "-"); err == nil {
		return false
	}

	// Simple version parsing (doesn't handle all cases)
	if n, err := parseVersionNumber(parts[0]); err == nil {
		kernelMajor = n
	}
	if n, err := parseVersionNumber(parts[1]); err == nil {
		kernelMinor = n
	}

	if kernelMajor > major {
		return true
	}
	if kernelMajor == major && kernelMinor >= minor {
		return true
	}

	return false
}

// parseVersionNumber parses a version number string
func parseVersionNumber(s string) (int, error) {
	// Remove any non-numeric suffix
	for i, r := range s {
		if r < '0' || r > '9' {
			s = s[:i]
			break
		}
	}

	var n int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}

	return n, nil
}