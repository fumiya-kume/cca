// Package observability provides monitoring dashboards for ccAgents
package observability

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/fumiya-kume/cca/pkg/performance"
)

// Dashboard provides web-based monitoring interface
type Dashboard struct {
	metricsCollector   *ApplicationMetricsCollector
	performanceMonitor *performance.PerformanceMonitor
	telemetryCollector *TelemetryCollector
	sessionTracker     *SessionTracker
	debugManager       *DebugManager
	logger             *logger.Logger
	config             DashboardConfig
	server             *http.Server
	templates          *template.Template
	mutex              sync.RWMutex
	startTime          time.Time
}

// DashboardConfig configures the monitoring dashboard
type DashboardConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	RefreshRate int    `yaml:"refresh_rate"`
	Theme       string `yaml:"theme"`
	Title       string `yaml:"title"`
}

// DashboardData contains data for dashboard rendering
type DashboardData struct {
	Title         string                 `json:"title"`
	Timestamp     time.Time              `json:"timestamp"`
	Uptime        string                 `json:"uptime"`
	Status        string                 `json:"status"`
	Metrics       map[string]interface{} `json:"metrics"`
	Performance   map[string]interface{} `json:"performance"`
	SessionInfo   map[string]interface{} `json:"session_info"`
	TelemetryInfo map[string]interface{} `json:"telemetry_info"`
	DebugInfo     map[string]interface{} `json:"debug_info"`
	RecentEvents  []interface{}          `json:"recent_events"`
	SystemInfo    map[string]interface{} `json:"system_info"`
	RefreshRate   int                    `json:"refresh_rate"`
}

// AlertDashboardData contains alert information for dashboard
type AlertDashboardData struct {
	ActiveAlerts map[string]interface{} `json:"active_alerts"`
	AlertHistory []interface{}          `json:"alert_history"`
	AlertRules   []interface{}          `json:"alert_rules"`
	AlertSummary map[string]interface{} `json:"alert_summary"`
}

// NewDashboard creates a new monitoring dashboard
func NewDashboard(
	metricsCollector *ApplicationMetricsCollector,
	performanceMonitor *performance.PerformanceMonitor,
	telemetryCollector *TelemetryCollector,
	sessionTracker *SessionTracker,
	debugManager *DebugManager,
	logger *logger.Logger,
) *Dashboard {
	return &Dashboard{
		metricsCollector:   metricsCollector,
		performanceMonitor: performanceMonitor,
		telemetryCollector: telemetryCollector,
		sessionTracker:     sessionTracker,
		debugManager:       debugManager,
		logger:             logger,
		config:             DefaultDashboardConfig(),
		startTime:          time.Now(),
	}
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() DashboardConfig {
	return DashboardConfig{
		Enabled:     false,
		Host:        "localhost",
		Port:        8080,
		RefreshRate: 5, // seconds
		Theme:       "dark",
		Title:       "ccAgents Monitoring Dashboard",
	}
}

// SetConfig updates the dashboard configuration
func (d *Dashboard) SetConfig(config DashboardConfig) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.config = config
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	if !d.config.Enabled {
		return nil
	}

	d.initializeTemplates()

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", d.handleDashboard)
	mux.HandleFunc("/api/data", d.handleAPIData)
	mux.HandleFunc("/api/metrics", d.handleAPIMetrics)
	mux.HandleFunc("/api/performance", d.handleAPIPerformance)
	mux.HandleFunc("/api/telemetry", d.handleAPITelemetry)
	mux.HandleFunc("/api/session", d.handleAPISession)
	mux.HandleFunc("/api/debug", d.handleAPIDebug)
	mux.HandleFunc("/api/alerts", d.handleAPIAlerts)
	mux.HandleFunc("/api/health", d.handleAPIHealth)
	mux.HandleFunc("/static/", d.handleStatic)

	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	d.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if d.logger != nil {
				d.logger.Error("Dashboard server failed (error: %v)", err)
			}
		}
	}()

	if d.logger != nil {
		d.logger.Info("Dashboard started (url: http://%s)", addr)
	}

	return nil
}

// Stop stops the dashboard server
func (d *Dashboard) Stop() error {
	if d.server != nil {
		if d.logger != nil {
			d.logger.Info("Stopping dashboard server")
		}
		return d.server.Close()
	}
	return nil
}

// handleDashboard serves the main dashboard page
func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := d.collectDashboardData()

	w.Header().Set("Content-Type", "text/html")

	if err := d.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		if d.logger != nil {
			d.logger.Error("Failed to render dashboard template (error: %v)", err)
		}
	}
}

// handleAPIData serves dashboard data as JSON
func (d *Dashboard) handleAPIData(w http.ResponseWriter, r *http.Request) {
	data := d.collectDashboardData()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		if d.logger != nil {
			d.logger.Error("Failed to encode dashboard data (error: %v)", err)
		}
	}
}

// handleAPIMetrics serves metrics data
func (d *Dashboard) handleAPIMetrics(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if d.metricsCollector != nil {
		data = d.metricsCollector.CollectApplicationMetrics()
	} else {
		data = map[string]interface{}{"error": "metrics collector not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPIPerformance serves performance data
func (d *Dashboard) handleAPIPerformance(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if d.performanceMonitor != nil {
		data = map[string]interface{}{
			"metrics": d.performanceMonitor.GetAllMetrics(),
			"summary": d.performanceMonitor.GetMetricsSummary(),
		}
	} else {
		data = map[string]interface{}{"error": "performance monitor not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPITelemetry serves telemetry data
func (d *Dashboard) handleAPITelemetry(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if d.telemetryCollector != nil {
		data = d.telemetryCollector.GetSummary()
	} else {
		data = map[string]interface{}{"error": "telemetry collector not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPISession serves session data
func (d *Dashboard) handleAPISession(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}

	if d.sessionTracker != nil {
		data = d.sessionTracker.GetSessionSummary()
	} else {
		data = map[string]interface{}{"error": "session tracker not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPIDebug serves debug information
func (d *Dashboard) handleAPIDebug(w http.ResponseWriter, r *http.Request) {
	var data interface{}

	if d.debugManager != nil {
		data = d.debugManager.GetDiagnosticInfo()
	} else {
		data = map[string]interface{}{"error": "debug manager not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPIAlerts serves alert information
func (d *Dashboard) handleAPIAlerts(w http.ResponseWriter, r *http.Request) {
	data := AlertDashboardData{
		ActiveAlerts: make(map[string]interface{}),
		AlertHistory: make([]interface{}, 0),
		AlertRules:   make([]interface{}, 0),
		AlertSummary: map[string]interface{}{
			"total_alerts":    0,
			"active_alerts":   0,
			"resolved_alerts": 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleAPIHealth serves health check
func (d *Dashboard) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(d.startTime),
		"components": map[string]bool{
			"metrics_collector":   d.metricsCollector != nil,
			"performance_monitor": d.performanceMonitor != nil,
			"telemetry_collector": d.telemetryCollector != nil,
			"session_tracker":     d.sessionTracker != nil,
			"debug_manager":       d.debugManager != nil,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleStatic serves static files
func (d *Dashboard) handleStatic(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would serve CSS, JS, and other static files
	// For now, we'll return a simple CSS for the dashboard
	if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Content-Type", "text/css")
		if _, err := w.Write([]byte(d.getDefaultCSS())); err != nil {
			// Log error but continue as this is HTTP response
			// Client will receive partial response
			d.logger.Warn("Failed to write CSS response: %v", err)
		}
	} else if strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
		if _, err := w.Write([]byte(d.getDefaultJS())); err != nil {
			// Log error but continue as this is HTTP response
			// Client will receive partial response
			d.logger.Warn("Failed to write JS response: %v", err)
		}
	} else {
		http.NotFound(w, r)
	}
}

// collectDashboardData collects all data for dashboard rendering
func (d *Dashboard) collectDashboardData() DashboardData {
	data := DashboardData{
		Title:       d.config.Title,
		Timestamp:   time.Now(),
		Uptime:      time.Since(d.startTime).String(),
		Status:      "running",
		RefreshRate: d.config.RefreshRate,
		SystemInfo:  d.getSystemInfo(),
	}

	// Collect metrics
	if d.metricsCollector != nil {
		data.Metrics = d.metricsCollector.GetMetricsSummary()
	}

	// Collect performance data
	if d.performanceMonitor != nil {
		data.Performance = d.performanceMonitor.GetMetricsSummary()
	}

	// Collect session info
	if d.sessionTracker != nil {
		data.SessionInfo = d.sessionTracker.GetSessionSummary()
	}

	// Collect telemetry info
	if d.telemetryCollector != nil {
		data.TelemetryInfo = d.telemetryCollector.GetSummary()
	}

	// Collect debug info
	if d.debugManager != nil {
		debugInfo := d.debugManager.GetDiagnosticInfo()
		data.DebugInfo = map[string]interface{}{
			"runtime":         debugInfo.Runtime,
			"memory":          debugInfo.Memory,
			"active_profiles": debugInfo.ActiveProfiles,
			"trace_count":     len(debugInfo.RecentTraces),
		}
	}

	// Collect recent events (simplified)
	data.RecentEvents = d.getRecentEvents()

	return data
}

// getSystemInfo returns system information
func (d *Dashboard) getSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"dashboard_version": "1.0.0",
		"start_time":        d.startTime,
		"config":            d.config,
	}
}

// getRecentEvents returns recent events for the dashboard
func (d *Dashboard) getRecentEvents() []interface{} {
	events := make([]interface{}, 0)

	// Add startup event
	events = append(events, map[string]interface{}{
		"timestamp": d.startTime,
		"type":      "system",
		"message":   "Dashboard started",
		"level":     "info",
	})

	return events
}

// initializeTemplates initializes HTML templates
func (d *Dashboard) initializeTemplates() {
	templateStr := d.getDashboardTemplate()
	d.templates = template.Must(template.New("dashboard.html").Parse(templateStr))
}

// getDashboardTemplate returns the HTML template for the dashboard
func (d *Dashboard) getDashboardTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/dashboard.css">
    <meta http-equiv="refresh" content="{{.RefreshRate}}">
</head>
<body>
    <div class="container">
        <header>
            <h1>{{.Title}}</h1>
            <div class="status">
                <span class="status-indicator {{.Status}}"></span>
                <span>{{.Status}}</span>
                <span class="uptime">Uptime: {{.Uptime}}</span>
            </div>
        </header>
        
        <main>
            <div class="grid">
                <div class="card">
                    <h3>System Overview</h3>
                    <div class="metric">
                        <span class="label">Status:</span>
                        <span class="value">{{.Status}}</span>
                    </div>
                    <div class="metric">
                        <span class="label">Uptime:</span>
                        <span class="value">{{.Uptime}}</span>
                    </div>
                    <div class="metric">
                        <span class="label">Last Updated:</span>
                        <span class="value">{{.Timestamp.Format "15:04:05"}}</span>
                    </div>
                </div>
                
                {{if .Performance}}
                <div class="card">
                    <h3>Performance</h3>
                    {{range $key, $value := .Performance}}
                    <div class="metric">
                        <span class="label">{{$key}}:</span>
                        <span class="value">{{$value}}</span>
                    </div>
                    {{end}}
                </div>
                {{end}}
                
                {{if .Metrics}}
                <div class="card">
                    <h3>Metrics</h3>
                    {{range $key, $value := .Metrics}}
                    <div class="metric">
                        <span class="label">{{$key}}:</span>
                        <span class="value">{{$value}}</span>
                    </div>
                    {{end}}
                </div>
                {{end}}
                
                {{if .SessionInfo}}
                <div class="card">
                    <h3>Session Info</h3>
                    {{range $key, $value := .SessionInfo}}
                    <div class="metric">
                        <span class="label">{{$key}}:</span>
                        <span class="value">{{$value}}</span>
                    </div>
                    {{end}}
                </div>
                {{end}}
                
                {{if .DebugInfo}}
                <div class="card">
                    <h3>Debug Info</h3>
                    {{range $key, $value := .DebugInfo}}
                    <div class="metric">
                        <span class="label">{{$key}}:</span>
                        <span class="value">{{$value}}</span>
                    </div>
                    {{end}}
                </div>
                {{end}}
                
                {{if .RecentEvents}}
                <div class="card wide">
                    <h3>Recent Events</h3>
                    <div class="events">
                        {{range .RecentEvents}}
                        <div class="event {{.level}}">
                            <span class="timestamp">{{.timestamp}}</span>
                            <span class="type">{{.type}}</span>
                            <span class="message">{{.message}}</span>
                        </div>
                        {{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </main>
    </div>
    
    <script src="/static/dashboard.js"></script>
</body>
</html>`
}

// getDefaultCSS returns default CSS for the dashboard
func (d *Dashboard) getDefaultCSS() string {
	return `
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    margin: 0;
    padding: 0;
    background-color: #1a1a1a;
    color: #ffffff;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;
    padding-bottom: 20px;
    border-bottom: 1px solid #333;
}

h1 {
    margin: 0;
    color: #4CAF50;
}

.status {
    display: flex;
    align-items: center;
    gap: 10px;
}

.status-indicator {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: #4CAF50;
}

.status-indicator.error {
    background-color: #f44336;
}

.status-indicator.warning {
    background-color: #ff9800;
}

.grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;
}

.card {
    background-color: #2d2d2d;
    border-radius: 8px;
    padding: 20px;
    border: 1px solid #333;
}

.card.wide {
    grid-column: 1 / -1;
}

.card h3 {
    margin: 0 0 15px 0;
    color: #4CAF50;
    border-bottom: 1px solid #333;
    padding-bottom: 10px;
}

.metric {
    display: flex;
    justify-content: space-between;
    margin-bottom: 10px;
    padding: 5px 0;
}

.metric .label {
    color: #bbb;
}

.metric .value {
    color: #fff;
    font-weight: bold;
}

.events {
    max-height: 300px;
    overflow-y: auto;
}

.event {
    display: flex;
    gap: 15px;
    margin-bottom: 10px;
    padding: 8px;
    border-left: 3px solid #4CAF50;
    background-color: #1a1a1a;
    border-radius: 4px;
}

.event.error {
    border-left-color: #f44336;
}

.event.warning {
    border-left-color: #ff9800;
}

.event .timestamp {
    color: #888;
    font-size: 0.9em;
    min-width: 120px;
}

.event .type {
    color: #4CAF50;
    font-weight: bold;
    min-width: 80px;
}

.event .message {
    color: #fff;
}

.uptime {
    color: #888;
    font-size: 0.9em;
}
`
}

// getDefaultJS returns default JavaScript for the dashboard
func (d *Dashboard) getDefaultJS() string {
	return `
// Dashboard JavaScript functionality
(function() {
    'use strict';
    
    // Auto-refresh functionality
    let refreshInterval;
    
    function startAutoRefresh() {
        const refreshRate = parseInt(document.querySelector('meta[http-equiv="refresh"]').content) * 1000;
        
        refreshInterval = setInterval(() => {
            fetch('/api/data')
                .then(response => response.json())
                .then(data => {
                    // Update specific dashboard elements
                    updateDashboard(data);
                })
                .catch(error => {
                    console.error('Failed to refresh dashboard:', error);
                });
        }, refreshRate);
    }
    
    function updateDashboard(data) {
        // Update timestamp
        const timestampElements = document.querySelectorAll('.timestamp');
        timestampElements.forEach(el => {
            if (el.textContent.includes('Last Updated')) {
                el.textContent = new Date(data.timestamp).toLocaleTimeString();
            }
        });
        
        // Update status indicator
        const statusIndicator = document.querySelector('.status-indicator');
        if (statusIndicator) {
            statusIndicator.className = 'status-indicator ' + data.status;
        }
        
        // Update uptime
        const uptimeElements = document.querySelectorAll('.uptime');
        uptimeElements.forEach(el => {
            el.textContent = 'Uptime: ' + data.uptime;
        });
    }
    
    // Initialize when DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
        startAutoRefresh();
        
        // Add click handlers for interactive elements
        addInteractivity();
    });
    
    function addInteractivity() {
        // Add click handlers for cards to expand/collapse
        const cards = document.querySelectorAll('.card');
        cards.forEach(card => {
            const header = card.querySelector('h3');
            if (header) {
                header.style.cursor = 'pointer';
                header.addEventListener('click', () => {
                    const content = card.querySelector('.metric, .events');
                    if (content) {
                        content.style.display = content.style.display === 'none' ? 'block' : 'none';
                    }
                });
            }
        });
    }
    
    // Cleanup on page unload
    window.addEventListener('beforeunload', () => {
        if (refreshInterval) {
            clearInterval(refreshInterval);
        }
    });
})();
`
}
