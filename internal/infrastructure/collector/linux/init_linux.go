//go:build linux

package linux

import (
	"github.com/reinaldosaraiva/goiotop/internal/domain/services"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/collector"
)

func init() {
	// Register the Linux collector factory with the registry
	collector.Register("linux", func() (services.MetricsCollector, error) {
		return NewLinuxMetricsCollector()
	})
}