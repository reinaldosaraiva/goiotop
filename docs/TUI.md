# GoIOTop TUI Documentation

## Overview

GoIOTop provides a comprehensive Terminal User Interface (TUI) mode that offers real-time, interactive monitoring of system resources similar to iotop. The TUI is built using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework and provides a modern, responsive interface for system monitoring.

## Quick Start

### Running the TUI

```bash
# Build and run TUI mode
make run-tui

# Or directly with the binary
./bin/goiotop tui

# With custom options
./bin/goiotop tui --refresh 1s --sort memory --theme light
```

### Command Line Options

```bash
goiotop tui [flags]

Flags:
  --refresh duration    Refresh interval (default 2s)
  --sort string        Initial sort order: cpu, memory, io, pid, name (default "cpu")
  --only-active        Show only active processes initially
  --show-kernel        Show kernel threads
  --show-system        Show system processes
  --limit int          Maximum number of processes to track (default 100)
  --theme string       Color theme: dark, light, none (default "dark")
  -h, --help          Help for tui command
```

## User Interface

### Views

The TUI provides three main views that can be switched using the Tab key or number keys:

#### 1. Process List View (Default)

Displays a real-time table of processes with the following columns:

- **PID**: Process ID
- **USER**: Process owner
- **PROCESS**: Process name
- **CPU%**: CPU usage percentage
- **MEM%**: Memory usage percentage
- **DISK R**: Disk read rate (MB/s)
- **DISK W**: Disk write rate (MB/s)
- **NET RX**: Network receive rate (MB/s)
- **NET TX**: Network transmit rate (MB/s)
- **STATE**: Process state (R=Running, S=Sleeping, etc.)

```
┌─────────────────────────────────────────────────────────────────────┐
│ GoIOTop - System Monitor        [Process List]   Last update: 14:23:45 │
├─────────────────────────────────────────────────────────────────────┤
│ PID     USER      PROCESS         CPU%   MEM%   DISK R    DISK W     │
│ 1234    root      systemd         0.5    1.2    0.00      0.00       │
│ 5678    user      chrome         25.3   15.7    1.24      0.53       │
│ 9012    user      node           10.2    8.3    0.00      0.12       │
│ ...                                                                   │
├─────────────────────────────────────────────────────────────────────┤
│ Showing 50 of 150 processes | Sort: cpu | Filter: none              │
└─────────────────────────────────────────────────────────────────────┘
```

#### 2. System Summary View

Provides an overview of system-wide metrics:

```
┌─────────────────────────────────────────────────────────────────────┐
│ GoIOTop - System Monitor      [System Summary]   Last update: 14:23:45 │
├─────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────┐  ┌──────────────────┐                         │
│ │   CPU Usage      │  │  Memory Usage    │                         │
│ │   ████░░░░ 45%   │  │  ██████░░ 67%    │                         │
│ │   I/O Wait: 2.3% │  │  Swap: ░░░░ 5%   │                         │
│ └──────────────────┘  └──────────────────┘                         │
│                                                                      │
│ ┌──────────────────┐  ┌──────────────────┐                         │
│ │    Disk I/O      │  │   Network I/O    │                         │
│ │ Read:  12.5 MB/s │  │ RX: 1.2 MB/s     │                         │
│ │ Write: 8.3 MB/s  │  │ TX: 0.5 MB/s     │                         │
│ └──────────────────┘  └──────────────────┘                         │
│                                                                      │
│ Load Average: 1.45, 1.23, 0.98    Uptime: 5d 12h 34m               │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3. Help View

Displays comprehensive keyboard shortcuts and usage information.

### Visual Indicators

- **High CPU Usage** (>80%): Displayed in red
- **High Memory Usage** (>80%): Displayed in yellow
- **High I/O Activity** (>100 MB/s): Displayed in blue
- **Selected Process**: Highlighted with inverse colors
- **Sorted Column**: Underlined header

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `↑`/`k` | Move selection up |
| `↓`/`j` | Move selection down |
| `PgUp` | Page up |
| `PgDown` | Page down |
| `Home` | Go to top |
| `End` | Go to bottom |

### View Switching

| Key | Action |
|-----|--------|
| `Tab` | Cycle through views |
| `1` | Process List view |
| `2` | System Summary view |
| `3`/`h` | Help view |

### Sorting (Process List View)

| Key | Action |
|-----|--------|
| `c` | Sort by CPU usage |
| `m` | Sort by Memory usage |
| `i` | Sort by I/O (disk + network) |
| `p` | Sort by Process ID |
| `n` | Sort by Process Name |

### Filtering (Process List View)

| Key | Action |
|-----|--------|
| `f` | Open filter dialog |
| `/` | Search for process |
| `Esc` | Clear filter/Cancel |
| `Enter` | Apply filter |

### General

| Key | Action |
|-----|--------|
| `r` | Refresh immediately |
| `q` | Quit GoIOTop |
| `h`/`?` | Show help |

## Filtering

The TUI supports multiple filtering options:

### Process Name Filter

Press `f` to open the filter dialog and enter a process name or pattern. The filter matches against:
- Process name
- Command line
- Username

Example filters:
- `chrome` - Show only Chrome processes
- `python` - Show only Python processes
- `root` - Show only root processes

### Threshold Filters

You can set minimum thresholds for displaying processes:

- CPU threshold: Only show processes using more than X% CPU
- Memory threshold: Only show processes using more than X% memory
- I/O threshold: Only show processes with combined I/O > X MB/s

## Themes

GoIOTop supports three color themes:

### Dark Theme (Default)

Optimized for dark terminals with high contrast colors.

```bash
goiotop tui --theme dark
```

### Light Theme

Designed for light terminal backgrounds.

```bash
goiotop tui --theme light
```

### No Colors

Disables all colors for monochrome terminals or accessibility.

```bash
goiotop tui --theme none
```

## Configuration

### Config File

GoIOTop looks for configuration in the following locations:
1. `~/.goiotop/config.yaml`
2. `./config.yaml`
3. Environment variables (prefix: `GOIOTOP_`)

Example configuration:

```yaml
display:
  refresh_interval: 2s
  default_sort: cpu
  theme: dark
  max_rows: 50

collection:
  process_limit: 100
  include_idle: false
  include_kernel: false
  include_system_procs: false
  min_cpu_threshold: 0.1
  min_mem_threshold: 0.1

log:
  level: info
  file: ~/.goiotop/goiotop.log
```

### Environment Variables

```bash
export GOIOTOP_DISPLAY_REFRESH_INTERVAL=1s
export GOIOTOP_DISPLAY_THEME=light
export GOIOTOP_COLLECTION_PROCESS_LIMIT=200
```

## Performance Considerations

### Refresh Rate

The refresh rate affects both system load and responsiveness:
- **Fast (< 1s)**: More responsive but higher CPU usage
- **Default (2s)**: Good balance for most systems
- **Slow (> 5s)**: Lower system impact but less responsive

### Process Limit

Limiting the number of tracked processes improves performance:
- **Default (100)**: Suitable for most systems
- **High (> 500)**: May cause lag on slower systems
- **Low (< 50)**: Better for resource-constrained environments

### Optimization Tips

1. **Use filters** to reduce the number of displayed processes
2. **Increase refresh interval** if experiencing high CPU usage
3. **Disable kernel threads** with `--show-kernel=false`
4. **Use `--only-active`** to hide idle processes

## Troubleshooting

### TUI Not Starting

```bash
# Check if terminal supports TUI
echo $TERM

# Try with explicit terminal settings
TERM=xterm-256color goiotop tui

# Run with debug logging
goiotop tui --log-level debug
```

### Display Issues

If the display appears corrupted:
1. Check terminal size: `echo $LINES $COLUMNS`
2. Try resizing the terminal window
3. Use a different terminal emulator
4. Disable colors: `goiotop tui --theme none`

### Performance Issues

If the TUI is slow or unresponsive:
1. Increase refresh interval: `--refresh 5s`
2. Reduce process limit: `--limit 50`
3. Use filters to show fewer processes
4. Check system load with other tools

### No Data Displayed

If processes aren't showing:
1. Check permissions (may need sudo for some metrics)
2. Verify no filters are active (press Esc)
3. Check sort order (might be showing idle processes last)
4. Review log file for errors

## Advanced Usage

### Scripting

While TUI mode is interactive, you can use it in scripts:

```bash
# Run for 60 seconds then exit
timeout 60 goiotop tui

# Capture TUI session
script -c "goiotop tui" session.log
```

### Remote Monitoring

Monitor remote systems over SSH:

```bash
# Direct SSH
ssh remote-host goiotop tui

# With compression for slow connections
ssh -C remote-host goiotop tui --refresh 5s
```

### Integration with tmux/screen

GoIOTop works well in terminal multiplexers:

```bash
# tmux
tmux new-session -s monitoring goiotop tui

# screen
screen -S monitoring goiotop tui
```

## Examples

### Monitor High CPU Processes

```bash
goiotop tui --sort cpu --only-active
```

### Focus on I/O Activity

```bash
goiotop tui --sort io --refresh 1s
```

### Light Theme with Custom Limits

```bash
goiotop tui --theme light --limit 50 --refresh 3s
```

### Monitor Specific User Processes

```bash
# Start TUI then press 'f' and enter username
goiotop tui
```

## Tips and Tricks

1. **Quick Process Kill**: Select a process and press `x` (SIGTERM) or `X` (SIGKILL) [Note: Feature planned]
2. **Export Data**: Press `e` to export current view to file [Note: Feature planned]
3. **Process Details**: Press `Enter` on a selected process for detailed view [Note: Feature planned]
4. **Historical Data**: Press `F5` to toggle between real-time and historical view [Note: Feature planned]
5. **Multiple Sorts**: Hold Shift while pressing sort keys for secondary sort [Note: Feature planned]

## Comparison with Other Tools

| Feature | GoIOTop TUI | iotop | htop | top |
|---------|------------|-------|------|-----|
| Real-time I/O | ✓ | ✓ | ✗ | ✗ |
| Network I/O | ✓ | ✗ | ✗ | ✗ |
| Modern TUI | ✓ | ✗ | ✓ | ✗ |
| Themes | ✓ | ✗ | ✓ | ✗ |
| Cross-platform | ✓ | Linux | Unix | Unix |
| Go-based | ✓ | ✗ | ✗ | ✗ |

## Known Limitations

1. **Root Access**: Some metrics require elevated privileges
2. **Platform Support**: Full features on Linux, limited on Windows/macOS
3. **Terminal Size**: Minimum 80x24 terminal size required
4. **SSH Sessions**: Some features may be limited over SSH

## Contributing

To contribute to the TUI development:

1. Fork the repository
2. Create a feature branch
3. Run tests: `make test-tui`
4. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## Support

For issues, questions, or feature requests:
- GitHub Issues: [github.com/reinaldosaraiva/goiotop/issues](https://github.com/reinaldosaraiva/goiotop/issues)
- Documentation: [github.com/reinaldosaraiva/goiotop/docs](https://github.com/reinaldosaraiva/goiotop/docs)