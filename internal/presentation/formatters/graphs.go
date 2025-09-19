package formatters

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
)

// GraphOptions defines options for graph rendering
type GraphOptions struct {
	Width      int
	Height     int
	ShowAxes   bool
	ShowLegend bool
	Title      string
	XLabel     string
	YLabel     string
	MinY       float64
	MaxY       float64
	AutoScale  bool
}

// DefaultGraphOptions returns default graph options
func DefaultGraphOptions() GraphOptions {
	return GraphOptions{
		Width:      80,
		Height:     20,
		ShowAxes:   true,
		ShowLegend: true,
		AutoScale:  true,
	}
}

// GraphRenderer renders various types of ASCII graphs
type GraphRenderer struct {
	options GraphOptions
}

// NewGraphRenderer creates a new graph renderer
func NewGraphRenderer(options GraphOptions) *GraphRenderer {
	return &GraphRenderer{
		options: options,
	}
}

// RenderLineGraph renders a line graph for time-series data
func (r *GraphRenderer) RenderLineGraph(values []float64, timestamps []time.Time) string {
	if len(values) == 0 {
		return "No data available"
	}

	// Ensure we have valid dimensions
	if r.options.Width < 20 {
		r.options.Width = 20
	}
	if r.options.Height < 5 {
		r.options.Height = 5
	}

	// Calculate min/max if auto-scaling
	minY, maxY := r.options.MinY, r.options.MaxY
	if r.options.AutoScale {
		minY, maxY = findMinMax(values)
		// Add 10% padding
		padding := (maxY - minY) * 0.1
		minY -= padding
		maxY += padding
	}

	// Create the graph canvas
	graphWidth := r.options.Width - 10 // Leave space for Y-axis labels
	graphHeight := r.options.Height - 3 // Leave space for X-axis and title

	canvas := make([][]rune, graphHeight)
	for i := range canvas {
		canvas[i] = make([]rune, graphWidth)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// Sample or interpolate data points to fit the width
	sampledValues := sampleData(values, graphWidth)

	// Plot the line
	for i := 0; i < len(sampledValues)-1; i++ {
		y1 := scaleValue(sampledValues[i], minY, maxY, graphHeight)
		y2 := scaleValue(sampledValues[i+1], minY, maxY, graphHeight)

		// Draw line between points
		drawLine(canvas, i, y1, i+1, y2)
	}

	// Build the output string
	var result strings.Builder

	// Title
	if r.options.Title != "" {
		result.WriteString(centerText(r.options.Title, r.options.Width))
		result.WriteString("\n")
	}

	// Y-axis labels and graph
	yStep := (maxY - minY) / float64(graphHeight-1)
	for i := 0; i < graphHeight; i++ {
		// Y-axis label
		yValue := maxY - float64(i)*yStep
		result.WriteString(fmt.Sprintf("%8.2f│", yValue))

		// Graph line
		for j := 0; j < graphWidth; j++ {
			result.WriteRune(canvas[i][j])
		}
		result.WriteString("\n")
	}

	// X-axis
	result.WriteString("        └")
	result.WriteString(strings.Repeat("─", graphWidth))
	result.WriteString("\n")

	// X-axis labels (time)
	if len(timestamps) > 0 {
		result.WriteString("         ")
		startTime := timestamps[0]
		endTime := timestamps[len(timestamps)-1]
		duration := endTime.Sub(startTime)
		result.WriteString(fmt.Sprintf("%-*s", graphWidth/2, formatDuration(startTime)))
		result.WriteString(fmt.Sprintf("%*s", graphWidth/2, formatDuration(endTime)))
		result.WriteString(fmt.Sprintf(" (%s)", duration.Round(time.Second)))
		result.WriteString("\n")
	}

	return result.String()
}

// RenderBarChart renders a bar chart
func (r *GraphRenderer) RenderBarChart(labels []string, values []float64) string {
	if len(values) == 0 {
		return "No data available"
	}

	// Find max value for scaling
	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	if maxVal == 0 {
		maxVal = 1
	}

	var result strings.Builder

	// Title
	if r.options.Title != "" {
		result.WriteString(centerText(r.options.Title, r.options.Width))
		result.WriteString("\n\n")
	}

	// Calculate bar width
	barWidth := (r.options.Width - 20) / len(values)
	if barWidth < 1 {
		barWidth = 1
	}
	if barWidth > 8 {
		barWidth = 8
	}

	// Render bars
	for i := r.options.Height; i > 0; i-- {
		threshold := float64(i) * maxVal / float64(r.options.Height)
		result.WriteString(fmt.Sprintf("%8.2f│", threshold))

		for j, val := range values {
			if val >= threshold {
				result.WriteString(strings.Repeat("█", barWidth))
			} else {
				result.WriteString(strings.Repeat(" ", barWidth))
			}
			result.WriteString(" ")
		}
		result.WriteString("\n")
	}

	// X-axis
	result.WriteString("        └")
	result.WriteString(strings.Repeat("─", len(values)*(barWidth+1)))
	result.WriteString("\n")

	// Labels
	if len(labels) > 0 {
		result.WriteString("         ")
		for i, label := range labels {
			if i < len(labels) {
				truncated := truncateString(label, barWidth)
				result.WriteString(fmt.Sprintf("%-*s ", barWidth, truncated))
			}
		}
		result.WriteString("\n")
	}

	// Values
	result.WriteString("         ")
	for _, val := range values {
		result.WriteString(fmt.Sprintf("%-*.1f ", barWidth, val))
	}
	result.WriteString("\n")

	return result.String()
}

// RenderSparkline renders a compact sparkline graph
func (r *GraphRenderer) RenderSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	minVal, maxVal := findMinMax(values)
	if maxVal == minVal {
		maxVal = minVal + 1
	}

	var result strings.Builder
	for _, v := range values {
		normalized := (v - minVal) / (maxVal - minVal)
		idx := int(normalized * float64(len(sparkChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		result.WriteRune(sparkChars[idx])
	}

	return result.String()
}

// RenderHeatmap renders a heatmap for 2D data
func (r *GraphRenderer) RenderHeatmap(data [][]float64, rowLabels, colLabels []string) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Find min/max for color scaling
	minVal, maxVal := data[0][0], data[0][0]
	for _, row := range data {
		for _, val := range row {
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	// Heat characters from cold to hot
	heatChars := []rune{' ', '·', '▪', '▫', '▬', '▭', '▮', '▯', '▰', '█'}

	var result strings.Builder

	// Title
	if r.options.Title != "" {
		result.WriteString(centerText(r.options.Title, r.options.Width))
		result.WriteString("\n\n")
	}

	// Column labels
	if len(colLabels) > 0 {
		result.WriteString("        ")
		for _, label := range colLabels {
			result.WriteString(fmt.Sprintf("%-6s", truncateString(label, 6)))
		}
		result.WriteString("\n")
	}

	// Data rows
	for i, row := range data {
		// Row label
		if i < len(rowLabels) {
			result.WriteString(fmt.Sprintf("%-8s", truncateString(rowLabels[i], 8)))
		} else {
			result.WriteString(fmt.Sprintf("%-8d", i))
		}

		// Heat values
		for _, val := range row {
			normalized := (val - minVal) / (maxVal - minVal)
			idx := int(normalized * float64(len(heatChars)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(heatChars) {
				idx = len(heatChars) - 1
			}
			result.WriteString(fmt.Sprintf("%c%c%c%c%c ",
				heatChars[idx], heatChars[idx], heatChars[idx],
				heatChars[idx], heatChars[idx]))
		}
		result.WriteString("\n")
	}

	// Legend
	result.WriteString("\nLegend: ")
	for i, char := range heatChars {
		val := minVal + float64(i)*(maxVal-minVal)/float64(len(heatChars)-1)
		result.WriteString(fmt.Sprintf("%c=%.1f ", char, val))
	}
	result.WriteString("\n")

	return result.String()
}

// RenderSystemMetricsTrend renders system metrics trend graph
func (r *GraphRenderer) RenderSystemMetricsTrend(metrics []*entities.SystemMetrics, metricType string) string {
	if len(metrics) == 0 {
		return "No metrics data available"
	}

	values := make([]float64, len(metrics))
	timestamps := make([]time.Time, len(metrics))

	for i, m := range metrics {
		timestamps[i] = m.Timestamp.Time()

		switch metricType {
		case "cpu":
			values[i] = m.CPUUsage.Total()
		case "memory":
			values[i] = float64(m.MemoryUsage.UsedPercent())
		case "load":
			values[i] = m.LoadAverage.Values()[0]
		case "swap":
			values[i] = float64(m.SwapUsage.UsedPercent())
		default:
			values[i] = 0
		}
	}

	r.options.Title = fmt.Sprintf("%s Usage Trend", strings.Title(metricType))
	r.options.YLabel = getMetricUnit(metricType)

	return r.RenderLineGraph(values, timestamps)
}

// RenderProcessMetricsComparison renders comparison of process metrics
func (r *GraphRenderer) RenderProcessMetricsComparison(processes []*dto.ProcessMetricsDTO, limit int) string {
	if len(processes) == 0 {
		return "No process data available"
	}

	if limit > 0 && len(processes) > limit {
		processes = processes[:limit]
	}

	labels := make([]string, len(processes))
	cpuValues := make([]float64, len(processes))
	memValues := make([]float64, len(processes))

	for i, p := range processes {
		labels[i] = p.Name
		cpuValues[i] = p.CPUPercent
		memValues[i] = p.MemoryPercent
	}

	// Render CPU chart
	r.options.Title = "Top Processes by CPU Usage"
	cpuChart := r.RenderBarChart(labels, cpuValues)

	// Render Memory chart
	r.options.Title = "Top Processes by Memory Usage"
	memChart := r.RenderBarChart(labels, memValues)

	return cpuChart + "\n" + memChart
}

// Helper functions

func findMinMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func scaleValue(value, min, max float64, height int) int {
	if max == min {
		return height / 2
	}
	scaled := (value - min) / (max - min)
	return height - 1 - int(scaled*float64(height-1))
}

func sampleData(values []float64, targetSize int) []float64 {
	if len(values) <= targetSize {
		return values
	}

	sampled := make([]float64, targetSize)
	step := float64(len(values)-1) / float64(targetSize-1)

	for i := 0; i < targetSize; i++ {
		idx := int(float64(i) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		sampled[i] = values[idx]
	}

	return sampled
}

func drawLine(canvas [][]rune, x1, y1, x2, y2 int) {
	// Simple line drawing using ASCII characters
	if x1 == x2 {
		// Vertical line
		minY, maxY := y1, y2
		if y1 > y2 {
			minY, maxY = y2, y1
		}
		for y := minY; y <= maxY; y++ {
			if y >= 0 && y < len(canvas) && x1 >= 0 && x1 < len(canvas[0]) {
				canvas[y][x1] = '│'
			}
		}
	} else if y1 == y2 {
		// Horizontal line
		for x := x1; x <= x2; x++ {
			if y1 >= 0 && y1 < len(canvas) && x >= 0 && x < len(canvas[0]) {
				canvas[y1][x] = '─'
			}
		}
	} else {
		// Diagonal line
		slope := float64(y2-y1) / float64(x2-x1)
		for x := x1; x <= x2; x++ {
			y := y1 + int(slope*float64(x-x1))
			if y >= 0 && y < len(canvas) && x >= 0 && x < len(canvas[0]) {
				if slope > 0 {
					canvas[y][x] = '\\'
				} else {
					canvas[y][x] = '/'
				}
			}
		}
	}

	// Mark data points
	if y1 >= 0 && y1 < len(canvas) && x1 >= 0 && x1 < len(canvas[0]) {
		canvas[y1][x1] = '●'
	}
	if y2 >= 0 && y2 < len(canvas) && x2 >= 0 && x2 < len(canvas[0]) {
		canvas[y2][x2] = '●'
	}
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return s[:maxLen-3] + "..."
	}
	return s[:maxLen]
}

func formatDuration(t time.Time) string {
	return t.Format("15:04:05")
}

func getMetricUnit(metricType string) string {
	switch metricType {
	case "cpu":
		return "CPU %"
	case "memory":
		return "Memory %"
	case "load":
		return "Load"
	case "swap":
		return "Swap %"
	default:
		return "Value"
	}
}

// RenderMultiLineGraph renders multiple lines on the same graph
func (r *GraphRenderer) RenderMultiLineGraph(series map[string][]float64, timestamps []time.Time) string {
	if len(series) == 0 {
		return "No data available"
	}

	// Find global min/max
	var allValues []float64
	for _, values := range series {
		allValues = append(allValues, values...)
	}

	minY, maxY := findMinMax(allValues)
	if r.options.AutoScale {
		padding := (maxY - minY) * 0.1
		minY -= padding
		maxY += padding
	}

	// Create canvas
	graphWidth := r.options.Width - 10
	graphHeight := r.options.Height - 3

	canvas := make([][]rune, graphHeight)
	for i := range canvas {
		canvas[i] = make([]rune, graphWidth)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// Line styles for different series
	lineStyles := []rune{'─', '═', '╌', '┅', '┄', '┈'}
	styleIdx := 0

	// Plot each series
	for name, values := range series {
		sampledValues := sampleData(values, graphWidth)
		lineStyle := lineStyles[styleIdx%len(lineStyles)]
		styleIdx++

		for i := 0; i < len(sampledValues)-1; i++ {
			y1 := scaleValue(sampledValues[i], minY, maxY, graphHeight)
			y2 := scaleValue(sampledValues[i+1], minY, maxY, graphHeight)

			// Simple line between points with style
			if math.Abs(float64(y1-y2)) < 2 {
				for x := i; x <= i+1; x++ {
					if y1 >= 0 && y1 < len(canvas) && x >= 0 && x < len(canvas[0]) {
						if canvas[y1][x] == ' ' {
							canvas[y1][x] = lineStyle
						}
					}
				}
			}
		}
	}

	// Build output
	var result strings.Builder

	if r.options.Title != "" {
		result.WriteString(centerText(r.options.Title, r.options.Width))
		result.WriteString("\n")
	}

	// Render canvas with Y-axis
	yStep := (maxY - minY) / float64(graphHeight-1)
	for i := 0; i < graphHeight; i++ {
		yValue := maxY - float64(i)*yStep
		result.WriteString(fmt.Sprintf("%8.2f│", yValue))
		for j := 0; j < graphWidth; j++ {
			result.WriteRune(canvas[i][j])
		}
		result.WriteString("\n")
	}

	// X-axis
	result.WriteString("        └")
	result.WriteString(strings.Repeat("─", graphWidth))
	result.WriteString("\n")

	// Legend
	if r.options.ShowLegend {
		result.WriteString("\nLegend: ")
		styleIdx = 0
		for name := range series {
			result.WriteString(fmt.Sprintf("%c %s  ", lineStyles[styleIdx%len(lineStyles)], name))
			styleIdx++
		}
		result.WriteString("\n")
	}

	return result.String()
}