package laravel

import (
	"encoding/json"
	"testing"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
)

func TestBoolString_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected BoolString
		wantErr  bool
	}{
		{
			name:     "boolean true",
			input:    `true`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "boolean false",
			input:    `false`,
			expected: BoolString(false),
			wantErr:  false,
		},
		{
			name:     "string enabled",
			input:    `"enabled"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "string true",
			input:    `"true"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "string on",
			input:    `"on"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "string yes",
			input:    `"yes"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "string cached",
			input:    `"cached"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:     "string disabled",
			input:    `"disabled"`,
			expected: BoolString(false),
			wantErr:  false,
		},
		{
			name:     "string off",
			input:    `"off"`,
			expected: BoolString(false),
			wantErr:  false,
		},
		{
			name:     "string case insensitive",
			input:    `"ENABLED"`,
			expected: BoolString(true),
			wantErr:  false,
		},
		{
			name:    "invalid input",
			input:   `123`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bs BoolString
			err := json.Unmarshal([]byte(tt.input), &bs)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if bs != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, bs)
			}
		})
	}
}

func TestStringOrSlice_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected StringOrSlice
		wantErr  bool
	}{
		{
			name:     "single string",
			input:    `"single"`,
			expected: StringOrSlice{"single"},
			wantErr:  false,
		},
		{
			name:     "array of strings",
			input:    `["first", "second", "third"]`,
			expected: StringOrSlice{"first", "second", "third"},
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: StringOrSlice{},
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: StringOrSlice{""},
			wantErr:  false,
		},
		{
			name:    "invalid input",
			input:   `123`,
			wantErr: true,
		},
		{
			name:    "mixed array",
			input:   `["string", 123]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sos StringOrSlice
			err := json.Unmarshal([]byte(tt.input), &sos)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(sos) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(sos))
				return
			}

			for i, v := range sos {
				if v != tt.expected[i] {
					t.Errorf("Expected %v at index %d, got %v", tt.expected[i], i, v)
				}
			}
		})
	}
}

func TestGetAppInfo(t *testing.T) {
	// Initialize logger to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})
	tests := []struct {
		name      string
		site      config.LaravelConfig
		phpBinary string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "app info disabled",
			site: config.LaravelConfig{
				Name:          "test",
				Path:          "/tmp/test",
				EnableAppInfo: false,
			},
			phpBinary: "php",
			wantErr:   false,
			wantNil:   true,
		},
		{
			name: "empty php binary",
			site: config.LaravelConfig{
				Name:          "test",
				Path:          "/tmp/test",
				EnableAppInfo: true,
			},
			phpBinary: "",
			wantErr:   true,
			wantNil:   false,
		},
		{
			name: "empty path",
			site: config.LaravelConfig{
				Name:          "test",
				Path:          "",
				EnableAppInfo: true,
			},
			phpBinary: "php",
			wantErr:   true,
			wantNil:   false,
		},
		{
			name: "valid config but no laravel app",
			site: config.LaravelConfig{
				Name:          "test",
				Path:          "/tmp/nonexistent",
				EnableAppInfo: true,
			},
			phpBinary: "php",
			wantErr:   true,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache before each test
			cacheMutex.Lock()
			appInfoCache = make(map[string]*AppInfo)
			cacheMutex.Unlock()

			result, err := GetAppInfo(tt.site, tt.phpBinary)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.wantNil && result != nil {
				t.Errorf("Expected nil result but got %+v", result)
			}

			if !tt.wantNil && !tt.wantErr && result == nil {
				t.Errorf("Expected non-nil result but got nil")
			}
		})
	}
}

func TestGetAppInfo_Caching(t *testing.T) {
	// Initialize logger to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})
	// Clear cache
	cacheMutex.Lock()
	appInfoCache = make(map[string]*AppInfo)
	cacheMutex.Unlock()

	site := config.LaravelConfig{
		Name:          "test",
		Path:          "/tmp/test-cache",
		EnableAppInfo: true,
	}

	// First call should attempt to run artisan (and fail)
	result1, err1 := GetAppInfo(site, "php")
	if err1 == nil {
		t.Errorf("Expected error on first call")
	}
	if result1 != nil {
		t.Errorf("Expected nil result on first call")
	}

	// Second call should use cache and return the same error
	result2, err2 := GetAppInfo(site, "php")
	if err2 == nil {
		t.Errorf("Expected error on second call")
	}
	if result2 != nil {
		t.Errorf("Expected nil result on second call")
	}

	// Error messages should indicate cached failure
	if err2.Error() != "app info was previously attempted but failed" {
		t.Errorf("Expected cached error message, got: %s", err2.Error())
	}
}

func TestAppInfo_JSONStructure(t *testing.T) {
	// Test that AppInfo struct can handle various JSON structures
	jsonInput := `{
		"environment": {
			"application_name": "Test App",
			"laravel_version": "10.0.0",
			"php_version": "8.2.0",
			"composer_version": "2.5.0",
			"environment": "production",
			"debug_mode": false,
			"url": "https://example.com",
			"maintenance_mode": "disabled",
			"timezone": "UTC",
			"locale": "en"
		},
		"cache": {
			"config": true,
			"events": "cached",
			"routes": "enabled",
			"views": false
		},
		"drivers": {
			"broadcasting": "redis",
			"cache": "redis",
			"database": "mysql",
			"logs": ["single", "daily"],
			"mail": "smtp",
			"queue": "redis",
			"session": "redis"
		},
		"livewire": {
			"version": "3.0.0"
		}
	}`

	var appInfo AppInfo
	err := json.Unmarshal([]byte(jsonInput), &appInfo)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
		return
	}

	// Test environment fields
	if appInfo.Environment.ApplicationName == nil || *appInfo.Environment.ApplicationName != "Test App" {
		t.Errorf("Expected ApplicationName to be 'Test App'")
	}

	if appInfo.Environment.LaravelVersion == nil || *appInfo.Environment.LaravelVersion != "10.0.0" {
		t.Errorf("Expected LaravelVersion to be '10.0.0'")
	}

	// Test BoolString fields
	if bool(appInfo.Cache.Config) != true {
		t.Errorf("Expected Cache.Config to be true")
	}

	if bool(appInfo.Cache.Events) != true {
		t.Errorf("Expected Cache.Events to be true (from 'cached')")
	}

	if bool(appInfo.Cache.Routes) != true {
		t.Errorf("Expected Cache.Routes to be true (from 'enabled')")
	}

	if bool(appInfo.Cache.Views) != false {
		t.Errorf("Expected Cache.Views to be false")
	}

	// Test StringOrSlice field
	if appInfo.Drivers.Logs == nil || len(*appInfo.Drivers.Logs) != 2 {
		t.Errorf("Expected Logs to have 2 elements")
	} else {
		logs := *appInfo.Drivers.Logs
		if logs[0] != "single" || logs[1] != "daily" {
			t.Errorf("Expected Logs to be ['single', 'daily'], got %v", logs)
		}
	}

	// Test optional fields
	if appInfo.Livewire == nil {
		t.Errorf("Expected Livewire to be present")
	} else {
		livewire := *appInfo.Livewire
		if livewire["version"] != "3.0.0" {
			t.Errorf("Expected Livewire version to be '3.0.0'")
		}
	}
}
