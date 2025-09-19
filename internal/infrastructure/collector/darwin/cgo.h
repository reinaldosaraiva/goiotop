//go:build darwin

#ifndef GOIOTOP_DARWIN_CGO_H
#define GOIOTOP_DARWIN_CGO_H

#include <sys/types.h>
#include <sys/sysctl.h>
#include <sys/resource.h>
#include <mach/mach.h>
#include <mach/task.h>
#include <mach/mach_init.h>
#include <libproc.h>
#include <errno.h>
#include <stdlib.h>
#include <string.h>

// Process resource usage structure
typedef struct {
    int64_t bytes_read;
    int64_t bytes_written;
    int64_t read_ops;
    int64_t write_ops;
    int64_t cpu_time_user;
    int64_t cpu_time_system;
    int64_t memory_rss;
    int error_code;
    char error_msg[256];
} process_io_stats;

// System information structure
typedef struct {
    int cpu_count;
    int64_t memory_total;
    int64_t memory_free;
    int64_t memory_used;
    double load_avg_1;
    double load_avg_5;
    double load_avg_15;
    int error_code;
    char error_msg[256];
} system_info;

// Function declarations
int get_process_io_stats(pid_t pid, process_io_stats *stats);
int get_system_info(system_info *info);
int get_process_list(pid_t **pids, int *count);
int get_cpu_usage(double *usage);
int get_memory_pressure(double *pressure);
int check_sandbox_restrictions(void);
int detect_capabilities(void);

// Helper functions
const char* get_error_string(int error_code);
void free_process_list(pid_t *pids);

#endif // GOIOTOP_DARWIN_CGO_H