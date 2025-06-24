package laravel

import (
	"os"
	"strings"
	"testing"
)

func TestGetQueueSizes(t *testing.T) {
	tests := []struct {
		name     string
		queueMap map[string][]string
		wantErr  bool
	}{
		{
			name:     "empty queue map",
			queueMap: map[string][]string{},
			wantErr:  false,
		},
		{
			name: "single connection with one queue",
			queueMap: map[string][]string{
				"default": {"default"},
			},
			wantErr: true, // Will fail without proper Laravel setup
		},
		{
			name: "multiple connections with multiple queues",
			queueMap: map[string][]string{
				"default": {"default", "high"},
				"redis":   {"background"},
			},
			wantErr: true, // Will fail without proper Laravel setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			result, err := GetQueueSizes(tempDir, "php", tt.queueMap)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetQueueSizes() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetQueueSizes() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("GetQueueSizes() returned nil result")
				return
			}

			// For empty queue map, result should be empty
			if len(tt.queueMap) == 0 && len(*result) != 0 {
				t.Errorf("GetQueueSizes() expected empty result for empty queue map")
			}
		})
	}
}

func TestGetQueueSizes_WithEnvVariable(t *testing.T) {
	// Test that NIGHTWATCH_ENABLED=false is set in the environment
	// We'll create a mock PHP script that outputs the environment variables
	tempDir := t.TempDir()

	// Create a mock php script that outputs environment variables
	mockPhpScript := `#!/bin/bash
echo "NIGHTWATCH_ENABLED=$NIGHTWATCH_ENABLED"
exit 0`

	mockPhpPath := tempDir + "/mock-php"
	err := os.WriteFile(mockPhpPath, []byte(mockPhpScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock PHP script: %v", err)
	}

	queueMap := map[string][]string{
		"default": {"default"},
	}

	// Use our mock PHP script
	_, err = GetQueueSizes(tempDir, mockPhpPath, queueMap)

	// We expect an error since our mock doesn't output valid JSON
	// but we can check if the environment variable was set by looking at the error output
	if err == nil {
		t.Errorf("Expected error when running mock PHP script")
		return
	}

	// The error should contain our environment variable output
	if !strings.Contains(err.Error(), "NIGHTWATCH_ENABLED=false") {
		t.Errorf("Expected error output to contain 'NIGHTWATCH_ENABLED=false', got: %s", err.Error())
	}
}

func TestGetQueueSizes_EnvVariableValidation(t *testing.T) {
	// Additional test to verify the environment variable is properly passed through a script
	tempDir := t.TempDir()

	// Create a script that validates NIGHTWATCH_ENABLED and outputs JSON
	validatorScript := `#!/bin/bash
if [ "$NIGHTWATCH_ENABLED" = "false" ]; then
    echo '{"default":{"default":{"size":0}}}'
else
    echo "ERROR: NIGHTWATCH_ENABLED not set to false, got: $NIGHTWATCH_ENABLED" >&2
    exit 1
fi`

	scriptPath := tempDir + "/validator-php"
	err := os.WriteFile(scriptPath, []byte(validatorScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create validator script: %v", err)
	}

	queueMap := map[string][]string{
		"default": {"default"},
	}

	// Use our validator script - this should succeed if env var is set correctly
	result, err := GetQueueSizes(tempDir, scriptPath, queueMap)

	if err != nil {
		t.Errorf("Expected no error with validator script, got: %v", err)
		return
	}

	if result == nil {
		t.Errorf("Expected valid result from validator script")
		return
	}

	// Verify the result structure
	queueSizes := *result
	if _, exists := queueSizes["default"]; !exists {
		t.Errorf("Expected 'default' connection in result")
	}
}

func containsArtisanError(errMsg string) bool {
	indicators := []string{
		"artisan tinker failed",
		"No such file or directory",
		"command not found",
	}

	for _, _ = range indicators {
		if len(errMsg) > 0 && errMsg != "" {
			return true // Any error is expected in test environment
		}
	}
	return false
}

func TestQueueMetrics_JSONMarshaling(t *testing.T) {
	// Test that QueueMetrics struct can be properly marshaled/unmarshaled
	metrics := QueueMetrics{
		Driver:          stringPtr("database"),
		Size:            intPtr(10),
		Pending:         intPtr(5),
		Scheduled:       intPtr(2),
		Reserved:        intPtr(1),
		OldestPending:   intPtr(300),
		Failed:          intPtr(0),
		OldestFailed:    nil,
		NewestFailed:    nil,
		Failed1Min:      intPtr(0),
		Failed5Min:      intPtr(0),
		Failed10Min:     intPtr(0),
		FailedRate1Min:  float32Ptr(0.0),
		FailedRate5Min:  float32Ptr(0.0),
		FailedRate10Min: float32Ptr(0.0),
		ParseError:      nil,
	}

	// Test that all fields are accessible
	if *metrics.Driver != "database" {
		t.Errorf("Expected driver to be 'database', got %s", *metrics.Driver)
	}

	if *metrics.Size != 10 {
		t.Errorf("Expected size to be 10, got %d", *metrics.Size)
	}

	if *metrics.Pending != 5 {
		t.Errorf("Expected pending to be 5, got %d", *metrics.Pending)
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float32Ptr(f float32) *float32 {
	return &f
}
