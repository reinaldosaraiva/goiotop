package valueobjects

import (
	"errors"
	"time"
)

var (
	ErrInvalidTimestamp = errors.New("invalid timestamp: cannot be zero or future time")
	MaxFutureSkew       = 2 * time.Second
)

type Timestamp struct {
	value time.Time
}

func NewTimestamp(t time.Time) (*Timestamp, error) {
	if t.IsZero() {
		return nil, ErrInvalidTimestamp
	}
	if t.After(time.Now().Add(MaxFutureSkew)) {
		return nil, ErrInvalidTimestamp
	}
	return &Timestamp{value: t}, nil
}

func NewTimestampNow() *Timestamp {
	return &Timestamp{value: time.Now()}
}

func (t Timestamp) Value() time.Time {
	return t.value
}

func (t Timestamp) Unix() int64 {
	return t.value.Unix()
}

func (t Timestamp) UnixMilli() int64 {
	return t.value.UnixMilli()
}

func (t Timestamp) Format(layout string) string {
	return t.value.Format(layout)
}

func (t Timestamp) String() string {
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) Equal(other Timestamp) bool {
	return t.value.Equal(other.value)
}

func (t Timestamp) Before(other Timestamp) bool {
	return t.value.Before(other.value)
}

func (t Timestamp) After(other Timestamp) bool {
	return t.value.After(other.value)
}

func (t Timestamp) Sub(other Timestamp) time.Duration {
	return t.value.Sub(other.value)
}

func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Add(d)}
}

func (t Timestamp) IsRecent(threshold time.Duration) bool {
	return time.Since(t.value) <= threshold
}

func (t Timestamp) DurationSince() time.Duration {
	return time.Since(t.value)
}

func (t Timestamp) Truncate(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Truncate(d)}
}

func (t Timestamp) Round(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Round(d)}
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return t.value.MarshalJSON()
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var tm time.Time
	if err := tm.UnmarshalJSON(data); err != nil {
		return err
	}
	ts, err := NewTimestamp(tm)
	if err != nil {
		return err
	}
	*t = *ts
	return nil
}