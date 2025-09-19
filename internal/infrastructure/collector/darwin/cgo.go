//go:build darwin

package darwin

/*
#include "cgo.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// CGOProcessIOStats wraps the C process_io_stats structure
type CGOProcessIOStats struct {
	BytesRead     int64
	BytesWritten  int64
	ReadOps       int64
	WriteOps      int64
	CPUTimeUser   int64
	CPUTimeSystem int64
	MemoryRSS     int64
}

// CGOSystemInfo wraps the C system_info structure
type CGOSystemInfo struct {
	CPUCount     int
	MemoryTotal  int64
	MemoryFree   int64
	MemoryUsed   int64
	LoadAvg1     float64
	LoadAvg5     float64
	LoadAvg15    float64
}

// GetProcessIOStats retrieves I/O statistics for a specific process
func GetProcessIOStats(pid int32) (*CGOProcessIOStats, error) {
	var stats C.process_io_stats
	ret := C.get_process_io_stats(C.pid_t(pid), &stats)

	if ret < 0 {
		if stats.error_code != 0 {
			return nil, fmt.Errorf("proc_pid_rusage failed (code %d): %s",
				int(stats.error_code), C.GoString(&stats.error_msg[0]))
		}
		return nil, fmt.Errorf("failed to get process I/O stats")
	}

	return &CGOProcessIOStats{
		BytesRead:     int64(stats.bytes_read),
		BytesWritten:  int64(stats.bytes_written),
		ReadOps:       int64(stats.read_ops),
		WriteOps:      int64(stats.write_ops),
		CPUTimeUser:   int64(stats.cpu_time_user),
		CPUTimeSystem: int64(stats.cpu_time_system),
		MemoryRSS:     int64(stats.memory_rss),
	}, nil
}

// GetSystemInfo retrieves system information
func GetSystemInfo() (*CGOSystemInfo, error) {
	var info C.system_info
	ret := C.get_system_info(&info)

	if ret < 0 {
		if info.error_code != 0 {
			return nil, fmt.Errorf("sysctl failed (code %d): %s",
				int(info.error_code), C.GoString(&info.error_msg[0]))
		}
		return nil, fmt.Errorf("failed to get system info")
	}

	return &CGOSystemInfo{
		CPUCount:     int(info.cpu_count),
		MemoryTotal:  int64(info.memory_total),
		MemoryFree:   int64(info.memory_free),
		MemoryUsed:   int64(info.memory_used),
		LoadAvg1:     float64(info.load_avg_1),
		LoadAvg5:     float64(info.load_avg_5),
		LoadAvg15:    float64(info.load_avg_15),
	}, nil
}

// GetProcessList retrieves a list of all process IDs
func GetProcessList() ([]int32, error) {
	var pids *C.pid_t
	var count C.int

	ret := C.get_process_list(&pids, &count)
	if ret < 0 {
		return nil, fmt.Errorf("failed to get process list")
	}
	defer C.free_process_list(pids)

	// Convert C array to Go slice
	pidSlice := make([]int32, int(count))
	if count > 0 {
		// Create a Go slice from the C array
		cPids := (*[1 << 30]C.pid_t)(unsafe.Pointer(pids))[:count:count]
		for i := 0; i < int(count); i++ {
			pidSlice[i] = int32(cPids[i])
		}
	}

	return pidSlice, nil
}

// GetCPUUsage retrieves the current CPU usage percentage
func GetCPUUsage() (float64, error) {
	var usage C.double
	ret := C.get_cpu_usage(&usage)

	if ret < 0 {
		return 0, fmt.Errorf("failed to get CPU usage")
	}

	return float64(usage), nil
}

// GetMemoryPressure retrieves an estimate of memory pressure
func GetMemoryPressure() (float64, error) {
	var pressure C.double
	ret := C.get_memory_pressure(&pressure)

	if ret < 0 {
		return 0, fmt.Errorf("failed to get memory pressure")
	}

	return float64(pressure), nil
}

// CheckSandboxRestrictions checks if the application is running in a sandbox
func CheckSandboxRestrictions() bool {
	ret := C.check_sandbox_restrictions()
	return ret == 1
}

// DetectCapabilities detects available capabilities
func DetectCapabilities() int {
	return int(C.detect_capabilities())
}

// LoadAverage represents system load averages
type LoadAverage struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

// GetLoadAverage retrieves the system load average
func GetLoadAverage() (*LoadAverage, error) {
	info, err := GetSystemInfo()
	if err != nil {
		return nil, err
	}

	return &LoadAverage{
		Load1:  info.LoadAvg1,
		Load5:  info.LoadAvg5,
		Load15: info.LoadAvg15,
	}, nil
}