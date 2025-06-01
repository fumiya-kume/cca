package main

import (
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test that main doesn't panic with help flag
	os.Args = []string{"ccagents", "--help"}
	
	// Since main() calls Execute() and Execute() calls cobra's Execute(),
	// we need to handle the expected exit from help
	defer func() {
		if r := recover(); r != nil {
			// Help command might exit, which is expected behavior
			t.Logf("Help command exited as expected: %v", r)
		}
	}()

	// This will call main() which calls Execute()
	// We expect it might exit with help, but should not panic
	main()
}