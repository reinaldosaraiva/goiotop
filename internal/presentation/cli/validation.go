package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ValidateDuration validates a duration string
func ValidateDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return 0, nil
	}

	// Try parsing as a duration string (e.g., "5s", "1m", "1h30m")
	d, err := time.ParseDuration(duration)
	if err == nil {
		if d < 0 {
			return 0, fmt.Errorf("duration must be positive: %s", duration)
		}
		return d, nil
	}

	// Try parsing as seconds if it's just a number
	if seconds, err := strconv.Atoi(duration); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("duration must be positive: %d", seconds)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (use format like '5s', '1m', '1h30m')", duration)
}

// ValidatePercentage validates a percentage value
func ValidatePercentage(value string) (float64, error) {
	// Remove % sign if present
	value = strings.TrimSuffix(value, "%")

	percentage, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid percentage value: %s", value)
	}

	if percentage < 0 || percentage > 100 {
		return 0, fmt.Errorf("percentage must be between 0 and 100: %.2f", percentage)
	}

	return percentage, nil
}

// ValidateFilePath validates a file path
func ValidateFilePath(path string, mustExist bool) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Check if file exists if required
	if mustExist {
		if _, err := os.Stat(absPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", absPath)
			}
			return fmt.Errorf("failed to access file: %w", err)
		}
	}

	return nil
}

// ValidateDirectory validates a directory path
func ValidateDirectory(path string, mustExist bool) error {
	if path == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// Check if directory exists if required
	if mustExist {
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("directory does not exist: %s", absPath)
			}
			return fmt.Errorf("failed to access directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", absPath)
		}
	}

	return nil
}

// ValidateProcessName validates a process name
func ValidateProcessName(name string) error {
	if name == "" {
		return fmt.Errorf("process name cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("process name contains invalid characters: %s", name)
	}

	// Check length
	if len(name) > 255 {
		return fmt.Errorf("process name too long (max 255 characters): %d", len(name))
	}

	return nil
}

// ValidateProcessNames validates multiple process names
func ValidateProcessNames(names []string) error {
	seen := make(map[string]bool)
	for _, name := range names {
		if err := ValidateProcessName(name); err != nil {
			return err
		}
		if seen[name] {
			return fmt.Errorf("duplicate process name: %s", name)
		}
		seen[name] = true
	}
	return nil
}

// ValidatePort validates a network port number
func ValidatePort(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535: %d", port)
	}
	return nil
}

// ValidateURL validates a URL string
func ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "s3://") {
		return fmt.Errorf("URL must start with http://, https://, or s3://: %s", url)
	}

	return nil
}

// ValidateTimeFormat validates a time format string
func ValidateTimeFormat(timeStr string, format string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("time string cannot be empty")
	}

	t, err := time.Parse(format, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format (expected %s): %s", format, timeStr)
	}

	return t, nil
}

// ValidateRFC3339Time validates an RFC3339 time string
func ValidateRFC3339Time(timeStr string) (time.Time, error) {
	return ValidateTimeFormat(timeStr, time.RFC3339)
}

// ValidatePositiveInt validates a positive integer
func ValidatePositiveInt(value int) error {
	if value <= 0 {
		return fmt.Errorf("value must be positive: %d", value)
	}
	return nil
}

// ValidateNonNegativeInt validates a non-negative integer
func ValidateNonNegativeInt(value int) error {
	if value < 0 {
		return fmt.Errorf("value must be non-negative: %d", value)
	}
	return nil
}

// ValidateIntRange validates an integer is within a range
func ValidateIntRange(value, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("value must be between %d and %d: %d", min, max, value)
	}
	return nil
}

// ValidateFloatRange validates a float is within a range
func ValidateFloatRange(value, min, max float64) error {
	if value < min || value > max {
		return fmt.Errorf("value must be between %.2f and %.2f: %.2f", min, max, value)
	}
	return nil
}

// ValidateRegex validates a regular expression pattern
func ValidateRegex(pattern string) error {
	if pattern == "" {
		return nil // Empty pattern is valid
	}

	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	return nil
}

// ValidateEnum validates a value is in a list of allowed values
func ValidateEnum(value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("value must be one of %v: %s", allowed, value)
}

// CobraValidators provides validation functions for cobra commands

// DurationValidator returns a cobra validator for duration flags
func DurationValidator(flagName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString(flagName)
		if value == "" {
			return nil
		}
		_, err := ValidateDuration(value)
		return err
	}
}

// FilePathValidator returns a cobra validator for file path flags
func FilePathValidator(flagName string, mustExist bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString(flagName)
		if value == "" {
			return nil
		}
		return ValidateFilePath(value, mustExist)
	}
}

// DirectoryValidator returns a cobra validator for directory flags
func DirectoryValidator(flagName string, mustExist bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString(flagName)
		if value == "" {
			return nil
		}
		return ValidateDirectory(value, mustExist)
	}
}

// PortValidator returns a cobra validator for port flags
func PortValidator(flagName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetInt(flagName)
		return ValidatePort(value)
	}
}

// URLValidator returns a cobra validator for URL flags
func URLValidator(flagName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString(flagName)
		if value == "" {
			return nil
		}
		return ValidateURL(value)
	}
}

// EnumValidator returns a cobra validator for enum flags
func EnumValidator(flagName string, allowed []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString(flagName)
		if value == "" {
			return nil
		}
		return ValidateEnum(value, allowed)
	}
}

// IntRangeValidator returns a cobra validator for integer range flags
func IntRangeValidator(flagName string, min, max int) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetInt(flagName)
		return ValidateIntRange(value, min, max)
	}
}

// FloatRangeValidator returns a cobra validator for float range flags
func FloatRangeValidator(flagName string, min, max float64) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetFloat64(flagName)
		return ValidateFloatRange(value, min, max)
	}
}