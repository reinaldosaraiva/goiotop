package valueobjects

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrInvalidProcessID = errors.New("process ID must be positive")
)

type ProcessID struct {
	value int32
}

func NewProcessID(pid int32) (ProcessID, error) {
	if pid <= 0 {
		return ProcessID{}, ErrInvalidProcessID
	}
	return ProcessID{value: pid}, nil
}

func NewProcessIDFromString(pidStr string) (ProcessID, error) {
	pid64, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		return ProcessID{}, fmt.Errorf("invalid process ID format: %w", err)
	}
	return NewProcessID(int32(pid64))
}

func NewProcessIDFromInt(pid int) (ProcessID, error) {
	return NewProcessID(int32(pid))
}

func (p ProcessID) Value() int32 {
	return p.value
}

func (p ProcessID) Int() int {
	return int(p.value)
}

func (p ProcessID) String() string {
	return strconv.Itoa(int(p.value))
}

func (p ProcessID) Equal(other ProcessID) bool {
	return p.value == other.value
}

func (p ProcessID) LessThan(other ProcessID) bool {
	return p.value < other.value
}

func (p ProcessID) GreaterThan(other ProcessID) bool {
	return p.value > other.value
}

func (p ProcessID) IsSystemProcess() bool {
	return p.value < 1000
}

func (p ProcessID) IsKernelThread() bool {
	return p.value < 300
}

func (p ProcessID) IsInit() bool {
	return p.value == 1
}

func (p ProcessID) IsValid() bool {
	const maxPID int32 = 4194304
	return p.value > 0 && p.value <= maxPID
}

func (p ProcessID) Hash() uint32 {
	return uint32(p.value)
}

func (p ProcessID) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *ProcessID) UnmarshalJSON(data []byte) error {
	var value int32
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	pid, err := NewProcessID(value)
	if err != nil {
		return err
	}
	*p = pid
	return nil
}