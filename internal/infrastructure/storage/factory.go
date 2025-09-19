package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/reinaldosaraiva/goiotop/internal/domain/repositories"
)

// RepositoryType defines the type of repository implementation
type RepositoryType string

const (
	// MemoryRepositoryType represents an in-memory repository
	MemoryRepositoryType RepositoryType = "memory"
	// SQLiteRepositoryType represents a SQLite repository
	SQLiteRepositoryType RepositoryType = "sqlite"
	// PostgresRepositoryType represents a PostgreSQL repository
	PostgresRepositoryType RepositoryType = "postgres"
)

// RepositoryFactory creates repository instances based on configuration
type RepositoryFactory struct {
	mu          sync.RWMutex
	registry    map[RepositoryType]func(config interface{}) (repositories.MetricsRepository, error)
	instances   map[string]repositories.MetricsRepository
	defaultType RepositoryType
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	factory := &RepositoryFactory{
		registry:    make(map[RepositoryType]func(config interface{}) (repositories.MetricsRepository, error)),
		instances:   make(map[string]repositories.MetricsRepository),
		defaultType: MemoryRepositoryType,
	}

	// Register default repository types
	factory.registerDefaults()

	return factory
}

// registerDefaults registers the default repository implementations
func (f *RepositoryFactory) registerDefaults() {
	// Register memory repository
	f.Register(MemoryRepositoryType, func(config interface{}) (repositories.MetricsRepository, error) {
		var cfg MemoryRepositoryConfig
		switch c := config.(type) {
		case MemoryRepositoryConfig:
			cfg = c
		case *MemoryRepositoryConfig:
			cfg = *c
		case nil:
			cfg = DefaultMemoryRepositoryConfig()
		default:
			return nil, fmt.Errorf("invalid configuration type for memory repository: %T", config)
		}
		return NewMemoryRepository(cfg), nil
	})

	// SQLite repository would be registered here
	// f.Register(SQLiteRepositoryType, func(config interface{}) (repositories.MetricsRepository, error) {
	//     // Implementation for SQLite repository
	//     return nil, fmt.Errorf("SQLite repository not yet implemented")
	// })

	// PostgreSQL repository would be registered here
	// f.Register(PostgresRepositoryType, func(config interface{}) (repositories.MetricsRepository, error) {
	//     // Implementation for PostgreSQL repository
	//     return nil, fmt.Errorf("PostgreSQL repository not yet implemented")
	// })
}

// Register registers a new repository type with its factory function
func (f *RepositoryFactory) Register(repoType RepositoryType, factory func(config interface{}) (repositories.MetricsRepository, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registry[repoType] = factory
}

// Create creates a new repository instance based on the specified type and configuration
func (f *RepositoryFactory) Create(repoType RepositoryType, config interface{}) (repositories.MetricsRepository, error) {
	f.mu.RLock()
	factory, exists := f.registry[repoType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("repository type %s not registered", repoType)
	}

	repo, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Initialize the repository
	if err := repo.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	return repo, nil
}

// CreateWithName creates a named repository instance that can be retrieved later
func (f *RepositoryFactory) CreateWithName(name string, repoType RepositoryType, config interface{}) (repositories.MetricsRepository, error) {
	repo, err := f.Create(repoType, config)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.instances[name] = repo
	f.mu.Unlock()

	return repo, nil
}

// Get retrieves a named repository instance
func (f *RepositoryFactory) Get(name string) (repositories.MetricsRepository, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	repo, exists := f.instances[name]
	return repo, exists
}

// GetOrCreate retrieves a named repository or creates it if it doesn't exist
func (f *RepositoryFactory) GetOrCreate(name string, repoType RepositoryType, config interface{}) (repositories.MetricsRepository, error) {
	if repo, exists := f.Get(name); exists {
		return repo, nil
	}
	return f.CreateWithName(name, repoType, config)
}

// CreateDefault creates a repository with default configuration
func (f *RepositoryFactory) CreateDefault() (repositories.MetricsRepository, error) {
	return f.Create(f.defaultType, nil)
}

// SetDefaultType sets the default repository type
func (f *RepositoryFactory) SetDefaultType(repoType RepositoryType) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultType = repoType
}

// ListTypes returns a list of registered repository types
func (f *RepositoryFactory) ListTypes() []RepositoryType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]RepositoryType, 0, len(f.registry))
	for t := range f.registry {
		types = append(types, t)
	}
	return types
}

// CloseAll closes all managed repository instances
func (f *RepositoryFactory) CloseAll(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var lastErr error
	for name, repo := range f.instances {
		if err := repo.Close(ctx); err != nil {
			lastErr = fmt.Errorf("failed to close repository %s: %w", name, err)
		}
		delete(f.instances, name)
	}

	return lastErr
}

// HealthCheckAll performs health checks on all managed repositories
func (f *RepositoryFactory) HealthCheckAll(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]error)
	for name, repo := range f.instances {
		results[name] = repo.HealthCheck(ctx)
	}

	return results
}

// RepositoryConfig is a generic configuration interface for repositories
type RepositoryConfig struct {
	Type   RepositoryType `json:"type" yaml:"type"`
	Config interface{}    `json:"config" yaml:"config"`
}

// CreateFromConfig creates a repository from a generic configuration
func (f *RepositoryFactory) CreateFromConfig(cfg RepositoryConfig) (repositories.MetricsRepository, error) {
	return f.Create(cfg.Type, cfg.Config)
}

// Registry is the global repository factory instance
var Registry = NewRepositoryFactory()

// CreateRepository is a convenience function to create a repository using the global registry
func CreateRepository(repoType RepositoryType, config interface{}) (repositories.MetricsRepository, error) {
	return Registry.Create(repoType, config)
}

// CreateDefaultRepository is a convenience function to create a default repository
func CreateDefaultRepository() (repositories.MetricsRepository, error) {
	return Registry.CreateDefault()
}