package ui

import (
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

// BeepProvider interface for sound beep functionality
type BeepProvider interface {
	Beep(frequency float64, duration int) error
}

// RealBeepProvider implements BeepProvider using actual beeep library
type RealBeepProvider struct{}

func (r *RealBeepProvider) Beep(frequency float64, duration int) error {
	return beeep.Beep(frequency, duration)
}

// FakeBeepProvider implements BeepProvider for testing without actual sound.
// Use this in unit tests to avoid playing actual sounds during test execution.
//
// Example usage:
//
//	config := DefaultSoundConfig()
//	fakeProvider := &FakeBeepProvider{}
//	notifier := NewSoundNotifierWithProvider(config, fakeProvider)
//
//	notifier.PlayConfirmationSound()
//	fakeProvider.WaitForCalls(1) // Wait for expected number of calls
//
//	assert.Equal(t, 1, fakeProvider.CallCount)
//	assert.Equal(t, 800.0, fakeProvider.LastFreq)
type FakeBeepProvider struct {
	CallCount int
	LastFreq  float64
	LastDur   int
	Calls     []BeepCall
	doneCh    chan struct{}
	mu        sync.Mutex
}

type BeepCall struct {
	Frequency float64
	Duration  int
}

func (f *FakeBeepProvider) Beep(frequency float64, duration int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CallCount++
	f.LastFreq = frequency
	f.LastDur = duration
	f.Calls = append(f.Calls, BeepCall{Frequency: frequency, Duration: duration})

	// Signal that a call was made
	if f.doneCh != nil {
		select {
		case f.doneCh <- struct{}{}:
		default:
		}
	}

	return nil
}

// Reset clears all recorded calls
func (f *FakeBeepProvider) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CallCount = 0
	f.LastFreq = 0
	f.LastDur = 0
	f.Calls = nil
	f.doneCh = make(chan struct{}, 10) // Buffered channel for multiple calls
}

// WaitForCalls waits for the specified number of beep calls to complete
func (f *FakeBeepProvider) WaitForCalls(expectedCalls int) {
	f.mu.Lock()
	if f.doneCh == nil {
		f.doneCh = make(chan struct{}, 10)
	}
	f.mu.Unlock()

	for i := 0; i < expectedCalls; i++ {
		<-f.doneCh
	}
}

// NewTestSoundNotifier creates a SoundNotifier with FakeBeepProvider for testing.
// This is a convenience function for tests that don't need to inspect beep calls.
func NewTestSoundNotifier(config SoundConfig) (*SoundNotifier, *FakeBeepProvider) {
	fakeProvider := &FakeBeepProvider{}
	notifier := NewSoundNotifierWithProvider(config, fakeProvider)
	return notifier, fakeProvider
}

// SoundConfig holds sound notification settings
type SoundConfig struct {
	Enabled   bool
	Frequency int           // Hz for beep sound
	Duration  time.Duration // Duration of the beep
	Volume    float64       // Volume (0.0 to 1.0) - for future use
	BeepDelay time.Duration // Delay between beeps in sequences
}

// DefaultSoundConfig returns default sound configuration
func DefaultSoundConfig() SoundConfig {
	return SoundConfig{
		Enabled:   true,
		Frequency: 800,                    // 800Hz - pleasant notification tone
		Duration:  200 * time.Millisecond, // 200ms beep
		Volume:    0.7,                    // 70% volume (not used by beeep yet)
		BeepDelay: 50 * time.Millisecond,  // 50ms delay between beeps
	}
}

// TestSoundConfig returns a sound configuration optimized for testing (no delays)
func TestSoundConfig() SoundConfig {
	return SoundConfig{
		Enabled:   true,
		Frequency: 800,
		Duration:  200 * time.Millisecond,
		Volume:    0.7,
		BeepDelay: 0, // No delay in tests
	}
}

// SoundNotifier handles audio notifications
type SoundNotifier struct {
	config   SoundConfig
	provider BeepProvider
}

// NewSoundNotifier creates a new sound notifier with real beep provider
func NewSoundNotifier(config SoundConfig) *SoundNotifier {
	return &SoundNotifier{
		config:   config,
		provider: &RealBeepProvider{},
	}
}

// NewSoundNotifierWithProvider creates a new sound notifier with custom provider
func NewSoundNotifierWithProvider(config SoundConfig, provider BeepProvider) *SoundNotifier {
	return &SoundNotifier{
		config:   config,
		provider: provider,
	}
}

// PlayConfirmationSound plays a sound when user confirmation is requested
func (s *SoundNotifier) PlayConfirmationSound() {
	if !s.config.Enabled {
		return
	}

	// Play beep in background to avoid blocking UI
	go func() {
		_ = s.provider.Beep(float64(s.config.Frequency), int(s.config.Duration.Milliseconds())) //nolint:errcheck // UI feedback errors are not critical
	}()
}

// PlaySuccessSound plays a sound for successful operations
func (s *SoundNotifier) PlaySuccessSound() {
	if !s.config.Enabled {
		return
	}

	// Two quick beeps for success (higher pitch)
	go func() {
		_ = s.provider.Beep(1000.0, 100) //nolint:errcheck // UI feedback errors are not critical
		if s.config.BeepDelay > 0 {
			timer := time.NewTimer(s.config.BeepDelay)
			<-timer.C
		}
		_ = s.provider.Beep(1200.0, 100) //nolint:errcheck // UI feedback errors are not critical
	}()
}

// PlayErrorSound plays a sound for errors
func (s *SoundNotifier) PlayErrorSound() {
	if !s.config.Enabled {
		return
	}

	// Lower pitch, longer beep for errors
	go func() {
		_ = s.provider.Beep(400.0, 400) //nolint:errcheck // UI feedback errors are not critical
	}()
}

// PlayWorkflowCompleteSound plays a sound when workflow completes
func (s *SoundNotifier) PlayWorkflowCompleteSound() {
	if !s.config.Enabled {
		return
	}

	// Ascending melody for completion
	go func() {
		frequencies := []int{600, 700, 800, 900}
		for i, freq := range frequencies {
			_ = s.provider.Beep(float64(freq), 150) //nolint:errcheck // UI feedback errors are not critical
			// Sleep between beeps except after the last one
			if i < len(frequencies)-1 && s.config.BeepDelay > 0 {
				timer := time.NewTimer(s.config.BeepDelay * 2) // Double delay for melody
				<-timer.C
			}
		}
	}()
}

// SetEnabled enables or disables sound notifications
func (s *SoundNotifier) SetEnabled(enabled bool) {
	s.config.Enabled = enabled
}

// IsEnabled returns whether sound notifications are enabled
func (s *SoundNotifier) IsEnabled() bool {
	return s.config.Enabled
}
