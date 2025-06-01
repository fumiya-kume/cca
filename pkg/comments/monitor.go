package comments

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CommentMonitor monitors for new comments and tracks resolution
type CommentMonitor struct {
	config       CommentMonitorConfig
	tracking     map[int]*CommentTracker
	callbacks    map[string]CommentCallback
	mutex        sync.RWMutex
	stopChannels map[string]chan bool
	isMonitoring bool

	// Goroutine lifecycle management
	activeGoroutines sync.WaitGroup
	shutdownCtx      context.Context
	shutdownCancel   context.CancelFunc
}

// CommentMonitorConfig configures the comment monitor
type CommentMonitorConfig struct {
	PollInterval       time.Duration
	TrackResolution    bool
	EscalationRules    []EscalationRule
	AlertOnFailure     bool
	MaxTrackingTime    time.Duration
	ResolutionTimeout  time.Duration
	RetryAttempts      int
	NotificationConfig NotificationConfig
}

// CommentTracker tracks the state of a comment
type CommentTracker struct {
	Comment       *Comment            `json:"comment"`
	FirstSeen     time.Time           `json:"first_seen"`
	LastUpdated   time.Time           `json:"last_updated"`
	ResponseCount int                 `json:"response_count"`
	Status        TrackingStatus      `json:"status"`
	Escalations   []EscalationEvent   `json:"escalations"`
	Resolution    *ResolutionEvent    `json:"resolution,omitempty"`
	Metrics       TrackingMetrics     `json:"metrics"`
	Notifications []NotificationEvent `json:"notifications"`
}

// CommentCallback is called when a comment event occurs
type CommentCallback func(ctx context.Context, comment *Comment, client GitHubClient) error

// TrackingStatus represents the tracking status of a comment
type TrackingStatus int

const (
	TrackingStatusActive TrackingStatus = iota
	TrackingStatusResolved
	TrackingStatusEscalated
	TrackingStatusExpired
	TrackingStatusIgnored
)

// EscalationEvent represents an escalation that occurred
type EscalationEvent struct {
	Rule        EscalationRule `json:"rule"`
	TriggeredAt time.Time      `json:"triggered_at"`
	Reason      string         `json:"reason"`
	Recipients  []string       `json:"recipients"`
	Status      string         `json:"status"`
	Response    string         `json:"response,omitempty"`
}

// ResolutionEvent represents the resolution of a comment
type ResolutionEvent struct {
	ResolvedAt     time.Time     `json:"resolved_at"`
	ResolvedBy     string        `json:"resolved_by"`
	Method         string        `json:"method"`
	TotalTime      time.Duration `json:"total_time"`
	ResponseTime   time.Duration `json:"response_time"`
	ResolutionTime time.Duration `json:"resolution_time"`
}

// TrackingMetrics contains metrics about comment tracking
type TrackingMetrics struct {
	TimeToFirstResponse time.Duration `json:"time_to_first_response"`
	TimeToResolution    time.Duration `json:"time_to_resolution"`
	EscalationCount     int           `json:"escalation_count"`
	NotificationCount   int           `json:"notification_count"`
	InteractionCount    int           `json:"interaction_count"`
}

// NotificationConfig configures notifications
type NotificationConfig struct {
	Enabled    bool              `json:"enabled"`
	Channels   []string          `json:"channels"`
	Recipients []string          `json:"recipients"`
	Templates  map[string]string `json:"templates"`
	Throttle   time.Duration     `json:"throttle"`
}

// NotificationEvent represents a notification that was sent
type NotificationEvent struct {
	Type       string    `json:"type"`
	Recipients []string  `json:"recipients"`
	Channel    string    `json:"channel"`
	Message    string    `json:"message"`
	SentAt     time.Time `json:"sent_at"`
	Status     string    `json:"status"`
	Response   string    `json:"response,omitempty"`
}

// NewCommentMonitor creates a new comment monitor
func NewCommentMonitor(config CommentMonitorConfig) *CommentMonitor {
	// Set defaults
	if config.PollInterval == 0 {
		config.PollInterval = time.Minute * 2
	}
	if config.MaxTrackingTime == 0 {
		config.MaxTrackingTime = time.Hour * 24 * 7 // 1 week
	}
	if config.ResolutionTimeout == 0 {
		config.ResolutionTimeout = time.Hour * 24 // 1 day
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &CommentMonitor{
		config:         config,
		tracking:       make(map[int]*CommentTracker),
		callbacks:      make(map[string]CommentCallback),
		stopChannels:   make(map[string]chan bool),
		isMonitoring:   false,
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
}

// StartMonitoring starts monitoring comments for a PR
func (cm *CommentMonitor) StartMonitoring(ctx context.Context, prNumber int, client GitHubClient, callback CommentCallback) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	monitorKey := fmt.Sprintf("pr-%d", prNumber)

	// Stop existing monitoring if running
	if stopChan, exists := cm.stopChannels[monitorKey]; exists {
		// Safely close channel by checking if it's already closed
		select {
		case <-stopChan:
			// Channel already closed
		default:
			close(stopChan)
		}
		delete(cm.stopChannels, monitorKey)
	}

	// Create new stop channel
	stopChan := make(chan bool, 1) // Buffered to prevent blocking
	cm.stopChannels[monitorKey] = stopChan
	cm.callbacks[monitorKey] = callback
	cm.isMonitoring = true

	// Track the goroutine for proper cleanup
	cm.activeGoroutines.Add(1)

	// Start monitoring goroutine with combined context
	monitorCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer func() {
			cancel()
			cm.activeGoroutines.Done()
		}()
		cm.monitorLoop(monitorCtx, prNumber, client, callback, stopChan)
	}()

	return nil
}

// StopMonitoring stops monitoring for a specific PR
func (cm *CommentMonitor) StopMonitoring(prNumber int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	monitorKey := fmt.Sprintf("pr-%d", prNumber)

	if stopChan, exists := cm.stopChannels[monitorKey]; exists {
		// Safely signal stop by checking if channel is still open
		select {
		case <-stopChan:
			// Channel already closed
		default:
			close(stopChan)
		}
		delete(cm.stopChannels, monitorKey)
		delete(cm.callbacks, monitorKey)
	}

	// Check if all monitoring stopped
	if len(cm.stopChannels) == 0 {
		cm.isMonitoring = false
	}
}

// StopAllMonitoring stops all monitoring
func (cm *CommentMonitor) StopAllMonitoring() {
	cm.mutex.Lock()

	// Safely close all channels
	for key, stopChan := range cm.stopChannels {
		select {
		case <-stopChan:
			// Channel already closed
		default:
			close(stopChan)
		}
		delete(cm.stopChannels, key)
	}

	cm.callbacks = make(map[string]CommentCallback)
	cm.isMonitoring = false
	cm.mutex.Unlock()

	// Wait for all monitoring goroutines to finish
	cm.activeGoroutines.Wait()
}

// Shutdown gracefully shuts down the comment monitor
func (cm *CommentMonitor) Shutdown() {
	// Signal shutdown to all goroutines
	cm.shutdownCancel()

	// Stop all monitoring
	cm.StopAllMonitoring()
}

// TrackComment starts tracking a specific comment
func (cm *CommentMonitor) TrackComment(comment *Comment) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.tracking[comment.ID]; !exists {
		tracker := &CommentTracker{
			Comment:       comment,
			FirstSeen:     time.Now(),
			LastUpdated:   time.Now(),
			Status:        TrackingStatusActive,
			Escalations:   []EscalationEvent{},
			Notifications: []NotificationEvent{},
			Metrics: TrackingMetrics{
				TimeToFirstResponse: 0,
				TimeToResolution:    0,
				EscalationCount:     0,
				NotificationCount:   0,
				InteractionCount:    0,
			},
		}
		cm.tracking[comment.ID] = tracker
	}
}

// UpdateCommentStatus updates the tracking status of a comment
func (cm *CommentMonitor) UpdateCommentStatus(commentID int, status CommentStatus) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if tracker, exists := cm.tracking[commentID]; exists {
		tracker.LastUpdated = time.Now()
		tracker.Comment.Status = status
		tracker.Metrics.InteractionCount++

		// Update tracking status based on comment status
		switch status {
		case CommentStatusResolved:
			tracker.Status = TrackingStatusResolved
			tracker.Resolution = &ResolutionEvent{
				ResolvedAt:     time.Now(),
				ResolvedBy:     "ccagents-ai",
				Method:         "automatic",
				TotalTime:      time.Since(tracker.FirstSeen),
				ResponseTime:   tracker.Metrics.TimeToFirstResponse,
				ResolutionTime: time.Since(tracker.FirstSeen),
			}
		case CommentStatusEscalated:
			tracker.Status = TrackingStatusEscalated
		case CommentStatusDismissed:
			tracker.Status = TrackingStatusIgnored
		}
	}
}

// GetTrackingStats returns tracking statistics
func (cm *CommentMonitor) GetTrackingStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interface{})

	totalTracked := len(cm.tracking)
	activeCount := 0
	resolvedCount := 0
	escalatedCount := 0

	var totalResolutionTime time.Duration
	var totalResponseTime time.Duration
	resolvedComments := 0

	for _, tracker := range cm.tracking {
		switch tracker.Status {
		case TrackingStatusActive:
			activeCount++
		case TrackingStatusResolved:
			resolvedCount++
			if tracker.Resolution != nil {
				totalResolutionTime += tracker.Resolution.TotalTime
				totalResponseTime += tracker.Resolution.ResponseTime
				resolvedComments++
			}
		case TrackingStatusEscalated:
			escalatedCount++
		}
	}

	stats["total_tracked"] = totalTracked
	stats["active"] = activeCount
	stats["resolved"] = resolvedCount
	stats["escalated"] = escalatedCount
	stats["is_monitoring"] = cm.isMonitoring

	if resolvedComments > 0 {
		stats["avg_resolution_time"] = totalResolutionTime / time.Duration(resolvedComments)
		stats["avg_response_time"] = totalResponseTime / time.Duration(resolvedComments)
	}

	return stats
}

// monitorLoop runs the monitoring loop for a specific PR
func (cm *CommentMonitor) monitorLoop(ctx context.Context, prNumber int, client GitHubClient, callback CommentCallback, stopChan chan bool) {
	ticker := time.NewTicker(cm.config.PollInterval)
	defer ticker.Stop()

	var lastSeen time.Time

	for {
		select {
		case <-stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cm.checkForNewComments(ctx, prNumber, client, callback, lastSeen); err != nil {
				if cm.config.AlertOnFailure {
					fmt.Printf("Warning: failed to check for new comments: %v\n", err)
				}
			} else {
				lastSeen = time.Now()
			}

			// Check for timeouts and escalations
			cm.checkTimeouts()
			cm.checkEscalations(ctx, prNumber, client)
		}
	}
}

// checkForNewComments checks for new comments since last check
func (cm *CommentMonitor) checkForNewComments(ctx context.Context, prNumber int, client GitHubClient, callback CommentCallback, since time.Time) error {
	comments, err := client.ListComments(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	for _, comment := range comments {
		// Skip old comments
		if !since.IsZero() && comment.CreatedAt.Before(since) {
			continue
		}

		// Start tracking if not already tracked
		cm.TrackComment(comment)

		// Call the callback for new or updated comments
		if err := callback(ctx, comment, client); err != nil {
			fmt.Printf("Warning: callback failed for comment %d: %v\n", comment.ID, err)
		}

		// Record first response time if this is a response
		cm.recordResponseTime(comment)
	}

	return nil
}

// recordResponseTime records the time to first response
func (cm *CommentMonitor) recordResponseTime(comment *Comment) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if tracker, exists := cm.tracking[comment.ID]; exists {
		// If this is a response from the AI and we haven't recorded first response time
		if comment.Author == "ccagents-ai" && tracker.Metrics.TimeToFirstResponse == 0 {
			tracker.Metrics.TimeToFirstResponse = time.Since(tracker.FirstSeen)
			tracker.ResponseCount++
		}
	}
}

// checkTimeouts checks for comments that have timed out
func (cm *CommentMonitor) checkTimeouts() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()

	for commentID, tracker := range cm.tracking {
		// Check if comment has expired
		if tracker.Status == TrackingStatusActive {
			if now.Sub(tracker.FirstSeen) > cm.config.MaxTrackingTime {
				tracker.Status = TrackingStatusExpired
				fmt.Printf("Comment %d expired after %v\n", commentID, cm.config.MaxTrackingTime)
			}

			// Check resolution timeout
			if cm.config.TrackResolution &&
				now.Sub(tracker.FirstSeen) > cm.config.ResolutionTimeout &&
				tracker.Comment.Status != CommentStatusResolved {

				fmt.Printf("Comment %d resolution timeout after %v\n", commentID, cm.config.ResolutionTimeout)
				// Could trigger escalation here
			}
		}
	}
}

// checkEscalations checks if any comments need escalation
func (cm *CommentMonitor) checkEscalations(ctx context.Context, prNumber int, client GitHubClient) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for commentID, tracker := range cm.tracking {
		if tracker.Status != TrackingStatusActive {
			continue
		}

		for _, rule := range cm.config.EscalationRules {
			if cm.shouldEscalateComment(tracker, rule) {
				if err := cm.escalateComment(ctx, prNumber, tracker, rule, client); err != nil {
					fmt.Printf("Warning: failed to escalate comment %d: %v\n", commentID, err)
				}
			}
		}
	}
}

// shouldEscalateComment determines if a comment should be escalated based on a rule
func (cm *CommentMonitor) shouldEscalateComment(tracker *CommentTracker, rule EscalationRule) bool {
	// Check if this rule has already been triggered
	for _, escalation := range tracker.Escalations {
		if escalation.Rule.Name == rule.Name {
			return false // Already escalated for this rule
		}
	}

	now := time.Now()

	switch rule.Condition {
	case "blocking":
		return tracker.Comment.Priority == CommentPriorityBlocking
	case "critical":
		return tracker.Comment.Priority == CommentPriorityCritical
	case "unresolved_time":
		return now.Sub(tracker.FirstSeen) > rule.Delay
	case "no_response":
		return tracker.ResponseCount == 0 && now.Sub(tracker.FirstSeen) > rule.Delay
	case "multiple_interactions":
		return tracker.Metrics.InteractionCount >= rule.Threshold
	default:
		return false
	}
}

// escalateComment escalates a comment according to a rule
func (cm *CommentMonitor) escalateComment(ctx context.Context, prNumber int, tracker *CommentTracker, rule EscalationRule, client GitHubClient) error {
	escalation := EscalationEvent{
		Rule:        rule,
		TriggeredAt: time.Now(),
		Reason:      fmt.Sprintf("Rule '%s' triggered", rule.Name),
		Recipients:  rule.Recipients,
		Status:      "triggered",
	}

	// Execute escalation action
	switch rule.Action {
	case "notify":
		if err := cm.sendNotification(tracker.Comment, rule); err != nil {
			escalation.Status = "failed"
			escalation.Response = err.Error()
		} else {
			escalation.Status = "sent"
		}
	case "comment":
		escalationMsg := cm.generateEscalationComment(tracker.Comment, rule)
		if _, err := client.CreateComment(ctx, prNumber, escalationMsg); err != nil {
			escalation.Status = "failed"
			escalation.Response = err.Error()
		} else {
			escalation.Status = "posted"
		}
	case "assign":
		// Would assign the PR to specific people
		escalation.Status = "assigned"
	}

	tracker.Escalations = append(tracker.Escalations, escalation)
	tracker.Metrics.EscalationCount++

	// Update comment status
	tracker.Comment.Status = CommentStatusEscalated
	tracker.Status = TrackingStatusEscalated

	return nil
}

// sendNotification sends a notification about the comment
func (cm *CommentMonitor) sendNotification(comment *Comment, rule EscalationRule) error {
	if !cm.config.NotificationConfig.Enabled {
		return nil
	}

	// This would integrate with notification systems (Slack, email, etc.)
	fmt.Printf("NOTIFICATION: Comment %d escalated via rule '%s' to %v\n",
		comment.ID, rule.Name, rule.Recipients)

	return nil
}

// generateEscalationComment generates an escalation comment
func (cm *CommentMonitor) generateEscalationComment(comment *Comment, rule EscalationRule) string {
	return fmt.Sprintf("🚨 **Escalation**: Comment requires attention\n\n"+
		"**Original comment** by @%s:\n> %s\n\n"+
		"**Escalation rule**: %s\n"+
		"**Reason**: %s\n"+
		"**Recipients**: %s\n\n"+
		"Please review and provide guidance.",
		comment.Author,
		comment.Body,
		rule.Name,
		rule.Condition,
		fmt.Sprintf("%v", rule.Recipients))
}

// String method for TrackingStatus
func (ts TrackingStatus) String() string {
	switch ts {
	case TrackingStatusActive:
		return "active"
	case TrackingStatusResolved:
		return "resolved"
	case TrackingStatusEscalated:
		return "escalated"
	case TrackingStatusExpired:
		return "expired"
	case TrackingStatusIgnored:
		return "ignored"
	default:
		return "unknown"
	}
}
