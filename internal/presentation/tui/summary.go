package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// SummaryModel represents the system summary view
type SummaryModel struct {
	displayResult   *dto.DisplayResultDTO
	systemSummary   *dto.DisplaySummaryDTO

	width          int
	height         int
}

// NewSummaryModel creates a new summary model
func NewSummaryModel() SummaryModel {
	return SummaryModel{}
}

// SetData updates the model with new data
func (m *SummaryModel) SetData(result *dto.DisplayResultDTO) {
	m.displayResult = result
	if result != nil && result.SystemSummary != nil {
		// Convert to DisplaySummaryDTO if needed
		if sm := result.SystemSummary; sm != nil {
			m.systemSummary = &dto.DisplaySummaryDTO{
				TotalDiskRead:    sm.NetworkRX,    // Map as needed
				TotalDiskWrite:   sm.NetworkTX,
				TotalNetRX:       sm.NetworkRX,
				TotalNetTX:       sm.NetworkTX,
				AvgCPUPercent:    sm.CPUUsage,
				AvgMemoryPercent: sm.MemoryUsage,
				ActiveProcesses:  sm.ProcessCount,
				TotalProcesses:   sm.ProcessCount,
				LoadAvg1:         sm.LoadAverage.Load1,
				LoadAvg5:         sm.LoadAverage.Load5,
				LoadAvg15:        sm.LoadAverage.Load15,
				Uptime:           sm.Uptime,
			}
		}
	}
}

// SetSize updates the display dimensions
func (m *SummaryModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model
func (m SummaryModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m SummaryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Summary view doesn't handle specific keys
	return m, nil
}

// View renders the system summary
func (m SummaryModel) View() string {
	if m.systemSummary == nil {
		return CenterStyle.Render("Loading system summary...")
	}

	sections := []string{
		m.renderCPUSection(),
		m.renderMemorySection(),
		m.renderIOSection(),
		m.renderNetworkSection(),
		m.renderLoadSection(),
		m.renderProcessSection(),
	}

	// Arrange sections in a grid
	grid := m.arrangeGrid(sections)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 4).
		Render(grid)
}

// renderCPUSection renders CPU information
func (m SummaryModel) renderCPUSection() string {
	title := SectionTitleStyle.Render("CPU Usage")

	cpuBar := m.renderProgressBar(m.systemSummary.AvgCPUPercent, 30)
	cpuText := fmt.Sprintf("%.1f%%", m.systemSummary.AvgCPUPercent)

	ioWaitBar := m.renderProgressBar(m.systemSummary.IOWaitPercent, 30)
	ioWaitText := fmt.Sprintf("I/O Wait: %.1f%%", m.systemSummary.IOWaitPercent)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fmt.Sprintf("Usage: %s %s", cpuBar, cpuText),
		fmt.Sprintf("%s %s", ioWaitBar, ioWaitText),
	)

	return SectionStyle.Render(content)
}

// renderMemorySection renders memory information
func (m SummaryModel) renderMemorySection() string {
	title := SectionTitleStyle.Render("Memory Usage")

	memBar := m.renderProgressBar(m.systemSummary.AvgMemoryPercent, 30)
	memText := fmt.Sprintf("%.1f%%", m.systemSummary.AvgMemoryPercent)

	swapBar := m.renderProgressBar(m.systemSummary.SwapPercent, 30)
	swapText := fmt.Sprintf("Swap: %.1f%%", m.systemSummary.SwapPercent)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fmt.Sprintf("Memory: %s %s", memBar, memText),
		fmt.Sprintf("%s %s", swapBar, swapText),
	)

	return SectionStyle.Render(content)
}

// renderIOSection renders I/O information
func (m SummaryModel) renderIOSection() string {
	title := SectionTitleStyle.Render("Disk I/O")

	readRate := fmt.Sprintf("Read:  %10.2f MB/s", m.systemSummary.TotalDiskRead)
	writeRate := fmt.Sprintf("Write: %10.2f MB/s", m.systemSummary.TotalDiskWrite)
	totalRate := fmt.Sprintf("Total: %10.2f MB/s",
		m.systemSummary.TotalDiskRead + m.systemSummary.TotalDiskWrite)

	// Create a simple graph representation
	readBar := m.renderRateBar(m.systemSummary.TotalDiskRead, 100.0, 20)
	writeBar := m.renderRateBar(m.systemSummary.TotalDiskWrite, 100.0, 20)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fmt.Sprintf("%s %s", readRate, readBar),
		fmt.Sprintf("%s %s", writeRate, writeBar),
		totalRate,
	)

	return SectionStyle.Render(content)
}

// renderNetworkSection renders network information
func (m SummaryModel) renderNetworkSection() string {
	title := SectionTitleStyle.Render("Network I/O")

	rxRate := fmt.Sprintf("RX: %10.2f MB/s", m.systemSummary.TotalNetRX)
	txRate := fmt.Sprintf("TX: %10.2f MB/s", m.systemSummary.TotalNetTX)
	totalRate := fmt.Sprintf("Total: %10.2f MB/s",
		m.systemSummary.TotalNetRX + m.systemSummary.TotalNetTX)

	// Create a simple graph representation
	rxBar := m.renderRateBar(m.systemSummary.TotalNetRX, 100.0, 20)
	txBar := m.renderRateBar(m.systemSummary.TotalNetTX, 100.0, 20)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		fmt.Sprintf("%s %s", rxRate, rxBar),
		fmt.Sprintf("%s %s", txRate, txBar),
		totalRate,
	)

	return SectionStyle.Render(content)
}

// renderLoadSection renders load average information
func (m SummaryModel) renderLoadSection() string {
	title := SectionTitleStyle.Render("System Load")

	load1 := fmt.Sprintf("1 min:  %.2f", m.systemSummary.LoadAvg1)
	load5 := fmt.Sprintf("5 min:  %.2f", m.systemSummary.LoadAvg5)
	load15 := fmt.Sprintf("15 min: %.2f", m.systemSummary.LoadAvg15)

	uptime := fmt.Sprintf("Uptime: %s", m.formatDuration(m.systemSummary.Uptime))

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		load1,
		load5,
		load15,
		"",
		uptime,
	)

	return SectionStyle.Render(content)
}

// renderProcessSection renders process summary information
func (m SummaryModel) renderProcessSection() string {
	title := SectionTitleStyle.Render("Process Summary")

	active := fmt.Sprintf("Active:  %d", m.systemSummary.ActiveProcesses)
	total := fmt.Sprintf("Total:   %d", m.systemSummary.TotalProcesses)

	// Top processes if available
	var topProcs []string
	if m.displayResult != nil && m.displayResult.SystemSummary != nil {
		for i, proc := range m.displayResult.SystemSummary.TopProcesses {
			if i >= 5 {
				break
			}
			topProcs = append(topProcs, fmt.Sprintf(
				"  %s (%.1f%% CPU)",
				truncate(proc.Name, 15),
				proc.CPUPercent,
			))
		}
	}

	content := []string{title, "", active, total}
	if len(topProcs) > 0 {
		content = append(content, "", "Top Processes:")
		content = append(content, topProcs...)
	}

	return SectionStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

// renderProgressBar renders a progress bar
func (m SummaryModel) renderProgressBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(float64(width) * percent / 100)
	empty := width - filled

	bar := ProgressBarFullStyle.Render(strings.Repeat("█", filled)) +
		ProgressBarEmptyStyle.Render(strings.Repeat("░", empty))

	// Color based on value
	if percent > 90 {
		return CriticalStyle.Render(bar)
	} else if percent > 70 {
		return WarningStyle.Render(bar)
	}
	return NormalStyle.Render(bar)
}

// renderRateBar renders a rate bar for I/O
func (m SummaryModel) renderRateBar(rate, maxRate float64, width int) string {
	if maxRate <= 0 {
		maxRate = 100
	}
	percent := (rate / maxRate) * 100
	return m.renderProgressBar(percent, width)
}

// arrangeGrid arranges sections in a grid layout
func (m SummaryModel) arrangeGrid(sections []string) string {
	if len(sections) == 0 {
		return ""
	}

	// Arrange in 2 columns
	var rows []string
	for i := 0; i < len(sections); i += 2 {
		if i+1 < len(sections) {
			row := lipgloss.JoinHorizontal(
				lipgloss.Top,
				sections[i],
				"  ",
				sections[i+1],
			)
			rows = append(rows, row)
		} else {
			rows = append(rows, sections[i])
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}

// formatDuration formats a duration in a human-readable way
func (m SummaryModel) formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}