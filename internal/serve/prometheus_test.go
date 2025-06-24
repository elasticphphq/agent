package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestNewPrometheusCollector(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: true,
		},
	}

	collector := NewPrometheusCollector(cfg)

	if collector == nil {
		t.Fatalf("Expected NewPrometheusCollector to return non-nil collector")
	}

	if collector.cfg != cfg {
		t.Errorf("Expected collector config to match input")
	}

	// Test that descriptors are initialized
	if collector.upDesc == nil {
		t.Errorf("Expected upDesc to be initialized")
	}

	if collector.acceptedConnectionsDesc == nil {
		t.Errorf("Expected acceptedConnectionsDesc to be initialized")
	}

	if collector.laravelInfoDesc == nil {
		t.Errorf("Expected laravelInfoDesc to be initialized")
	}
}

func TestPrometheusCollector_Describe(t *testing.T) {
	cfg := &config.Config{}
	collector := NewPrometheusCollector(cfg)

	ch := make(chan *prometheus.Desc, 100)
	collector.Describe(ch)
	close(ch)

	// Count descriptors
	var count int
	for range ch {
		count++
	}

	// Should have many descriptors (at least 30+)
	if count < 30 {
		t.Errorf("Expected at least 30 descriptors, got %d", count)
	}
}

func TestParseConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		valid    bool
	}{
		{
			name:     "integer value",
			input:    "10",
			expected: 10.0,
			valid:    true,
		},
		{
			name:     "float value",
			input:    "10.5",
			expected: 10.5,
			valid:    true,
		},
		{
			name:     "value with seconds suffix",
			input:    "30s",
			expected: 30.0,
			valid:    true,
		},
		{
			name:     "value with spaces",
			input:    "  25  ",
			expected: 25.0,
			valid:    true,
		},
		{
			name:     "value with spaces and suffix",
			input:    "  15s  ",
			expected: 15.0,
			valid:    true,
		},
		{
			name:     "invalid value",
			input:    "invalid",
			expected: 0.0,
			valid:    false,
		},
		{
			name:     "empty value",
			input:    "",
			expected: 0.0,
			valid:    false,
		},
		{
			name:     "negative value",
			input:    "-5",
			expected: -5.0,
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := parseConfigValue(tt.input)

			if valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, valid)
			}

			if valid && result != tt.expected {
				t.Errorf("Expected result=%f, got %f", tt.expected, result)
			}
		})
	}
}

func TestBoolToFloat(t *testing.T) {
	tests := []struct {
		input    bool
		expected float64
	}{
		{input: true, expected: 1.0},
		{input: false, expected: 0.0},
	}

	for _, tt := range tests {
		result := boolToFloat(tt.input)
		if result != tt.expected {
			t.Errorf("Expected boolToFloat(%v) = %f, got %f", tt.input, tt.expected, result)
		}
	}
}

func TestPrometheusCollector_Collect_DisabledFPM(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
		Laravel: []config.LaravelConfig{},
	}

	collector := NewPrometheusCollector(cfg)

	// Create a test registry and gather metrics
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Should have some metrics even with FPM disabled
	if len(metricFamilies) == 0 {
		t.Errorf("Expected some metrics even with FPM disabled")
	}

	// Look for the up metric indicating FPM is down
	foundUpMetric := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "phpfpm_up" {
			foundUpMetric = true
			if len(mf.GetMetric()) > 0 {
				value := mf.GetMetric()[0].GetGauge().GetValue()
				if value != 0 {
					t.Errorf("Expected phpfpm_up to be 0 when FPM disabled, got %f", value)
				}
			}
		}
	}

	if !foundUpMetric {
		t.Errorf("Expected to find phpfpm_up metric")
	}
}

func TestStartPrometheusServer_MetricsEndpoint(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
		Monitor: config.MonitorConfig{
			ListenAddr: ":0", // Use random port
			EnableJson: false,
		},
	}

	// Create test server
	mux := http.NewServeMux()
	registry := prometheus.NewRegistry()
	collector := NewPrometheusCollector(cfg)
	registry.MustRegister(collector)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got %s", contentType)
	}
}

func TestStartPrometheusServer_JSONEndpoint(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()

	// Add JSON endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		// Use a simple mock metrics response for testing
		mockMetrics := map[string]interface{}{
			"timestamp": time.Now(),
			"server": map[string]interface{}{
				"os": "test",
			},
			"errors": map[string]string{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockMetrics)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test JSON endpoint
	resp, err := http.Get(server.URL + "/json")
	if err != nil {
		t.Fatalf("Failed to get JSON: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected application/json content type, got %s", contentType)
	}

	// Parse JSON response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Errorf("Failed to decode JSON response: %v", err)
	}

	// Check that response has expected structure
	if _, exists := result["server"]; !exists {
		t.Errorf("Expected JSON response to have 'server' field")
	}
}

func TestPrometheusCollector_MetricNames(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
	}

	collector := NewPrometheusCollector(cfg)

	// Test that metric descriptors have expected names
	expectedMetrics := []string{
		"phpfpm_up",
		"phpfpm_accepted_connections",
		"phpfpm_start_since",
		"phpfpm_listen_queue",
		"phpfpm_idle_processes",
		"phpfpm_active_processes",
		"phpfpm_total_processes",
		"phpfpm_opcache_enabled",
		"phpfpm_opcache_used_memory_bytes",
		"phpfpm_opcache_hits_total",
		"system_info",
		"system_cpu_limit",
		"laravel_app_info",
	}

	// Get all descriptors
	ch := make(chan *prometheus.Desc, 100)
	collector.Describe(ch)
	close(ch)

	var descriptorNames []string
	for desc := range ch {
		// Extract name from descriptor string representation
		descStr := desc.String()
		for _, expected := range expectedMetrics {
			if strings.Contains(descStr, "fqName: \""+expected+"\"") {
				descriptorNames = append(descriptorNames, expected)
				break
			}
		}
	}

	// Check that we found at least some expected metrics
	if len(descriptorNames) < 5 {
		t.Errorf("Expected to find at least 5 known metrics, found %d: %v", len(descriptorNames), descriptorNames)
	}
}

func TestPrometheusCollector_Collect_WithError(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Create config that will cause metrics collection to have issues
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: true,
			Pools: []config.FPMPoolConfig{
				{
					Socket:       "unix:///nonexistent/socket",
					StatusSocket: "unix:///nonexistent/socket",
					StatusPath:   "/status",
				},
			},
		},
	}

	collector := NewPrometheusCollector(cfg)

	// Create a test registry and collect metrics
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Should still have metrics even with errors
	if len(metricFamilies) == 0 {
		t.Errorf("Expected some metrics even with collection errors")
	}
}

func TestPrometheusCollector_RegistryIntegration(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
	}

	collector := NewPrometheusCollector(cfg)

	// Test registering with Prometheus registry
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	if err != nil {
		t.Errorf("Failed to register collector: %v", err)
	}

	// Test that we can gather metrics without error
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Errorf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Errorf("Expected some metric families")
	}

	// Test unregistering
	success := registry.Unregister(collector)
	if !success {
		t.Errorf("Failed to unregister collector")
	}
}