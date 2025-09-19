package valueobjects

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidPercentage = errors.New("percentage must be between 0 and 100")
)

type Percentage struct {
	value float64
}

func NewPercentage(value float64) (Percentage, error) {
	if value < 0 || value > 100 {
		return Percentage{}, ErrInvalidPercentage
	}
	return Percentage{value: value}, nil
}

func NewPercentageFromRatio(numerator, denominator float64) (Percentage, error) {
	if denominator == 0 {
		return Percentage{}, errors.New("division by zero")
	}
	value := (numerator / denominator) * 100
	return NewPercentage(value)
}

func NewPercentageZero() Percentage {
	return Percentage{value: 0}
}

func NewPercentageFull() Percentage {
	return Percentage{value: 100}
}

func (p Percentage) Value() float64 {
	return p.value
}

func (p Percentage) Ratio() float64 {
	return p.value / 100
}

func (p Percentage) Add(other Percentage) (Percentage, error) {
	return NewPercentage(p.value + other.value)
}

func (p Percentage) Subtract(other Percentage) (Percentage, error) {
	return NewPercentage(p.value - other.value)
}

func (p Percentage) Multiply(factor float64) (Percentage, error) {
	return NewPercentage(p.value * factor)
}

func (p Percentage) Divide(divisor float64) (Percentage, error) {
	if divisor == 0 {
		return Percentage{}, errors.New("division by zero")
	}
	return NewPercentage(p.value / divisor)
}

func (p Percentage) Average(others ...Percentage) Percentage {
	sum := p.value
	count := 1.0
	for _, other := range others {
		sum += other.value
		count++
	}
	avg := sum / count
	result, _ := NewPercentage(avg)
	return result
}

func (p Percentage) Equal(other Percentage) bool {
	const epsilon = 0.0001
	diff := p.value - other.value
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func (p Percentage) LessThan(other Percentage) bool {
	return p.value < other.value
}

func (p Percentage) GreaterThan(other Percentage) bool {
	return p.value > other.value
}

func (p Percentage) IsZero() bool {
	return p.value == 0
}

func (p Percentage) IsFull() bool {
	return p.value == 100
}

func (p Percentage) IsHigh(threshold float64) bool {
	return p.value >= threshold
}

func (p Percentage) IsLow(threshold float64) bool {
	return p.value <= threshold
}

func (p Percentage) IsCritical() bool {
	return p.value >= 90
}

func (p Percentage) IsWarning() bool {
	return p.value >= 70 && p.value < 90
}

func (p Percentage) IsNormal() bool {
	return p.value < 70
}

func (p Percentage) String() string {
	return fmt.Sprintf("%.2f%%", p.value)
}

func (p Percentage) StringWithPrecision(precision int) string {
	format := fmt.Sprintf("%%.%df%%%%", precision)
	return fmt.Sprintf(format, p.value)
}

func (p Percentage) ProgressBar(width int) string {
	if width <= 0 {
		return ""
	}

	filled := int(p.value * float64(width) / 100)
	if filled > width {
		filled = width
	}

	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}
	bar += "]"

	return bar
}

func (p Percentage) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *Percentage) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	pct, err := NewPercentage(value)
	if err != nil {
		return err
	}
	*p = pct
	return nil
}