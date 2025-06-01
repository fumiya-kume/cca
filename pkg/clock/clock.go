// Package clock provides time utilities and clock interfaces for testing and time management in ccAgents
package clock

import (
	"context"
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	NewTicker(d time.Duration) Ticker
	Sleep(d time.Duration)
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

type RealClock struct{}

func NewRealClock() Clock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *RealClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(d)}
}

func (c *RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *RealClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

type realTicker struct {
	ticker *time.Ticker
}

func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}

func (t *realTicker) Reset(d time.Duration) {
	t.ticker.Reset(d)
}

type FakeClock struct {
	mu        sync.RWMutex
	now       time.Time
	waiters   []waiter
	tickers   []*fakeTicker
	advanceMu sync.Mutex
}

type waiter struct {
	ch       chan time.Time
	deadline time.Time
}

func NewFakeClock(now time.Time) *FakeClock {
	return &FakeClock{
		now: now,
	}
}

func (c *FakeClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan time.Time, 1)
	deadline := c.now.Add(d)

	if d <= 0 {
		ch <- c.now
		return ch
	}

	c.waiters = append(c.waiters, waiter{ch: ch, deadline: deadline})
	return ch
}

func (c *FakeClock) NewTicker(d time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()

	ticker := &fakeTicker{
		clock:    c,
		interval: d,
		ch:       make(chan time.Time, 1),
		next:     c.now.Add(d),
		stopped:  false,
	}

	c.tickers = append(c.tickers, ticker)
	return ticker
}

func (c *FakeClock) Sleep(d time.Duration) {
	<-c.After(d)
}

func (c *FakeClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

func (c *FakeClock) Until(t time.Time) time.Duration {
	return t.Sub(c.Now())
}

func (c *FakeClock) Advance(d time.Duration) {
	c.advanceMu.Lock()
	defer c.advanceMu.Unlock()

	c.mu.Lock()
	c.now = c.now.Add(d)
	newTime := c.now
	c.mu.Unlock()

	c.fireWaiters(newTime)
	c.fireTickers(newTime)
}

func (c *FakeClock) Set(t time.Time) {
	c.advanceMu.Lock()
	defer c.advanceMu.Unlock()

	c.mu.Lock()
	c.now = t
	c.mu.Unlock()

	c.fireWaiters(t)
	c.fireTickers(t)
}

func (c *FakeClock) fireWaiters(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	remaining := c.waiters[:0]
	for _, w := range c.waiters {
		if now.After(w.deadline) || now.Equal(w.deadline) {
			select {
			case w.ch <- now:
			default:
			}
		} else {
			remaining = append(remaining, w)
		}
	}
	c.waiters = remaining
}

func (c *FakeClock) fireTickers(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ticker := range c.tickers {
		if ticker.stopped {
			continue
		}

		for now.After(ticker.next) || now.Equal(ticker.next) {
			select {
			case ticker.ch <- ticker.next:
			default:
			}
			ticker.next = ticker.next.Add(ticker.interval)
		}
	}
}

type fakeTicker struct {
	clock    *FakeClock
	interval time.Duration
	ch       chan time.Time
	next     time.Time
	stopped  bool
	mu       sync.RWMutex
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
}

func (t *fakeTicker) Reset(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.interval = d
	t.next = t.clock.Now().Add(d)
	t.stopped = false
}

func WithTimeout(ctx context.Context, clock Clock, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-clock.After(timeout):
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}
