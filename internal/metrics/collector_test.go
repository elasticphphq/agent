package metrics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elasticphphq/agent/internal/config"
)

func TestNewCollector(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: true,
		},
	}
	interval := time.Second

	collector := NewCollector(cfg, interval)

	if collector == nil {
		t.Fatalf("Expected NewCollector to return non-nil collector")
	}

	if collector.cfg != cfg {
		t.Errorf("Expected collector config to match input")
	}

	if collector.interval != interval {
		t.Errorf("Expected collector interval to match input")
	}

	if collector.listeners == nil {
		t.Errorf("Expected listeners to be initialized")
	}

	if len(collector.listeners) != 0 {
		t.Errorf("Expected listeners slice to be empty initially")
	}

	if collector.results == nil {
		t.Errorf("Expected results map to be initialized")
	}

	if len(collector.results) != 0 {
		t.Errorf("Expected results map to be empty initially")
	}
}

func TestCollector_AddListener(t *testing.T) {
	cfg := &config.Config{}
	collector := NewCollector(cfg, time.Second)

	// Test adding listeners
	listenerCalled := false
	listener := func(m *Metrics) {
		listenerCalled = true
	}

	collector.AddListener(listener)

	if len(collector.listeners) != 1 {
		t.Errorf("Expected 1 listener, got %d", len(collector.listeners))
	}

	// Add another listener
	listener2Called := false
	listener2 := func(m *Metrics) {
		listener2Called = true
	}

	collector.AddListener(listener2)

	if len(collector.listeners) != 2 {
		t.Errorf("Expected 2 listeners, got %d", len(collector.listeners))
	}

	// Test notify calls all listeners
	metrics := &Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	collector.notify(metrics)

	if !listenerCalled {
		t.Errorf("Expected first listener to be called")
	}

	if !listener2Called {
		t.Errorf("Expected second listener to be called")
	}
}

func TestCollector_Notify(t *testing.T) {
	cfg := &config.Config{}
	collector := NewCollector(cfg, time.Second)

	var receivedMetrics *Metrics
	listener := func(m *Metrics) {
		receivedMetrics = m
	}

	collector.AddListener(listener)

	testMetrics := &Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	collector.notify(testMetrics)

	if receivedMetrics != testMetrics {
		t.Errorf("Expected listener to receive the same metrics instance")
	}
}

func TestCollector_ConcurrentListeners(t *testing.T) {
	cfg := &config.Config{}
	collector := NewCollector(cfg, time.Second)

	// Add listeners concurrently
	var wg sync.WaitGroup
	numListeners := 10

	for i := 0; i < numListeners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.AddListener(func(m *Metrics) {})
		}()
	}

	wg.Wait()

	if len(collector.listeners) != numListeners {
		t.Errorf("Expected %d listeners, got %d", numListeners, len(collector.listeners))
	}
}

func TestCollector_Collect(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false, // Disable to avoid requiring real FPM
		},
		Laravel: []config.LaravelConfig{}, // Empty to avoid requiring real Laravel
	}
	collector := NewCollector(cfg, time.Second)

	ctx := context.Background()
	metrics, err := collector.Collect(ctx)

	if err != nil {
		t.Errorf("Unexpected error from Collect: %v", err)
	}

	if metrics == nil {
		t.Errorf("Expected Collect to return non-nil metrics")
	}

	if metrics.Errors == nil {
		t.Errorf("Expected metrics to have initialized Errors map")
	}
}

func TestCollector_RunCancellation(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
	}
	collector := NewCollector(cfg, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	// Start collector in goroutine
	done := make(chan bool)
	go func() {
		collector.Run(ctx)
		done <- true
	}()

	// Cancel after short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for Run to finish
	select {
	case <-done:
		// Success - Run returned
	case <-time.After(time.Second):
		t.Errorf("Expected Run to return after context cancellation")
	}
}

func TestCollector_RunPerPoolCollector(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled:      true,
			PollInterval: 100 * time.Millisecond,
			Pools: []config.FPMPoolConfig{
				{
					Socket:       "unix:///tmp/test1.sock",
					StatusSocket: "unix:///tmp/test1.sock",
					StatusPath:   "/status",
					PollInterval: 50 * time.Millisecond,
					Timeout:      time.Second,
				},
				{
					Socket:       "unix:///tmp/test2.sock",
					StatusSocket: "unix:///tmp/test2.sock",
					StatusPath:   "/status",
					// No PollInterval set - should use global
					Timeout: time.Second,
				},
			},
		},
	}
	collector := NewCollector(cfg, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the per-pool collector
	collector.RunPerPoolCollector(ctx)

	// Give it a moment to start
	time.Sleep(200 * time.Millisecond)

	// Cancel and give it time to stop
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Check that results were attempted (they'll be error results since sockets don't exist)
	collector.mu.Lock()
	defer collector.mu.Unlock()

	if len(collector.results) != 2 {
		t.Errorf("Expected 2 results (one per pool), got %d", len(collector.results))
	}

	// Check that both sockets have results
	if _, exists := collector.results["unix:///tmp/test1.sock"]; !exists {
		t.Errorf("Expected result for test1.sock")
	}

	if _, exists := collector.results["unix:///tmp/test2.sock"]; !exists {
		t.Errorf("Expected result for test2.sock")
	}
}

func TestGetMetrics(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false, // Disable to avoid requiring real FPM
		},
		Laravel: []config.LaravelConfig{}, // Empty Laravel configs
	}

	ctx := context.Background()
	metrics, err := GetMetrics(ctx, cfg)

	if err != nil {
		t.Errorf("Unexpected error from GetMetrics: %v", err)
	}

	if metrics == nil {
		t.Errorf("Expected GetMetrics to return non-nil metrics")
	}

	// Should have timestamp
	if metrics.Timestamp.IsZero() {
		t.Errorf("Expected metrics to have non-zero timestamp")
	}

	// Should have initialized errors map
	if metrics.Errors == nil {
		t.Errorf("Expected metrics to have initialized Errors map")
	}

	// Should have server info
	if metrics.Server == nil {
		t.Errorf("Expected metrics to have server info")
	}

	// Should not have FPM data since it's disabled
	if metrics.Fpm != nil {
		t.Errorf("Expected no FPM data when FPM is disabled")
	}

	// Should not have Laravel data since no configs
	if metrics.Laravel != nil {
		t.Errorf("Expected no Laravel data when no Laravel configs")
	}
}

func TestGetMetrics_WithLaravelConfig(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
		Laravel: []config.LaravelConfig{
			{
				Name:          "TestApp",
				Path:          "/tmp/nonexistent", // This will cause errors, which is expected
				EnableAppInfo: true,
			},
		},
	}

	ctx := context.Background()
	metrics, err := GetMetrics(ctx, cfg)

	if err != nil {
		t.Errorf("Unexpected error from GetMetrics: %v", err)
	}

	if metrics == nil {
		t.Errorf("Expected GetMetrics to return non-nil metrics")
	}

	// Should have Laravel data structure initialized
	if metrics.Laravel == nil {
		t.Errorf("Expected metrics to have Laravel data structure")
	}

	// Should have errors due to nonexistent path
	if len(metrics.Errors) == 0 {
		t.Errorf("Expected errors due to nonexistent Laravel path")
	}
}

func TestListener_FunctionType(t *testing.T) {
	// Test that Listener function type works as expected
	var listener Listener = func(m *Metrics) {
		// This test just verifies the function type compiles
		if m == nil {
			t.Errorf("Metrics should not be nil")
		}
	}

	// Call the listener
	testMetrics := &Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	listener(testMetrics)
}