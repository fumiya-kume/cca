package performance

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// PerformanceMonitor tracks performance metrics across the application
type PerformanceMonitor struct {
	metrics     map[string]*Metric
	collectors  []MetricCollector
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	interval    time.Duration
	logger      *logger.Logger
	running     bool
	subscribers []MetricSubscriber
}

// Metric represents a performance metric
type Metric struct {
	Name       string            `json:"name"`
	Type       MetricType        `json:"type"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
	Timestamp  time.Time         `json:"timestamp"`
	Tags       map[string]string `json:"tags"`
	History    []MetricSample    `json:"history,omitempty"`
	maxHistory int
	mutex      sync.RWMutex
}

// MetricSample represents a historical metric sample
type MetricSample struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricType represents the type of metric
type MetricType int

const (
	MetricTypeGauge MetricType = iota
	MetricTypeCounter
	MetricTypeHistogram
	MetricTypeTimer
)

func (mt MetricType) String() string {
	switch mt {
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeCounter:
		return "counter"
	case MetricTypeHistogram:
		return "histogram"
	case MetricTypeTimer:
		return "timer"
	default:
		return "unknown"
	}
}

// MetricCollector collects performance metrics
type MetricCollector interface {
	CollectMetrics() map[string]float64
	GetName() string
}

// MetricSubscriber receives metric updates
type MetricSubscriber interface {
	OnMetricUpdate(metric *Metric)
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(interval time.Duration, logger *logger.Logger) *PerformanceMonitor {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	pm := &PerformanceMonitor{
		metrics:     make(map[string]*Metric),
		collectors:  make([]MetricCollector, 0),
		subscribers: make([]MetricSubscriber, 0),
		ctx:         ctx,
		cancel:      cancel,
		interval:    interval,
		logger:      logger,
	}

	// Add built-in collectors
	pm.AddCollector(&SystemMetricCollector{})
	pm.AddCollector(&RuntimeMetricCollector{})

	return pm
}

// AddCollector adds a metric collector
func (pm *PerformanceMonitor) AddCollector(collector MetricCollector) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.collectors = append(pm.collectors, collector)
}

// AddSubscriber adds a metric subscriber
func (pm *PerformanceMonitor) AddSubscriber(subscriber MetricSubscriber) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.subscribers = append(pm.subscribers, subscriber)
}

// Start starts the performance monitoring
func (pm *PerformanceMonitor) Start() {
	pm.mutex.Lock()
	if pm.running {
		pm.mutex.Unlock()
		return
	}
	pm.running = true
	pm.mutex.Unlock()

	go pm.monitorLoop()

	if pm.logger != nil {
		pm.logger.Info("Performance monitoring started (interval: %v)", pm.interval)
	}
}

// Stop stops the performance monitoring
func (pm *PerformanceMonitor) Stop() {
	pm.cancel()

	pm.mutex.Lock()
	pm.running = false
	pm.mutex.Unlock()

	if pm.logger != nil {
		pm.logger.Info("Performance monitoring stopped")
	}
}

// RecordMetric records a metric value
func (pm *PerformanceMonitor) RecordMetric(name string, value float64, metricType MetricType, unit string, tags map[string]string) {
	pm.mutex.Lock()
	metric, exists := pm.metrics[name]
	if !exists {
		metric = &Metric{
			Name:       name,
			Type:       metricType,
			Unit:       unit,
			Tags:       tags,
			History:    make([]MetricSample, 0),
			maxHistory: 100, // Keep last 100 samples
		}
		pm.metrics[name] = metric
	}
	pm.mutex.Unlock()

	metric.mutex.Lock()
	metric.Value = value
	metric.Timestamp = time.Now()

	// Add to history
	sample := MetricSample{
		Value:     value,
		Timestamp: metric.Timestamp,
	}
	metric.History = append(metric.History, sample)

	// Trim history if too long
	if len(metric.History) > metric.maxHistory {
		metric.History = metric.History[1:]
	}
	metric.mutex.Unlock()

	// Notify subscribers
	pm.mutex.RLock()
	subscribers := make([]MetricSubscriber, len(pm.subscribers))
	copy(subscribers, pm.subscribers)
	pm.mutex.RUnlock()

	for _, subscriber := range subscribers {
		go subscriber.OnMetricUpdate(metric)
	}
}

// GetMetric returns a metric by name
func (pm *PerformanceMonitor) GetMetric(name string) (*Metric, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	metric, exists := pm.metrics[name]
	if !exists {
		return nil, errors.ValidationError("metric not found: " + name)
	}

	// Return a copy to prevent concurrent access issues
	metric.mutex.RLock()
	defer metric.mutex.RUnlock()

	metricCopy := Metric{
		Name:       metric.Name,
		Type:       metric.Type,
		Value:      metric.Value,
		Unit:       metric.Unit,
		Timestamp:  metric.Timestamp,
		Tags:       metric.Tags,
		History:    make([]MetricSample, len(metric.History)),
		maxHistory: metric.maxHistory,
	}
	copy(metricCopy.History, metric.History)

	return &metricCopy, nil
}

// GetAllMetrics returns all metrics
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*Metric {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]*Metric)
	for name, metric := range pm.metrics {
		metric.mutex.RLock()
		metricCopy := Metric{
			Name:       metric.Name,
			Type:       metric.Type,
			Value:      metric.Value,
			Unit:       metric.Unit,
			Timestamp:  metric.Timestamp,
			Tags:       metric.Tags,
			History:    make([]MetricSample, len(metric.History)),
			maxHistory: metric.maxHistory,
		}
		copy(metricCopy.History, metric.History)
		metric.mutex.RUnlock()

		result[name] = &metricCopy
	}

	return result
}

// GetMetricsSummary returns a summary of all metrics
func (pm *PerformanceMonitor) GetMetricsSummary() map[string]interface{} {
	metrics := pm.GetAllMetrics()

	summary := map[string]interface{}{
		"total_metrics": len(metrics),
		"collectors":    len(pm.collectors),
		"subscribers":   len(pm.subscribers),
		"running":       pm.running,
		"interval":      pm.interval,
		"metrics":       make(map[string]interface{}),
	}

	metricsMap := make(map[string]interface{})
	for name, metric := range metrics {
		metricsMap[name] = map[string]interface{}{
			"type":      metric.Type.String(),
			"value":     metric.Value,
			"unit":      metric.Unit,
			"timestamp": metric.Timestamp,
			"tags":      metric.Tags,
			"samples":   len(metric.History),
		}
	}
	summary["metrics"] = metricsMap

	return summary
}

// ExportMetrics exports metrics in JSON format
func (pm *PerformanceMonitor) ExportMetrics() ([]byte, error) {
	metrics := pm.GetAllMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// monitorLoop runs the metric collection loop
func (pm *PerformanceMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.collectMetrics()
		case <-pm.ctx.Done():
			return
		}
	}
}

// collectMetrics collects metrics from all collectors
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mutex.RLock()
	collectors := make([]MetricCollector, len(pm.collectors))
	copy(collectors, pm.collectors)
	pm.mutex.RUnlock()

	for _, collector := range collectors {
		metrics := collector.CollectMetrics()
		for name, value := range metrics {
			fullName := collector.GetName() + "." + name
			pm.RecordMetric(fullName, value, MetricTypeGauge, "", map[string]string{
				"collector": collector.GetName(),
			})
		}
	}
}

// SystemMetricCollector collects system-level metrics
type SystemMetricCollector struct{}

func (smc *SystemMetricCollector) GetName() string {
	return "system"
}

func (smc *SystemMetricCollector) CollectMetrics() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]float64{
		"memory.allocated":   float64(m.Alloc),
		"memory.total_alloc": float64(m.TotalAlloc),
		"memory.sys":         float64(m.Sys),
		"memory.heap_alloc":  float64(m.HeapAlloc),
		"memory.heap_sys":    float64(m.HeapSys),
		"memory.heap_idle":   float64(m.HeapIdle),
		"memory.heap_inuse":  float64(m.HeapInuse),
		"gc.num_gc":          float64(m.NumGC),
		"gc.pause_total_ns":  float64(m.PauseTotalNs),
		"goroutines":         float64(runtime.NumGoroutine()),
		"cpu.num_cpu":        float64(runtime.NumCPU()),
	}
}

// RuntimeMetricCollector collects Go runtime metrics
type RuntimeMetricCollector struct{}

func (rmc *RuntimeMetricCollector) GetName() string {
	return "runtime"
}

func (rmc *RuntimeMetricCollector) CollectMetrics() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]float64{
		"gc.cpu_fraction": m.GCCPUFraction,
		"gc.next_gc":      float64(m.NextGC),
		"gc.last_gc":      float64(m.LastGC),
		"malloc.lookups":  float64(m.Lookups),
		"malloc.mallocs":  float64(m.Mallocs),
		"malloc.frees":    float64(m.Frees),
		"stack.inuse":     float64(m.StackInuse),
		"stack.sys":       float64(m.StackSys),
	}
}

// TimerMetric provides timing measurements
type TimerMetric struct {
	name      string
	startTime time.Time
	monitor   *PerformanceMonitor
	tags      map[string]string
}

// StartTimer creates and starts a timer metric
func (pm *PerformanceMonitor) StartTimer(name string, tags map[string]string) *TimerMetric {
	return &TimerMetric{
		name:      name,
		startTime: time.Now(),
		monitor:   pm,
		tags:      tags,
	}
}

// Stop stops the timer and records the duration
func (tm *TimerMetric) Stop() time.Duration {
	duration := time.Since(tm.startTime)
	tm.monitor.RecordMetric(tm.name, float64(duration.Nanoseconds()), MetricTypeTimer, "nanoseconds", tm.tags)
	return duration
}

// CounterMetric provides counter operations
type CounterMetric struct {
	name    string
	value   int64
	monitor *PerformanceMonitor
	tags    map[string]string
	mutex   sync.RWMutex
}

// GetCounter returns or creates a counter metric
func (pm *PerformanceMonitor) GetCounter(name string, tags map[string]string) *CounterMetric {
	return &CounterMetric{
		name:    name,
		monitor: pm,
		tags:    tags,
	}
}

// Increment increments the counter
func (cm *CounterMetric) Increment() {
	cm.Add(1)
}

// Add adds a value to the counter
func (cm *CounterMetric) Add(delta int64) {
	cm.mutex.Lock()
	cm.value += delta
	newValue := cm.value
	cm.mutex.Unlock()

	cm.monitor.RecordMetric(cm.name, float64(newValue), MetricTypeCounter, "count", cm.tags)
}

// Reset resets the counter to zero
func (cm *CounterMetric) Reset() {
	cm.mutex.Lock()
	cm.value = 0
	cm.mutex.Unlock()

	cm.monitor.RecordMetric(cm.name, 0, MetricTypeCounter, "count", cm.tags)
}

// Value returns the current counter value
func (cm *CounterMetric) Value() int64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.value
}

// GaugeMetric provides gauge operations
type GaugeMetric struct {
	name    string
	value   float64
	monitor *PerformanceMonitor
	tags    map[string]string
	mutex   sync.RWMutex
}

// GetGauge returns or creates a gauge metric
func (pm *PerformanceMonitor) GetGauge(name string, tags map[string]string) *GaugeMetric {
	return &GaugeMetric{
		name:    name,
		monitor: pm,
		tags:    tags,
	}
}

// Set sets the gauge value
func (gm *GaugeMetric) Set(value float64) {
	gm.mutex.Lock()
	gm.value = value
	gm.mutex.Unlock()

	gm.monitor.RecordMetric(gm.name, value, MetricTypeGauge, "", gm.tags)
}

// Add adds to the gauge value
func (gm *GaugeMetric) Add(delta float64) {
	gm.mutex.Lock()
	gm.value += delta
	newValue := gm.value
	gm.mutex.Unlock()

	gm.monitor.RecordMetric(gm.name, newValue, MetricTypeGauge, "", gm.tags)
}

// Value returns the current gauge value
func (gm *GaugeMetric) Value() float64 {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return gm.value
}

// HistogramMetric provides histogram operations
type HistogramMetric struct {
	name      string
	values    []float64
	buckets   []float64
	monitor   *PerformanceMonitor
	tags      map[string]string
	mutex     sync.RWMutex
	maxValues int
}

// GetHistogram returns or creates a histogram metric
func (pm *PerformanceMonitor) GetHistogram(name string, buckets []float64, tags map[string]string) *HistogramMetric {
	if buckets == nil {
		buckets = []float64{0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99}
	}

	return &HistogramMetric{
		name:      name,
		values:    make([]float64, 0),
		buckets:   buckets,
		monitor:   pm,
		tags:      tags,
		maxValues: 1000,
	}
}

// Observe adds an observation to the histogram
func (hm *HistogramMetric) Observe(value float64) {
	hm.mutex.Lock()
	hm.values = append(hm.values, value)

	// Trim values if too many
	if len(hm.values) > hm.maxValues {
		hm.values = hm.values[len(hm.values)-hm.maxValues/2:]
	}

	// Calculate percentiles
	percentiles := hm.calculatePercentiles()
	hm.mutex.Unlock()

	// Record percentile metrics
	for percentile, value := range percentiles {
		metricName := hm.name + ".percentile_" + percentile
		hm.monitor.RecordMetric(metricName, value, MetricTypeHistogram, "", hm.tags)
	}
}

// calculatePercentiles calculates percentiles for the histogram
func (hm *HistogramMetric) calculatePercentiles() map[string]float64 {
	if len(hm.values) == 0 {
		return map[string]float64{}
	}

	// Simple percentile calculation (should use proper sorting for production)
	result := make(map[string]float64)

	if len(hm.values) == 1 {
		value := hm.values[0]
		for _, bucket := range hm.buckets {
			result[formatFloat(bucket*100)] = value
		}
		return result
	}

	// For now, just return min/max/avg
	min := hm.values[0]
	max := hm.values[0]
	sum := hm.values[0]

	for i := 1; i < len(hm.values); i++ {
		val := hm.values[i]
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
		sum += val
	}

	avg := sum / float64(len(hm.values))

	result["0"] = min
	result["50"] = avg
	result["100"] = max

	return result
}

// formatFloat formats a float for use as a metric name
func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int64(f)))
	}
	return string(rune(int64(f * 100)))
}

// AlertManager handles performance alerts
type AlertManager struct {
	rules       []AlertRule
	alerts      map[string]*Alert
	mutex       sync.RWMutex
	logger      *logger.Logger
	subscribers []AlertSubscriber
}

// AlertRule defines conditions for generating alerts
type AlertRule struct {
	Name        string
	MetricName  string
	Condition   AlertCondition
	Threshold   float64
	Duration    time.Duration
	Severity    AlertSeverity
	Description string
}

// AlertCondition represents alert conditions
type AlertCondition int

const (
	AlertConditionGreaterThan AlertCondition = iota
	AlertConditionLessThan
	AlertConditionEqual
)

// AlertSeverity represents alert severity levels
type AlertSeverity int

const (
	AlertSeverityInfo AlertSeverity = iota
	AlertSeverityWarning
	AlertSeverityError
	AlertSeverityCritical
)

// Alert represents an active alert
type Alert struct {
	Rule        AlertRule
	StartTime   time.Time
	LastSeen    time.Time
	Value       float64
	IsActive    bool
	Occurrences int64
}

// AlertSubscriber receives alert notifications
type AlertSubscriber interface {
	OnAlert(alert *Alert)
	OnAlertResolved(alert *Alert)
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *logger.Logger) *AlertManager {
	return &AlertManager{
		rules:       make([]AlertRule, 0),
		alerts:      make(map[string]*Alert),
		logger:      logger,
		subscribers: make([]AlertSubscriber, 0),
	}
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.rules = append(am.rules, rule)
}

// AddSubscriber adds an alert subscriber
func (am *AlertManager) AddSubscriber(subscriber AlertSubscriber) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.subscribers = append(am.subscribers, subscriber)
}

// CheckMetric checks a metric against alert rules
func (am *AlertManager) CheckMetric(metric *Metric) {
	am.mutex.RLock()
	rules := make([]AlertRule, len(am.rules))
	copy(rules, am.rules)
	am.mutex.RUnlock()

	for _, rule := range rules {
		if rule.MetricName == metric.Name {
			am.evaluateRule(rule, metric)
		}
	}
}

// evaluateRule evaluates an alert rule against a metric
func (am *AlertManager) evaluateRule(rule AlertRule, metric *Metric) {
	triggered := false

	switch rule.Condition {
	case AlertConditionGreaterThan:
		triggered = metric.Value > rule.Threshold
	case AlertConditionLessThan:
		triggered = metric.Value < rule.Threshold
	case AlertConditionEqual:
		triggered = metric.Value == rule.Threshold
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	alert, exists := am.alerts[rule.Name]
	if !exists && triggered {
		// New alert
		alert = &Alert{
			Rule:        rule,
			StartTime:   time.Now(),
			LastSeen:    time.Now(),
			Value:       metric.Value,
			IsActive:    true,
			Occurrences: 1,
		}
		am.alerts[rule.Name] = alert

		// Notify subscribers
		for _, subscriber := range am.subscribers {
			go subscriber.OnAlert(alert)
		}

		if am.logger != nil {
			am.logger.Warn("Alert triggered (rule: %s, metric: %s, value: %v, threshold: %v)", rule.Name, metric.Name, metric.Value, rule.Threshold)
		}
	} else if exists && triggered {
		// Update existing alert
		alert.LastSeen = time.Now()
		alert.Value = metric.Value
		alert.Occurrences++
	} else if exists && !triggered {
		// Resolve alert
		alert.IsActive = false

		// Notify subscribers
		for _, subscriber := range am.subscribers {
			go subscriber.OnAlertResolved(alert)
		}

		if am.logger != nil {
			am.logger.Info("Alert resolved (rule: %s, duration: %v, occurrences: %d)", rule.Name, time.Since(alert.StartTime), alert.Occurrences)
		}

		delete(am.alerts, rule.Name)
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() map[string]*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	result := make(map[string]*Alert)
	for name, alert := range am.alerts {
		if alert.IsActive {
			alertCopy := *alert
			result[name] = &alertCopy
		}
	}

	return result
}
