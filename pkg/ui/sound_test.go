package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSoundNotifier(t *testing.T) {
	config := DefaultSoundConfig()
	notifier := NewSoundNotifier(config)

	// Test initial state
	assert.True(t, notifier.IsEnabled())

	// Test enable/disable
	notifier.SetEnabled(false)
	assert.False(t, notifier.IsEnabled())

	notifier.SetEnabled(true)
	assert.True(t, notifier.IsEnabled())
}

func TestDefaultSoundConfig(t *testing.T) {
	config := DefaultSoundConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 800, config.Frequency)
	assert.Equal(t, 200*time.Millisecond, config.Duration)
	assert.Equal(t, 0.7, config.Volume)
}

func TestSoundNotifierPlaySounds(t *testing.T) {
	config := TestSoundConfig() // Use test config with no delays
	fakeProvider := &FakeBeepProvider{}
	notifier := NewSoundNotifierWithProvider(config, fakeProvider)

	// Test actual sound calls using fake provider
	t.Run("PlayConfirmationSound", func(t *testing.T) {
		fakeProvider.Reset()
		notifier.PlayConfirmationSound()

		// Wait for exactly 1 call instead of sleeping
		fakeProvider.WaitForCalls(1)

		assert.Equal(t, 1, fakeProvider.CallCount)
		assert.Equal(t, float64(config.Frequency), fakeProvider.LastFreq)
		assert.Equal(t, int(config.Duration.Milliseconds()), fakeProvider.LastDur)
	})

	t.Run("PlaySuccessSound", func(t *testing.T) {
		fakeProvider.Reset()
		notifier.PlaySuccessSound()

		// Wait for exactly 2 calls instead of sleeping
		fakeProvider.WaitForCalls(2)

		assert.Equal(t, 2, fakeProvider.CallCount) // Two beeps for success
		assert.Len(t, fakeProvider.Calls, 2)
		assert.Equal(t, 1000.0, fakeProvider.Calls[0].Frequency)
		assert.Equal(t, 1200.0, fakeProvider.Calls[1].Frequency)
	})

	t.Run("PlayErrorSound", func(t *testing.T) {
		fakeProvider.Reset()
		notifier.PlayErrorSound()

		// Wait for exactly 1 call instead of sleeping
		fakeProvider.WaitForCalls(1)

		assert.Equal(t, 1, fakeProvider.CallCount)
		assert.Equal(t, 400.0, fakeProvider.LastFreq)
		assert.Equal(t, 400, fakeProvider.LastDur)
	})

	t.Run("PlayWorkflowCompleteSound", func(t *testing.T) {
		fakeProvider.Reset()
		notifier.PlayWorkflowCompleteSound()

		// Wait for exactly 4 calls instead of sleeping
		fakeProvider.WaitForCalls(4)

		assert.Equal(t, 4, fakeProvider.CallCount) // Four beeps in melody
		expectedFreqs := []float64{600, 700, 800, 900}
		assert.Len(t, fakeProvider.Calls, 4)
		for i, expectedFreq := range expectedFreqs {
			assert.Equal(t, expectedFreq, fakeProvider.Calls[i].Frequency)
			assert.Equal(t, 150, fakeProvider.Calls[i].Duration)
		}
	})
}

func TestSoundNotifierDisabled(t *testing.T) {
	config := DefaultSoundConfig()
	config.Enabled = false
	fakeProvider := &FakeBeepProvider{}
	notifier := NewSoundNotifierWithProvider(config, fakeProvider)

	// When disabled, all sound functions should work but not call beep
	notifier.PlayConfirmationSound()
	notifier.PlaySuccessSound()
	notifier.PlayErrorSound()
	notifier.PlayWorkflowCompleteSound()

	// No calls should be made when disabled - no need to wait

	// Should not have called beep when disabled
	assert.Equal(t, 0, fakeProvider.CallCount)
}

func TestFakeBeepProvider(t *testing.T) {
	fake := &FakeBeepProvider{}

	err := fake.Beep(440.0, 200)
	assert.NoError(t, err)
	assert.Equal(t, 1, fake.CallCount)
	assert.Equal(t, 440.0, fake.LastFreq)
	assert.Equal(t, 200, fake.LastDur)
	assert.Len(t, fake.Calls, 1)
	assert.Equal(t, 440.0, fake.Calls[0].Frequency)
	assert.Equal(t, 200, fake.Calls[0].Duration)
}

func TestNewTestSoundNotifier(t *testing.T) {
	config := TestSoundConfig() // Use test config with no delays
	notifier, fakeProvider := NewTestSoundNotifier(config)

	assert.NotNil(t, notifier)
	assert.NotNil(t, fakeProvider)
	assert.True(t, notifier.IsEnabled())
	assert.Equal(t, 0, fakeProvider.CallCount)

	// Test that it works as expected
	notifier.PlayConfirmationSound()
	fakeProvider.WaitForCalls(1)

	assert.Equal(t, 1, fakeProvider.CallCount)
}
