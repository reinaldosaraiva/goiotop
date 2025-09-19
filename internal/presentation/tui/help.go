package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModel represents the help screen
type HelpModel struct {
	content      []string
	scrollOffset int
	width        int
	height       int
	visibleLines int
}

// NewHelpModel creates a new help model
func NewHelpModel() HelpModel {
	content := []string{
		"",
		"GoIOTop - System Monitor Help",
		"==============================",
		"",
		"NAVIGATION",
		"----------",
		"  ↑/↓, j/k     Move selection up/down",
		"  PgUp/PgDown  Move page up/down",
		"  Home/End     Go to top/bottom",
		"  Tab          Switch between views",
		"  1            Process List view",
		"  2            System Summary view",
		"  3/h          Help view (this screen)",
		"",
		"SORTING (Process List View)",
		"----------------------------",
		"  c            Sort by CPU usage",
		"  m            Sort by Memory usage",
		"  i            Sort by I/O (disk + network)",
		"  p            Sort by Process ID (PID)",
		"  n            Sort by Process Name",
		"",
		"FILTERING (Process List View)",
		"------------------------------",
		"  f            Open filter dialog",
		"  /            Search for process",
		"  ESC          Clear filter/search",
		"",
		"DISPLAY OPTIONS",
		"---------------",
		"  r            Refresh display immediately",
		"  t            Toggle between actual/accumulated values",
		"  k            Toggle kernel threads visibility",
		"  o            Toggle only active processes",
		"",
		"PROCESS ACTIONS (Process List View)",
		"------------------------------------",
		"  Enter        Show process details",
		"  d            Show process disk I/O details",
		"  l            Show process network connections",
		"  s            Send signal to process",
		"  x            Kill process (SIGTERM)",
		"  X            Force kill process (SIGKILL)",
		"",
		"VIEW OPTIONS",
		"------------",
		"  F1           Show CPU cores view",
		"  F2           Show Memory details",
		"  F3           Show Disk I/O details",
		"  F4           Show Network details",
		"",
		"GENERAL",
		"-------",
		"  q            Quit GoIOTop",
		"  h/?          Show this help screen",
		"  v            Show version information",
		"",
		"COLUMN DESCRIPTIONS",
		"-------------------",
		"  PID          Process ID",
		"  USER         Process owner",
		"  PROCESS      Process name",
		"  CPU%         CPU usage percentage",
		"  MEM%         Memory usage percentage",
		"  DISK R       Disk read rate (MB/s)",
		"  DISK W       Disk write rate (MB/s)",
		"  NET RX       Network receive rate (MB/s)",
		"  NET TX       Network transmit rate (MB/s)",
		"  STATE        Process state (R=Running, S=Sleeping, etc.)",
		"",
		"PROCESS STATES",
		"--------------",
		"  R            Running",
		"  S            Sleeping",
		"  D            Disk sleep (uninterruptible)",
		"  Z            Zombie",
		"  T            Stopped",
		"  I            Idle",
		"",
		"TIPS",
		"----",
		"  • High CPU usage is highlighted in red",
		"  • High memory usage is highlighted in yellow",
		"  • High I/O activity is highlighted in blue",
		"  • Press 'f' to filter processes by name",
		"  • Use Tab to quickly switch between views",
		"  • The display refreshes automatically every 2 seconds",
		"",
		"CONFIGURATION",
		"-------------",
		"  Config file: ~/.goiotop/config.yaml",
		"  Log file:    ~/.goiotop/goiotop.log",
		"",
		"For more information, visit:",
		"https://github.com/reinaldosaraiva/goiotop",
		"",
	}

	return HelpModel{
		content:      content,
		visibleLines: 20,
	}
}

// SetSize updates the display dimensions
func (m *HelpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.visibleLines = height - 4 // Account for header and footer
}

// Init initializes the model
func (m HelpModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}

		case "down", "j":
			maxScroll := len(m.content) - m.visibleLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}

		case "pgup":
			m.scrollOffset -= m.visibleLines
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}

		case "pgdown":
			maxScroll := len(m.content) - m.visibleLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.scrollOffset += m.visibleLines
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}

		case "home":
			m.scrollOffset = 0

		case "end":
			m.scrollOffset = len(m.content) - m.visibleLines
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	}

	return m, nil
}

// View renders the help screen
func (m HelpModel) View() string {
	// Calculate visible content
	endIdx := m.scrollOffset + m.visibleLines
	if endIdx > len(m.content) {
		endIdx = len(m.content)
	}

	visibleContent := m.content[m.scrollOffset:endIdx]

	// Apply styling to different sections
	var styledLines []string
	for _, line := range visibleContent {
		styled := m.styleLine(line)
		styledLines = append(styledLines, styled)
	}

	// Add scroll indicator if needed
	scrollIndicator := ""
	if len(m.content) > m.visibleLines {
		current := m.scrollOffset + 1
		total := len(m.content) - m.visibleLines + 1
		scrollIndicator = ScrollIndicatorStyle.Render(
			fmt.Sprintf("[%d/%d]", current, total),
		)
	}

	// Combine content
	helpContent := strings.Join(styledLines, "\n")

	// Add padding to fill the view
	for len(styledLines) < m.visibleLines {
		helpContent += "\n"
	}

	// Create the final view with scroll indicator
	if scrollIndicator != "" {
		return lipgloss.JoinVertical(
			lipgloss.Top,
			helpContent,
			scrollIndicator,
		)
	}

	return helpContent
}

// styleLine applies appropriate styling to a help line
func (m HelpModel) styleLine(line string) string {
	// Headers
	if strings.Contains(line, "======") || strings.Contains(line, "------") {
		return DimStyle.Render(line)
	}

	// Section titles (all caps)
	if line != "" && line == strings.ToUpper(line) && !strings.Contains(line, " ") {
		return HelpSectionStyle.Render(line)
	}

	// Key bindings (lines starting with spaces)
	if strings.HasPrefix(line, "  ") && strings.Contains(line, "  ") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 {
			key := HelpKeyStyle.Render(parts[0])
			desc := HelpDescriptionStyle.Render(parts[1])
			return key + "  " + desc
		}
	}

	// URLs
	if strings.HasPrefix(line, "http") {
		return URLStyle.Render(line)
	}

	// Tips (bullet points)
	if strings.HasPrefix(line, "  •") {
		return HelpTipStyle.Render(line)
	}

	// Default
	return line
}