package optimization

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sirupsen/logrus"
)

// CacheEntry represents a cached value with metadata
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	AccessTime time.Time
	HitCount   atomic.Int64
}

// IsExpired checks if the cache entry is expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// MultiLevelCache implements a multi-level caching system
type MultiLevelCache struct {
	mu sync.RWMutex

	// L1 cache (hot data, small)
	l1Cache *lru.Cache[string, *CacheEntry]
	l1TTL   time.Duration

	// L2 cache (warm data, larger)
	l2Cache *lru.Cache[string, *CacheEntry]
	l2TTL   time.Duration

	// Statistics
	stats CacheStats

	// Configuration
	config CacheConfig

	// Logger
	logger *logrus.Logger
}

// CacheStats tracks cache performance statistics
type CacheStats struct {
	L1Hits      atomic.Int64
	L1Misses    atomic.Int64
	L2Hits      atomic.Int64
	L2Misses    atomic.Int64
	Evictions   atomic.Int64
	Sets        atomic.Int64
	Deletes     atomic.Int64
	TotalBytes  atomic.Int64
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	L1Size          int           `json:"l1Size" yaml:"l1Size"`
	L2Size          int           `json:"l2Size" yaml:"l2Size"`
	L1TTL           time.Duration `json:"l1TTL" yaml:"l1TTL"`
	L2TTL           time.Duration `json:"l2TTL" yaml:"l2TTL"`
	WarmupEnabled   bool          `json:"warmupEnabled" yaml:"warmupEnabled"`
	CleanupInterval time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
	MaxMemoryBytes  int64         `json:"maxMemoryBytes" yaml:"maxMemoryBytes"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		L1Size:          100,
		L2Size:          1000,
		L1TTL:           10 * time.Second,
		L2TTL:           60 * time.Second,
		WarmupEnabled:   true,
		CleanupInterval: 5 * time.Minute,
		MaxMemoryBytes:  100 * 1024 * 1024, // 100MB
	}
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(config CacheConfig, logger *logrus.Logger) (*MultiLevelCache, error) {
	cache := &MultiLevelCache{
		l1TTL:   config.L1TTL,
		l2TTL:   config.L2TTL,
		config:  config,
		logger:  logger,
	}

	// Create L2 cache first (no eviction callback needed)
	l2Cache, err := lru.New[string, *CacheEntry](config.L2Size)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 cache: %w", err)
	}
	cache.l2Cache = l2Cache

	// Create L1 cache with eviction callback to move to L2
	l1Cache, err := lru.NewWithEvict[string, *CacheEntry](config.L1Size,
		func(key string, value *CacheEntry) {
			// Move evicted item to L2 if it's still valid
			if !value.IsExpired() {
				cache.l2Cache.Add(key, value)
				cache.stats.Evictions.Add(1)
			}
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}
	cache.l1Cache = l1Cache


	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go cache.startCleanup()
	}

	return cache, nil
}

// Get retrieves a value from the cache
func (c *MultiLevelCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check L1 cache
	if entry, ok := c.l1Cache.Get(key); ok {
		if !entry.IsExpired() {
			c.stats.L1Hits.Add(1)
			entry.HitCount.Add(1)
			entry.AccessTime = time.Now()
			return entry.Value, true
		}
		// Remove expired entry
		c.l1Cache.Remove(key)
	}
	c.stats.L1Misses.Add(1)

	// Check L2 cache
	if entry, ok := c.l2Cache.Get(key); ok {
		if !entry.IsExpired() {
			c.stats.L2Hits.Add(1)
			entry.HitCount.Add(1)
			entry.AccessTime = time.Now()

			// Promote to L1 cache if frequently accessed
			if entry.HitCount.Load() > 3 {
				c.promoteToL1(key, entry)
			}

			return entry.Value, true
		}
		// Remove expired entry
		c.l2Cache.Remove(key)
	}
	c.stats.L2Misses.Add(1)

	return nil, false
}

// Set stores a value in the cache with the default TTL
func (c *MultiLevelCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.l1TTL)
}

// SetWithTTL stores a value in the cache with a specific TTL
func (c *MultiLevelCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		AccessTime: time.Now(),
	}

	// Add to L1 cache (eviction callback will handle moving to L2)
	c.l1Cache.Add(key, entry)

	c.stats.Sets.Add(1)
}

// Delete removes a value from the cache
func (c *MultiLevelCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.l1Cache.Remove(key)
	c.l2Cache.Remove(key)
	c.stats.Deletes.Add(1)
}

// Clear removes all values from the cache
func (c *MultiLevelCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.l1Cache.Purge()
	c.l2Cache.Purge()
}

// promoteToL1 promotes an entry from L2 to L1 cache
func (c *MultiLevelCache) promoteToL1(key string, entry *CacheEntry) {
	// Create a new entry for L1 with shorter TTL
	l1Entry := &CacheEntry{
		Value:      entry.Value,
		ExpiresAt:  time.Now().Add(c.l1TTL),
		AccessTime: entry.AccessTime,
		HitCount:   atomic.Int64{},
	}
	l1Entry.HitCount.Store(entry.HitCount.Load())

	// Promote from L2 to L1 (eviction callback will handle any evicted item)
	c.l1Cache.Add(key, l1Entry)
}

// startCleanup runs periodic cleanup of expired entries
func (c *MultiLevelCache) startCleanup() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries from both cache levels
func (c *MultiLevelCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean L1 cache
	for _, key := range c.l1Cache.Keys() {
		if entry, ok := c.l1Cache.Get(key); ok && entry.IsExpired() {
			c.l1Cache.Remove(key)
		}
	}

	// Clean L2 cache
	for _, key := range c.l2Cache.Keys() {
		if entry, ok := c.l2Cache.Get(key); ok && entry.IsExpired() {
			c.l2Cache.Remove(key)
		}
	}
}

// GetStatistics returns cache statistics
func (c *MultiLevelCache) GetStatistics() CacheStats {
	stats := CacheStats{}
	stats.L1Hits.Store(c.stats.L1Hits.Load())
	stats.L1Misses.Store(c.stats.L1Misses.Load())
	stats.L2Hits.Store(c.stats.L2Hits.Load())
	stats.L2Misses.Store(c.stats.L2Misses.Load())
	stats.Evictions.Store(c.stats.Evictions.Load())
	stats.Sets.Store(c.stats.Sets.Load())
	stats.Deletes.Store(c.stats.Deletes.Load())
	return stats
}

// GetHitRate returns the cache hit rate
func (c *MultiLevelCache) GetHitRate() float64 {
	hits := c.stats.L1Hits.Load() + c.stats.L2Hits.Load()
	misses := c.stats.L1Misses.Load() + c.stats.L2Misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// Warmup preloads the cache with initial data
func (c *MultiLevelCache) Warmup(ctx context.Context, loader func() map[string]interface{}) error {
	if !c.config.WarmupEnabled {
		return nil
	}

	data := loader()
	for key, value := range data {
		c.Set(key, value)
	}

	c.logger.WithField("items", len(data)).Info("Cache warmup completed")
	return nil
}

// SpecializedCaches provides optimized caching for specific data types

// ProcessCache caches process information
type ProcessCache struct {
	cache   *MultiLevelCache
	logger  *logrus.Logger
	keyFunc func(int32) string
}

// NewProcessCache creates a new process cache
func NewProcessCache(config CacheConfig, logger *logrus.Logger) (*ProcessCache, error) {
	cache, err := NewMultiLevelCache(config, logger)
	if err != nil {
		return nil, err
	}

	return &ProcessCache{
		cache:   cache,
		logger:  logger,
		keyFunc: func(pid int32) string { return fmt.Sprintf("proc_%d", pid) },
	}, nil
}

// Get retrieves process information from the cache
func (pc *ProcessCache) Get(pid int32) (interface{}, bool) {
	return pc.cache.Get(pc.keyFunc(pid))
}

// Set stores process information in the cache
func (pc *ProcessCache) Set(pid int32, value interface{}) {
	pc.cache.Set(pc.keyFunc(pid), value)
}

// Delete removes process information from the cache
func (pc *ProcessCache) Delete(pid int32) {
	pc.cache.Delete(pc.keyFunc(pid))
}

// SystemInfoCache caches system information
type SystemInfoCache struct {
	cache  *MultiLevelCache
	logger *logrus.Logger
}

// NewSystemInfoCache creates a new system info cache
func NewSystemInfoCache(config CacheConfig, logger *logrus.Logger) (*SystemInfoCache, error) {
	// System info doesn't change frequently, so use longer TTLs
	config.L1TTL = 30 * time.Second
	config.L2TTL = 5 * time.Minute

	cache, err := NewMultiLevelCache(config, logger)
	if err != nil {
		return nil, err
	}

	return &SystemInfoCache{
		cache:  cache,
		logger: logger,
	}, nil
}

// GetCPUInfo retrieves cached CPU information
func (s *SystemInfoCache) GetCPUInfo() (interface{}, bool) {
	return s.cache.Get("cpu_info")
}

// SetCPUInfo stores CPU information in the cache
func (s *SystemInfoCache) SetCPUInfo(info interface{}) {
	s.cache.SetWithTTL("cpu_info", info, 5*time.Minute)
}

// GetMemoryInfo retrieves cached memory information
func (s *SystemInfoCache) GetMemoryInfo() (interface{}, bool) {
	return s.cache.Get("memory_info")
}

// SetMemoryInfo stores memory information in the cache
func (s *SystemInfoCache) SetMemoryInfo(info interface{}) {
	s.cache.SetWithTTL("memory_info", info, 1*time.Minute)
}

// GetDiskInfo retrieves cached disk information
func (s *SystemInfoCache) GetDiskInfo() (interface{}, bool) {
	return s.cache.Get("disk_info")
}

// SetDiskInfo stores disk information in the cache
func (s *SystemInfoCache) SetDiskInfo(info interface{}) {
	s.cache.SetWithTTL("disk_info", info, 5*time.Minute)
}

// MetricsCalculationCache caches expensive metric calculations
type MetricsCalculationCache struct {
	cache  *MultiLevelCache
	logger *logrus.Logger
}

// NewMetricsCalculationCache creates a new metrics calculation cache
func NewMetricsCalculationCache(config CacheConfig, logger *logrus.Logger) (*MetricsCalculationCache, error) {
	// Calculations are valid for short periods
	config.L1TTL = 5 * time.Second
	config.L2TTL = 30 * time.Second

	cache, err := NewMultiLevelCache(config, logger)
	if err != nil {
		return nil, err
	}

	return &MetricsCalculationCache{
		cache:  cache,
		logger: logger,
	}, nil
}

// GetAggregation retrieves cached aggregation result
func (m *MetricsCalculationCache) GetAggregation(key string) (interface{}, bool) {
	return m.cache.Get(fmt.Sprintf("agg_%s", key))
}

// SetAggregation stores aggregation result in the cache
func (m *MetricsCalculationCache) SetAggregation(key string, value interface{}) {
	m.cache.Set(fmt.Sprintf("agg_%s", key), value)
}

// GetTopProcesses retrieves cached top processes list
func (m *MetricsCalculationCache) GetTopProcesses(metric string, limit int) (interface{}, bool) {
	key := fmt.Sprintf("top_%s_%d", metric, limit)
	return m.cache.Get(key)
}

// SetTopProcesses stores top processes list in the cache
func (m *MetricsCalculationCache) SetTopProcesses(metric string, limit int, value interface{}) {
	key := fmt.Sprintf("top_%s_%d", metric, limit)
	m.cache.Set(key, value)
}

// InvalidationStrategy defines cache invalidation strategies
type InvalidationStrategy interface {
	ShouldInvalidate(key string, entry *CacheEntry) bool
}

// TTLInvalidation invalidates based on time-to-live
type TTLInvalidation struct{}

func (t *TTLInvalidation) ShouldInvalidate(key string, entry *CacheEntry) bool {
	return entry.IsExpired()
}

// LRUInvalidation invalidates least recently used entries
type LRUInvalidation struct {
	maxAge time.Duration
}

func (l *LRUInvalidation) ShouldInvalidate(key string, entry *CacheEntry) bool {
	return time.Since(entry.AccessTime) > l.maxAge
}

// SizeInvalidation invalidates when cache exceeds size limit
type SizeInvalidation struct {
	maxBytes int64
}

func (s *SizeInvalidation) ShouldInvalidate(key string, entry *CacheEntry) bool {
	// Implementation would check total cache size
	return false
}