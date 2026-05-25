package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathResolutionBenchmark(t *testing.T) {
	// Set up a temporary home and sandbox for testing
	homeDir, err := os.MkdirTemp("", "golem-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(homeDir)
	
	sandboxDir, err := os.MkdirTemp("", "golem-sandbox-*")
	if err != nil {
		t.Fatalf("Failed to create temp sandbox: %v", err)
	}
	defer os.RemoveAll(sandboxDir)

	// Mock sandboxRoot
	sandboxRoot = sandboxDir

	// Create some expected structure
	os.MkdirAll(filepath.Join(homeDir, "dev", "project-overviews"), 0755)
	os.MkdirAll(filepath.Join(homeDir, "personal"), 0755)
	os.MkdirAll(filepath.Join(sandboxDir, "robocode-ai"), 0755)
	os.MkdirAll(filepath.Join(homeDir, "Downloads"), 0755)
	os.MkdirAll(filepath.Join(homeDir, "Documents"), 0755)

	// Test Cases
	cases := []struct {
		Name          string
		RawPath       string
		IsDest        bool
		Expected      string
		ExpectedError bool
	}{
		{
			Name:     "Resolve to home personal dir",
			RawPath:  "personal",
			IsDest:   false,
			Expected: filepath.Join(homeDir, "personal"),
		},
		{
			Name:     "Resolve to local sandbox dir",
			RawPath:  "robocode-ai",
			IsDest:   false,
			Expected: filepath.Join(sandboxDir, "robocode-ai"),
		},
		{
			Name:     "Resolve dot prefix to home",
			RawPath:  ".Downloads",
			IsDest:   false,
			Expected: filepath.Join(homeDir, "Downloads"),
		},
		{
			Name:     "Resolve dest path even if it doesnt exist",
			RawPath:  "dev/project-dump",
			IsDest:   true,
			Expected: filepath.Join(homeDir, "dev/project-dump"),
		},
	}

	successCount := 0
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// Override home dir for tests
			originalHome, _ := os.UserHomeDir()
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", originalHome)

			resolved, _, err := resolveSmartPath(c.RawPath, "test", c.IsDest, false)
			if c.ExpectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resolved != c.Expected {
					t.Errorf("Mismatch. Expected: %s, Got: %s", c.Expected, resolved)
				} else {
					successCount++
				}
			}
		})
	}
	t.Logf("Path Resolution Reliability: %.1f%% (%d/%d)", float64(successCount)/float64(len(cases))*100, successCount, len(cases))
}
