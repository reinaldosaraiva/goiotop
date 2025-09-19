# goiotop CLI Documentation

## Overview

goiotop is a command-line tool to monitor I/O usage by processes on Linux and macOS. It provides real-time metrics for disk I/O, network I/O, memory usage, and CPU usage, similar to iotop but with additional features and cross-platform support.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Global Options](#global-options)
- [Output Formats](#output-formats)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Automation](#automation)
- [Troubleshooting](#troubleshooting)

## Installation

### From Source

```bash
git clone https://github.com/reinaldosaraiva/goiotop.git
cd goiotop
make build
sudo make install
```

### Pre-built Binaries

Download pre-built binaries from the releases page:

```bash
# Linux
wget https://github.com/reinaldosaraiva/goiotop/releases/latest/download/goiotop-linux-amd64
chmod +x goiotop-linux-amd64
sudo mv goiotop-linux-amd64 /usr/local/bin/goiotop

# macOS
wget https://github.com/reinaldosaraiva/goiotop/releases/latest/download/goiotop-darwin-amd64
chmod +x goiotop-darwin-amd64
sudo mv goiotop-darwin-amd64 /usr/local/bin/goiotop
```

## Quick Start

```bash
# Display real-time I/O metrics (similar to iotop)
goiotop display

# Show version information
goiotop version

# Collect metrics once
goiotop collect

# Export metrics to JSON
goiotop export --format json --destination metrics.json

# Show system information
goiotop system info
```

## Commands

### `display` - Display Metrics

Display system and process metrics in various modes.

```bash
goiotop display [flags]
```

#### Flags

- `--mode, -m string`: Display mode (`realtime`, `historical`, `summary`) (default: `realtime`)
- `--refresh, -r int`: Refresh interval in seconds (default: 1)
- `--top, -t int`: Number of top processes to display (default: 20)
- `--sort, -s string`: Sort by field (`total`, `disk-read`, `disk-write`, `net-rx`, `net-tx`, `cpu`, `memory`) (default: `total`)
- `--reverse`: Reverse sort order
- `--processes, -p string`: Filter by process names (comma-separated)
- `--threshold float`: Only show processes above threshold
- `--show-threads`: Show individual threads
- `--show-paths`: Show full executable paths
- `--aggregate`: Aggregate by process name

#### Examples

```bash
# Real-time display with 5-second refresh
goiotop display --refresh 5

# Top 10 processes by disk read
goiotop display --top 10 --sort disk-read

# Monitor specific processes
goiotop display --processes postgres,mysql

# Summary mode
goiotop display --mode summary
```

### `collect` - Collect Metrics

Collect various system and process metrics.

```bash
goiotop collect [flags]
```

#### Flags

- `--continuous, -C`: Continuous collection mode
- `--interval, -i int`: Collection interval in seconds (default: 1)
- `--duration, -d int`: Collection duration in seconds (0=infinite)
- `--processes, -p string`: Filter by process names
- `--disk-only`: Collect only disk I/O metrics
- `--network-only`: Collect only network I/O metrics
- `--memory-only`: Collect only memory metrics
- `--cpu-only`: Collect only CPU metrics
- `--output, -o string`: Output file path

#### Examples

```bash
# Single collection
goiotop collect

# Continuous collection every 5 seconds
goiotop collect --continuous --interval 5

# Collect for 60 seconds
goiotop collect --duration 60

# Collect only disk metrics
goiotop collect --disk-only

# Save to file
goiotop collect --output metrics.json --format json
```

### `export` - Export Metrics

Export collected metrics to various formats and destinations.

```bash
goiotop export [flags]
```

#### Flags

- `--format, -f string`: Export format (`json`, `csv`, `prometheus`) (default: `json`)
- `--destination, -d string`: Export destination (required)
- `--start string`: Start time (RFC3339 format)
- `--end string`: End time (RFC3339 format)
- `--interval int`: Sampling interval in seconds
- `--compress`: Compress output
- `--filter string`: Export filters
- `--include-raw`: Include raw metrics
- `--http-method string`: HTTP method for HTTP destination (default: `POST`)
- `--http-header string`: HTTP headers (format: Key:Value)

#### Examples

```bash
# Export to JSON file
goiotop export --format json --destination metrics.json

# Export to CSV with time range
goiotop export --format csv --destination metrics.csv \
  --start "2024-01-01T00:00:00Z" --end "2024-01-02T00:00:00Z"

# Export to HTTP endpoint
goiotop export --format json \
  --destination "http://metrics.example.com/api/metrics" \
  --http-method POST

# Export with compression
goiotop export --format json --destination metrics.json.gz --compress
```

### `version` - Version Information

Display version information for goiotop.

```bash
goiotop version
```

Example output:
```
goiotop version dev
commit: unknown
built at: unknown
```

### `system` - System Information

Display system information and health status.

```bash
goiotop system [subcommand]
```

#### Subcommands

##### `info` - System Information

```bash
goiotop system info [flags]
```

Flags:
- `--detailed, -d`: Show detailed information
- `--modules, -m string`: Specific modules (cpu, memory, disk, network)

##### `health` - Health Check

```bash
goiotop system health [flags]
```

Flags:
- `--all, -a`: Run all health checks
- `--checks, -c string`: Specific health checks

##### `capabilities` - Collector Capabilities

```bash
goiotop system capabilities [flags]
```

Flags:
- `--collectors, -c string`: Specific collectors

##### `config` - Configuration

```bash
goiotop system config [flags]
```

Flags:
- `--defaults`: Show default configuration
- `--validate`: Validate current configuration

#### Examples

```bash
# Basic system info
goiotop system info

# Detailed system info
goiotop system info --detailed

# Health check
goiotop system health

# Show capabilities
goiotop system capabilities

# Validate configuration
goiotop system config --validate
```

## Global Options

These options are available for all commands:

- `--config, -c string`: Config file path (default: `$HOME/.goiotop.yaml`)
- `--log-level string`: Log level (`debug`, `info`, `warn`, `error`, `fatal`) (default: `info`)
- `--format string`: Output format (`text`, `json`, `csv`) (default: `text`)
- `--no-color`: Disable colored output
- `--batch`: Run in batch mode (non-interactive)
- `--help, -h`: Show help

## Output Formats

### Text Format (Default)

Human-readable format similar to iotop:

```
goiotop - 14:32:05
────────────────────────────────────────────────────────────────────────
Total DISK READ: 125.4 MB/s | Total DISK WRITE: 45.2 MB/s
Total NET RX: 10.5 MB/s | Total NET TX: 5.2 MB/s
CPU: 45.2% | Memory: 62.3% | Active Processes: 125
────────────────────────────────────────────────────────────────────────
PID     COMMAND              USER      DISK READ  DISK WRITE  NET RX     NET TX     CPU%   MEM%
────────────────────────────────────────────────────────────────────────
1234    postgres             postgres  50.2 MB    10.5 MB     1.2 MB     0.5 MB     12.5%  8.2%
5678    mysql                mysql     30.1 MB    15.2 MB     2.3 MB     1.1 MB     8.3%   15.4%
```

### JSON Format

Machine-readable JSON format for programmatic processing:

```json
{
  "timestamp": "2024-01-15T14:32:05Z",
  "processes": [
    {
      "pid": 1234,
      "name": "postgres",
      "user": "postgres",
      "diskReadRate": 52633600,
      "diskWriteRate": 11010048,
      "networkRxRate": 1258291,
      "networkTxRate": 524288,
      "cpuPercent": 12.5,
      "memoryPercent": 8.2
    }
  ],
  "summary": {
    "totalDiskRead": 131563520,
    "totalDiskWrite": 47395840,
    "totalNetworkRx": 11010048,
    "totalNetworkTx": 5452800,
    "avgCPUPercent": 45.2,
    "avgMemoryPercent": 62.3,
    "activeProcesses": 125
  }
}
```

### CSV Format

Spreadsheet-compatible CSV format:

```csv
timestamp,pid,name,user,cpu_percent,memory_percent,disk_read,disk_write,network_rx,network_tx
2024-01-15T14:32:05Z,1234,postgres,postgres,12.5,8.2,52633600,11010048,1258291,524288
2024-01-15T14:32:05Z,5678,mysql,mysql,8.3,15.4,31563520,15938560,2411520,1153024
```

## Configuration

goiotop can be configured using a YAML configuration file.

### Default Configuration Location

- Linux/macOS: `$HOME/.goiotop.yaml`
- System-wide: `/etc/goiotop/config.yaml`

### Configuration File Example

```yaml
# goiotop configuration
logLevel: info

collection:
  defaultInterval: 1s
  bufferSize: 1000
  maxProcesses: 500
  includeThreads: false
  includeKernelThreads: false

storage:
  type: memory
  path: /var/lib/goiotop
  maxSize: 1000  # MB
  retention: 24h
  compression: true

export:
  defaultFormat: json
  compression: false
  includeRawMetrics: false

filters:
  minCPUPercent: 0.1
  minMemoryPercent: 0.1
  minIORate: 1024  # bytes/sec
  excludeProcesses:
    - kernel
    - systemd

display:
  refreshRate: 1s
  topN: 20
  sortBy: total
  showThreads: false
  colorOutput: true

monitoring:
  enabled: true
  metricsPort: 9090
  healthCheckInterval: 30s
```

### Environment Variables

Configuration can also be set using environment variables:

```bash
export GOIOTOP_LOG_LEVEL=debug
export GOIOTOP_COLLECTION_INTERVAL=5s
export GOIOTOP_DISPLAY_TOP_N=50
```

## Usage Examples

### Basic Monitoring

```bash
# Real-time monitoring (like iotop)
goiotop display

# Monitor with custom refresh rate
goiotop display --refresh 5

# Monitor top 10 processes
goiotop display --top 10
```

### Filtering and Sorting

```bash
# Sort by disk write
goiotop display --sort disk-write

# Filter specific processes
goiotop display --processes postgres,mysql,redis

# Show only high I/O processes
goiotop display --threshold 1000000
```

### Data Collection

```bash
# Collect metrics once
goiotop collect

# Continuous collection
goiotop collect --continuous --interval 5

# Collect for specific duration
goiotop collect --duration 3600
```

### Data Export

```bash
# Export to JSON
goiotop export --format json --destination metrics.json

# Export to CSV with time range
goiotop export --format csv --destination report.csv \
  --start "2024-01-01T00:00:00Z" --end "2024-01-02T00:00:00Z"

# Export to HTTP endpoint
goiotop export --format json \
  --destination "http://metrics-server.com/api/metrics" \
  --http-method POST \
  --http-header "Authorization: Bearer token123"
```

### Batch Mode

```bash
# Non-interactive JSON output
goiotop display --batch --format json

# Pipe to other tools
goiotop collect --format json | jq '.processes[] | select(.cpuPercent > 10)'

# Save to file
goiotop collect --continuous --interval 5 --output metrics.log
```

## Automation

### Systemd Service

Create `/etc/systemd/system/goiotop-monitor.service`:

```ini
[Unit]
Description=goiotop Monitoring Service
After=network.target

[Service]
Type=simple
User=monitor
ExecStart=/usr/local/bin/goiotop collect --continuous --interval 5 --output /var/log/goiotop/metrics.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable goiotop-monitor
sudo systemctl start goiotop-monitor
```

### Cron Job

Add to crontab for periodic collection:

```cron
# Collect metrics every hour
0 * * * * /usr/local/bin/goiotop collect --duration 3600 --interval 10 --output /var/log/goiotop/hourly-$(date +\%Y\%m\%d-\%H).json

# Daily summary report
0 0 * * * /usr/local/bin/goiotop export --format csv --destination /var/reports/daily-$(date +\%Y\%m\%d).csv
```

### Docker

Run goiotop in a container:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goiotop cmd/goiotop/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/goiotop /usr/local/bin/
ENTRYPOINT ["goiotop"]
```

Run:

```bash
docker run --pid=host --network=host -v /proc:/proc:ro goiotop:latest display
```

### Kubernetes DaemonSet

Deploy as DaemonSet to monitor all nodes:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: goiotop-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: goiotop
  template:
    metadata:
      labels:
        app: goiotop
    spec:
      hostPID: true
      hostNetwork: true
      containers:
      - name: goiotop
        image: goiotop:latest
        command: ["goiotop", "collect", "--continuous", "--format", "json"]
        volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        resources:
          limits:
            memory: "128Mi"
            cpu: "100m"
      volumes:
      - name: proc
        hostPath:
          path: /proc
```

## Troubleshooting

### Common Issues

#### Permission Denied

Some metrics require elevated privileges:

```bash
# Run with sudo
sudo goiotop display

# Or add user to necessary groups
sudo usermod -a -G disk,network $USER
```

#### High CPU Usage

Reduce collection frequency:

```bash
# Increase interval
goiotop display --refresh 5

# Reduce number of processes
goiotop display --top 10
```

#### No Data Displayed

Check if metrics are being collected:

```bash
# Enable debug logging
goiotop --log-level debug collect

# Check system capabilities
goiotop system capabilities
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
goiotop --log-level debug display
```

### Health Check

Verify system health:

```bash
goiotop system health --all
```

## Performance Considerations

- **Collection Interval**: Lower intervals provide more real-time data but increase CPU usage
- **Process Filtering**: Use filters to reduce the number of monitored processes
- **Output Format**: JSON/CSV formats use less CPU than formatted text display
- **Batch Mode**: Use batch mode for scripts and automation to reduce overhead

## Security

- goiotop requires read access to `/proc` filesystem
- Some metrics may require root or specific capabilities
- Use appropriate file permissions for output files containing metrics
- Sanitize data before exporting to external systems

## License

goiotop is licensed under the MIT License. See LICENSE file for details.

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/reinaldosaraiva/goiotop/issues
- Documentation: https://github.com/reinaldosaraiva/goiotop/wiki