package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ProgressBar represents a progress bar for CLI output
type ProgressBar struct {
	total    int
	current  int
	width    int
	prefix   string
	writer   io.Writer
	lastDraw time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, prefix string, writer io.Writer) *ProgressBar {
	width := 40 // Default width
	if IsTerminal() {
		if termWidth, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			// Use 50% of terminal width for progress bar
			width = termWidth / 2
			if width > 60 {
				width = 60
			}
			if width < 20 {
				width = 20
			}
		}
	}

	return &ProgressBar{
		total:  total,
		prefix: prefix,
		width:  width,
		writer: writer,
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int) {
	p.current = current
	p.Draw()
}

// Increment increments the progress bar by one
func (p *ProgressBar) Increment() {
	p.current++
	p.Draw()
}

// Draw draws the progress bar
func (p *ProgressBar) Draw() {
	// Rate limit drawing to avoid flickering
	if time.Since(p.lastDraw) < 100*time.Millisecond {
		return
	}
	p.lastDraw = time.Now()

	percentage := float64(p.current) / float64(p.total)
	filled := int(float64(p.width) * percentage)
	if filled > p.width {
		filled = p.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	fmt.Fprintf(p.writer, "\r%s [%s] %d/%d (%.1f%%)",
		p.prefix, bar, p.current, p.total, percentage*100)

	if p.current >= p.total {
		fmt.Fprintln(p.writer) // New line when complete
	}
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.current = p.total
	p.Draw()
}

// Spinner represents a loading spinner for CLI output
type Spinner struct {
	frames  []string
	current int
	prefix  string
	writer  io.Writer
	active  bool
	stopCh  chan bool
}

// NewSpinner creates a new spinner
func NewSpinner(prefix string, writer io.Writer) *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		prefix: prefix,
		writer: writer,
		stopCh: make(chan bool),
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	if s.active {
		return
	}
	s.active = true

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				fmt.Fprintf(s.writer, "\r%s%s\n", s.prefix, strings.Repeat(" ", 10))
				return
			case <-ticker.C:
				s.current = (s.current + 1) % len(s.frames)
				fmt.Fprintf(s.writer, "\r%s %s", s.frames[s.current], s.prefix)
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	if s.active {
		s.active = false
		s.stopCh <- true
	}
}

// IsTerminal checks if the output is a terminal
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// IsBatchMode checks if running in batch mode (non-interactive)
func IsBatchMode() bool {
	// Check if output is not a terminal
	if !IsTerminal() {
		return true
	}

	// Check for CI environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "JENKINS_URL", "GITHUB_ACTIONS"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}

// SupportsColor checks if the terminal supports color output
func SupportsColor() bool {
	// Check if explicitly disabled
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if not a terminal
	if !IsTerminal() {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Most modern terminals support color
	return true
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	if IsTerminal() {
		fmt.Print("\033[H\033[2J")
	}
}

// MoveCursor moves the cursor to a specific position
func MoveCursor(x, y int) {
	if IsTerminal() {
		fmt.Printf("\033[%d;%dH", y, x)
	}
}

// HideCursor hides the terminal cursor
func HideCursor() {
	if IsTerminal() {
		fmt.Print("\033[?25l")
	}
}

// ShowCursor shows the terminal cursor
func ShowCursor() {
	if IsTerminal() {
		fmt.Print("\033[?25h")
	}
}

// GetTerminalSize returns the terminal width and height
func GetTerminalSize() (width, height int, err error) {
	if !IsTerminal() {
		return 80, 24, nil // Default size
	}
	return term.GetSize(int(os.Stdout.Fd()))
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadString pads a string to a specific length
func PadString(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(string(padChar), length-len(s))
	return s + padding
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// FormatBytes formats bytes in a human-readable way
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ConfirmPrompt asks the user for confirmation
func ConfirmPrompt(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// SelectPrompt presents a selection menu to the user
func SelectPrompt(prompt string, options []string) (int, error) {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Print("Select option: ")

	var selection int
	_, err := fmt.Scanln(&selection)
	if err != nil {
		return -1, err
	}

	if selection < 1 || selection > len(options) {
		return -1, fmt.Errorf("invalid selection: %d", selection)
	}

	return selection - 1, nil
}

// InputPrompt prompts the user for input
func InputPrompt(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	var input string
	fmt.Scanln(&input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}
	return input
}

// PrintError prints an error message in red if color is supported
func PrintError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if SupportsColor() {
		fmt.Fprintf(os.Stderr, "\033[31mError: %s\033[0m\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
}

// PrintWarning prints a warning message in yellow if color is supported
func PrintWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if SupportsColor() {
		fmt.Printf("\033[33mWarning: %s\033[0m\n", message)
	} else {
		fmt.Printf("Warning: %s\n", message)
	}
}

// PrintSuccess prints a success message in green if color is supported
func PrintSuccess(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if SupportsColor() {
		fmt.Printf("\033[32m%s\033[0m\n", message)
	} else {
		fmt.Println(message)
	}
}

// PrintInfo prints an info message in blue if color is supported
func PrintInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if SupportsColor() {
		fmt.Printf("\033[34m%s\033[0m\n", message)
	} else {
		fmt.Println(message)
	}
}