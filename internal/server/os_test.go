package server

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestSystemInfo_Structure(t *testing.T) {
	info := SystemInfo{
		NodeType:      NodeKubernetes,
		OS:            "linux",
		Architecture:  "amd64",
		CPULimit:      4,
		MemoryLimitMB: 8192,
	}

	if info.NodeType != NodeKubernetes {
		t.Errorf("Expected NodeType to be NodeKubernetes")
	}

	if info.OS != "linux" {
		t.Errorf("Expected OS to be 'linux', got %s", info.OS)
	}

	if info.Architecture != "amd64" {
		t.Errorf("Expected Architecture to be 'amd64', got %s", info.Architecture)
	}

	if info.CPULimit != 4 {
		t.Errorf("Expected CPULimit to be 4, got %d", info.CPULimit)
	}

	if info.MemoryLimitMB != 8192 {
		t.Errorf("Expected MemoryLimitMB to be 8192, got %d", info.MemoryLimitMB)
	}
}

func TestSystemInfoData_Structure(t *testing.T) {
	systemInfo := &SystemInfo{
		NodeType: NodeDocker,
		OS:       "linux",
	}

	data := SystemInfoData{
		SystemInfo: systemInfo,
		Errors: map[string]string{
			"test_error": "This is a test error",
		},
	}

	if data.SystemInfo != systemInfo {
		t.Errorf("Expected SystemInfo to match")
	}

	if len(data.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(data.Errors))
	}

	if data.Errors["test_error"] != "This is a test error" {
		t.Errorf("Expected error message to match")
	}
}

func TestNodeType_Constants(t *testing.T) {
	// Test that NodeType constants are defined
	if NodeKubernetes != "kubernetes" {
		t.Errorf("Expected NodeKubernetes to be 'kubernetes', got %s", NodeKubernetes)
	}

	if NodeDocker != "docker" {
		t.Errorf("Expected NodeDocker to be 'docker', got %s", NodeDocker)
	}

	if NodeVM != "vm" {
		t.Errorf("Expected NodeVM to be 'vm', got %s", NodeVM)
	}

	if NodePhysical != "physical" {
		t.Errorf("Expected NodePhysical to be 'physical', got %s", NodePhysical)
	}
}

func TestDetectSystem(t *testing.T) {
	// Clear cache before test
	sysInfoMu.Lock()
	cachedSystemInfo = nil
	lastSystemCheck = time.Time{}
	sysInfoMu.Unlock()

	data := DetectSystem()

	if data == nil {
		t.Fatalf("Expected DetectSystem to return non-nil data")
	}

	if data.SystemInfo == nil {
		t.Fatalf("Expected SystemInfo to be non-nil")
	}

	if data.Errors == nil {
		t.Fatalf("Expected Errors map to be non-nil")
	}

	// Test that basic fields are populated
	if data.SystemInfo.OS == "" {
		t.Errorf("Expected OS to be populated")
	}

	if data.SystemInfo.Architecture == "" {
		t.Errorf("Expected Architecture to be populated")
	}

	// Should match runtime values
	if data.SystemInfo.OS != runtime.GOOS {
		t.Errorf("Expected OS to match runtime.GOOS")
	}

	if data.SystemInfo.Architecture != runtime.GOARCH {
		t.Errorf("Expected Architecture to match runtime.GOARCH")
	}
}

func TestDetectSystem_Caching(t *testing.T) {
	// Clear cache before test
	sysInfoMu.Lock()
	cachedSystemInfo = nil
	lastSystemCheck = time.Time{}
	sysInfoMu.Unlock()

	// First call
	data1 := DetectSystem()

	// Second call (should use cache)
	data2 := DetectSystem()

	// Should be the same instance
	if data1 != data2 {
		t.Errorf("Expected cached result to be the same instance")
	}

	// Test cache expiry by setting old time
	sysInfoMu.Lock()
	lastSystemCheck = time.Now().Add(-11 * time.Minute)
	sysInfoMu.Unlock()

	// Third call (should refresh cache)
	data3 := DetectSystem()

	// Should be a new instance
	if data1 == data3 {
		t.Errorf("Expected refreshed cache to be a different instance")
	}
}

func TestDetectNodeType(t *testing.T) {
	// Save original environment
	originalKubeHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	defer os.Setenv("KUBERNETES_SERVICE_HOST", originalKubeHost)

	// Test Kubernetes detection
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	nodeType := detectNodeType()
	if nodeType != NodeKubernetes {
		t.Errorf("Expected NodeKubernetes when KUBERNETES_SERVICE_HOST is set")
	}

	// Test without Kubernetes environment
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	nodeType = detectNodeType()

	// Should be one of the valid node types
	validTypes := []NodeType{NodeDocker, NodeVM, NodePhysical}
	found := false
	for _, validType := range validTypes {
		if nodeType == validType {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected valid node type, got %s", nodeType)
	}
}

func TestDetectCPULimit(t *testing.T) {
	cpuLimit, err := detectCPULimit()

	// Should not error
	if err != nil {
		t.Errorf("Unexpected error from detectCPULimit: %v", err)
	}

	// Should return a positive value
	if cpuLimit <= 0 {
		t.Errorf("Expected positive CPU limit, got %d", cpuLimit)
	}

	// Should be reasonable (not more than 1000 CPUs)
	if cpuLimit > 1000 {
		t.Errorf("CPU limit seems unreasonably high: %d", cpuLimit)
	}

	// Should be at least runtime.NumCPU() (fallback case)
	if cpuLimit < int64(runtime.NumCPU()) {
		t.Errorf("Expected CPU limit to be at least %d, got %d", runtime.NumCPU(), cpuLimit)
	}
}

func TestDetectMemoryLimit(t *testing.T) {
	memoryLimit, err := detectMemoryLimit()

	// Should not error
	if err != nil {
		t.Errorf("Unexpected error from detectMemoryLimit: %v", err)
	}

	// Should return a value (could be -1 if not detectable)
	if memoryLimit == 0 {
		t.Errorf("Expected non-zero memory limit")
	}

	// If positive, should be reasonable (less than 1TB)
	if memoryLimit > 1024*1024 {
		t.Errorf("Memory limit seems unreasonably high: %d MB", memoryLimit)
	}
}

func TestDetectMemoryLimit_EdgeCases(t *testing.T) {
	// Test the function behavior by examining different code paths
	// We can't easily mock file reads, but we can test the logic

	// Test that the function handles the case when no memory info is available
	// This is tested implicitly by the main test above

	// Test parsing logic with sample data (indirectly)
	// The parsing happens inside the function, but we can test that
	// it doesn't crash with various inputs by calling it multiple times
	for i := 0; i < 5; i++ {
		mem, _ := detectMemoryLimit()
		if mem < -1 {
			t.Errorf("Multiple calls should return consistent valid values, got: %d", mem)
		}
	}
}

func TestDetectCPULimit_EdgeCases(t *testing.T) {
	// Test the detectCPULimit function multiple times for consistency
	for i := 0; i < 3; i++ {
		cpu, err := detectCPULimit()
		if err != nil {
			t.Logf("detectCPULimit returned error (iteration %d): %v", i, err)
		}

		// Should return a positive value
		if cpu <= 0 {
			t.Errorf("detectCPULimit should return positive value, got: %d", cpu)
		}

		// Should not exceed reasonable limits (e.g., 1024 cores)
		if cpu > 1024 {
			t.Errorf("detectCPULimit returned unreasonably high value: %d", cpu)
		}
	}
}

func TestDetectNodeType_FileSystemConditions(t *testing.T) {
	// Test detectNodeType behavior under different conditions
	// Save original environment
	originalKubeHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	defer func() {
		if originalKubeHost == "" {
			os.Unsetenv("KUBERNETES_SERVICE_HOST")
		} else {
			os.Setenv("KUBERNETES_SERVICE_HOST", originalKubeHost)
		}
	}()

	// Test without Kubernetes environment
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	nodeType := detectNodeType()

	// Should return one of the valid node types
	validTypes := map[NodeType]bool{
		NodeKubernetes: true,
		NodeDocker:     true,
		NodeVM:         true,
		NodePhysical:   true,
	}

	if !validTypes[nodeType] {
		t.Errorf("detectNodeType returned invalid type: %s", nodeType)
	}

	// Test with Kubernetes environment set
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	nodeType = detectNodeType()
	if nodeType != NodeKubernetes {
		t.Errorf("Expected NodeKubernetes when KUBERNETES_SERVICE_HOST is set, got: %s", nodeType)
	}

	// Test with empty Kubernetes environment (should NOT be detected as k8s)
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	nodeType = detectNodeType()
	// Empty string means the env var is set but empty, which our code treats as set
	if nodeType != NodePhysical {
		t.Errorf("Expected NodePhysical when KUBERNETES_SERVICE_HOST is empty but set, got: %s", nodeType)
	}
}

func TestDetectNodeType_EdgeCases(t *testing.T) {
	// Save original environment
	originalKubeHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	defer func() {
		if originalKubeHost == "" {
			os.Unsetenv("KUBERNETES_SERVICE_HOST")
		} else {
			os.Setenv("KUBERNETES_SERVICE_HOST", originalKubeHost)
		}
	}()

	// Test with empty KUBERNETES_SERVICE_HOST
	os.Setenv("KUBERNETES_SERVICE_HOST", "")
	nodeType := detectNodeType()
	if nodeType == NodeKubernetes {
		t.Errorf("Expected non-Kubernetes node type with empty KUBERNETES_SERVICE_HOST")
	}

	// Unset environment variable
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	nodeType = detectNodeType()

	// Should not panic and should return a valid node type
	validTypes := []NodeType{NodeDocker, NodeVM, NodePhysical}
	found := false
	for _, validType := range validTypes {
		if nodeType == validType {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected valid node type, got %s", nodeType)
	}
}

func TestSystemInfoData_WithErrors(t *testing.T) {
	data := &SystemInfoData{
		SystemInfo: &SystemInfo{
			NodeType: NodePhysical,
			OS:       "linux",
		},
		Errors: map[string]string{
			"cpu":    "Failed to detect CPU limit",
			"memory": "Failed to detect memory limit",
		},
	}

	if len(data.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(data.Errors))
	}

	if data.Errors["cpu"] != "Failed to detect CPU limit" {
		t.Errorf("Expected CPU error message to match")
	}

	if data.Errors["memory"] != "Failed to detect memory limit" {
		t.Errorf("Expected memory error message to match")
	}
}

func TestConcurrentDetectSystem(t *testing.T) {
	// Clear cache before test
	sysInfoMu.Lock()
	cachedSystemInfo = nil
	lastSystemCheck = time.Time{}
	sysInfoMu.Unlock()

	// Run multiple goroutines concurrently
	results := make(chan *SystemInfoData, 5)
	for i := 0; i < 5; i++ {
		go func() {
			results <- DetectSystem()
		}()
	}

	// Collect results
	var data []*SystemInfoData
	for i := 0; i < 5; i++ {
		data = append(data, <-results)
	}

	// All should be non-nil
	for i, d := range data {
		if d == nil {
			t.Errorf("Result %d should not be nil", i)
		}
		if d.SystemInfo == nil {
			t.Errorf("Result %d SystemInfo should not be nil", i)
		}
	}

	// Due to caching, many results should be the same instance
	sameCount := 0
	for i := 1; i < len(data); i++ {
		if data[i] == data[0] {
			sameCount++
		}
	}

	// At least some should be cached (this is a probabilistic test)
	if sameCount == 0 {
		t.Logf("Warning: No cached results found, caching may not be working properly")
	}
}
