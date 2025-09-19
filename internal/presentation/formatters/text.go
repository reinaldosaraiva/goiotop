package formatters

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// TextFormatter formats DTOs as human-readable text output
type TextFormatter struct {
	colorEnabled   bool
	showHeaders    bool
	columnWidth    int
	showProgress   bool
	showGraphs     bool
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(options ...TextOption) *TextFormatter {
	f := &TextFormatter{
		colorEnabled: false, // Disabled by default for compatibility
		showHeaders:  true,
		columnWidth:  10,
		showProgress: false,
		showGraphs:   false,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

// TextOption is a configuration option for text formatter
type TextOption func(*TextFormatter)

// WithColor enables or disables colored output
func WithColor(enabled bool) TextOption {
	return func(f *TextFormatter) {
		f.colorEnabled = enabled
	}
}

// WithTableHeaders enables or disables table headers
func WithTableHeaders(show bool) TextOption {
	return func(f *TextFormatter) {
		f.showHeaders = show
	}
}

// WithColumnWidth sets the column width for tables
func WithColumnWidth(width int) TextOption {
	return func(f *TextFormatter) {
		f.columnWidth = width
	}
}

// WithProgressBars enables progress bar display
func WithProgressBars(show bool) TextOption {
	return func(f *TextFormatter) {
		f.showProgress = show
	}
}

// WithGraphs enables graph display
func WithGraphs(show bool) TextOption {
	return func(f *TextFormatter) {
		f.showGraphs = show
	}
}

// FormatDisplayResult formats a display result as text (iotop-style)
func (f *TextFormatter) FormatDisplayResult(result dto.DisplayResultDTO, writer io.Writer) error {
	// Clear screen escape sequence (can be disabled in batch mode)
	// fmt.Fprint(writer, "\033[H\033[2J")

	// Header
	fmt.Fprintf(writer, "goiotop - %s\n", result.Timestamp.Format("15:04:05"))
	fmt.Fprintln(writer, strings.Repeat("─", 100))

	// System summary
	if result.Summary != nil {
		f.formatSummary(result.Summary, writer)
		fmt.Fprintln(writer, strings.Repeat("─", 100))
	}

	// Process table
	f.formatProcessTable(result.Processes, writer)

	return nil
}

// formatSummary formats the summary section
func (f *TextFormatter) formatSummary(summary *dto.DisplaySummaryDTO, writer io.Writer) {
	fmt.Fprintf(writer, "Total DISK READ: %s/s | Total DISK WRITE: %s/s",
		f.formatBytes(summary.TotalDiskRead),
		f.formatBytes(summary.TotalDiskWrite))

	if f.showProgress {
		fmt.Fprintf(writer, " [%s]", f.createProgressBar(summary.TotalDiskRead, 1000000))
	}
	fmt.Fprintln(writer)

	fmt.Fprintf(writer, "Total NET RX: %s/s | Total NET TX: %s/s",
		f.formatBytes(summary.TotalNetworkRx),
		f.formatBytes(summary.TotalNetworkTx))

	if f.showProgress {
		fmt.Fprintf(writer, " [%s]", f.createProgressBar(summary.TotalNetworkTx, 1000000))
	}
	fmt.Fprintln(writer)

	fmt.Fprintf(writer, "CPU: %.1f%% | Memory: %.1f%% | Active Processes: %d\n",
		summary.AvgCPUPercent,
		summary.AvgMemoryPercent,
		summary.ActiveProcesses)
}

// formatProcessTable formats the process table
func (f *TextFormatter) formatProcessTable(processes []dto.ProcessDisplayDTO, writer io.Writer) {
	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)

	// Table headers
	if f.showHeaders {
		if f.colorEnabled {
			fmt.Fprintf(w, "\033[1m%-7s\t%-20s\t%8s\t%10s\t%10s\t%10s\t%10s\t%7s\t%7s\033[0m\n",
				"PID", "COMMAND", "USER", "DISK READ", "DISK WRITE", "NET RX", "NET TX", "CPU%", "MEM%")
		} else {
			fmt.Fprintf(w, "%-7s\t%-20s\t%8s\t%10s\t%10s\t%10s\t%10s\t%7s\t%7s\n",
				"PID", "COMMAND", "USER", "DISK READ", "DISK WRITE", "NET RX", "NET TX", "CPU%", "MEM%")
		}
		fmt.Fprintln(w, strings.Repeat("─", 100))
	}

	// Process rows
	for _, p := range processes {
		name := f.truncateString(p.Name, 20)

		// Color coding based on I/O activity
		rowColor := ""
		if f.colorEnabled {
			totalIO := p.DiskReadRate + p.DiskWriteRate + p.NetworkRxRate + p.NetworkTxRate
			switch {
			case totalIO > 10000000: // > 10 MB/s
				rowColor = "\033[31m" // Red for high I/O
			case totalIO > 1000000: // > 1 MB/s
				rowColor = "\033[33m" // Yellow for medium I/O
			case totalIO > 0:
				rowColor = "\033[32m" // Green for low I/O
			default:
				rowColor = "\033[90m" // Gray for no I/O
			}
		}

		if rowColor != "" {
			fmt.Fprintf(w, "%s%-7d\t%-20s\t%8s\t%10s\t%10s\t%10s\t%10s\t%6.1f%%\t%6.1f%%\033[0m\n",
				rowColor,
				p.PID,
				name,
				f.truncateString(p.User, 8),
				f.formatBytes(p.DiskReadRate),
				f.formatBytes(p.DiskWriteRate),
				f.formatBytes(p.NetworkRxRate),
				f.formatBytes(p.NetworkTxRate),
				p.CPUPercent,
				p.MemoryPercent)
		} else {
			fmt.Fprintf(w, "%-7d\t%-20s\t%8s\t%10s\t%10s\t%10s\t%10s\t%6.1f%%\t%6.1f%%\n",
				p.PID,
				name,
				f.truncateString(p.User, 8),
				f.formatBytes(p.DiskReadRate),
				f.formatBytes(p.DiskWriteRate),
				f.formatBytes(p.NetworkRxRate),
				f.formatBytes(p.NetworkTxRate),
				p.CPUPercent,
				p.MemoryPercent)
		}
	}

	w.Flush()
}

// FormatCollectionResult formats a collection result as text
func (f *TextFormatter) FormatCollectionResult(result dto.CollectionResultDTO, writer io.Writer) error {
	fmt.Fprintf(writer, "=== Collection at %s ===\n", result.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(writer, "Status: %s\n", result.Status)

	if result.SystemMetrics != nil {
		fmt.Fprintln(writer, "\nSystem Metrics:")
		fmt.Fprintf(writer, "  CPU Usage: %.2f%%\n", result.SystemMetrics.CPUPercent)
		fmt.Fprintf(writer, "  Memory Usage: %.2f%% (%s/%s)\n",
			result.SystemMetrics.MemoryPercent,
			f.formatBytes(float64(result.SystemMetrics.MemoryUsed)),
			f.formatBytes(float64(result.SystemMetrics.MemoryTotal)))
		fmt.Fprintf(writer, "  Load Average: %.2f %.2f %.2f\n",
			result.SystemMetrics.LoadAverage1,
			result.SystemMetrics.LoadAverage5,
			result.SystemMetrics.LoadAverage15)
	}

	fmt.Fprintf(writer, "\nMetrics Collected:\n")
	fmt.Fprintf(writer, "  Process Metrics: %d\n", len(result.ProcessMetrics))
	fmt.Fprintf(writer, "  Disk Metrics: %d\n", len(result.DiskMetrics))
	fmt.Fprintf(writer, "  Network Metrics: %d\n", len(result.NetworkMetrics))

	// Top processes by I/O
	if len(result.ProcessMetrics) > 0 {
		fmt.Fprintln(writer, "\nTop Processes by I/O:")
		f.formatTopProcesses(result.ProcessMetrics, writer, 5)
	}

	return nil
}

// formatTopProcesses formats the top N processes by total I/O
func (f *TextFormatter) formatTopProcesses(processes []dto.ProcessMetricDTO, writer io.Writer, topN int) {
	// Sort by total I/O
	sort.Slice(processes, func(i, j int) bool {
		totalI := processes[i].DiskReadRate + processes[i].DiskWriteRate +
			processes[i].NetworkRxRate + processes[i].NetworkTxRate
		totalJ := processes[j].DiskReadRate + processes[j].DiskWriteRate +
			processes[j].NetworkRxRate + processes[j].NetworkTxRate
		return totalI > totalJ
	})

	// Display top N
	limit := topN
	if len(processes) < limit {
		limit = len(processes)
	}

	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %-20s\t%10s\t%10s\t%10s\n", "PROCESS", "DISK I/O", "NET I/O", "TOTAL I/O")
	fmt.Fprintf(w, "  %-20s\t%10s\t%10s\t%10s\n",
		strings.Repeat("─", 20),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10),
		strings.Repeat("─", 10))

	for i := 0; i < limit; i++ {
		p := processes[i]
		diskIO := p.DiskReadRate + p.DiskWriteRate
		netIO := p.NetworkRxRate + p.NetworkTxRate
		totalIO := diskIO + netIO

		fmt.Fprintf(w, "  %-20s\t%10s\t%10s\t%10s\n",
			f.truncateString(p.Name, 20),
			f.formatBytes(diskIO),
			f.formatBytes(netIO),
			f.formatBytes(totalIO))
	}
	w.Flush()
}

// FormatSystemInfo formats system info as text
func (f *TextFormatter) FormatSystemInfo(info dto.SystemInfoDTO, writer io.Writer) error {
	fmt.Fprintln(writer, "=== System Information ===")
	fmt.Fprintf(writer, "Hostname: %s\n", info.Hostname)
	fmt.Fprintf(writer, "OS: %s %s\n", info.OS, info.Platform)
	fmt.Fprintf(writer, "Architecture: %s\n", info.Architecture)
	fmt.Fprintf(writer, "Kernel: %s\n", info.KernelVersion)
	fmt.Fprintf(writer, "Uptime: %s\n", f.formatDuration(info.Uptime))

	fmt.Fprintln(writer, "\n=== CPU ===")
	fmt.Fprintf(writer, "Model: %s\n", info.CPU.Model)
	fmt.Fprintf(writer, "Cores: %d physical, %d logical\n", info.CPU.PhysicalCores, info.CPU.LogicalCores)
	fmt.Fprintf(writer, "Frequency: %.2f GHz\n", info.CPU.Frequency/1000)

	fmt.Fprintln(writer, "\n=== Memory ===")
	fmt.Fprintf(writer, "Total: %s\n", f.formatBytes(float64(info.Memory.Total)))
	fmt.Fprintf(writer, "Available: %s (%.1f%%)\n",
		f.formatBytes(float64(info.Memory.Available)),
		float64(info.Memory.Available)/float64(info.Memory.Total)*100)

	if f.showGraphs {
		memoryBar := f.createMemoryBar(info.Memory.Used, info.Memory.Total)
		fmt.Fprintf(writer, "Usage: [%s] %.1f%%\n", memoryBar, info.Memory.UsedPercent)
	} else {
		fmt.Fprintf(writer, "Used: %s (%.1f%%)\n",
			f.formatBytes(float64(info.Memory.Used)),
			info.Memory.UsedPercent)
	}

	return nil
}

// Helper functions

func (f *TextFormatter) formatBytes(bytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	value := bytes

	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%.0f%s", value, units[unitIndex])
	}
	return fmt.Sprintf("%.2f%s", value, units[unitIndex])
}

func (f *TextFormatter) formatDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
}

func (f *TextFormatter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (f *TextFormatter) createProgressBar(value, max float64) string {
	barLength := 20
	filled := int(float64(barLength) * (value / max))
	if filled > barLength {
		filled = barLength
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)
	return bar
}

func (f *TextFormatter) createMemoryBar(used, total uint64) string {
	barLength := 30
	percentage := float64(used) / float64(total)
	filled := int(float64(barLength) * percentage)

	if filled > barLength {
		filled = barLength
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)
	return bar
}