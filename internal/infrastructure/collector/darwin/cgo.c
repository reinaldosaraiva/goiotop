//go:build darwin

#include "cgo.h"
#include <stdio.h>
#include <unistd.h>
#include <sys/proc_info.h>

// Get process I/O statistics using proc_pid_rusage
int get_process_io_stats(pid_t pid, process_io_stats *stats) {
    if (stats == NULL) {
        return -1;
    }

    memset(stats, 0, sizeof(process_io_stats));

    // Get resource usage info
    struct rusage_info_v4 rusage_info;
    int ret = proc_pid_rusage(pid, RUSAGE_INFO_V4, (rusage_info_t *)&rusage_info);

    if (ret < 0) {
        stats->error_code = errno;
        snprintf(stats->error_msg, sizeof(stats->error_msg),
                "proc_pid_rusage failed: %s", strerror(errno));
        return -1;
    }

    // Fill in the statistics
    stats->bytes_read = rusage_info.ri_diskio_bytesread;
    stats->bytes_written = rusage_info.ri_diskio_byteswritten;
    stats->cpu_time_user = rusage_info.ri_user_time;
    stats->cpu_time_system = rusage_info.ri_system_time;
    stats->memory_rss = rusage_info.ri_resident_size;

    // Note: macOS doesn't provide direct read/write operation counts
    // We'll estimate based on bytes (assuming 4KB operations)
    stats->read_ops = stats->bytes_read / 4096;
    stats->write_ops = stats->bytes_written / 4096;

    return 0;
}

// Get system information using sysctl
int get_system_info(system_info *info) {
    if (info == NULL) {
        return -1;
    }

    memset(info, 0, sizeof(system_info));

    // Get CPU count
    int mib[2] = {CTL_HW, HW_NCPU};
    size_t len = sizeof(info->cpu_count);
    if (sysctl(mib, 2, &info->cpu_count, &len, NULL, 0) < 0) {
        info->error_code = errno;
        snprintf(info->error_msg, sizeof(info->error_msg),
                "sysctl HW_NCPU failed: %s", strerror(errno));
        return -1;
    }

    // Get physical memory
    int64_t memsize;
    mib[1] = HW_MEMSIZE;
    len = sizeof(memsize);
    if (sysctl(mib, 2, &memsize, &len, NULL, 0) < 0) {
        info->error_code = errno;
        snprintf(info->error_msg, sizeof(info->error_msg),
                "sysctl HW_MEMSIZE failed: %s", strerror(errno));
        return -1;
    }
    info->memory_total = memsize;

    // Get memory usage from Mach
    vm_size_t page_size;
    mach_port_t mach_port = mach_host_self();
    vm_statistics64_data_t vm_stat;
    mach_msg_type_number_t host_size = sizeof(vm_stat) / sizeof(natural_t);

    if (host_page_size(mach_port, &page_size) == KERN_SUCCESS &&
        host_statistics64(mach_port, HOST_VM_INFO,
                         (host_info64_t)&vm_stat, &host_size) == KERN_SUCCESS) {
        info->memory_free = (int64_t)(vm_stat.free_count * page_size);
        info->memory_used = info->memory_total - info->memory_free;
    }

    // Get load average
    double loadavg[3];
    if (getloadavg(loadavg, 3) != -1) {
        info->load_avg_1 = loadavg[0];
        info->load_avg_5 = loadavg[1];
        info->load_avg_15 = loadavg[2];
    }

    return 0;
}

// Get list of all process IDs
int get_process_list(pid_t **pids, int *count) {
    if (pids == NULL || count == NULL) {
        return -1;
    }

    // Get buffer size needed
    int mib[4] = {CTL_KERN, KERN_PROC, KERN_PROC_ALL, 0};
    size_t size;

    if (sysctl(mib, 4, NULL, &size, NULL, 0) < 0) {
        return -1;
    }

    // Allocate buffer
    struct kinfo_proc *procs = malloc(size);
    if (procs == NULL) {
        return -1;
    }

    // Get process list
    if (sysctl(mib, 4, procs, &size, NULL, 0) < 0) {
        free(procs);
        return -1;
    }

    // Count processes and extract PIDs
    int nprocs = size / sizeof(struct kinfo_proc);
    *pids = malloc(nprocs * sizeof(pid_t));
    if (*pids == NULL) {
        free(procs);
        return -1;
    }

    *count = 0;
    for (int i = 0; i < nprocs; i++) {
        if (procs[i].kp_proc.p_pid > 0) {
            (*pids)[(*count)++] = procs[i].kp_proc.p_pid;
        }
    }

    free(procs);
    return 0;
}

// Get CPU usage percentage
int get_cpu_usage(double *usage) {
    if (usage == NULL) {
        return -1;
    }

    mach_port_t host_port = mach_host_self();
    host_cpu_load_info_data_t cpu_info;
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;

    if (host_statistics(host_port, HOST_CPU_LOAD_INFO,
                       (host_info_t)&cpu_info, &count) != KERN_SUCCESS) {
        return -1;
    }

    unsigned int total_ticks = 0;
    for (int i = 0; i < CPU_STATE_MAX; i++) {
        total_ticks += cpu_info.cpu_ticks[i];
    }

    if (total_ticks == 0) {
        *usage = 0.0;
        return 0;
    }

    unsigned int idle_ticks = cpu_info.cpu_ticks[CPU_STATE_IDLE];
    *usage = 100.0 * (1.0 - ((double)idle_ticks / (double)total_ticks));

    return 0;
}

// Estimate memory pressure (macOS doesn't have direct PSI)
int get_memory_pressure(double *pressure) {
    if (pressure == NULL) {
        return -1;
    }

    vm_size_t page_size;
    mach_port_t mach_port = mach_host_self();
    vm_statistics64_data_t vm_stat;
    mach_msg_type_number_t host_size = sizeof(vm_stat) / sizeof(natural_t);

    if (host_page_size(mach_port, &page_size) != KERN_SUCCESS ||
        host_statistics64(mach_port, HOST_VM_INFO,
                         (host_info64_t)&vm_stat, &host_size) != KERN_SUCCESS) {
        return -1;
    }

    // Calculate pressure based on memory statistics
    // This is a simplified estimation
    uint64_t total_pages = vm_stat.free_count + vm_stat.active_count +
                          vm_stat.inactive_count + vm_stat.wire_count;

    if (total_pages == 0) {
        *pressure = 0.0;
        return 0;
    }

    // Higher pressure when free memory is low and pageouts are high
    double free_ratio = (double)vm_stat.free_count / (double)total_pages;
    double pageout_ratio = (double)vm_stat.pageouts / (double)total_pages;

    *pressure = (1.0 - free_ratio) * 50.0 + pageout_ratio * 50.0;
    if (*pressure > 100.0) *pressure = 100.0;
    if (*pressure < 0.0) *pressure = 0.0;

    return 0;
}

// Check for sandbox restrictions
int check_sandbox_restrictions(void) {
    // Try to access a restricted resource
    struct rusage_info_v4 rusage_info;
    int ret = proc_pid_rusage(1, RUSAGE_INFO_V4, (rusage_info_t *)&rusage_info);

    if (ret < 0 && errno == EPERM) {
        // Likely sandboxed
        return 1;
    }

    return 0;
}

// Detect available capabilities
int detect_capabilities(void) {
    int capabilities = 0;

    // Check if we can read process info
    struct rusage_info_v4 rusage_info;
    if (proc_pid_rusage(getpid(), RUSAGE_INFO_V4, (rusage_info_t *)&rusage_info) == 0) {
        capabilities |= 1; // Has process I/O
    }

    // Check if we can access system info
    int cpu_count;
    size_t len = sizeof(cpu_count);
    int mib[2] = {CTL_HW, HW_NCPU};
    if (sysctl(mib, 2, &cpu_count, &len, NULL, 0) == 0) {
        capabilities |= 2; // Has sysctl access
    }

    // Check for Docker (simple check - could be improved)
    if (access("/var/run/docker.sock", F_OK) == 0) {
        capabilities |= 4; // Has Docker
    }

    return capabilities;
}

// Get error string for error code
const char* get_error_string(int error_code) {
    return strerror(error_code);
}

// Free process list memory
void free_process_list(pid_t *pids) {
    if (pids != NULL) {
        free(pids);
    }
}