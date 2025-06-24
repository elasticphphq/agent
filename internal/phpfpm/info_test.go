package phpfpm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
)

func TestInfo_Structure(t *testing.T) {
	// Test Info structure
	info := Info{
		Version:    "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)",
		Extensions: []string{"Core", "date", "filter", "hash", "json", "pcre", "Reflection", "SPL"},
		Opcache:    nil,
	}

	// Verify structure
	if info.Version != "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)" {
		t.Errorf("Expected Version to be set correctly")
	}

	if len(info.Extensions) != 8 {
		t.Errorf("Expected 8 extensions, got %d", len(info.Extensions))
	}

	if info.Extensions[0] != "Core" {
		t.Errorf("Expected first extension to be 'Core', got '%s'", info.Extensions[0])
	}

	if info.Opcache != nil {
		t.Errorf("Expected Opcache to be nil")
	}

	// Test with Opcache
	opcacheStatus := &OpcacheStatus{
		Enabled: true,
		MemoryUsage: Memory{
			UsedMemory:       1024000,
			FreeMemory:       512000,
			WastedMemory:     1000,
			CurrentWastedPct: 0.1,
		},
	}

	info.Opcache = opcacheStatus
	if info.Opcache == nil {
		t.Errorf("Expected Opcache to be set")
	}

	if !info.Opcache.Enabled {
		t.Errorf("Expected Opcache to be enabled")
	}
}

func TestGetPHPStats_MockBinary(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create mock PHP binary
	tempDir := t.TempDir()
	mockPhpPath := tempDir + "/mock-php"

	// Create mock PHP script that responds to both -v and -m
	mockScript := `#!/bin/bash
if [[ "$1" == "-v" ]]; then
    echo "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)"
    echo "Copyright (c) The PHP Group"
    echo "Zend Engine v4.2.10, Copyright (c) Zend Technologies"
elif [[ "$1" == "-m" ]]; then
    echo "[PHP Modules]"
    echo "Core"
    echo "date"
    echo "filter"
    echo "hash"
    echo "json"
    echo "pcre"
    echo "Reflection"
    echo "SPL"
    echo ""
    echo "[Zend Modules]"
    echo "Zend OPcache"
fi
`

	err := os.WriteFile(mockPhpPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock PHP binary: %v", err)
	}

	// Clear cache to ensure fresh call
	phpInfoMu.Lock()
	cachedPHPInfo = nil
	phpInfoErr = nil
	lastPHPInfoTime = time.Time{}
	phpInfoMu.Unlock()

	// Create test config
	cfg := config.FPMPoolConfig{
		Binary: mockPhpPath,
	}

	ctx := context.Background()
	info, err := GetPHPStats(ctx, cfg)
	if err != nil {
		t.Fatalf("GetPHPStats failed: %v", err)
	}

	// Verify version
	expectedVersion := "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)"
	if info.Version != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, info.Version)
	}

	// Verify extensions (may have an extra empty line)
	expectedExtensions := []string{"Core", "date", "filter", "hash", "json", "pcre", "Reflection", "SPL"}
	if len(info.Extensions) < len(expectedExtensions) {
		t.Errorf("Expected at least %d extensions, got %d", len(expectedExtensions), len(info.Extensions))
	}

	for i, expected := range expectedExtensions {
		if i >= len(info.Extensions) || info.Extensions[i] != expected {
			t.Errorf("Expected extension[%d] to be '%s', got '%s'", i, expected, info.Extensions[i])
		}
	}
}

func TestGetPHPStats_Caching(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create mock PHP binary
	tempDir := t.TempDir()
	mockPhpPath := tempDir + "/mock-php-cache"

	mockScript := `#!/bin/bash
if [[ "$1" == "-v" ]]; then
    echo "PHP 8.1.0 (cli) (built: Jan  1 2023 10:30:45)"
elif [[ "$1" == "-m" ]]; then
    echo "[PHP Modules]"
    echo "Core"
    echo "json"
fi
`

	err := os.WriteFile(mockPhpPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock PHP binary: %v", err)
	}

	// Clear cache
	phpInfoMu.Lock()
	cachedPHPInfo = nil
	phpInfoErr = nil
	lastPHPInfoTime = time.Time{}
	phpInfoMu.Unlock()

	cfg := config.FPMPoolConfig{
		Binary: mockPhpPath,
	}

	ctx := context.Background()

	// First call
	info1, err := GetPHPStats(ctx, cfg)
	if err != nil {
		t.Fatalf("First GetPHPStats failed: %v", err)
	}

	// Second call (should use cache)
	info2, err := GetPHPStats(ctx, cfg)
	if err != nil {
		t.Fatalf("Second GetPHPStats failed: %v", err)
	}

	// Should be the same instance
	if info1 != info2 {
		t.Errorf("Expected cached result to be the same instance")
	}

	// Verify cache is working by checking time
	phpInfoMu.Lock()
	cacheTime := lastPHPInfoTime
	phpInfoMu.Unlock()

	if cacheTime.IsZero() {
		t.Errorf("Expected cache time to be set")
	}

	// Test cache expiry by setting old time
	phpInfoMu.Lock()
	lastPHPInfoTime = time.Now().Add(-2 * time.Hour)
	phpInfoMu.Unlock()

	// Third call (should refresh cache)
	info3, err := GetPHPStats(ctx, cfg)
	if err != nil {
		t.Fatalf("Third GetPHPStats failed: %v", err)
	}

	// Should be a new instance
	if info1 == info3 {
		t.Errorf("Expected refreshed cache to be a different instance")
	}

	// But content should be the same
	if info1.Version != info3.Version {
		t.Errorf("Expected version to be the same after cache refresh")
	}
}

func TestGetPHPVersion(t *testing.T) {
	tests := []struct {
		name           string
		phpOutput      string
		expectedResult string
		expectError    bool
	}{
		{
			name: "standard PHP version",
			phpOutput: `PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)
Copyright (c) The PHP Group
Zend Engine v4.2.10, Copyright (c) Zend Technologies`,
			expectedResult: "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)",
			expectError:    false,
		},
		{
			name: "PHP 7.4 version",
			phpOutput: `PHP 7.4.33 (cli) (built: May 16 2023 10:30:45)
Copyright (c) The PHP Group
Zend Engine v3.4.0, Copyright (c) Zend Technologies`,
			expectedResult: "PHP 7.4.33 (cli) (built: May 16 2023 10:30:45)",
			expectError:    false,
		},
		{
			name:           "empty output",
			phpOutput:      "",
			expectedResult: "", // Empty output results in empty string, not "unknown"
			expectError:    false,
		},
		{
			name:           "single line",
			phpOutput:      "PHP 8.0.0 (cli)",
			expectedResult: "PHP 8.0.0 (cli)",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock binary
			tempDir := t.TempDir()
			mockPhpPath := tempDir + "/mock-php-version"

			mockScript := `#!/bin/bash
cat << 'EOF'
` + tt.phpOutput + `
EOF`

			err := os.WriteFile(mockPhpPath, []byte(mockScript), 0755)
			if err != nil {
				t.Fatalf("Failed to create mock PHP binary: %v", err)
			}

			result, err := getPHPVersion(mockPhpPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if result != tt.expectedResult {
					t.Errorf("Expected '%s', got '%s'", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetPHPExtensions(t *testing.T) {
	tests := []struct {
		name           string
		phpOutput      string
		expectedResult []string
		expectError    bool
	}{
		{
			name: "standard extensions output",
			phpOutput: `[PHP Modules]
Core
date
filter
hash
json
pcre
Reflection
SPL

[Zend Modules]
Zend OPcache`,
			expectedResult: []string{"Core", "date", "filter", "hash", "json", "pcre", "Reflection", "SPL", "Zend OPcache"},
			expectError:    false,
		},
		{
			name: "minimal extensions",
			phpOutput: `[PHP Modules]
Core
json

[Zend Modules]`,
			expectedResult: []string{"Core", "json"},
			expectError:    false,
		},
		{
			name:           "empty output",
			phpOutput:      "",
			expectedResult: []string{},
			expectError:    false,
		},
		{
			name: "only sections",
			phpOutput: `[PHP Modules]

[Zend Modules]`,
			expectedResult: []string{},
			expectError:    false,
		},
		{
			name: "mixed content",
			phpOutput: `[PHP Modules]
Core
filter
[Some Other Section]
Other content
hash
json`,
			expectedResult: []string{"Core", "filter", "Other content", "hash", "json"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock binary
			tempDir := t.TempDir()
			mockPhpPath := tempDir + "/mock-php-extensions"

			mockScript := `#!/bin/bash
cat << 'EOF'
` + tt.phpOutput + `
EOF`

			err := os.WriteFile(mockPhpPath, []byte(mockScript), 0755)
			if err != nil {
				t.Fatalf("Failed to create mock PHP binary: %v", err)
			}

			result, err := getPHPExtensions(mockPhpPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if len(result) != len(tt.expectedResult) {
						t.Errorf("Expected %d extensions, got %d", len(tt.expectedResult), len(result))
					} else {
						for i, expected := range tt.expectedResult {
							if result[i] != expected {
								t.Errorf("Expected extension[%d] to be '%s', got '%s'", i, expected, result[i])
							}
						}
					}
				}
			}
		})
	}
}

func TestGetPHPStats_ErrorHandling(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Clear cache
	phpInfoMu.Lock()
	cachedPHPInfo = nil
	phpInfoErr = nil
	lastPHPInfoTime = time.Time{}
	phpInfoMu.Unlock()

	// Test with non-existent binary
	cfg := config.FPMPoolConfig{
		Binary: "/non/existent/php",
	}

	ctx := context.Background()
	_, err := GetPHPStats(ctx, cfg)
	if err == nil {
		t.Errorf("Expected error for non-existent binary")
	}

	// Test that error is cached
	phpInfoMu.Lock()
	cachedErr := phpInfoErr
	phpInfoMu.Unlock()

	if cachedErr == nil {
		t.Errorf("Expected error to be cached")
	}

	// Second call should return cached error
	_, err2 := GetPHPStats(ctx, cfg)
	if err2 == nil {
		t.Errorf("Expected cached error on second call")
	}

	// Both errors should be non-nil (we can't guarantee they're the same instance due to error wrapping)
	if err2 == nil || cachedErr == nil {
		t.Errorf("Expected both errors to be non-nil")
	}
}

func TestGetPHPConfig_Structure(t *testing.T) {
	// Note: getPHPConfig requires a real FastCGI connection, so we primarily test
	// that the function exists and has the correct signature

	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	cfg := config.FPMPoolConfig{
		StatusSocket: "unix:///non/existent/socket",
		StatusPath:   "/status",
	}

	ctx := context.Background()
	_, err := getPHPConfig(ctx, cfg)

	// We expect this to fail since we don't have a real FPM socket
	if err == nil {
		t.Errorf("Expected error when connecting to non-existent socket")
	}

	// Check that error mentions FastCGI or socket
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "fastcgi") && !strings.Contains(errStr, "socket") && !strings.Contains(errStr, "dial") {
		t.Errorf("Expected error to mention FastCGI or socket connection issue, got: %s", err.Error())
	}
}

func TestPHPStats_ConcurrentAccess(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create mock PHP binary
	tempDir := t.TempDir()
	mockPhpPath := tempDir + "/mock-php-concurrent"

	mockScript := `#!/bin/bash
if [[ "$1" == "-v" ]]; then
    echo "PHP 8.0.0 (cli)"
elif [[ "$1" == "-m" ]]; then
    echo "[PHP Modules]"
    echo "Core"
    echo "json"
fi
sleep 0.1
`

	err := os.WriteFile(mockPhpPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock PHP binary: %v", err)
	}

	// Clear cache
	phpInfoMu.Lock()
	cachedPHPInfo = nil
	phpInfoErr = nil
	lastPHPInfoTime = time.Time{}
	phpInfoMu.Unlock()

	cfg := config.FPMPoolConfig{
		Binary: mockPhpPath,
	}

	ctx := context.Background()

	// Launch multiple goroutines to test concurrent access
	results := make(chan *Info, 5)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func() {
			info, err := GetPHPStats(ctx, cfg)
			if err != nil {
				errors <- err
				return
			}
			results <- info
		}()
	}

	// Collect results
	var infos []*Info
	for i := 0; i < 5; i++ {
		select {
		case info := <-results:
			infos = append(infos, info)
		case err := <-errors:
			t.Fatalf("Concurrent GetPHPStats failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for concurrent GetPHPStats")
		}
	}

	// All should be the same instance (cached)
	for i := 1; i < len(infos); i++ {
		if infos[i] != infos[0] {
			t.Errorf("Expected all concurrent calls to return the same cached instance")
		}
	}
}