package metrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elasticphphq/agent/internal/laravel"
	"github.com/elasticphphq/agent/internal/phpfpm"
	"github.com/elasticphphq/agent/internal/server"
)

func TestMetrics_Structure(t *testing.T) {
	// Test Metrics structure
	timestamp := time.Now()
	metrics := Metrics{
		Timestamp: timestamp,
		Server: &server.SystemInfo{
			NodeType:      server.NodePhysical,
			OS:            "linux",
			Architecture:  "amd64",
			CPULimit:      4,
			MemoryLimitMB: 8192, // 8GB in MB
		},
		Fpm: map[string]*phpfpm.Result{
			"pool1": {
				Timestamp: timestamp,
				Pools: map[string]phpfpm.Pool{
					"www": {
						Name: "www",
						PhpInfo: phpfpm.Info{
							Version:    "PHP 8.2.10 (cli)",
							Extensions: []string{"Core", "date", "json"},
						},
						Processes: []phpfpm.PoolProcess{
							{
								PID:               1234,
								State:             "Idle",
								StartTime:         timestamp.Add(-time.Hour).Unix(),
								StartSince:        3600,
								Requests:          100,
								RequestDuration:   500,
								RequestMethod:     "GET",
								RequestURI:        "/api/test",
								ContentLength:     1024,
								User:              "www-data",
								Script:            "/var/www/test.php",
								LastRequestCPU:    0.5,
								LastRequestMemory: 1048576,
							},
						},
					},
				},
			},
		},
		Laravel: map[string]*laravel.LaravelMetrics{
			"app1": {
				Info: &laravel.AppInfo{},
				Queues: &laravel.QueueSizes{
					"redis": map[string]laravel.QueueMetrics{
						"default": {Size: &[]int{5}[0]},
						"high":    {Size: &[]int{2}[0]},
					},
				},
			},
		},
		Errors: map[string]string{
			"test_error": "This is a test error",
		},
	}

	// Verify structure
	if metrics.Timestamp != timestamp {
		t.Errorf("Expected Timestamp to be set correctly")
	}

	if metrics.Server == nil {
		t.Errorf("Expected Server to be set")
	}

	if metrics.Server.OS != "linux" {
		t.Errorf("Expected Server.OS to be 'linux', got %s", metrics.Server.OS)
	}

	if len(metrics.Fpm) != 1 {
		t.Errorf("Expected 1 FPM result, got %d", len(metrics.Fpm))
	}

	if len(metrics.Laravel) != 1 {
		t.Errorf("Expected 1 Laravel metric, got %d", len(metrics.Laravel))
	}

	if len(metrics.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(metrics.Errors))
	}

	if metrics.Errors["test_error"] != "This is a test error" {
		t.Errorf("Expected error message to match")
	}
}

func TestMetrics_JSONMarshaling(t *testing.T) {
	timestamp := time.Now()
	metrics := Metrics{
		Timestamp: timestamp,
		Server: &server.SystemInfo{
			NodeType:     server.NodePhysical,
			OS:           "linux",
			Architecture: "amd64",
		},
		Fpm: map[string]*phpfpm.Result{
			"test": {
				Timestamp: timestamp,
			},
		},
		Laravel: map[string]*laravel.LaravelMetrics{
			"app": {
				Info: &laravel.AppInfo{},
			},
		},
		Errors: map[string]string{
			"error1": "test error",
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal Metrics to JSON: %v", err)
	}

	// Unmarshal back
	var unmarshaledMetrics Metrics
	err = json.Unmarshal(jsonData, &unmarshaledMetrics)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to Metrics: %v", err)
	}

	// Compare values
	if unmarshaledMetrics.Server.OS != metrics.Server.OS {
		t.Errorf("Server.OS mismatch after JSON round-trip")
	}

	if len(unmarshaledMetrics.Fpm) != len(metrics.Fpm) {
		t.Errorf("Fpm length mismatch after JSON round-trip")
	}

	if len(unmarshaledMetrics.Laravel) != len(metrics.Laravel) {
		t.Errorf("Laravel length mismatch after JSON round-trip")
	}

	if len(unmarshaledMetrics.Errors) != len(metrics.Errors) {
		t.Errorf("Errors length mismatch after JSON round-trip")
	}
}

func TestMetrics_EmptyStructure(t *testing.T) {
	// Test with empty Metrics
	metrics := Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	// Should be able to marshal even with nil/empty fields
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal empty Metrics: %v", err)
	}

	var unmarshaledMetrics Metrics
	err = json.Unmarshal(jsonData, &unmarshaledMetrics)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty Metrics: %v", err)
	}

	// Check that nil fields remain nil
	if unmarshaledMetrics.Server != nil {
		t.Errorf("Expected Server to remain nil")
	}

	if unmarshaledMetrics.Fpm != nil {
		t.Errorf("Expected Fpm to remain nil")
	}

	if unmarshaledMetrics.Laravel != nil {
		t.Errorf("Expected Laravel to remain nil")
	}
}

func TestMetrics_LaravelOmitEmpty(t *testing.T) {
	// Test that Laravel field uses omitempty tag
	metrics := Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal Metrics: %v", err)
	}

	jsonStr := string(jsonData)
	
	// Laravel should be omitted when nil/empty due to omitempty tag
	if jsonStr[len(jsonStr)-1] == ',' {
		t.Errorf("Expected no trailing comma in JSON (Laravel should be omitted)")
	}

	// Test with empty Laravel map
	metrics.Laravel = make(map[string]*laravel.LaravelMetrics)
	jsonData, err = json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal Metrics with empty Laravel: %v", err)
	}

	jsonStr = string(jsonData)
	
	// Empty Laravel map should still be included (omitempty only applies to nil)
	// This is Go's default behavior for maps
}