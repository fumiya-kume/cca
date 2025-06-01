// Package observability provides comprehensive monitoring and observability for ccAgents
package observability

import (
	"context"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/fumiya-kume/cca/pkg/performance"
)

// ObservabilityManager orchestrates all observability components
type ObservabilityManager struct {
	config             ObservabilityConfig
	logger             *logger.Logger
	performanceMonitor *performance.PerformanceMonitor

	// Observability components
	structuredLogger   *StructuredLogger
	metricsCollector   *ApplicationMetricsCollector
	telemetryCollector *TelemetryCollector
	sessionTracker     *SessionTracker
	debugManager       *DebugManager
	dashboard          *Dashboard
	logAggregator      *LogAggregator

	// State
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	mutex   sync.RWMutex
}

// ObservabilityConfig configures all observability features
type ObservabilityConfig struct {
	// General settings
	Enabled bool `yaml:"enabled"`

	// Component configurations
	Logging   LoggingConfig   `yaml:"logging"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
	Debug     DebugConfig     `yaml:"debug"`
	Dashboard DashboardConfig `yaml:"dashboard"`

	// Integration settings
	EnableIntegration bool          `yaml:"enable_integration"`
	FlushInterval     time.Duration `yaml:"flush_interval"`
	RetentionPeriod   time.Duration `yaml:"retention_period"`
	MaxConcurrentOps  int           `yaml:"max_concurrent_ops"`
}

// LoggingConfig configures structured logging
type LoggingConfig struct {
	Enabled           bool          `yaml:"enabled"`
	Level             string        `yaml:"level"`
	Format            string        `yaml:"format"`
	OutputPath        string        `yaml:"output_path"`
	MaxSize           int64         `yaml:"max_size"`
	MaxAge            time.Duration `yaml:"max_age"`
	MaxBackups        int           `yaml:"max_backups"`
	EnableRotation    bool          `yaml:"enable_rotation"`
	EnableAggregation bool          `yaml:"enable_aggregation"`
	MaxEntries        int           `yaml:"max_entries"`
}

// NewObservabilityManager creates a new observability manager
func NewObservabilityManager(
	config ObservabilityConfig,
	logger *logger.Logger,
	performanceMonitor *performance.PerformanceMonitor,
) *ObservabilityManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ObservabilityManager{
		config:             config,
		logger:             logger,
		performanceMonitor: performanceMonitor,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// DefaultObservabilityConfig returns default observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Enabled: true,
		Logging: LoggingConfig{
			Enabled:           true,
			Level:             "INFO",
			Format:            "json",
			OutputPath:        "./logs",
			MaxSize:           100 * 1024 * 1024, // 100MB
			MaxAge:            24 * time.Hour,
			MaxBackups:        7,
			EnableRotation:    true,
			EnableAggregation: true,
			MaxEntries:        10000,
		},
		Metrics:           DefaultMetricsConfig(),
		Telemetry:         DefaultTelemetryConfig(),
		Debug:             DefaultDebugConfig(),
		Dashboard:         DefaultDashboardConfig(),
		EnableIntegration: true,
		FlushInterval:     30 * time.Second,
		RetentionPeriod:   7 * 24 * time.Hour,
		MaxConcurrentOps:  100,
	}
}

// Initialize initializes all observability components
func (om *ObservabilityManager) Initialize() error {
	if !om.config.Enabled {
		return om.handleDisabledObservability()
	}

	om.mutex.Lock()
	defer om.mutex.Unlock()

	if err := om.initializeLoggingComponents(); err != nil {
		return err
	}

	if err := om.initializeCollectors(); err != nil {
		return err
	}

	if err := om.initializeTrackerAndDebug(); err != nil {
		return err
	}

	if err := om.initializeDashboard(); err != nil {
		return err
	}

	if om.logger != nil {
		om.logger.Info("Observability manager initialized successfully")
	}

	return nil
}

// handleDisabledObservability handles the case when observability is disabled
func (om *ObservabilityManager) handleDisabledObservability() error {
	if om.logger != nil {
		om.logger.Info("Observability disabled")
	}
	return nil
}

// initializeLoggingComponents initializes logging-related components
func (om *ObservabilityManager) initializeLoggingComponents() error {
	if !om.config.Logging.Enabled {
		return nil
	}

	om.structuredLogger = NewStructuredLogger(om.logger)
	om.logAggregator = NewLogAggregator(om.config.Logging.MaxEntries)

	// Add filters and processors
	if om.config.Logging.Level != "" {
		om.logAggregator.AddFilter(&LevelFilter{MinLevel: om.config.Logging.Level})
	}

	om.logAggregator.AddProcessor(&SanitizationProcessor{
		SensitiveFields: []string{"password", "token", "secret", "key"},
	})

	// Add file output if path is specified
	if om.config.Logging.OutputPath != "" {
		if output, err := NewFileLogOutput(om.config.Logging.OutputPath + "/observability.log"); err == nil {
			om.logAggregator.AddOutput(output)
		}
	}

	if om.logger != nil {
		om.logger.Info("Structured logging initialized")
	}

	return nil
}

// initializeCollectors initializes metrics and telemetry collectors
func (om *ObservabilityManager) initializeCollectors() error {
	// Initialize metrics collector
	if om.config.Metrics.Enabled {
		om.metricsCollector = NewApplicationMetricsCollector(om.performanceMonitor, om.logAggregator, om.logger)
		om.metricsCollector.SetConfig(om.config.Metrics)

		if om.logger != nil {
			om.logger.Info("Metrics collector initialized")
		}
	}

	// Initialize telemetry collector
	if om.config.Telemetry.Enabled {
		om.telemetryCollector = NewTelemetryCollector(om.config.Telemetry, om.logger)

		if om.logger != nil {
			om.logger.Info("Telemetry collector initialized")
		}
	}

	return nil
}

// initializeTrackerAndDebug initializes session tracker and debug manager
func (om *ObservabilityManager) initializeTrackerAndDebug() error {
	// Initialize session tracker
	om.sessionTracker = NewSessionTracker(om.logger)
	if om.logger != nil {
		om.logger.Info("Session tracker initialized")
	}

	// Initialize debug manager
	if om.config.Debug.Enabled {
		om.debugManager = NewDebugManager(om.config.Debug, om.logger)

		if om.logger != nil {
			om.logger.Info("Debug manager initialized")
		}
	}

	return nil
}

// initializeDashboard initializes the dashboard component
func (om *ObservabilityManager) initializeDashboard() error {
	if !om.config.Dashboard.Enabled {
		return nil
	}

	om.dashboard = NewDashboard(
		om.metricsCollector,
		om.performanceMonitor,
		om.telemetryCollector,
		om.sessionTracker,
		om.debugManager,
		om.logger,
	)
	om.dashboard.SetConfig(om.config.Dashboard)

	if om.logger != nil {
		om.logger.Info("Dashboard initialized")
	}

	return nil
}

// Start starts all observability components
func (om *ObservabilityManager) Start() error {
	if !om.config.Enabled {
		return nil
	}

	om.mutex.Lock()
	if om.running {
		om.mutex.Unlock()
		return errors.ValidationError("observability manager already running")
	}
	om.running = true
	om.mutex.Unlock()

	// Start metrics collector
	if om.metricsCollector != nil {
		if err := om.metricsCollector.Start(); err != nil {
			return errors.NewError(errors.ErrorTypeValidation).
				WithMessage("failed to start metrics collector").
				WithCause(err).
				Build()
		}
	}

	// Start telemetry collector
	if om.telemetryCollector != nil {
		om.telemetryCollector.Enable()
	}

	// Start debug manager
	if om.debugManager != nil {
		om.debugManager.Enable()
	}

	// Start dashboard
	if om.dashboard != nil {
		if err := om.dashboard.Start(); err != nil {
			if om.logger != nil {
				om.logger.Warn("Failed to start dashboard (error: %v)", err)
			}
		}
	}

	// Start background tasks
	go om.backgroundTasks()

	if om.logger != nil {
		om.logger.Info("Observability manager started")
	}

	return nil
}

// Stop stops all observability components
func (om *ObservabilityManager) Stop() error {
	om.cancel()

	om.mutex.Lock()
	defer om.mutex.Unlock()

	if !om.running {
		return nil
	}

	// Stop components in reverse order
	if om.dashboard != nil {
		if err := om.dashboard.Stop(); err != nil && om.logger != nil {
			om.logger.Warn("Failed to stop dashboard (error: %v)", err)
		}
	}

	if om.debugManager != nil {
		om.debugManager.Disable()
	}

	if om.telemetryCollector != nil {
		om.telemetryCollector.Disable()
		// Flush any remaining telemetry data
		if err := om.telemetryCollector.FlushEvents(); err != nil && om.logger != nil {
			om.logger.Warn("Failed to flush telemetry (error: %v)", err)
		}
	}

	if om.metricsCollector != nil {
		om.metricsCollector.Stop()
		// Export final metrics
		if err := om.metricsCollector.ExportMetrics(); err != nil && om.logger != nil {
			om.logger.Warn("Failed to export final metrics (error: %v)", err)
		}
	}

	om.running = false

	if om.logger != nil {
		om.logger.Info("Observability manager stopped")
	}

	return nil
}

// GetStructuredLogger returns the structured logger
func (om *ObservabilityManager) GetStructuredLogger() *StructuredLogger {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.structuredLogger
}

// GetMetricsCollector returns the metrics collector
func (om *ObservabilityManager) GetMetricsCollector() *ApplicationMetricsCollector {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.metricsCollector
}

// GetTelemetryCollector returns the telemetry collector
func (om *ObservabilityManager) GetTelemetryCollector() *TelemetryCollector {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.telemetryCollector
}

// GetSessionTracker returns the session tracker
func (om *ObservabilityManager) GetSessionTracker() *SessionTracker {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.sessionTracker
}

// GetDebugManager returns the debug manager
func (om *ObservabilityManager) GetDebugManager() *DebugManager {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.debugManager
}

// GetDashboard returns the dashboard
func (om *ObservabilityManager) GetDashboard() *Dashboard {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	return om.dashboard
}

// RecordActivity records an activity across multiple components
func (om *ObservabilityManager) RecordActivity(activityType string, success bool, duration time.Duration, err error, metadata map[string]interface{}) {
	// Record in session tracker
	if om.sessionTracker != nil {
		om.sessionTracker.RecordTimedActivity(activityType, duration, success, err, metadata)
	}

	// Record in telemetry (if enabled)
	if om.telemetryCollector != nil && om.telemetryCollector.IsEnabled() {
		telemetryData := make(map[string]interface{})
		for k, v := range metadata {
			telemetryData[k] = v
		}
		telemetryData["success"] = success
		telemetryData["duration_ms"] = duration.Milliseconds()

		if err != nil {
			om.telemetryCollector.RecordError(activityType, err, telemetryData)
		} else {
			om.telemetryCollector.RecordTiming(activityType, duration, telemetryData)
		}
	}

	// Record in structured logs
	if om.structuredLogger != nil {
		fields := []interface{}{
			"activity_type", activityType,
			"success", success,
			"duration_ms", duration.Milliseconds(),
		}

		for k, v := range metadata {
			fields = append(fields, k, v)
		}

		if err != nil {
			om.structuredLogger.LogError(err, "Activity completed with error", fields...)
		} else {
			om.structuredLogger.Info("Activity completed successfully", fields...)
		}
	}
}

// GetOverallStatus returns the overall observability status
func (om *ObservabilityManager) GetOverallStatus() map[string]interface{} {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	status := map[string]interface{}{
		"enabled": om.config.Enabled,
		"running": om.running,
		"components": map[string]bool{
			"structured_logger":   om.structuredLogger != nil,
			"metrics_collector":   om.metricsCollector != nil,
			"telemetry_collector": om.telemetryCollector != nil,
			"session_tracker":     om.sessionTracker != nil,
			"debug_manager":       om.debugManager != nil,
			"dashboard":           om.dashboard != nil,
			"log_aggregator":      om.logAggregator != nil,
		},
	}

	// Add component-specific status
	if om.telemetryCollector != nil {
		status["telemetry_enabled"] = om.telemetryCollector.IsEnabled()
	}

	if om.sessionTracker != nil {
		status["session_id"] = om.sessionTracker.GetSessionID()
	}

	return status
}

// backgroundTasks runs background maintenance tasks
func (om *ObservabilityManager) backgroundTasks() {
	ticker := time.NewTicker(om.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			om.performMaintenance()
		case <-om.ctx.Done():
			return
		}
	}
}

// performMaintenance performs periodic maintenance tasks
func (om *ObservabilityManager) performMaintenance() {
	// Flush telemetry data
	if om.telemetryCollector != nil && om.telemetryCollector.IsEnabled() {
		if err := om.telemetryCollector.FlushEvents(); err != nil && om.logger != nil {
			om.logger.Warn("Failed to flush telemetry during maintenance (error: %v)", err)
		}
	}

	// Export metrics
	if om.metricsCollector != nil {
		if err := om.metricsCollector.ExportMetrics(); err != nil && om.logger != nil {
			om.logger.Warn("Failed to export metrics during maintenance (error: %v)", err)
		}
	}

	// Flush log aggregator
	if om.logAggregator != nil {
		for _, output := range om.logAggregator.outputs {
			if err := output.Flush(); err != nil && om.logger != nil {
				om.logger.Warn("Failed to flush logs during maintenance (error: %v)", err)
			}
		}
	}

	if om.logger != nil {
		om.logger.Debug("Completed observability maintenance")
	}
}
