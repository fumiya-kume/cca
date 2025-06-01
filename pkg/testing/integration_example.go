package testing

import (
	"context"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// Example showing how existing code can be refactored to use Clock interface
// This demonstrates the pattern for updating pkg/agents/health.go

type HealthMonitor struct {
	clock         clock.Clock
	checkInterval time.Duration
	timeout       time.Duration
	mu            sync.RWMutex
	agents        map[string]*AgentHealth
	running       bool
	stopCh        chan struct{}
}

type AgentHealth struct {
	ID           string
	LastCheck    time.Time
	Status       HealthStatus
	ResponseTime time.Duration
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

func NewHealthMonitor(clk clock.Clock) *HealthMonitor {
	return &HealthMonitor{
		clock:         clk,
		checkInterval: 30 * time.Second,
		timeout:       10 * time.Second,
		agents:        make(map[string]*AgentHealth),
		stopCh:        make(chan struct{}),
	}
}

func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.mu.Lock()
	if hm.running {
		hm.mu.Unlock()
		return nil
	}
	hm.running = true
	hm.mu.Unlock()

	ticker := hm.clock.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			hm.performHealthChecks(ctx)
		case <-hm.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		close(hm.stopCh)
		hm.running = false
	}
}

func (hm *HealthMonitor) performHealthChecks(ctx context.Context) {
	hm.mu.RLock()
	agents := make([]string, 0, len(hm.agents))
	for id := range hm.agents {
		agents = append(agents, id)
	}
	hm.mu.RUnlock()

	for _, agentID := range agents {
		go hm.checkAgent(ctx, agentID)
	}
}

func (hm *HealthMonitor) checkAgent(ctx context.Context, agentID string) {
	start := hm.clock.Now()

	// Create timeout context using our clock
	timeoutCtx, cancel := clock.WithTimeout(ctx, hm.clock, hm.timeout)
	defer cancel()

	// Simulate health check
	result := hm.performActualHealthCheck(timeoutCtx, agentID)

	responseTime := hm.clock.Since(start)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if agent, exists := hm.agents[agentID]; exists {
		agent.LastCheck = hm.clock.Now()
		agent.Status = result
		agent.ResponseTime = responseTime
	}
}

func (hm *HealthMonitor) performActualHealthCheck(ctx context.Context, agentID string) HealthStatus {
	// Simulate some work that might take time
	select {
	case <-hm.clock.After(100 * time.Millisecond):
		return HealthStatusHealthy
	case <-ctx.Done():
		return HealthStatusUnhealthy
	}
}

func (hm *HealthMonitor) RegisterAgent(agentID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.agents[agentID] = &AgentHealth{
		ID:        agentID,
		LastCheck: hm.clock.Now(),
		Status:    HealthStatusUnknown,
	}
}

func (hm *HealthMonitor) GetAgentHealth(agentID string) (*AgentHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	agent, exists := hm.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &AgentHealth{
		ID:           agent.ID,
		LastCheck:    agent.LastCheck,
		Status:       agent.Status,
		ResponseTime: agent.ResponseTime,
	}, true
}

// RetryConfig demonstrates retry logic that would benefit from fake clock
type RetryConfig struct {
	MaxAttempts         int
	InitialInterval     time.Duration
	Multiplier          float64
	MaxInterval         time.Duration
	RandomizationFactor float64
}

func RetryWithClock(ctx context.Context, clk clock.Clock, config RetryConfig, operation func() error) error {
	return RetryWithClockAndNotify(ctx, clk, config, operation, nil)
}

func RetryWithClockAndNotify(ctx context.Context, clk clock.Clock, config RetryConfig, operation func() error, waitingChan chan<- struct{}) error {
	var lastErr error
	interval := config.InitialInterval

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Signal that we're about to wait (for testing)
			if waitingChan != nil {
				select {
				case waitingChan <- struct{}{}:
				default:
				}
			}

			// Wait before retry (except first attempt)
			select {
			case <-clk.After(interval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Calculate next interval with exponential backoff
		interval = time.Duration(float64(interval) * config.Multiplier)
		if interval > config.MaxInterval {
			interval = config.MaxInterval
		}
	}

	return lastErr
}

// CacheEntry demonstrates cache expiration that would benefit from fake clock
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

type TimeBasedCache struct {
	clock   clock.Clock
	entries map[string]*CacheEntry
	mu      sync.RWMutex
}

func NewTimeBasedCache(clk clock.Clock) *TimeBasedCache {
	return &TimeBasedCache{
		clock:   clk,
		entries: make(map[string]*CacheEntry),
	}
}

func (c *TimeBasedCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Value:      value,
		Expiration: c.clock.Now().Add(ttl),
	}
}

func (c *TimeBasedCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if c.clock.Now().After(entry.Expiration) {
		// Entry has expired
		return nil, false
	}

	return entry.Value, true
}

func (c *TimeBasedCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()
	for key, entry := range c.entries {
		if now.After(entry.Expiration) {
			delete(c.entries, key)
		}
	}
}
