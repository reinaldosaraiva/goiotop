//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

const (
	psiCPUPath    = "/proc/pressure/cpu"
	psiIOPath     = "/proc/pressure/io"
	psiMemoryPath = "/proc/pressure/memory"
)

// psiParser handles parsing of PSI (Pressure Stall Information) metrics
type psiParser struct {
	cpuPath    string
	ioPath     string
	memoryPath string
}

// newPSIParser creates a new PSI parser
func newPSIParser() *psiParser {
	return &psiParser{
		cpuPath:    psiCPUPath,
		ioPath:     psiIOPath,
		memoryPath: psiMemoryPath,
	}
}

// collectCPUPressure collects CPU pressure information
func (p *psiParser) collectCPUPressure() (*entities.CPUPressure, error) {
	file, err := os.Open(p.cpuPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPSINotSupported
		}
		return nil, fmt.Errorf("failed to open PSI CPU file: %w", err)
	}
	defer file.Close()

	var some10s, some60s, some300s float64
	var full10s, full60s, full300s float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "some") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'some' line: %w", err)
			}
			some10s = values.avg10
			some60s = values.avg60
			some300s = values.avg300
		} else if strings.HasPrefix(line, "full") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'full' line: %w", err)
			}
			full10s = values.avg10
			full60s = values.avg60
			full300s = values.avg300
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read PSI CPU file: %w", err)
	}

	// Create Percentage value objects
	some10sPercent, err := valueobjects.NewPercentage(some10s)
	if err != nil {
		some10sPercent = valueobjects.NewPercentageZero()
	}

	some60sPercent, err := valueobjects.NewPercentage(some60s)
	if err != nil {
		some60sPercent = valueobjects.NewPercentageZero()
	}

	some300sPercent, err := valueobjects.NewPercentage(some300s)
	if err != nil {
		some300sPercent = valueobjects.NewPercentageZero()
	}

	// For CPU, "full" metrics may not exist on all kernels
	if full10s > 0 || full60s > 0 || full300s > 0 {
		full10sPercent, err := valueobjects.NewPercentage(full10s)
		if err != nil {
			full10sPercent = valueobjects.NewPercentageZero()
		}
		full60sPercent, err := valueobjects.NewPercentage(full60s)
		if err != nil {
			full60sPercent = valueobjects.NewPercentageZero()
		}
		full300sPercent, err := valueobjects.NewPercentage(full300s)
		if err != nil {
			full300sPercent = valueobjects.NewPercentageZero()
		}

		// Use constructor with full metrics
		return entities.NewCPUPressureWithFull(
			some10sPercent,
			some60sPercent,
			some300sPercent,
			full10sPercent,
			full60sPercent,
			full300sPercent,
		)
	}

	// Use constructor without full metrics
	return entities.NewCPUPressure(
		some10sPercent,
		some60sPercent,
		some300sPercent,
	)
}

// collectIOPressure collects I/O pressure information
func (p *psiParser) collectIOPressure() (*entities.IOPressure, error) {
	file, err := os.Open(p.ioPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPSINotSupported
		}
		return nil, fmt.Errorf("failed to open PSI I/O file: %w", err)
	}
	defer file.Close()

	var some10s, some60s, some300s float64
	var full10s, full60s, full300s float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "some") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'some' line: %w", err)
			}
			some10s = values.avg10
			some60s = values.avg60
			some300s = values.avg300
		} else if strings.HasPrefix(line, "full") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'full' line: %w", err)
			}
			full10s = values.avg10
			full60s = values.avg60
			full300s = values.avg300
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read PSI I/O file: %w", err)
	}

	// Create Percentage value objects
	some10sPercent, err := valueobjects.NewPercentage(some10s)
	if err != nil {
		some10sPercent = valueobjects.NewPercentageZero()
	}
	some60sPercent, err := valueobjects.NewPercentage(some60s)
	if err != nil {
		some60sPercent = valueobjects.NewPercentageZero()
	}
	some300sPercent, err := valueobjects.NewPercentage(some300s)
	if err != nil {
		some300sPercent = valueobjects.NewPercentageZero()
	}
	full10sPercent, err := valueobjects.NewPercentage(full10s)
	if err != nil {
		full10sPercent = valueobjects.NewPercentageZero()
	}
	full60sPercent, err := valueobjects.NewPercentage(full60s)
	if err != nil {
		full60sPercent = valueobjects.NewPercentageZero()
	}
	full300sPercent, err := valueobjects.NewPercentage(full300s)
	if err != nil {
		full300sPercent = valueobjects.NewPercentageZero()
	}

	return entities.NewIOPressure(
		some10sPercent,
		some60sPercent,
		some300sPercent,
		full10sPercent,
		full60sPercent,
		full300sPercent,
	), nil
}

// collectMemoryPressure collects memory pressure information
func (p *psiParser) collectMemoryPressure() (*entities.MemoryPressure, error) {
	file, err := os.Open(p.memoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrPSINotSupported
		}
		return nil, fmt.Errorf("failed to open PSI memory file: %w", err)
	}
	defer file.Close()

	var some10s, some60s, some300s float64
	var full10s, full60s, full300s float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "some") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'some' line: %w", err)
			}
			some10s = values.avg10
			some60s = values.avg60
			some300s = values.avg300
		} else if strings.HasPrefix(line, "full") {
			values, err := p.parsePSILine(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PSI 'full' line: %w", err)
			}
			full10s = values.avg10
			full60s = values.avg60
			full300s = values.avg300
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read PSI memory file: %w", err)
	}

	// Create Percentage value objects
	some10sPercent, err := valueobjects.NewPercentage(some10s)
	if err != nil {
		some10sPercent = valueobjects.NewPercentageZero()
	}
	some60sPercent, err := valueobjects.NewPercentage(some60s)
	if err != nil {
		some60sPercent = valueobjects.NewPercentageZero()
	}
	some300sPercent, err := valueobjects.NewPercentage(some300s)
	if err != nil {
		some300sPercent = valueobjects.NewPercentageZero()
	}
	full10sPercent, err := valueobjects.NewPercentage(full10s)
	if err != nil {
		full10sPercent = valueobjects.NewPercentageZero()
	}
	full60sPercent, err := valueobjects.NewPercentage(full60s)
	if err != nil {
		full60sPercent = valueobjects.NewPercentageZero()
	}
	full300sPercent, err := valueobjects.NewPercentage(full300s)
	if err != nil {
		full300sPercent = valueobjects.NewPercentageZero()
	}

	return entities.NewMemoryPressure(
		some10sPercent,
		some60sPercent,
		some300sPercent,
		full10sPercent,
		full60sPercent,
		full300sPercent,
	), nil
}

// psiValues holds parsed PSI line values
type psiValues struct {
	avg10  float64
	avg60  float64
	avg300 float64
	total  uint64
}

// parsePSILine parses a single line from PSI file
// Format: some avg10=0.00 avg60=0.00 avg300=0.00 total=0
func (p *psiParser) parsePSILine(line string) (*psiValues, error) {
	values := &psiValues{}

	// Split by space to get key=value pairs
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid PSI line format: %s", line)
	}

	// Skip the first part (some/full)
	for _, part := range parts[1:] {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "avg10":
			val, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg10: %w", err)
			}
			values.avg10 = val

		case "avg60":
			val, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg60: %w", err)
			}
			values.avg60 = val

		case "avg300":
			val, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse avg300: %w", err)
			}
			values.avg300 = val

		case "total":
			val, err := strconv.ParseUint(kv[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse total: %w", err)
			}
			values.total = val
		}
	}

	return values, nil
}

// checkPSISupport checks if PSI is available on the system
func (p *psiParser) checkPSISupport() bool {
	_, err := os.Stat(p.cpuPath)
	return err == nil
}