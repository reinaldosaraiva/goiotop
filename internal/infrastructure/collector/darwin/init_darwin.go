//go:build darwin

package darwin

import (
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/collector"
)

func init() {
	// Register the Darwin collector factory with the registry
	collector.Register("darwin", func() (services.MetricsCollector, error) {
		return NewDarwinMetricsCollector()
	})
}