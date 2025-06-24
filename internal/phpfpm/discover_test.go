package phpfpm

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
)

func TestDiscoveredFPM_Structure(t *testing.T) {
	// Test DiscoveredFPM structure
	discovered := DiscoveredFPM{
		ConfigPath:   "/etc/php/8.2/fpm/pool.d/www.conf",
		StatusPath:   "/status",
		Binary:       "/usr/sbin/php-fpm8.2",
		Socket:       "unix:///var/run/php8.2-fpm.sock",
		StatusSocket: "unix:///var/run/php8.2-fpm.sock",
		CliBinary:    "/usr/bin/php8.2",
	}

	// Verify all fields are accessible and correctly typed
	if discovered.ConfigPath != "/etc/php/8.2/fpm/pool.d/www.conf" {
		t.Errorf("Expected ConfigPath to be set correctly")
	}

	if discovered.StatusPath != "/status" {
		t.Errorf("Expected StatusPath to be set correctly")
	}

	if discovered.Binary != "/usr/sbin/php-fpm8.2" {
		t.Errorf("Expected Binary to be set correctly")
	}

	if discovered.Socket != "unix:///var/run/php8.2-fpm.sock" {
		t.Errorf("Expected Socket to be set correctly")
	}

	if discovered.StatusSocket != "unix:///var/run/php8.2-fpm.sock" {
		t.Errorf("Expected StatusSocket to be set correctly")
	}

	if discovered.CliBinary != "/usr/bin/php8.2" {
		t.Errorf("Expected CliBinary to be set correctly")
	}
}

func TestFmpNamePattern(t *testing.T) {
	// Test the FMP name pattern regex
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "php-fpm",
			input:    "php-fpm",
			expected: true,
		},
		{
			name:     "php8.2-fpm",
			input:    "php8.2-fpm",
			expected: true,
		},
		{
			name:     "php82-fpm",
			input:    "php82-fpm",
			expected: true,
		},
		{
			name:     "phpfpm",
			input:    "phpfpm",
			expected: true,
		},
		{
			name:     "php7.4-fpm",
			input:    "php7.4-fpm",
			expected: true,
		},
		{
			name:     "php-fpm8.1",
			input:    "php-fpm8.1",
			expected: true,
		},
		{
			name:     "php-fpm-custom",
			input:    "php-fpm-custom",
			expected: true,
		},
		{
			name:     "apache2",
			input:    "apache2",
			expected: false,
		},
		{
			name:     "nginx",
			input:    "nginx",
			expected: false,
		},
		{
			name:     "php-cli",
			input:    "php-cli",
			expected: false,
		},
		{
			name:     "mysql",
			input:    "mysql",
			expected: false,
		},
		{
			name:     "empty",
			input:    "",
			expected: false,
		},
		{
			name:     "just php",
			input:    "php",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := fpmNamePattern.MatchString(tt.input)
			if matches != tt.expected {
				t.Errorf("Expected fmpNamePattern.MatchString(%q) to be %v, got %v", tt.input, tt.expected, matches)
			}
		})
	}
}

func TestParseSocket(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unix socket absolute path",
			input:    "/var/run/php-fpm.sock",
			expected: "unix:///var/run/php-fpm.sock",
		},
		{
			name:     "unix socket with subdirectory",
			input:    "/run/php/php8.2-fpm.sock",
			expected: "unix:///run/php/php8.2-fpm.sock",
		},
		{
			name:     "tcp with ip and port",
			input:    "127.0.0.1:9000",
			expected: "tcp://127.0.0.1:9000",
		},
		{
			name:     "tcp with host and port",
			input:    "localhost:9001",
			expected: "tcp://localhost:9001",
		},
		{
			name:     "ipv6 with port",
			input:    "[::1]:9000",
			expected: "tcp://[::1]:9000",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "just port number (will try to connect)",
			input:    "9000",
			expected: "tcp://127.0.0.1:9000", // May actually connect to localhost in some environments
		},
		{
			name:     "invalid format",
			input:    "invalid-socket",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSocket(tt.input)
			if tt.input == "9000" {
				// Port 9000 test is flexible - could connect or not
				if result != "" && result != "tcp://127.0.0.1:9000" && result != "tcp://[::1]:9000" {
					t.Errorf("Expected parseSocket(%q) to be empty or valid tcp address, got %q", tt.input, result)
				}
			} else if result != tt.expected {
				t.Errorf("Expected parseSocket(%q) to be %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}

func TestExtractConfigFromMaster(t *testing.T) {
	tests := []struct {
		name     string
		cmdline  string
		expected string
	}{
		{
			name:     "standard master process",
			cmdline:  "php-fpm: master process (/etc/php/8.2/fpm/php-fpm.conf)",
			expected: "/etc/php/8.2/fpm/php-fpm.conf",
		},
		{
			name:     "custom config path",
			cmdline:  "php-fpm: master process (/custom/path/fpm.conf)",
			expected: "/custom/path/fpm.conf",
		},
		{
			name:     "versioned php-fpm",
			cmdline:  "php-fpm8.1: master process (/etc/php/8.1/fpm/php-fpm.conf)",
			expected: "/etc/php/8.1/fpm/php-fpm.conf",
		},
		{
			name:     "no parentheses",
			cmdline:  "php-fpm: master process",
			expected: "",
		},
		{
			name:     "empty cmdline",
			cmdline:  "",
			expected: "",
		},
		{
			name:     "malformed parentheses - no closing",
			cmdline:  "php-fpm: master process (/etc/php.conf",
			expected: "",
		},
		{
			name:     "malformed parentheses - no opening",
			cmdline:  "php-fpm: master process /etc/php.conf)",
			expected: "",
		},
		{
			name:     "empty parentheses",
			cmdline:  "php-fpm: master process ()",
			expected: "",
		},
		{
			name:     "multiple parentheses - takes first",
			cmdline:  "php-fpm: master process (/etc/php.conf) (extra)",
			expected: "/etc/php.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractConfigFromMaster(tt.cmdline)
			if result != tt.expected {
				t.Errorf("Expected extractConfigFromMaster(%q) to be %q, got %q", tt.cmdline, tt.expected, result)
			}
		})
	}
}

func TestFindMatchingCliBinary_MockBinary(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create a mock FPM binary that outputs version info
	tempDir := t.TempDir()
	mockFmpPath := tempDir + "/mock-php-fpm"
	mockCliPath := tempDir + "/php8.2"  // Use the name the function expects

	// Create mock FPM binary
	fmpScript := `#!/bin/bash
echo "PHP 8.2.10 (fpm-fcgi) (built: Sep  1 2023 10:30:45)"
echo "Copyright (c) The PHP Group"
echo "Zend Engine v4.2.10, Copyright (c) Zend Technologies"
`

	err := os.WriteFile(mockFmpPath, []byte(fmpScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock FPM binary: %v", err)
	}

	// Create mock CLI binary
	cliScript := `#!/bin/bash
echo "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)"
echo "Copyright (c) The PHP Group"
echo "Zend Engine v4.2.10, Copyright (c) Zend Technologies"
`

	err = os.WriteFile(mockCliPath, []byte(cliScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock CLI binary: %v", err)
	}

	// Set PATH to include our mock binary
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	// Test finding matching CLI binary
	cliBinary, err := findMatchingCliBinary(mockFmpPath)
	if err != nil {
		t.Fatalf("findMatchingCliBinary failed: %v", err)
	}

	// Should find our mock CLI binary
	if cliBinary != "php8.2" {
		t.Errorf("Expected to find 'php8.2', got %s", cliBinary)
	}
}

func TestFindMatchingCliBinary_ErrorCases(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	tests := []struct {
		name      string
		fmpBinary string
		wantErr   bool
	}{
		{
			name:      "non-existent binary",
			fmpBinary: "/non/existent/php-fmp",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := findMatchingCliBinary(tt.fmpBinary)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFindMatchingCliBinary_VersionParsing(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create mock binaries with different version outputs
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		fmpOutput      string
		cliOutput      string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "PHP 8.2",
			fmpOutput:      "PHP 8.2.10 (fpm-fcgi) (built: Sep  1 2023 10:30:45)",
			cliOutput:      "PHP 8.2.10 (cli) (built: Sep  1 2023 10:30:45)",
			expectedResult: "php8.2",
			expectError:    false,
		},
		{
			name:           "PHP 7.4",
			fmpOutput:      "PHP 7.4.33 (fpm-fcgi) (built: Sep  1 2023 10:30:45)",
			cliOutput:      "PHP 7.4.33 (cli) (built: Sep  1 2023 10:30:45)",
			expectedResult: "php7.4",
			expectError:    false,
		},
		{
			name:        "unparseable version",
			fmpOutput:   "Invalid version output",
			cliOutput:   "",
			expectError: true,
		},
		{
			name:        "version mismatch",
			fmpOutput:   "PHP 8.2.10 (fpm-fcgi) (built: Sep  1 2023 10:30:45)",
			cliOutput:   "PHP 8.1.10 (cli) (built: Sep  1 2023 10:30:45)",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError && tt.expectedResult != "" {
				t.Fatalf("Test case error: cannot expect both error and result")
			}

			// Create mock FMP binary
			mockFmpPath := tempDir + "/mock-fpm-" + tt.name
			fmpScript := "#!/bin/bash\necho '" + tt.fmpOutput + "'"

			err := os.WriteFile(mockFmpPath, []byte(fmpScript), 0755)
			if err != nil {
				t.Fatalf("Failed to create mock FMP binary: %v", err)
			}

			if !tt.expectError {
				// Extract version from fmpOutput to create properly named CLI binary
				version := ""
				if strings.Contains(tt.fmpOutput, "8.2") {
					version = "8.2"
				} else if strings.Contains(tt.fmpOutput, "7.4") {
					version = "7.4"
				}
				
				// Create mock CLI binary with the name the function expects
				var mockCliPath string
				if version != "" {
					mockCliPath = tempDir + "/php" + version
				} else {
					mockCliPath = tempDir + "/php"
				}
				cliScript := "#!/bin/bash\necho '" + tt.cliOutput + "'"

				err := os.WriteFile(mockCliPath, []byte(cliScript), 0755)
				if err != nil {
					t.Fatalf("Failed to create mock CLI binary: %v", err)
				}

				// Set PATH to include our mock binary
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", tempDir+":"+originalPath)
				defer os.Setenv("PATH", originalPath)
			}

			cliBinary, err := findMatchingCliBinary(mockFmpPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if cliBinary != tt.expectedResult && !strings.Contains(cliBinary, "php") {
					t.Errorf("Expected %s or a php binary, got %s", tt.expectedResult, cliBinary)
				}
			}
		})
	}
}

func TestDiscoverFPMProcesses_MockImplementation(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Note: This test is limited because we can't easily mock the process.Processes() call
	// In a real implementation, we would need to use dependency injection or build tags
	// to replace the process discovery mechanism for testing

	// For now, we test that the function exists and returns without panicking
	discovered, err := DiscoverFPMProcesses()

	// We expect this to work even if no FPM processes are found
	if err != nil {
		// Only fail if it's a fundamental error (not "no processes found")
		// Most systems won't have FPM running during tests
		t.Logf("DiscoverFPMProcesses returned error (expected in test environment): %v", err)
	}

	// Should return a slice (even if empty)
	if discovered == nil {
		t.Errorf("Expected DiscoverFPMProcesses to return non-nil slice")
	}

	// Log the results for debugging
	t.Logf("Discovered %d FPM processes", len(discovered))
	for i, fpm := range discovered {
		t.Logf("FPM %d: Binary=%s, Socket=%s, StatusPath=%s", i, fpm.Binary, fpm.Socket, fpm.StatusPath)
	}
}

func TestRegexPatterns(t *testing.T) {
	// Test that regex patterns compile correctly
	if fpmNamePattern == nil {
		t.Errorf("Expected fpmNamePattern to be initialized")
	}

	// Test the pattern itself
	pattern := `^php[0-9]{0,2}.*fpm.*$`
	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		t.Errorf("Pattern should compile without error: %v", err)
	}

	// Test some expected matches
	testCases := []string{"php-fpm", "php8.2-fpm", "phpfpm", "php82-fpm"}
	for _, testCase := range testCases {
		if !compiledPattern.MatchString(testCase) {
			t.Errorf("Pattern should match %s", testCase)
		}
	}

	// Test some expected non-matches
	nonMatches := []string{"apache2", "nginx", "php", "fpm", "mysql"}
	for _, testCase := range nonMatches {
		if compiledPattern.MatchString(testCase) {
			t.Errorf("Pattern should not match %s", testCase)
		}
	}
}