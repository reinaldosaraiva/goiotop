package collector

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
)

// CollectorFactory is a function that creates a MetricsCollector
type CollectorFactory func() (services.MetricsCollector, error)

// Registry manages platform-specific collectors
type Registry struct {
	mu              sync.RWMutex
	collectors      map[string]CollectorFactory
	defaultPlatform string
}

// globalRegistry is the singleton registry instance
var globalRegistry = &Registry{
	collectors: make(map[string]CollectorFactory),
}

// Register registers a collector factory for a platform
func Register(platform string, factory CollectorFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.collectors[platform] = factory
}

// SetDefault sets the default platform
func SetDefault(platform string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.defaultPlatform = platform
}

// Get returns a collector for the specified platform
func Get(platform string) (services.MetricsCollector, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	factory, exists := globalRegistry.collectors[platform]
	if !exists {
		return nil, fmt.Errorf("no collector registered for platform: %s", platform)
	}

	return factory()
}

// GetDefault returns the collector for the current platform
func GetDefault() (services.MetricsCollector, error) {
	platform := runtime.GOOS

	// Check if a default is explicitly set
	globalRegistry.mu.RLock()
	if globalRegistry.defaultPlatform != "" {
		platform = globalRegistry.defaultPlatform
	}
	globalRegistry.mu.RUnlock()

	return Get(platform)
}

// ListPlatforms returns all registered platforms
func ListPlatforms() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	platforms := make([]string, 0, len(globalRegistry.collectors))
	for platform := range globalRegistry.collectors {
		platforms = append(platforms, platform)
	}

	return platforms
}

// IsSupported checks if a platform is supported
func IsSupported(platform string) bool {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	_, exists := globalRegistry.collectors[platform]
	return exists
}

// CurrentPlatform returns the current runtime platform
func CurrentPlatform() string {
	return runtime.GOOS
}

// AutoDetect creates a collector for the current platform
func AutoDetect() (services.MetricsCollector, error) {
	platform := runtime.GOOS

	collector, err := Get(platform)
	if err != nil {
		return nil, fmt.Errorf("platform %s is not supported: %w", platform, err)
	}

	return collector, nil
}

// Clear removes all registered collectors (mainly for testing)
func Clear() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.collectors = make(map[string]CollectorFactory)
	globalRegistry.defaultPlatform = ""
}

// CollectorInfo provides information about a registered collector
type CollectorInfo struct {
	Platform     string
	IsDefault    bool
	IsCurrentOS  bool
	IsRegistered bool
}

// GetInfo returns information about a collector
func GetInfo(platform string) CollectorInfo {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	_, registered := globalRegistry.collectors[platform]

	return CollectorInfo{
		Platform:     platform,
		IsDefault:    platform == globalRegistry.defaultPlatform,
		IsCurrentOS:  platform == runtime.GOOS,
		IsRegistered: registered,
	}
}

// GetAllInfo returns information about all possible collectors
func GetAllInfo() []CollectorInfo {
	platforms := []string{"linux", "darwin", "windows", "freebsd"}
	info := make([]CollectorInfo, 0, len(platforms))

	for _, platform := range platforms {
		info = append(info, GetInfo(platform))
	}

	return info
}