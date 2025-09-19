package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
)

// ProcessListModel represents the process list view
type ProcessListModel struct {
	processes      []dto.ProcessDisplayDTO
	filtered       []dto.ProcessDisplayDTO
	displayResult  *dto.DisplayResultDTO

	// Sorting
	sortBy         string
	sortOrder      string

	// Filtering
	filterActive   bool
	filterInput    textinput.Model
	processFilter  string
	ioThreshold    float64

	// Selection
	selectedIndex  int
	scrollOffset   int

	// Display
	width          int
	height         int
	visibleRows    int
}

// NewProcessListModel creates a new process list model
func NewProcessListModel() ProcessListModel {
	ti := textinput.New()
	ti.Placeholder = "Filter processes..."
	ti.CharLimit = 100
	ti.Width = 50

	return ProcessListModel{
		sortBy:       "cpu",
		sortOrder:    "desc",
		filterInput:  ti,
		ioThreshold:  0.0,
		visibleRows:  20,
	}
}

// SetData updates the model with new data
func (m *ProcessListModel) SetData(result *dto.DisplayResultDTO) {
	m.displayResult = result

	// Convert processes to display format
	m.processes = make([]dto.ProcessDisplayDTO, 0, len(result.Processes))
	for _, p := range result.Processes {
		displayProc := dto.FromProcessMetricsDTO(&p)
		if displayProc != nil {
			m.processes = append(m.processes, *displayProc)
		}
	}

	m.applyFilters()
	m.sortProcesses()
}

// SetSize updates the display dimensions
func (m *ProcessListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.visibleRows = height - 8 // Account for header, footer, etc.
}

// Init initializes the model
func (m ProcessListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ProcessListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "enter":
				m.processFilter = m.filterInput.Value()
				m.applyFilters()
				m.filterActive = false
				m.filterInput.Blur()
				return m, nil

			case "esc":
				m.filterActive = false
				m.filterInput.Blur()
				m.filterInput.SetValue(m.processFilter)
				return m, nil

			default:
				m.filterInput, cmd = m.filterInput.Update(msg)
				return m, cmd
			}
		}

		// Normal key handling
		switch msg.String() {
		case "c":
			m.sortBy = "cpu"
			m.sortProcesses()

		case "m":
			m.sortBy = "memory"
			m.sortProcesses()

		case "i":
			m.sortBy = "io"
			m.sortProcesses()

		case "p":
			m.sortBy = "pid"
			m.sortProcesses()

		case "n":
			m.sortBy = "name"
			m.sortProcesses()

		case "f":
			m.filterActive = true
			m.filterInput.Focus()
			m.filterInput.SetValue(m.processFilter)
			return m, m.filterInput.Focus()

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.selectedIndex < m.scrollOffset {
					m.scrollOffset = m.selectedIndex
				}
			}

		case "down", "j":
			if m.selectedIndex < len(m.filtered)-1 {
				m.selectedIndex++
				if m.selectedIndex >= m.scrollOffset+m.visibleRows {
					m.scrollOffset = m.selectedIndex - m.visibleRows + 1
				}
			}

		case "pgup":
			m.selectedIndex -= m.visibleRows
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
			m.scrollOffset = m.selectedIndex

		case "pgdown":
			m.selectedIndex += m.visibleRows
			if m.selectedIndex >= len(m.filtered) {
				m.selectedIndex = len(m.filtered) - 1
			}
			if m.selectedIndex >= m.scrollOffset+m.visibleRows {
				m.scrollOffset = m.selectedIndex - m.visibleRows + 1
			}

		case "home":
			m.selectedIndex = 0
			m.scrollOffset = 0

		case "end":
			m.selectedIndex = len(m.filtered) - 1
			if m.selectedIndex >= m.visibleRows {
				m.scrollOffset = m.selectedIndex - m.visibleRows + 1
			}
		}
	}

	return m, cmd
}

// View renders the process list
func (m ProcessListModel) View() string {
	if m.filterActive {
		return m.renderFilterInput()
	}

	// Build header
	header := m.renderHeader()

	// Build process rows
	var rows []string
	endIdx := m.scrollOffset + m.visibleRows
	if endIdx > len(m.filtered) {
		endIdx = len(m.filtered)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		proc := m.filtered[i]
		row := m.renderProcessRow(proc, i == m.selectedIndex)
		rows = append(rows, row)
	}

	// Add empty rows if needed
	for len(rows) < m.visibleRows {
		rows = append(rows, "")
	}

	// Combine header and rows
	content := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		strings.Join(rows, "\n"),
	)

	// Add summary line
	summary := m.renderSummary()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		content,
		summary,
	)
}

// renderHeader renders the table header
func (m ProcessListModel) renderHeader() string {
	headers := []string{
		fmt.Sprintf("%-8s", "PID"),
		fmt.Sprintf("%-15s", "USER"),
		fmt.Sprintf("%-20s", "PROCESS"),
		fmt.Sprintf("%7s", "CPU%"),
		fmt.Sprintf("%7s", "MEM%"),
		fmt.Sprintf("%10s", "DISK R"),
		fmt.Sprintf("%10s", "DISK W"),
		fmt.Sprintf("%10s", "NET RX"),
		fmt.Sprintf("%10s", "NET TX"),
		fmt.Sprintf("%-8s", "STATE"),
	}

	// Highlight sorted column
	switch m.sortBy {
	case "pid":
		headers[0] = HeaderHighlightStyle.Render(headers[0])
	case "name":
		headers[2] = HeaderHighlightStyle.Render(headers[2])
	case "cpu":
		headers[3] = HeaderHighlightStyle.Render(headers[3])
	case "memory":
		headers[4] = HeaderHighlightStyle.Render(headers[4])
	case "io":
		headers[5] = HeaderHighlightStyle.Render(headers[5])
		headers[6] = HeaderHighlightStyle.Render(headers[6])
	}

	return TableHeaderStyle.Render(strings.Join(headers, " "))
}

// renderProcessRow renders a single process row
func (m ProcessListModel) renderProcessRow(proc dto.ProcessDisplayDTO, selected bool) string {
	row := fmt.Sprintf(
		"%-8d %-15s %-20s %7.1f %7.1f %10.2f %10.2f %10.2f %10.2f %-8s",
		proc.PID,
		truncate(proc.User, 15),
		truncate(proc.Name, 20),
		proc.CPUPercent,
		proc.MemoryPercent,
		proc.DiskReadRate,
		proc.DiskWriteRate,
		proc.NetRXRate,
		proc.NetTXRate,
		proc.State,
	)

	if selected {
		return SelectedRowStyle.Render(row)
	}

	// Color code by resource usage
	if proc.CPUPercent > 80 {
		return HighCPUStyle.Render(row)
	} else if proc.MemoryPercent > 80 {
		return HighMemoryStyle.Render(row)
	} else if proc.DiskReadRate+proc.DiskWriteRate > 100 {
		return HighIOStyle.Render(row)
	}

	return row
}

// renderSummary renders the summary line
func (m ProcessListModel) renderSummary() string {
	total := len(m.processes)
	filtered := len(m.filtered)

	summary := fmt.Sprintf(
		"Showing %d of %d processes | Sort: %s | Filter: %s",
		filtered,
		total,
		m.sortBy,
		m.processFilter,
	)

	if m.processFilter != "" {
		summary += fmt.Sprintf(" (filtered)")
	}

	return SummaryStyle.Render(summary)
}

// renderFilterInput renders the filter input dialog
func (m ProcessListModel) renderFilterInput() string {
	return FilterDialogStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			"Enter process filter:",
			m.filterInput.View(),
			"",
			"Press Enter to apply, ESC to cancel",
		),
	)
}

// applyFilters applies current filters to the process list
func (m *ProcessListModel) applyFilters() {
	m.filtered = make([]dto.ProcessDisplayDTO, 0, len(m.processes))

	for _, proc := range m.processes {
		// Process name filter
		if m.processFilter != "" {
			if !strings.Contains(strings.ToLower(proc.Name), strings.ToLower(m.processFilter)) &&
			   !strings.Contains(strings.ToLower(proc.Command), strings.ToLower(m.processFilter)) {
				continue
			}
		}

		// I/O threshold filter
		if m.ioThreshold > 0 {
			totalIO := proc.DiskReadRate + proc.DiskWriteRate + proc.NetRXRate + proc.NetTXRate
			if totalIO < m.ioThreshold {
				continue
			}
		}

		m.filtered = append(m.filtered, proc)
	}

	// Reset selection if needed
	if m.selectedIndex >= len(m.filtered) {
		m.selectedIndex = 0
		m.scrollOffset = 0
	}
}

// sortProcesses sorts the filtered process list
func (m *ProcessListModel) sortProcesses() {
	ascending := m.sortOrder == "asc"

	sort.Slice(m.filtered, func(i, j int) bool {
		switch m.sortBy {
		case "pid":
			if ascending {
				return m.filtered[i].PID < m.filtered[j].PID
			}
			return m.filtered[i].PID > m.filtered[j].PID

		case "name":
			if ascending {
				return m.filtered[i].Name < m.filtered[j].Name
			}
			return m.filtered[i].Name > m.filtered[j].Name

		case "cpu":
			if ascending {
				return m.filtered[i].CPUPercent < m.filtered[j].CPUPercent
			}
			return m.filtered[i].CPUPercent > m.filtered[j].CPUPercent

		case "memory":
			if ascending {
				return m.filtered[i].MemoryPercent < m.filtered[j].MemoryPercent
			}
			return m.filtered[i].MemoryPercent > m.filtered[j].MemoryPercent

		case "io":
			totalI := m.filtered[i].DiskReadRate + m.filtered[i].DiskWriteRate +
				m.filtered[i].NetRXRate + m.filtered[i].NetTXRate
			totalJ := m.filtered[j].DiskReadRate + m.filtered[j].DiskWriteRate +
				m.filtered[j].NetRXRate + m.filtered[j].NetTXRate
			if ascending {
				return totalI < totalJ
			}
			return totalI > totalJ

		default:
			return false
		}
	})
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}