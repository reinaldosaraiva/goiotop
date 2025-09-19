package optimization

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/reinaldosaraiva/goiotop/internal/application/dto"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/domain/valueobjects"
)

// ObjectPool is a generic object pool implementation
type ObjectPool[T any] struct {
	pool      sync.Pool
	newFunc   func() T
	resetFunc func(T)
	stats     PoolStats
}

// PoolStats tracks pool usage statistics
type PoolStats struct {
	Gets    atomic.Int64
	Puts    atomic.Int64
	Misses  atomic.Int64
	Created atomic.Int64
}

// NewObjectPool creates a new object pool
func NewObjectPool[T any](newFunc func() T, resetFunc func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		newFunc:   newFunc,
		resetFunc: resetFunc,
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool[T]) Get() T {
	p.stats.Gets.Add(1)
	obj := p.pool.Get()
	if obj == nil {
		p.stats.Misses.Add(1)
		p.stats.Created.Add(1)
		return p.newFunc()
	}
	return obj.(T)
}

// Put returns an object to the pool
func (p *ObjectPool[T]) Put(obj T) {
	if p.resetFunc != nil {
		p.resetFunc(obj)
	}
	p.stats.Puts.Add(1)
	p.pool.Put(obj)
}

// GetStats returns pool statistics
func (p *ObjectPool[T]) GetStats() PoolStats {
	return PoolStats{
		Gets:    p.stats.Gets,
		Puts:    p.stats.Puts,
		Misses:  p.stats.Misses,
		Created: p.stats.Created,
	}
}

// ByteSlice is a wrapper for []byte to enable proper pooling
type ByteSlice struct {
	Data []byte
}

// StringSlice is a wrapper for []string to enable proper pooling
type StringSlice struct {
	Data []string
}

// MetricsPoolManager manages object pools for metrics entities
type MetricsPoolManager struct {
	systemMetricsPool  *ObjectPool[*entities.SystemMetrics]
	processMetricsPool *ObjectPool[*entities.ProcessMetrics]
	cpuPressurePool    *ObjectPool[*entities.CPUPressure]
	diskIOPool         *ObjectPool[*entities.DiskIO]

	// DTO pools
	systemMetricsDTOPool  *ObjectPool[*dto.SystemMetricsDTO]
	processMetricsDTOPool *ObjectPool[*dto.ProcessMetricsDTO]
	collectionResultPool  *ObjectPool[*dto.CollectionResultDTO]

	// Value object pools
	bytesPool      *ObjectPool[*ByteSlice]
	stringSlicePool *ObjectPool[*StringSlice]
}

// NewMetricsPoolManager creates a new metrics pool manager
func NewMetricsPoolManager() *MetricsPoolManager {
	return &MetricsPoolManager{
		// Entity pools
		systemMetricsPool: NewObjectPool(
			func() *entities.SystemMetrics { return &entities.SystemMetrics{} },
			func(m *entities.SystemMetrics) { resetSystemMetrics(m) },
		),
		processMetricsPool: NewObjectPool(
			func() *entities.ProcessMetrics { return &entities.ProcessMetrics{} },
			func(m *entities.ProcessMetrics) { resetProcessMetrics(m) },
		),
		cpuPressurePool: NewObjectPool(
			func() *entities.CPUPressure { return &entities.CPUPressure{} },
			func(p *entities.CPUPressure) { resetCPUPressure(p) },
		),
		diskIOPool: NewObjectPool(
			func() *entities.DiskIO { return &entities.DiskIO{} },
			func(d *entities.DiskIO) { resetDiskIO(d) },
		),

		// DTO pools
		systemMetricsDTOPool: NewObjectPool(
			func() *dto.SystemMetricsDTO { return &dto.SystemMetricsDTO{} },
			func(d *dto.SystemMetricsDTO) { resetSystemMetricsDTO(d) },
		),
		processMetricsDTOPool: NewObjectPool(
			func() *dto.ProcessMetricsDTO { return &dto.ProcessMetricsDTO{} },
			func(d *dto.ProcessMetricsDTO) { resetProcessMetricsDTO(d) },
		),
		collectionResultPool: NewObjectPool(
			func() *dto.CollectionResultDTO { return &dto.CollectionResultDTO{} },
			func(r *dto.CollectionResultDTO) { resetCollectionResultDTO(r) },
		),

		// Utility pools with proper reset
		bytesPool: NewObjectPool(
			func() *ByteSlice { return &ByteSlice{Data: make([]byte, 0, 4096)} },
			func(b *ByteSlice) { b.Data = b.Data[:0] },
		),
		stringSlicePool: NewObjectPool(
			func() *StringSlice { return &StringSlice{Data: make([]string, 0, 100)} },
			func(s *StringSlice) { s.Data = s.Data[:0] },
		),
	}
}

// GetSystemMetrics gets a SystemMetrics from the pool
func (m *MetricsPoolManager) GetSystemMetrics() *entities.SystemMetrics {
	return m.systemMetricsPool.Get()
}

// PutSystemMetrics returns a SystemMetrics to the pool
func (m *MetricsPoolManager) PutSystemMetrics(metrics *entities.SystemMetrics) {
	m.systemMetricsPool.Put(metrics)
}

// GetProcessMetrics gets a ProcessMetrics from the pool
func (m *MetricsPoolManager) GetProcessMetrics() *entities.ProcessMetrics {
	return m.processMetricsPool.Get()
}

// PutProcessMetrics returns a ProcessMetrics to the pool
func (m *MetricsPoolManager) PutProcessMetrics(metrics *entities.ProcessMetrics) {
	m.processMetricsPool.Put(metrics)
}

// GetCPUPressure gets a CPUPressure from the pool
func (m *MetricsPoolManager) GetCPUPressure() *entities.CPUPressure {
	return m.cpuPressurePool.Get()
}

// PutCPUPressure returns a CPUPressure to the pool
func (m *MetricsPoolManager) PutCPUPressure(pressure *entities.CPUPressure) {
	m.cpuPressurePool.Put(pressure)
}

// GetDiskIO gets a DiskIO from the pool
func (m *MetricsPoolManager) GetDiskIO() *entities.DiskIO {
	return m.diskIOPool.Get()
}

// PutDiskIO returns a DiskIO to the pool
func (m *MetricsPoolManager) PutDiskIO(diskIO *entities.DiskIO) {
	m.diskIOPool.Put(diskIO)
}

// GetSystemMetricsDTO gets a SystemMetricsDTO from the pool
func (m *MetricsPoolManager) GetSystemMetricsDTO() *dto.SystemMetricsDTO {
	return m.systemMetricsDTOPool.Get()
}

// PutSystemMetricsDTO returns a SystemMetricsDTO to the pool
func (m *MetricsPoolManager) PutSystemMetricsDTO(d *dto.SystemMetricsDTO) {
	m.systemMetricsDTOPool.Put(d)
}

// GetProcessMetricsDTO gets a ProcessMetricsDTO from the pool
func (m *MetricsPoolManager) GetProcessMetricsDTO() *dto.ProcessMetricsDTO {
	return m.processMetricsDTOPool.Get()
}

// PutProcessMetricsDTO returns a ProcessMetricsDTO to the pool
func (m *MetricsPoolManager) PutProcessMetricsDTO(d *dto.ProcessMetricsDTO) {
	m.processMetricsDTOPool.Put(d)
}

// GetCollectionResult gets a CollectionResultDTO from the pool
func (m *MetricsPoolManager) GetCollectionResult() *dto.CollectionResultDTO {
	return m.collectionResultPool.Get()
}

// PutCollectionResult returns a CollectionResultDTO to the pool
func (m *MetricsPoolManager) PutCollectionResult(result *dto.CollectionResultDTO) {
	m.collectionResultPool.Put(result)
}

// GetByteBuffer gets a byte buffer from the pool
func (m *MetricsPoolManager) GetByteBuffer() []byte {
	bs := m.bytesPool.Get()
	return bs.Data
}

// PutByteBuffer returns a byte buffer to the pool
func (m *MetricsPoolManager) PutByteBuffer(buf []byte) {
	bs := &ByteSlice{Data: buf}
	bs.Data = bs.Data[:0] // Reset before putting back
	m.bytesPool.Put(bs)
}

// GetStringSlice gets a string slice from the pool
func (m *MetricsPoolManager) GetStringSlice() []string {
	ss := m.stringSlicePool.Get()
	return ss.Data
}

// PutStringSlice returns a string slice to the pool
func (m *MetricsPoolManager) PutStringSlice(slice []string) {
	ss := &StringSlice{Data: slice}
	ss.Data = ss.Data[:0] // Reset before putting back
	m.stringSlicePool.Put(ss)
}

// GetStatistics returns statistics for all pools
func (m *MetricsPoolManager) GetStatistics() map[string]PoolStats {
	return map[string]PoolStats{
		"systemMetrics":      m.systemMetricsPool.GetStats(),
		"processMetrics":     m.processMetricsPool.GetStats(),
		"cpuPressure":        m.cpuPressurePool.GetStats(),
		"diskIO":             m.diskIOPool.GetStats(),
		"systemMetricsDTO":   m.systemMetricsDTOPool.GetStats(),
		"processMetricsDTO":  m.processMetricsDTOPool.GetStats(),
		"collectionResult":   m.collectionResultPool.GetStats(),
		"byteBuffer":         m.bytesPool.GetStats(),
		"stringSlice":        m.stringSlicePool.GetStats(),
	}
}

// Reset functions for entities

func resetSystemMetrics(m *entities.SystemMetrics) {
	*m = entities.SystemMetrics{}
}

func resetProcessMetrics(m *entities.ProcessMetrics) {
	*m = entities.ProcessMetrics{}
}

func resetCPUPressure(p *entities.CPUPressure) {
	*p = entities.CPUPressure{}
}

func resetDiskIO(d *entities.DiskIO) {
	*d = entities.DiskIO{}
}

func resetSystemMetricsDTO(d *dto.SystemMetricsDTO) {
	*d = dto.SystemMetricsDTO{}
}

func resetProcessMetricsDTO(d *dto.ProcessMetricsDTO) {
	*d = dto.ProcessMetricsDTO{}
}

func resetCollectionResultDTO(r *dto.CollectionResultDTO) {
	r.SystemMetrics = nil
	r.ProcessMetrics = r.ProcessMetrics[:0]
	r.Errors = r.Errors[:0]
}

// GlobalPools is the global pool manager instance
var GlobalPools = NewMetricsPoolManager()

// ProcessMetricsPool provides a specialized pool for process metrics slices
type ProcessMetricsPool struct {
	pool sync.Pool
}

// NewProcessMetricsPool creates a new pool for process metrics slices
func NewProcessMetricsPool(initialCapacity int) *ProcessMetricsPool {
	return &ProcessMetricsPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]*entities.ProcessMetrics, 0, initialCapacity)
			},
		},
	}
}

// Get retrieves a process metrics slice from the pool
func (p *ProcessMetricsPool) Get() []*entities.ProcessMetrics {
	v := p.pool.Get()
	if v == nil {
		return make([]*entities.ProcessMetrics, 0, 100)
	}
	return v.([]*entities.ProcessMetrics)
}

// Put returns a process metrics slice to the pool
func (p *ProcessMetricsPool) Put(slice []*entities.ProcessMetrics) {
	// Clear the slice but keep the capacity
	for i := range slice {
		slice[i] = nil
	}
	slice = slice[:0]
	p.pool.Put(slice)
}

// BufferPool manages reusable byte buffers
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		size: bufferSize,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf []byte) {
	if cap(buf) == p.size {
		buf = buf[:p.size]
		p.pool.Put(buf)
	}
}

// StringBuilderPool manages reusable string builders
type StringBuilderPool struct {
	pool sync.Pool
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool() *StringBuilderPool {
	return &StringBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
	}
}

// Get retrieves a string builder from the pool
func (p *StringBuilderPool) Get() *strings.Builder {
	sb := p.pool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// Put returns a string builder to the pool
func (p *StringBuilderPool) Put(sb *strings.Builder) {
	sb.Reset()
	p.pool.Put(sb)
}