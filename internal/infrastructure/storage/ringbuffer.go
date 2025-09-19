package storage

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// RingBuffer is a thread-safe circular buffer optimized for time-series metrics storage
type RingBuffer[T any] struct {
	mu          sync.RWMutex
	buffer      []T
	timestamps  []time.Time
	capacity    int
	head        int
	tail        int
	size        int
	totalWrites int64
	totalReads  int64
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RingBuffer[T]{
		buffer:     make([]T, capacity),
		timestamps: make([]time.Time, capacity),
		capacity:   capacity,
		head:       0,
		tail:       0,
		size:       0,
	}
}

// Push adds an item to the ring buffer with the current timestamp
func (rb *RingBuffer[T]) Push(item T) {
	rb.PushWithTime(item, time.Now())
}

// PushWithTime adds an item to the ring buffer with a specific timestamp
func (rb *RingBuffer[T]) PushWithTime(item T, timestamp time.Time) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = item
	rb.timestamps[rb.head] = timestamp
	rb.head = (rb.head + 1) % rb.capacity

	if rb.size < rb.capacity {
		rb.size++
	} else {
		// Buffer is full, move tail forward (oldest item is evicted)
		rb.tail = (rb.tail + 1) % rb.capacity
	}
	rb.totalWrites++
}

// Pop removes and returns the oldest item from the buffer
func (rb *RingBuffer[T]) Pop() (T, time.Time, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var zero T
	if rb.size == 0 {
		return zero, time.Time{}, false
	}

	item := rb.buffer[rb.tail]
	timestamp := rb.timestamps[rb.tail]
	rb.tail = (rb.tail + 1) % rb.capacity
	rb.size--
	rb.totalReads++

	return item, timestamp, true
}

// Peek returns the oldest item without removing it
func (rb *RingBuffer[T]) Peek() (T, time.Time, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var zero T
	if rb.size == 0 {
		return zero, time.Time{}, false
	}

	return rb.buffer[rb.tail], rb.timestamps[rb.tail], true
}

// GetAll returns all items in the buffer in chronological order
func (rb *RingBuffer[T]) GetAll() ([]T, []time.Time) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []T{}, []time.Time{}
	}

	items := make([]T, rb.size)
	timestamps := make([]time.Time, rb.size)

	for i := 0; i < rb.size; i++ {
		idx := (rb.tail + i) % rb.capacity
		items[i] = rb.buffer[idx]
		timestamps[i] = rb.timestamps[idx]
	}
	rb.totalReads += int64(rb.size)

	return items, timestamps
}

// GetByTimeRange returns items within the specified time range
// Uses binary search for optimization when dealing with large buffers
func (rb *RingBuffer[T]) GetByTimeRange(start, end time.Time) ([]T, []time.Time) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []T{}, []time.Time{}
	}

	// For small buffers, use linear search
	if rb.size < 100 {
		return rb.getByTimeRangeLinear(start, end)
	}

	// For larger buffers, use binary search to find range bounds
	// First, collect all timestamps in order for binary search
	orderedTimestamps := make([]time.Time, rb.size)
	for i := 0; i < rb.size; i++ {
		idx := (rb.tail + i) % rb.capacity
		orderedTimestamps[i] = rb.timestamps[idx]
	}

	// Binary search for start index
	startIdx := sort.Search(rb.size, func(i int) bool {
		return !orderedTimestamps[i].Before(start)
	})

	// If start index is beyond our range, return empty
	if startIdx >= rb.size {
		return []T{}, []time.Time{}
	}

	// Binary search for end index
	endIdx := sort.Search(rb.size, func(i int) bool {
		return orderedTimestamps[i].After(end)
	})

	// Collect items in the range [startIdx, endIdx)
	items := make([]T, 0, endIdx-startIdx)
	timestamps := make([]time.Time, 0, endIdx-startIdx)

	for i := startIdx; i < endIdx; i++ {
		idx := (rb.tail + i) % rb.capacity
		items = append(items, rb.buffer[idx])
		timestamps = append(timestamps, rb.timestamps[idx])
	}

	rb.totalReads += int64(len(items))
	return items, timestamps
}

// getByTimeRangeLinear is the linear search implementation for small buffers
func (rb *RingBuffer[T]) getByTimeRangeLinear(start, end time.Time) ([]T, []time.Time) {
	var items []T
	var timestamps []time.Time

	for i := 0; i < rb.size; i++ {
		idx := (rb.tail + i) % rb.capacity
		ts := rb.timestamps[idx]
		if (ts.Equal(start) || ts.After(start)) && (ts.Equal(end) || ts.Before(end)) {
			items = append(items, rb.buffer[idx])
			timestamps = append(timestamps, ts)
			rb.totalReads++
		}
	}

	return items, timestamps
}

// GetLatest returns the most recent n items
func (rb *RingBuffer[T]) GetLatest(n int) ([]T, []time.Time) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []T{}, []time.Time{}
	}

	if n > rb.size {
		n = rb.size
	}

	items := make([]T, n)
	timestamps := make([]time.Time, n)

	// Start from the most recent items
	startIdx := (rb.tail + rb.size - n) % rb.capacity
	for i := 0; i < n; i++ {
		idx := (startIdx + i) % rb.capacity
		items[i] = rb.buffer[idx]
		timestamps[i] = rb.timestamps[idx]
	}
	rb.totalReads += int64(n)

	return items, timestamps
}

// Size returns the current number of items in the buffer
func (rb *RingBuffer[T]) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Capacity returns the maximum capacity of the buffer
func (rb *RingBuffer[T]) Capacity() int {
	return rb.capacity
}

// Clear removes all items from the buffer
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.size = 0
}

// Stats returns statistics about the buffer usage
func (rb *RingBuffer[T]) Stats() map[string]interface{} {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	utilization := float64(rb.size) / float64(rb.capacity) * 100

	return map[string]interface{}{
		"capacity":    rb.capacity,
		"size":        rb.size,
		"utilization": fmt.Sprintf("%.2f%%", utilization),
		"totalReads":  rb.totalReads,
		"totalWrites": rb.totalWrites,
		"isFull":      rb.size == rb.capacity,
	}
}

// EvictOlderThan removes items older than the specified time
func (rb *RingBuffer[T]) EvictOlderThan(cutoff time.Time) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	evicted := 0
	for rb.size > 0 {
		if rb.timestamps[rb.tail].Before(cutoff) {
			rb.tail = (rb.tail + 1) % rb.capacity
			rb.size--
			evicted++
		} else {
			break
		}
	}

	return evicted
}

// GetWithTimestamp wraps an item with its timestamp for aggregation
type ItemWithTimestamp[T any] struct {
	Item      T
	Timestamp valueobjects.Timestamp
}

// Aggregate applies an aggregation function to items in a time range
func (rb *RingBuffer[T]) Aggregate(start, end time.Time, aggFunc func([]T) interface{}) interface{} {
	items, _ := rb.GetByTimeRange(start, end)
	if len(items) == 0 {
		return nil
	}
	return aggFunc(items)
}

// MemoryUsage estimates the memory usage of the buffer in bytes
func (rb *RingBuffer[T]) MemoryUsage() int64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	// Estimate based on buffer capacity and overhead
	// This is a rough estimate as actual size depends on T
	baseSize := int64(rb.capacity) * 32 // Assume 32 bytes per item average
	timestampSize := int64(rb.capacity) * 24 // time.Time is typically 24 bytes
	overhead := int64(1024) // Mutex, counters, etc.

	return baseSize + timestampSize + overhead
}

// Resize changes the capacity of the buffer, preserving recent data
func (rb *RingBuffer[T]) Resize(newCapacity int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if newCapacity <= 0 || newCapacity == rb.capacity {
		return
	}

	// Get all current items
	items := make([]T, 0, rb.size)
	timestamps := make([]time.Time, 0, rb.size)

	for i := 0; i < rb.size; i++ {
		idx := (rb.tail + i) % rb.capacity
		items = append(items, rb.buffer[idx])
		timestamps = append(timestamps, rb.timestamps[idx])
	}

	// Create new buffers
	rb.buffer = make([]T, newCapacity)
	rb.timestamps = make([]time.Time, newCapacity)
	rb.capacity = newCapacity
	rb.head = 0
	rb.tail = 0
	rb.size = 0

	// Re-add items, keeping the most recent if there are too many
	start := 0
	if len(items) > newCapacity {
		start = len(items) - newCapacity
	}

	for i := start; i < len(items); i++ {
		rb.buffer[rb.head] = items[i]
		rb.timestamps[rb.head] = timestamps[i]
		rb.head = (rb.head + 1) % rb.capacity
		rb.size++
	}
}