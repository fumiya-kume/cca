package errors

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestErrorBuilder(t *testing.T) {
	err := NewError(ErrorTypeValidation).
		WithMessage("invalid input").
		WithSeverity(SeverityLow).
		WithContext("field", "username").
		WithSuggestion("Use alphanumeric characters only").
		WithRecoverable(true).
		Build()

	ccErr, ok := err.(*ccAgentsError)
	if !ok {
		t.Fatal("Expected *ccAgentsError")
	}

	if ccErr.Type() != ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation, got %v", ccErr.Type())
	}

	if ccErr.Severity() != SeverityLow {
		t.Errorf("Expected SeverityLow, got %v", ccErr.Severity())
	}

	if !ccErr.IsRecoverable() {
		t.Error("Expected error to be recoverable")
	}

	suggestions := ccErr.Suggestions()
	if len(suggestions) != 1 || suggestions[0] != "Use alphanumeric characters only" {
		t.Errorf("Expected suggestion not found: %v", suggestions)
	}

	context := ccErr.Context()
	if context["field"] != "username" {
		t.Errorf("Expected context field 'username', got %v", context["field"])
	}
}

func TestErrorMessage(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewError(ErrorTypeNetwork).
		WithMessage("connection failed").
		WithCause(cause).
		WithSeverity(SeverityMedium).
		Build()

	expectedMsg := "[network:medium] connection failed caused by: underlying error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestConvenienceErrors(t *testing.T) {
	tests := []struct {
		name                string
		errFunc             func() error
		expectedType        ErrorType
		expectedRecoverable bool
	}{
		{
			name:                "ValidationError",
			errFunc:             func() error { return ValidationError("test validation") },
			expectedType:        ErrorTypeValidation,
			expectedRecoverable: true,
		},
		{
			name:                "NetworkError",
			errFunc:             func() error { return NetworkError(fmt.Errorf("connection timeout")) },
			expectedType:        ErrorTypeNetwork,
			expectedRecoverable: true,
		},
		{
			name:                "AuthenticationError",
			errFunc:             func() error { return AuthenticationError("github") },
			expectedType:        ErrorTypeAuthentication,
			expectedRecoverable: true,
		},
		{
			name:                "ProcessError",
			errFunc:             func() error { return ProcessError("git", 1, fmt.Errorf("command failed")) },
			expectedType:        ErrorTypeProcess,
			expectedRecoverable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errFunc()

			if !IsType(err, tt.expectedType) {
				t.Errorf("Expected error type %v", tt.expectedType)
			}

			if IsRecoverable(err) != tt.expectedRecoverable {
				t.Errorf("Expected recoverable %v, got %v", tt.expectedRecoverable, IsRecoverable(err))
			}
		})
	}
}

func TestErrorTypeChecking(t *testing.T) {
	networkErr := NetworkError(fmt.Errorf("timeout"))

	if !IsType(networkErr, ErrorTypeNetwork) {
		t.Error("Expected network error to be of type Network")
	}

	if IsType(networkErr, ErrorTypeValidation) {
		t.Error("Expected network error not to be of type Validation")
	}

	if !IsRecoverable(networkErr) {
		t.Error("Expected network error to be recoverable")
	}

	suggestions := GetSuggestions(networkErr)
	if len(suggestions) == 0 {
		t.Error("Expected network error to have suggestions")
	}

	// Test with non-ccAgentsError
	regularErr := fmt.Errorf("regular error")
	if IsType(regularErr, ErrorTypeNetwork) {
		t.Error("Expected regular error not to be typed")
	}

	if IsRecoverable(regularErr) {
		t.Error("Expected regular error not to be recoverable")
	}
}

func TestRetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping time-dependent retry test in short mode")
	}

	ctx := context.Background()

	// Test successful retry
	attempts := 0
	err := RetryWithExponentialBackoff(ctx, 3, func() error {
		attempts++
		if attempts < 3 {
			return NetworkError(fmt.Errorf("temporary failure"))
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Test non-recoverable error (should not retry)
	attempts = 0
	err = RetryWithExponentialBackoff(ctx, 3, func() error {
		attempts++
		return fmt.Errorf("non-recoverable error")
	})

	if err == nil {
		t.Error("Expected error to be returned")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-recoverable error, got %d", attempts)
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = RetryWithExponentialBackoff(ctx, 3, func() error {
		return NetworkError(fmt.Errorf("should not retry"))
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}

	if config.InitialInterval != time.Second {
		t.Errorf("Expected InitialInterval 1s, got %v", config.InitialInterval)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier 2.0, got %f", config.Multiplier)
	}
}

func TestShouldRetryFunctions(t *testing.T) {
	networkErr := NetworkError(fmt.Errorf("timeout"))
	validationErr := ValidationError("invalid")
	regularErr := fmt.Errorf("regular error")

	// Test DefaultShouldRetry
	if !DefaultShouldRetry(networkErr) {
		t.Error("Expected network error to be retryable")
	}

	if !DefaultShouldRetry(validationErr) {
		t.Error("Expected validation error to be retryable")
	}

	if DefaultShouldRetry(regularErr) {
		t.Error("Expected regular error not to be retryable")
	}

	// Test NetworkShouldRetry
	if !NetworkShouldRetry(networkErr) {
		t.Error("Expected network error to be retryable by NetworkShouldRetry")
	}

	if NetworkShouldRetry(validationErr) {
		t.Error("Expected validation error not to be retryable by NetworkShouldRetry")
	}

	if NetworkShouldRetry(regularErr) {
		t.Error("Expected regular error not to be retryable by NetworkShouldRetry")
	}
}
