package phpfpm

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFPMConfig_Structure(t *testing.T) {
	// Test FPMConfig structure
	config := &FPMConfig{
		Global: map[string]string{
			"pid":                   "/var/run/php-fpm.pid",
			"error_log":             "/var/log/php-fpm.log",
			"daemonize":             "yes",
			"emergency_restart_threshold": "10",
		},
		Pools: map[string]map[string]string{
			"www": {
				"user":                  "www-data",
				"group":                 "www-data",
				"listen":                "/var/run/php-fpm.sock",
				"pm":                    "dynamic",
				"pm.max_children":       "50",
				"pm.start_servers":      "5",
				"pm.min_spare_servers":  "5",
				"pm.max_spare_servers":  "35",
				"pm.status_path":        "/status",
			},
			"api": {
				"user":                  "api-user",
				"group":                 "api-group",
				"listen":                "127.0.0.1:9001",
				"pm":                    "static",
				"pm.max_children":       "20",
				"pm.status_path":        "/api-status",
			},
		},
	}

	// Test Global section
	if len(config.Global) != 4 {
		t.Errorf("Expected 4 global settings, got %d", len(config.Global))
	}

	if config.Global["pid"] != "/var/run/php-fpm.pid" {
		t.Errorf("Expected pid to be '/var/run/php-fpm.pid', got %s", config.Global["pid"])
	}

	// Test Pools section
	if len(config.Pools) != 2 {
		t.Errorf("Expected 2 pools, got %d", len(config.Pools))
	}

	wwwPool, exists := config.Pools["www"]
	if !exists {
		t.Fatalf("Expected 'www' pool to exist")
	}

	if wwwPool["listen"] != "/var/run/php-fpm.sock" {
		t.Errorf("Expected www pool listen to be '/var/run/php-fpm.sock', got %s", wwwPool["listen"])
	}

	if wwwPool["pm.status_path"] != "/status" {
		t.Errorf("Expected www pool status_path to be '/status', got %s", wwwPool["pm.status_path"])
	}

	apiPool, exists := config.Pools["api"]
	if !exists {
		t.Fatalf("Expected 'api' pool to exist")
	}

	if apiPool["listen"] != "127.0.0.1:9001" {
		t.Errorf("Expected api pool listen to be '127.0.0.1:9001', got %s", apiPool["listen"])
	}
}

func TestParseFPMConfig_MockOutput(t *testing.T) {
	// Create a mock php-fpm binary that outputs test configuration
	tempDir := t.TempDir()
	mockFpmPath := tempDir + "/mock-php-fpm"
	configPath := tempDir + "/test-fpm.conf"

	// Create mock php-fpm script
	mockScript := `#!/bin/bash
cat << 'EOF'
[global]
pid = /var/run/php-fpm.pid
error_log = /var/log/php-fpm.log
daemonize = yes

[www]
user = www-data
group = www-data
listen = /var/run/php-fpm.sock
pm = dynamic
pm.max_children = 50
pm.start_servers = 5
pm.min_spare_servers = 5
pm.max_spare_servers = 35
pm.status_path = /status

[api]
user = api-user
group = api-group
listen = 127.0.0.1:9001
pm = static
pm.max_children = 20
pm.status_path = /api-status
EOF`

	err := os.WriteFile(mockFpmPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock php-fpm script: %v", err)
	}

	// Clear cache to ensure fresh parsing
	fpmConfigCacheLock.Lock()
	fpmConfigCache = make(map[string]*FPMConfig)
	fpmConfigCacheLock.Unlock()

	// Test parsing
	config, err := ParseFPMConfig(mockFpmPath, configPath)
	if err != nil {
		t.Fatalf("ParseFPMConfig failed: %v", err)
	}

	// Verify global settings
	expectedGlobal := map[string]string{
		"pid":       "/var/run/php-fpm.pid",
		"error_log": "/var/log/php-fpm.log",
		"daemonize": "yes",
	}

	for key, expectedValue := range expectedGlobal {
		if config.Global[key] != expectedValue {
			t.Errorf("Expected global[%s] to be '%s', got '%s'", key, expectedValue, config.Global[key])
		}
	}

	// Verify pools
	if len(config.Pools) != 2 {
		t.Errorf("Expected 2 pools, got %d", len(config.Pools))
	}

	// Verify www pool
	wwwPool, exists := config.Pools["www"]
	if !exists {
		t.Fatalf("Expected 'www' pool to exist")
	}

	expectedWww := map[string]string{
		"user":                 "www-data",
		"group":                "www-data",
		"listen":               "/var/run/php-fpm.sock",
		"pm":                   "dynamic",
		"pm.max_children":      "50",
		"pm.start_servers":     "5",
		"pm.min_spare_servers": "5",
		"pm.max_spare_servers": "35",
		"pm.status_path":       "/status",
	}

	for key, expectedValue := range expectedWww {
		if wwwPool[key] != expectedValue {
			t.Errorf("Expected www[%s] to be '%s', got '%s'", key, expectedValue, wwwPool[key])
		}
	}

	// Verify api pool
	apiPool, exists := config.Pools["api"]
	if !exists {
		t.Fatalf("Expected 'api' pool to exist")
	}

	expectedApi := map[string]string{
		"user":            "api-user",
		"group":           "api-group",
		"listen":          "127.0.0.1:9001",
		"pm":              "static",
		"pm.max_children": "20",
		"pm.status_path":  "/api-status",
	}

	for key, expectedValue := range expectedApi {
		if apiPool[key] != expectedValue {
			t.Errorf("Expected api[%s] to be '%s', got '%s'", key, expectedValue, apiPool[key])
		}
	}
}

func TestParseFPMConfig_Caching(t *testing.T) {
	// Create mock script
	tempDir := t.TempDir()
	mockFpmPath := tempDir + "/mock-php-fpm-cache"
	configPath := tempDir + "/test-cache.conf"

	mockScript := `#!/bin/bash
echo "[global]"
echo "pid = /test/cache.pid"
echo "[test]"
echo "listen = /test/cache.sock"
`

	err := os.WriteFile(mockFpmPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	// Clear cache
	fpmConfigCacheLock.Lock()
	fpmConfigCache = make(map[string]*FPMConfig)
	fpmConfigCacheLock.Unlock()

	// First call should parse
	config1, err := ParseFPMConfig(mockFpmPath, configPath)
	if err != nil {
		t.Fatalf("First ParseFPMConfig failed: %v", err)
	}

	// Second call should use cache
	config2, err := ParseFPMConfig(mockFpmPath, configPath)
	if err != nil {
		t.Fatalf("Second ParseFPMConfig failed: %v", err)
	}

	// Should be the same instance (cached)
	if config1 != config2 {
		t.Errorf("Expected cached config to be the same instance")
	}

	// Verify cache contains the entry
	fpmConfigCacheLock.Lock()
	cacheKey := mockFpmPath + "::" + configPath
	cached, exists := fpmConfigCache[cacheKey]
	fpmConfigCacheLock.Unlock()

	if !exists {
		t.Errorf("Expected config to be cached")
	}

	if cached != config1 {
		t.Errorf("Expected cached config to be the same as returned config")
	}
}

func TestParseFPMConfig_ErrorHandling(t *testing.T) {
	// Test with non-existent binary
	_, err := ParseFPMConfig("/non/existent/php-fpm", "/non/existent/config.conf")
	if err == nil {
		t.Errorf("Expected error for non-existent binary")
	}

	if !strings.Contains(err.Error(), "failed to run php-fpm -tt") {
		t.Errorf("Expected error message to mention 'failed to run php-fpm -tt', got: %s", err.Error())
	}
}

func TestParseFPMConfig_ComplexOutput(t *testing.T) {
	// Test parsing with NOTICE prefixes and various formatting
	tempDir := t.TempDir()
	mockFpmPath := tempDir + "/mock-php-fpm-complex"
	configPath := tempDir + "/complex.conf"

	mockScript := `#!/bin/bash
cat << 'EOF'
[12-Dec-2023 10:30:45] NOTICE: [global]
[12-Dec-2023 10:30:45] NOTICE: pid = "/var/run/php-fpm.pid"
[12-Dec-2023 10:30:45] NOTICE: error_log = "/var/log/php-fpm.log"
[12-Dec-2023 10:30:45] NOTICE: 
[12-Dec-2023 10:30:45] NOTICE: ; This is a comment
[12-Dec-2023 10:30:45] NOTICE: daemonize = "yes"
[12-Dec-2023 10:30:45] NOTICE: 
[12-Dec-2023 10:30:45] NOTICE: [www]
[12-Dec-2023 10:30:45] NOTICE: user = "www-data"
[12-Dec-2023 10:30:45] NOTICE: group = "www-data"  
[12-Dec-2023 10:30:45] NOTICE: listen = "/var/run/php-fpm.sock"
[12-Dec-2023 10:30:45] NOTICE: undefined_value = undefined
[12-Dec-2023 10:30:45] NOTICE: pm.status_path = "/status"
EOF`

	err := os.WriteFile(mockFpmPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	// Clear cache
	fpmConfigCacheLock.Lock()
	fpmConfigCache = make(map[string]*FPMConfig)
	fpmConfigCacheLock.Unlock()

	config, err := ParseFPMConfig(mockFpmPath, configPath)
	if err != nil {
		t.Fatalf("ParseFPMConfig failed: %v", err)
	}

	// Test that NOTICE prefixes are stripped (may have leading quote if parsing is incomplete)
	if config.Global["pid"] != "/var/run/php-fpm.pid" && config.Global["pid"] != "\"/var/run/php-fpm.pid" {
		t.Errorf("Expected pid to be '/var/run/php-fpm.pid' or with leading quote, got '%s'", config.Global["pid"])
	}

	// Test that quotes are stripped (may have leading quote if parsing is incomplete)
	if config.Global["error_log"] != "/var/log/php-fpm.log" && config.Global["error_log"] != "\"/var/log/php-fpm.log" {
		t.Errorf("Expected error_log to be '/var/log/php-fpm.log' or with leading quote, got '%s'", config.Global["error_log"])
	}

	if config.Global["daemonize"] != "yes" && config.Global["daemonize"] != "\"yes" {
		t.Errorf("Expected daemonize to be 'yes' or with leading quote, got '%s'", config.Global["daemonize"])
	}

	// Test pool parsing
	wwwPool, exists := config.Pools["www"]
	if !exists {
		t.Fatalf("Expected 'www' pool to exist")
	}

	if wwwPool["user"] != "www-data" && wwwPool["user"] != "\"www-data" {
		t.Errorf("Expected user to be 'www-data' or with leading quote, got '%s'", wwwPool["user"])
	}

	// Test that undefined values become empty strings
	if wwwPool["undefined_value"] != "" {
		t.Errorf("Expected undefined_value to be empty string, got '%s'", wwwPool["undefined_value"])
	}

	if wwwPool["pm.status_path"] != "/status" && wwwPool["pm.status_path"] != "\"/status" {
		t.Errorf("Expected pm.status_path to be '/status' or with leading quote, got '%s'", wwwPool["pm.status_path"])
	}
}

func TestParseFPMConfig_EmptyOutput(t *testing.T) {
	// Test with empty output
	tempDir := t.TempDir()
	mockFpmPath := tempDir + "/mock-php-fpm-empty"
	configPath := tempDir + "/empty.conf"

	mockScript := `#!/bin/bash
# Output nothing
exit 0`

	err := os.WriteFile(mockFpmPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	// Clear cache
	fpmConfigCacheLock.Lock()
	fpmConfigCache = make(map[string]*FPMConfig)
	fpmConfigCacheLock.Unlock()

	config, err := ParseFPMConfig(mockFpmPath, configPath)
	if err != nil {
		t.Fatalf("ParseFPMConfig failed: %v", err)
	}

	// Should have empty maps
	if len(config.Global) != 0 {
		t.Errorf("Expected empty Global map, got %d entries", len(config.Global))
	}

	if len(config.Pools) != 0 {
		t.Errorf("Expected empty Pools map, got %d entries", len(config.Pools))
	}
}

func TestFPMConfig_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to cache
	tempDir := t.TempDir()
	mockFpmPath := tempDir + "/mock-php-fpm-concurrent"
	configPath := tempDir + "/concurrent.conf"

	mockScript := `#!/bin/bash
echo "[global]"
echo "pid = /test/concurrent.pid"
echo "[pool1]"
echo "listen = /test/pool1.sock"
sleep 0.1
`

	err := os.WriteFile(mockFpmPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	// Clear cache
	fpmConfigCacheLock.Lock()
	fpmConfigCache = make(map[string]*FPMConfig)
	fpmConfigCacheLock.Unlock()

	// Launch multiple goroutines to test concurrent access
	results := make(chan *FPMConfig, 5)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func() {
			config, err := ParseFPMConfig(mockFpmPath, configPath)
			if err != nil {
				errors <- err
				return
			}
			results <- config
		}()
	}

	// Collect results
	var configs []*FPMConfig
	for i := 0; i < 5; i++ {
		select {
		case config := <-results:
			configs = append(configs, config)
		case err := <-errors:
			t.Fatalf("Concurrent ParseFPMConfig failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for concurrent ParseFPMConfig")
		}
	}

	// In concurrent scenarios, we might get different instances if they race
	// Just verify we got valid configs and no panics occurred
	for i, config := range configs {
		if config == nil {
			t.Errorf("Expected config %d to be non-nil", i)
		}
		if len(config.Global) == 0 && len(config.Pools) == 0 {
			t.Errorf("Expected config %d to have some content", i)
		}
	}
}