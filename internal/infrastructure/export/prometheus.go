package export

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/sirupsen/logrus"
)

// PrometheusExporter exports metrics to Prometheus
type PrometheusExporter struct {
	mu       sync.RWMutex
	registry *prometheus.Registry
	server   *http.Server
	config   PrometheusConfig

	// Previous values for delta calculation
	prevProcessIO map[string]struct {
		ReadBytes  uint64
		WriteBytes uint64
	}
	prevDiskIO map[string]struct {
		ReadBytes  uint64
		WriteBytes uint64
		ReadCount  uint64
		WriteCount uint64
		ReadTime   uint64
		WriteTime  uint64
	}

	// System metrics
	cpuUsage      *prometheus.GaugeVec
	memoryUsage   *prometheus.GaugeVec
	diskUsage     *prometheus.GaugeVec
	loadAverage   *prometheus.GaugeVec
	temperature   *prometheus.GaugeVec
	fanSpeed      *prometheus.GaugeVec
	powerUsage    *prometheus.GaugeVec

	// Process metrics
	processCount      prometheus.Gauge
	processCPU        *prometheus.GaugeVec
	processMemory     *prometheus.GaugeVec
	processThreads    *prometheus.GaugeVec
	processOpenFiles  *prometheus.GaugeVec
	processIORead     *prometheus.CounterVec
	processIOWrite    *prometheus.CounterVec

	// CPU pressure metrics
	cpuPressureSome10  prometheus.Gauge
	cpuPressureSome60  prometheus.Gauge
	cpuPressureSome300 prometheus.Gauge
	cpuPressureFull10  prometheus.Gauge
	cpuPressureFull60  prometheus.Gauge
	cpuPressureFull300 prometheus.Gauge

	// Disk I/O metrics
	diskReadBytes    *prometheus.CounterVec
	diskWriteBytes   *prometheus.CounterVec
	diskReadCount    *prometheus.CounterVec
	diskWriteCount   *prometheus.CounterVec
	diskReadTime     *prometheus.CounterVec
	diskWriteTime    *prometheus.CounterVec
	diskQueueLength  *prometheus.GaugeVec
	diskUtilization  *prometheus.GaugeVec

	// Collection statistics
	collectionDuration   prometheus.Histogram
	collectionErrors     prometheus.Counter
	lastCollectionTime   prometheus.Gauge
	metricsExported      prometheus.Counter

	logger *logrus.Logger
}

// PrometheusConfig holds configuration for the Prometheus exporter
type PrometheusConfig struct {
	Enabled         bool          `json:"enabled" yaml:"enabled"`
	Port            int           `json:"port" yaml:"port"`
	Path            string        `json:"path" yaml:"path"`
	UpdateInterval  time.Duration `json:"updateInterval" yaml:"updateInterval"`
	IncludeProcesses bool         `json:"includeProcesses" yaml:"includeProcesses"`
	ProcessLimit    int           `json:"processLimit" yaml:"processLimit"`
}

// DefaultPrometheusConfig returns default Prometheus configuration
func DefaultPrometheusConfig() PrometheusConfig {
	return PrometheusConfig{
		Enabled:         true,
		Port:            9090,
		Path:            "/metrics",
		UpdateInterval:  10 * time.Second,
		IncludeProcesses: true,
		ProcessLimit:    100,
	}
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(config PrometheusConfig, logger *logrus.Logger) *PrometheusExporter {
	registry := prometheus.NewRegistry()

	exporter := &PrometheusExporter{
		registry: registry,
		config:   config,
		logger:   logger,
		prevProcessIO: make(map[string]struct {
			ReadBytes  uint64
			WriteBytes uint64
		}),
		prevDiskIO: make(map[string]struct {
			ReadBytes  uint64
			WriteBytes uint64
			ReadCount  uint64
			WriteCount uint64
			ReadTime   uint64
			WriteTime  uint64
		}),
	}

	// Initialize metrics
	exporter.initializeMetrics()

	// Register metrics with registry
	exporter.registerMetrics()

	return exporter
}

// initializeMetrics initializes all Prometheus metrics
func (e *PrometheusExporter) initializeMetrics() {
	// System metrics
	e.cpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_cpu_usage_percent",
			Help: "CPU usage percentage per core",
		},
		[]string{"core"},
	)

	e.memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
		[]string{"type"}, // used, free, cached, available
	)

	e.diskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_disk_usage_bytes",
			Help: "Disk usage in bytes",
		},
		[]string{"path", "type"}, // used, free, total
	)

	e.loadAverage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_load_average",
			Help: "System load average",
		},
		[]string{"period"}, // 1min, 5min, 15min
	)

	e.temperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_temperature_celsius",
			Help: "System temperature in Celsius",
		},
		[]string{"sensor"},
	)

	e.fanSpeed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_fan_speed_rpm",
			Help: "Fan speed in RPM",
		},
		[]string{"fan"},
	)

	e.powerUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_system_power_usage_watts",
			Help: "Power usage in watts",
		},
		[]string{"component"},
	)

	// Process metrics
	e.processCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_process_count",
			Help: "Total number of processes",
		},
	)

	e.processCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_process_cpu_percent",
			Help: "Process CPU usage percentage",
		},
		[]string{"pid", "name", "user"},
	)

	e.processMemory = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_process_memory_bytes",
			Help: "Process memory usage in bytes",
		},
		[]string{"pid", "name", "user", "type"}, // rss, vms, shared
	)

	e.processThreads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_process_threads",
			Help: "Number of threads per process",
		},
		[]string{"pid", "name", "user"},
	)

	e.processOpenFiles = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_process_open_files",
			Help: "Number of open files per process",
		},
		[]string{"pid", "name", "user"},
	)

	e.processIORead = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_process_io_read_bytes_total",
			Help: "Total bytes read by process",
		},
		[]string{"pid", "name", "user"},
	)

	e.processIOWrite = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_process_io_write_bytes_total",
			Help: "Total bytes written by process",
		},
		[]string{"pid", "name", "user"},
	)

	// CPU pressure metrics
	e.cpuPressureSome10 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_some_10s",
			Help: "CPU pressure some avg10",
		},
	)

	e.cpuPressureSome60 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_some_60s",
			Help: "CPU pressure some avg60",
		},
	)

	e.cpuPressureSome300 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_some_300s",
			Help: "CPU pressure some avg300",
		},
	)

	e.cpuPressureFull10 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_full_10s",
			Help: "CPU pressure full avg10",
		},
	)

	e.cpuPressureFull60 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_full_60s",
			Help: "CPU pressure full avg60",
		},
	)

	e.cpuPressureFull300 = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_cpu_pressure_full_300s",
			Help: "CPU pressure full avg300",
		},
	)

	// Disk I/O metrics
	e.diskReadBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_read_bytes_total",
			Help: "Total bytes read from disk",
		},
		[]string{"device"},
	)

	e.diskWriteBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_write_bytes_total",
			Help: "Total bytes written to disk",
		},
		[]string{"device"},
	)

	e.diskReadCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_read_operations_total",
			Help: "Total read operations",
		},
		[]string{"device"},
	)

	e.diskWriteCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_write_operations_total",
			Help: "Total write operations",
		},
		[]string{"device"},
	)

	e.diskReadTime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_read_time_seconds_total",
			Help: "Total time spent reading from disk",
		},
		[]string{"device"},
	)

	e.diskWriteTime = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goiotop_disk_write_time_seconds_total",
			Help: "Total time spent writing to disk",
		},
		[]string{"device"},
	)

	e.diskQueueLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_disk_queue_length",
			Help: "Disk queue length",
		},
		[]string{"device"},
	)

	e.diskUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goiotop_disk_utilization_percent",
			Help: "Disk utilization percentage",
		},
		[]string{"device"},
	)

	// Collection statistics
	e.collectionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "goiotop_collection_duration_seconds",
			Help:    "Duration of metrics collection",
			Buckets: prometheus.DefBuckets,
		},
	)

	e.collectionErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "goiotop_collection_errors_total",
			Help: "Total number of collection errors",
		},
	)

	e.lastCollectionTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goiotop_last_collection_timestamp",
			Help: "Timestamp of last collection",
		},
	)

	e.metricsExported = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "goiotop_metrics_exported_total",
			Help: "Total number of metrics exported",
		},
	)
}

// registerMetrics registers all metrics with the Prometheus registry
func (e *PrometheusExporter) registerMetrics() {
	// Register standard Go process and runtime collectors
	e.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	e.registry.MustRegister(prometheus.NewGoCollector())

	// System metrics
	e.registry.MustRegister(e.cpuUsage)
	e.registry.MustRegister(e.memoryUsage)
	e.registry.MustRegister(e.diskUsage)
	e.registry.MustRegister(e.loadAverage)
	e.registry.MustRegister(e.temperature)
	e.registry.MustRegister(e.fanSpeed)
	e.registry.MustRegister(e.powerUsage)

	// Process metrics
	if e.config.IncludeProcesses {
		e.registry.MustRegister(e.processCount)
		e.registry.MustRegister(e.processCPU)
		e.registry.MustRegister(e.processMemory)
		e.registry.MustRegister(e.processThreads)
		e.registry.MustRegister(e.processOpenFiles)
		e.registry.MustRegister(e.processIORead)
		e.registry.MustRegister(e.processIOWrite)
	}

	// CPU pressure metrics
	e.registry.MustRegister(e.cpuPressureSome10)
	e.registry.MustRegister(e.cpuPressureSome60)
	e.registry.MustRegister(e.cpuPressureSome300)
	e.registry.MustRegister(e.cpuPressureFull10)
	e.registry.MustRegister(e.cpuPressureFull60)
	e.registry.MustRegister(e.cpuPressureFull300)

	// Disk I/O metrics
	e.registry.MustRegister(e.diskReadBytes)
	e.registry.MustRegister(e.diskWriteBytes)
	e.registry.MustRegister(e.diskReadCount)
	e.registry.MustRegister(e.diskWriteCount)
	e.registry.MustRegister(e.diskReadTime)
	e.registry.MustRegister(e.diskWriteTime)
	e.registry.MustRegister(e.diskQueueLength)
	e.registry.MustRegister(e.diskUtilization)

	// Collection statistics
	e.registry.MustRegister(e.collectionDuration)
	e.registry.MustRegister(e.collectionErrors)
	e.registry.MustRegister(e.lastCollectionTime)
	e.registry.MustRegister(e.metricsExported)

	// Register Go runtime metrics
	e.registry.MustRegister(prometheus.NewGoCollector())
}

// Start starts the Prometheus HTTP server
func (e *PrometheusExporter) Start(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(e.config.Path, promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		ErrorLog: logrus.StandardLogger(),
	}))

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	e.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", e.config.Port),
		Handler: mux,
	}

	// Start server in background with context monitoring
	go func() {
		e.logger.WithFields(logrus.Fields{
			"port": e.config.Port,
			"path": e.config.Path,
		}).Info("Starting Prometheus metrics server")

		// Start the HTTP server
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- e.server.ListenAndServe()
		}()

		// Wait for either context cancellation or server error
		select {
		case <-ctx.Done():
			e.logger.Info("Context cancelled, initiating Prometheus server shutdown")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := e.server.Shutdown(shutdownCtx); err != nil {
				e.logger.WithError(err).Error("Error during Prometheus server shutdown")
			}
		case err := <-serverErr:
			if err != nil && err != http.ErrServerClosed {
				e.logger.WithError(err).Error("Prometheus server error")
			}
		}
	}()

	return nil
}

// Stop stops the Prometheus HTTP server
func (e *PrometheusExporter) Stop(ctx context.Context) error {
	if e.server == nil {
		return nil
	}

	e.logger.Info("Stopping Prometheus metrics server")
	return e.server.Shutdown(ctx)
}

// UpdateSystemMetrics updates system metrics in Prometheus
func (e *PrometheusExporter) UpdateSystemMetrics(metrics *entities.SystemMetrics) {
	if metrics == nil {
		return
	}

	start := time.Now()

	// Update CPU metrics
	for i, usage := range metrics.CPUUsage.PerCoreUsage() {
		e.cpuUsage.WithLabelValues(fmt.Sprintf("cpu%d", i)).Set(usage)
	}

	// Update memory metrics
	e.memoryUsage.WithLabelValues("used").Set(float64(metrics.MemoryUsage.Used()))
	e.memoryUsage.WithLabelValues("free").Set(float64(metrics.MemoryUsage.Available()))
	e.memoryUsage.WithLabelValues("cached").Set(float64(metrics.MemoryUsage.Cached()))
	e.memoryUsage.WithLabelValues("total").Set(float64(metrics.MemoryUsage.Total()))

	// Update load average
	loads := metrics.LoadAverage.Values()
	e.loadAverage.WithLabelValues("1min").Set(loads[0])
	e.loadAverage.WithLabelValues("5min").Set(loads[1])
	e.loadAverage.WithLabelValues("15min").Set(loads[2])

	// Update collection statistics
	e.lastCollectionTime.SetToCurrentTime()
	e.collectionDuration.Observe(time.Since(start).Seconds())
	e.metricsExported.Inc()
}

// UpdateProcessMetrics updates process metrics in Prometheus
func (e *PrometheusExporter) UpdateProcessMetrics(metrics []*entities.ProcessMetrics) {
	if !e.config.IncludeProcesses || metrics == nil {
		return
	}

	// Reset process metrics to clear stale data
	e.processCPU.Reset()
	e.processMemory.Reset()
	e.processThreads.Reset()
	e.processOpenFiles.Reset()

	// Update process count
	e.processCount.Set(float64(len(metrics)))

	// Update individual process metrics (limit to configured maximum)
	limit := len(metrics)
	if e.config.ProcessLimit > 0 && limit > e.config.ProcessLimit {
		limit = e.config.ProcessLimit
	}

	for i := 0; i < limit; i++ {
		p := metrics[i]
		pidStr := fmt.Sprintf("%d", p.PID.Value())
		name := p.Name.Value()
		user := p.User.Value()

		e.processCPU.WithLabelValues(pidStr, name, user).Set(p.CPUPercent.Value())
		e.processMemory.WithLabelValues(pidStr, name, user, "rss").Set(float64(p.MemoryRSS.Value()))
		e.processMemory.WithLabelValues(pidStr, name, user, "vms").Set(float64(p.MemoryVMS.Value()))
		e.processThreads.WithLabelValues(pidStr, name, user).Set(float64(p.NumThreads.Value()))
		e.processOpenFiles.WithLabelValues(pidStr, name, user).Set(float64(p.OpenFiles.Value()))

		// Update I/O counters (only add the delta)
		if p.IOCounters != nil {
			readBytes := p.IOCounters.ReadBytes()
			writeBytes := p.IOCounters.WriteBytes()

			// Calculate deltas
			if prev, exists := e.prevProcessIO[pidStr]; exists {
				if readBytes >= prev.ReadBytes {
					e.processIORead.WithLabelValues(pidStr, name, user).Add(float64(readBytes - prev.ReadBytes))
				}
				if writeBytes >= prev.WriteBytes {
					e.processIOWrite.WithLabelValues(pidStr, name, user).Add(float64(writeBytes - prev.WriteBytes))
				}
			}

			// Store current values for next update
			e.prevProcessIO[pidStr] = struct {
				ReadBytes  uint64
				WriteBytes uint64
			}{
				ReadBytes:  readBytes,
				WriteBytes: writeBytes,
			}
		}
	}

	e.metricsExported.Add(float64(limit))
}

// UpdateCPUPressure updates CPU pressure metrics in Prometheus
func (e *PrometheusExporter) UpdateCPUPressure(pressure *entities.CPUPressure) {
	if pressure == nil {
		return
	}

	e.cpuPressureSome10.Set(pressure.SomeAvg10.Value())
	e.cpuPressureSome60.Set(pressure.SomeAvg60.Value())
	e.cpuPressureSome300.Set(pressure.SomeAvg300.Value())
	e.cpuPressureFull10.Set(pressure.FullAvg10.Value())
	e.cpuPressureFull60.Set(pressure.FullAvg60.Value())
	e.cpuPressureFull300.Set(pressure.FullAvg300.Value())

	e.metricsExported.Inc()
}

// UpdateDiskIO updates disk I/O metrics in Prometheus
func (e *PrometheusExporter) UpdateDiskIO(diskIOs []*entities.DiskIO) {
	if diskIOs == nil {
		return
	}

	// Reset disk metrics to clear stale data
	e.diskQueueLength.Reset()
	e.diskUtilization.Reset()

	for _, d := range diskIOs {
		device := d.Device.String()

		// Update counters (only add the delta)
		readBytes := uint64(d.ReadBytes.Value())
		writeBytes := uint64(d.WriteBytes.Value())
		readCount := uint64(d.ReadCount.Value())
		writeCount := uint64(d.WriteCount.Value())

		// Calculate deltas
		if prev, exists := e.prevDiskIO[device]; exists {
			if readBytes >= prev.ReadBytes {
				e.diskReadBytes.WithLabelValues(device).Add(float64(readBytes - prev.ReadBytes))
			}
			if writeBytes >= prev.WriteBytes {
				e.diskWriteBytes.WithLabelValues(device).Add(float64(writeBytes - prev.WriteBytes))
			}
			if readCount >= prev.ReadCount {
				e.diskReadCount.WithLabelValues(device).Add(float64(readCount - prev.ReadCount))
			}
			if writeCount >= prev.WriteCount {
				e.diskWriteCount.WithLabelValues(device).Add(float64(writeCount - prev.WriteCount))
			}
		}

		// Store current values for next update
		e.prevDiskIO[device] = struct {
			ReadBytes  uint64
			WriteBytes uint64
			ReadCount  uint64
			WriteCount uint64
			ReadTime   uint64
			WriteTime  uint64
		}{
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
			ReadCount:  readCount,
			WriteCount: writeCount,
			ReadTime:   0, // Will be updated if we add time tracking
			WriteTime:  0,
		}
		e.diskReadTime.WithLabelValues(device).Add(float64(d.ReadTime.Value()) / 1000.0) // Convert ms to seconds
		e.diskWriteTime.WithLabelValues(device).Add(float64(d.WriteTime.Value()) / 1000.0)

		// Update gauges
		e.diskQueueLength.WithLabelValues(device).Set(float64(d.QueueLength.Value()))
		e.diskUtilization.WithLabelValues(device).Set(d.Utilization.Value())
	}

	e.metricsExported.Add(float64(len(diskIOs)))
}

// RecordError records a collection error
func (e *PrometheusExporter) RecordError(err error) {
	e.collectionErrors.Inc()
	if e.logger != nil {
		e.logger.WithError(err).Error("Collection error")
	}
}

// GetMetricsHandler returns the HTTP handler for metrics
func (e *PrometheusExporter) GetMetricsHandler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		ErrorLog: logrus.StandardLogger(),
	})
}

// IsEnabled returns whether the exporter is enabled
func (e *PrometheusExporter) IsEnabled() bool {
	return e.config.Enabled
}

// GetConfig returns the exporter configuration
func (e *PrometheusExporter) GetConfig() PrometheusConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// UpdateConfig updates the exporter configuration
func (e *PrometheusExporter) UpdateConfig(config PrometheusConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// If port or path changed, need to restart server
	if e.config.Port != config.Port || e.config.Path != config.Path {
		if e.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := e.Stop(ctx); err != nil {
				return err
			}
		}

		e.config = config

		if config.Enabled {
			return e.Start(context.Background())
		}
	} else {
		e.config = config
	}

	return nil
}