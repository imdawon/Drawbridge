package main

import (
	"testing"
)

// TestVersionFormatting is a simple test to ensure the project can be tested from root
// This ensures that `go test` runs successfully at the project root
func TestVersionFormatting(t *testing.T) {
	// Simple test just to verify testing works from root
	t.Run("Version format check", func(t *testing.T) {
		testVersion := "v1.0.0"
		if len(testVersion) < 2 || testVersion[0] != 'v' {
			t.Errorf("Version %s doesn't match expected format (should start with 'v')", testVersion)
		}
	})
}