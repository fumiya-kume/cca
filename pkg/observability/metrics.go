// Package observability provides metrics collection and telemetry for ccAgents
package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/fumiya-kume/cca/pkg/performance"
)

// ApplicationMetricsCollector integrates with performance monitoring for comprehensive metrics
type ApplicationMetricsCollector struct {
	performanceMonitor *performance.PerformanceMonitor
	aggregator         *LogAggregator
	logger             *logger.Logger
	config             MetricsConfig
	registry           map[string]*MetricDefinition
	mutex              sync.RWMutex
	running            bool
	ctx                context.Context
	cancel             context.CancelFunc
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval"`
	RetentionPeriod    time.Duration `yaml:"retention_period"`
	EnablePrometheus   bool          `yaml:"enable_prometheus"`
	EnableJSONExport   bool          `yaml:"enable_json_export"`
	ExportPath         string        `yaml:"export_path"`
	MaxMetrics         int           `yaml:"max_metrics"`
	EnableSampling     bool          `yaml:"enable_sampling"`
	SamplingRate       float64       `yaml:"sampling_rate"`
}

// MetricDefinition defines a metric with metadata
type MetricDefinition struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Unit        string            `json:"unit"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   time.Time         `json:"created_at"`
	LastUpdated time.Time         `json:"last_updated"`
}

// NewApplicationMetricsCollector creates a new application metrics collector
func NewApplicationMetricsCollector(performanceMonitor *performance.PerformanceMonitor, aggregator *LogAggregator, logger *logger.Logger) *ApplicationMetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &ApplicationMetricsCollector{
		performanceMonitor: performanceMonitor,
		aggregator:         aggregator,
		logger:             logger,
		config:             DefaultMetricsConfig(),
		registry:           make(map[string]*MetricDefinition),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:            true,
		CollectionInterval: 30 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		EnablePrometheus:   false,
		EnableJSONExport:   true,
		ExportPath:         "./metrics",
		MaxMetrics:         10000,
		EnableSampling:     false,
		SamplingRate:       1.0,
	}
}

// SetConfig updates the metrics configuration
func (mc *ApplicationMetricsCollector) SetConfig(config MetricsConfig) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.config = config
}

// RegisterMetric registers a new metric definition
func (mc *ApplicationMetricsCollector) RegisterMetric(definition MetricDefinition) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.registry[definition.Name]; exists {
		return errors.ValidationError("metric already registered: " + definition.Name)
	}

	definition.CreatedAt = time.Now()
	definition.LastUpdated = time.Now()
	mc.registry[definition.Name] = &definition

	if mc.logger != nil {
		mc.logger.Debug("Registered metric (name: %s, type: %s)", definition.Name, definition.Type)
	}

	return nil
}

// Start starts the metrics collection
func (mc *ApplicationMetricsCollector) Start() error {
	mc.mutex.Lock()
	if mc.running {
		mc.mutex.Unlock()
		return errors.ValidationError("metrics collector already running")
	}
	mc.running = true
	mc.mutex.Unlock()

	// Start collection routine
	go mc.collectionLoop()

	if mc.logger != nil {
		mc.logger.Info("Metrics collection started (interval: %v)", mc.config.CollectionInterval)
	}

	return nil
}

// Stop stops the metrics collection
func (mc *ApplicationMetricsCollector) Stop() {
	mc.cancel()

	mc.mutex.Lock()
	mc.running = false
	mc.mutex.Unlock()

	if mc.logger != nil {
		mc.logger.Info("Metrics collection stopped")
	}
}

// CollectApplicationMetrics collects application-specific metrics
func (mc *ApplicationMetricsCollector) CollectApplicationMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["runtime"] = map[string]interface{}{
		"memory_alloc": m.Alloc,
		"memory_sys":   m.Sys,
		"memory_heap":  m.HeapAlloc,
		"gc_runs":      m.NumGC,
		"goroutines":   runtime.NumGoroutine(),
		"cgo_calls":    runtime.NumCgoCall(),
	}

	// Performance monitor metrics
	if mc.performanceMonitor != nil {
		if pmMetrics := mc.performanceMonitor.GetAllMetrics(); pmMetrics != nil {
			metrics["performance"] = pmMetrics
		}
	}

	// Log aggregator metrics
	if mc.aggregator != nil {
		if logStats := mc.aggregator.GetStats(); logStats != nil {
			metrics["logging"] = logStats
		}
	}

	// Custom metrics from registry
	mc.mutex.RLock()
	registryMetrics := make(map[string]*MetricDefinition)
	for k, v := range mc.registry {
		registryMetrics[k] = v
	}
	mc.mutex.RUnlock()

	if len(registryMetrics) > 0 {
		metrics["custom"] = registryMetrics
	}

	return metrics
}

// ExportMetrics exports metrics in various formats
func (mc *ApplicationMetricsCollector) ExportMetrics() error {
	metrics := mc.CollectApplicationMetrics()

	if mc.config.EnableJSONExport {
		if err := mc.exportJSON(metrics); err != nil {
			return err
		}
	}

	if mc.config.EnablePrometheus {
		if err := mc.exportPrometheus(metrics); err != nil {
			return err
		}
	}

	return nil
}

// exportJSON exports metrics to JSON format
func (mc *ApplicationMetricsCollector) exportJSON(metrics map[string]interface{}) error {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("metrics-%s.json", timestamp)
	filepath := mc.config.ExportPath + "/" + filename

	// Ensure directory exists
	if err := ensureDir(mc.config.ExportPath); err != nil {
		return err
	}

	// Export data
	data, err := json.MarshalIndent(map[string]interface{}{
		"timestamp": time.Now(),
		"metrics":   metrics,
	}, "", "  ")
	if err != nil {
		return err
	}

	if err := writeFile(filepath, data); err != nil {
		return err
	}

	if mc.logger != nil {
		mc.logger.Debug("Exported metrics to JSON (file: %s)", filepath)
	}

	return nil
}

// exportPrometheus exports metrics in Prometheus format
func (mc *ApplicationMetricsCollector) exportPrometheus(metrics map[string]interface{}) error {
	var output []string
	timestamp := time.Now().Unix()

	// Convert metrics to Prometheus format
	for category, data := range metrics {
		if dataMap, ok := data.(map[string]interface{}); ok {
			for name, value := range dataMap {
				promName := fmt.Sprintf("ccagents_%s_%s", category, name)
				if numValue, ok := value.(float64); ok {
					output = append(output, fmt.Sprintf("%s %f %d", promName, numValue, timestamp))
				} else if intValue, ok := value.(int64); ok {
					output = append(output, fmt.Sprintf("%s %d %d", promName, intValue, timestamp))
				} else if uintValue, ok := value.(uint64); ok {
					output = append(output, fmt.Sprintf("%s %d %d", promName, uintValue, timestamp))
				}
			}
		}
	}

	// Write to file
	filename := fmt.Sprintf("metrics-%s.prom", time.Now().Format("20060102-150405"))
	filepath := mc.config.ExportPath + "/" + filename

	data := []byte(fmt.Sprintf("# ccAgents metrics export\n# TIMESTAMP: %s\n%s\n",
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%s", output)))

	if err := writeFile(filepath, data); err != nil {
		return err
	}

	if mc.logger != nil {
		mc.logger.Debug("Exported metrics to Prometheus format (file: %s)", filepath)
	}

	return nil
}

// GetMetricsSummary returns a summary of collected metrics
func (mc *ApplicationMetricsCollector) GetMetricsSummary() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return map[string]interface{}{
		"running":             mc.running,
		"collection_interval": mc.config.CollectionInterval,
		"retention_period":    mc.config.RetentionPeriod,
		"registered_metrics":  len(mc.registry),
		"max_metrics":         mc.config.MaxMetrics,
		"sampling_enabled":    mc.config.EnableSampling,
		"prometheus_enabled":  mc.config.EnablePrometheus,
		"json_export_enabled": mc.config.EnableJSONExport,
		"export_path":         mc.config.ExportPath,
	}
}

// collectionLoop runs the metrics collection loop
func (mc *ApplicationMetricsCollector) collectionLoop() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := mc.ExportMetrics(); err != nil && mc.logger != nil {
				mc.logger.Error("Failed to export metrics (error: %v)", err)
			}
		case <-mc.ctx.Done():
			return
		}
	}
}

// MetricsHTTPHandler provides HTTP endpoint for metrics
type MetricsHTTPHandler struct {
	collector *ApplicationMetricsCollector
	logger    *logger.Logger
}

// NewMetricsHTTPHandler creates a new HTTP handler for metrics
func NewMetricsHTTPHandler(collector *ApplicationMetricsCollector, logger *logger.Logger) *MetricsHTTPHandler {
	return &MetricsHTTPHandler{
		collector: collector,
		logger:    logger,
	}
}

// ServeHTTP handles HTTP requests for metrics
func (mh *MetricsHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/metrics":
		mh.handleMetrics(w, r)
	case "/metrics/summary":
		mh.handleSummary(w, r)
	case "/health":
		mh.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleMetrics serves the current metrics
func (mh *MetricsHTTPHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := mh.collector.CollectApplicationMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"timestamp": time.Now(),
		"metrics":   metrics,
	}); err != nil && mh.logger != nil {
		mh.logger.Error("Failed to encode metrics response (error: %v)", err)
	}
}

// handleSummary serves the metrics summary
func (mh *MetricsHTTPHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary := mh.collector.GetMetricsSummary()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(summary); err != nil && mh.logger != nil {
		mh.logger.Error("Failed to encode summary response (error: %v)", err)
	}
}

// handleHealth serves health check
func (mh *MetricsHTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()), // This would be calculated from start time
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(health); err != nil && mh.logger != nil {
		mh.logger.Error("Failed to encode health response (error: %v)", err)
	}
}

// Utility functions
func ensureDir(path string) error {
	return os.MkdirAll(path, 0750)
}

func writeFile(filepath string, data []byte) error {
	return os.WriteFile(filepath, data, 0600)
}
