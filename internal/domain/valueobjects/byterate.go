package valueobjects

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNegativeByteRate = errors.New("byte rate cannot be negative")
)

type ByteRate struct {
	bytesPerSecond float64
}

func NewByteRate(bytesPerSecond float64) (ByteRate, error) {
	if bytesPerSecond < 0 {
		return ByteRate{}, ErrNegativeByteRate
	}
	return ByteRate{bytesPerSecond: bytesPerSecond}, nil
}

func NewByteRateZero() ByteRate {
	return ByteRate{bytesPerSecond: 0}
}

func NewByteRateFromKB(kbPerSecond float64) (ByteRate, error) {
	return NewByteRate(kbPerSecond * 1024)
}

func NewByteRateFromMB(mbPerSecond float64) (ByteRate, error) {
	return NewByteRate(mbPerSecond * 1024 * 1024)
}

func NewByteRateFromGB(gbPerSecond float64) (ByteRate, error) {
	return NewByteRate(gbPerSecond * 1024 * 1024 * 1024)
}

func (b ByteRate) BytesPerSecond() float64 {
	return b.bytesPerSecond
}

func (b ByteRate) KBPerSecond() float64 {
	return b.bytesPerSecond / 1024
}

func (b ByteRate) MBPerSecond() float64 {
	return b.bytesPerSecond / (1024 * 1024)
}

func (b ByteRate) GBPerSecond() float64 {
	return b.bytesPerSecond / (1024 * 1024 * 1024)
}

func (b ByteRate) Add(other ByteRate) ByteRate {
	return ByteRate{bytesPerSecond: b.bytesPerSecond + other.bytesPerSecond}
}

func (b ByteRate) Subtract(other ByteRate) (ByteRate, error) {
	result := b.bytesPerSecond - other.bytesPerSecond
	if result < 0 {
		return ByteRate{}, ErrNegativeByteRate
	}
	return ByteRate{bytesPerSecond: result}, nil
}

func (b ByteRate) Multiply(factor float64) (ByteRate, error) {
	if factor < 0 {
		return ByteRate{}, ErrNegativeByteRate
	}
	return ByteRate{bytesPerSecond: b.bytesPerSecond * factor}, nil
}

func (b ByteRate) Divide(divisor float64) (ByteRate, error) {
	if divisor == 0 {
		return ByteRate{}, errors.New("division by zero")
	}
	result := b.bytesPerSecond / divisor
	if result < 0 {
		return ByteRate{}, ErrNegativeByteRate
	}
	return ByteRate{bytesPerSecond: result}, nil
}

func (b ByteRate) Equal(other ByteRate) bool {
	const epsilon = 0.0001
	diff := b.bytesPerSecond - other.bytesPerSecond
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func (b ByteRate) LessThan(other ByteRate) bool {
	return b.bytesPerSecond < other.bytesPerSecond
}

func (b ByteRate) GreaterThan(other ByteRate) bool {
	return b.bytesPerSecond > other.bytesPerSecond
}

func (b ByteRate) IsZero() bool {
	return b.bytesPerSecond == 0
}

func (b ByteRate) String() string {
	return b.HumanReadable()
}

func (b ByteRate) HumanReadable() string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	rate := b.bytesPerSecond

	switch {
	case rate >= GB:
		return fmt.Sprintf("%.2f GB/s", rate/GB)
	case rate >= MB:
		return fmt.Sprintf("%.2f MB/s", rate/MB)
	case rate >= KB:
		return fmt.Sprintf("%.2f KB/s", rate/KB)
	default:
		return fmt.Sprintf("%.0f B/s", rate)
	}
}

func (b ByteRate) HumanReadableWithPrecision(precision int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	rate := b.bytesPerSecond
	format := fmt.Sprintf("%%.%df", precision)

	switch {
	case rate >= GB:
		return fmt.Sprintf(format+" GB/s", rate/GB)
	case rate >= MB:
		return fmt.Sprintf(format+" MB/s", rate/MB)
	case rate >= KB:
		return fmt.Sprintf(format+" KB/s", rate/KB)
	default:
		return fmt.Sprintf("%.0f B/s", rate)
	}
}

func (b ByteRate) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.bytesPerSecond)
}

func (b *ByteRate) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	br, err := NewByteRate(value)
	if err != nil {
		return err
	}
	*b = br
	return nil
}