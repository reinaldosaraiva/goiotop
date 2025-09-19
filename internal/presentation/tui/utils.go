package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
	"github.com/charmbracelet/lipgloss"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// GetTerminalSize returns the current terminal dimensions
func GetTerminalSize() (width, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Return default size if we can't get terminal size
		return 80, 24
	}
	return width, height
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// HasColorSupport checks if the terminal supports colors
func HasColorSupport() bool {
	// Check common environment variables
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}

	// Basic color support
	return strings.Contains(term, "color") ||
	       strings.Contains(term, "256") ||
	       strings.Contains(term, "xterm")
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatRate formats rate in bytes/sec to human-readable format
func FormatRate(bytesPerSec float64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytesPerSec >= GB:
		return fmt.Sprintf("%.2f GB/s", bytesPerSec/GB)
	case bytesPerSec >= MB:
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/MB)
	case bytesPerSec >= KB:
		return fmt.Sprintf("%.2f KB/s", bytesPerSec/KB)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

// FormatPercent formats a percentage value
func FormatPercent(value float64) string {
	if value < 10 {
		return fmt.Sprintf("%.2f%%", value)
	} else if value < 100 {
		return fmt.Sprintf("%.1f%%", value)
	}
	return fmt.Sprintf("%.0f%%", value)
}

// CreateTable creates a formatted table from data
func CreateTable(headers []string, rows [][]string, widths []int) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Ensure we have widths for all columns
	if len(widths) != len(headers) {
		widths = make([]int, len(headers))
		for i := range widths {
			widths[i] = 15 // Default width
		}
	}

	// Create header row
	headerRow := ""
	for i, header := range headers {
		headerRow += padString(header, widths[i]) + " "
	}
	headerRow = strings.TrimSpace(headerRow)

	// Create separator
	separator := strings.Repeat("-", len(headerRow))

	// Create data rows
	var dataRows []string
	for _, row := range rows {
		dataRow := ""
		for i, cell := range row {
			if i < len(widths) {
				dataRow += padString(cell, widths[i]) + " "
			}
		}
		dataRows = append(dataRows, strings.TrimSpace(dataRow))
	}

	// Combine all parts
	result := []string{headerRow, separator}
	result = append(result, dataRows...)

	return strings.Join(result, "\n")
}

// padString pads or truncates a string to the specified width
func padString(s string, width int) string {
	if len(s) >= width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// SortProcesses sorts processes based on the specified criteria
func SortProcesses(processes []dto.ProcessDisplayDTO, sortBy string, ascending bool) []dto.ProcessDisplayDTO {
	sorted := make([]dto.ProcessDisplayDTO, len(processes))
	copy(sorted, processes)

	sort.Slice(sorted, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "cpu":
			less = sorted[i].CPUPercent < sorted[j].CPUPercent
		case "memory":
			less = sorted[i].MemoryPercent < sorted[j].MemoryPercent
		case "io":
			totalI := sorted[i].DiskReadRate + sorted[i].DiskWriteRate +
				sorted[i].NetRXRate + sorted[i].NetTXRate
			totalJ := sorted[j].DiskReadRate + sorted[j].DiskWriteRate +
				sorted[j].NetRXRate + sorted[j].NetTXRate
			less = totalI < totalJ
		case "pid":
			less = sorted[i].PID < sorted[j].PID
		case "name":
			less = sorted[i].Name < sorted[j].Name
		case "user":
			less = sorted[i].User < sorted[j].User
		case "disk_read":
			less = sorted[i].DiskReadRate < sorted[j].DiskReadRate
		case "disk_write":
			less = sorted[i].DiskWriteRate < sorted[j].DiskWriteRate
		case "net_rx":
			less = sorted[i].NetRXRate < sorted[j].NetRXRate
		case "net_tx":
			less = sorted[i].NetTXRate < sorted[j].NetTXRate
		default:
			less = sorted[i].CPUPercent < sorted[j].CPUPercent
		}

		if ascending {
			return less
		}
		return !less
	})

	return sorted
}

// FilterProcesses filters processes based on criteria
func FilterProcesses(processes []dto.ProcessDisplayDTO, filter string, cpuThreshold, memThreshold, ioThreshold float64) []dto.ProcessDisplayDTO {
	if filter == "" && cpuThreshold == 0 && memThreshold == 0 && ioThreshold == 0 {
		return processes
	}

	var filtered []dto.ProcessDisplayDTO
	filterLower := strings.ToLower(filter)

	for _, proc := range processes {
		// Name filter
		if filter != "" {
			nameLower := strings.ToLower(proc.Name)
			cmdLower := strings.ToLower(proc.Command)
			userLower := strings.ToLower(proc.User)

			if !strings.Contains(nameLower, filterLower) &&
			   !strings.Contains(cmdLower, filterLower) &&
			   !strings.Contains(userLower, filterLower) {
				continue
			}
		}

		// CPU threshold
		if cpuThreshold > 0 && proc.CPUPercent < cpuThreshold {
			continue
		}

		// Memory threshold
		if memThreshold > 0 && proc.MemoryPercent < memThreshold {
			continue
		}

		// I/O threshold
		if ioThreshold > 0 {
			totalIO := proc.DiskReadRate + proc.DiskWriteRate + proc.NetRXRate + proc.NetTXRate
			if totalIO < ioThreshold {
				continue
			}
		}

		filtered = append(filtered, proc)
	}

	return filtered
}

// CalculateLayout calculates optimal layout for the given terminal size
type Layout struct {
	HeaderHeight  int
	FooterHeight  int
	ContentHeight int
	ContentWidth  int
	SidebarWidth  int
}

// CalculateLayout determines the optimal layout for the terminal
func CalculateLayout(width, height int) Layout {
	layout := Layout{
		HeaderHeight:  2,
		FooterHeight:  2,
		ContentHeight: height - 4, // Header + Footer
		ContentWidth:  width,
		SidebarWidth:  0,
	}

	// Adjust for small terminals
	if height < 20 {
		layout.HeaderHeight = 1
		layout.FooterHeight = 1
		layout.ContentHeight = height - 2
	}

	// Add sidebar for wide terminals
	if width > 120 {
		layout.SidebarWidth = 30
		layout.ContentWidth = width - layout.SidebarWidth - 1 // -1 for border
	}

	return layout
}

// RenderProgressBar creates a visual progress bar
func RenderProgressBar(percent float64, width int, filled, empty string) string {
	if width <= 0 {
		return ""
	}

	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	filledCount := int(float64(width) * percent / 100)
	emptyCount := width - filledCount

	if filled == "" {
		filled = "█"
	}
	if empty == "" {
		empty = "░"
	}

	bar := strings.Repeat(filled, filledCount) + strings.Repeat(empty, emptyCount)

	// Apply color based on percentage
	if percent > 90 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#DC143C")).Render(bar)
	} else if percent > 70 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render(bar)
	} else if percent > 50 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Render(bar)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32")).Render(bar)
}

// WrapText wraps text to fit within the specified width
func WrapText(text string, width int) []string {
	if width <= 0 || text == "" {
		return []string{}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}