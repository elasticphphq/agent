package server

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	sysInfoMu        sync.Mutex
	cachedSystemInfo *SystemInfoData
	lastSystemCheck  time.Time
)

type NodeType string

const (
	NodeKubernetes NodeType = "kubernetes"
	NodeDocker     NodeType = "docker"
	NodeVM         NodeType = "vm"
	NodePhysical   NodeType = "physical"
)

type SystemInfo struct {
	NodeType      NodeType
	OS            string
	Architecture  string
	CPULimit      int64 // Logical CPUs (possibly limited)
	MemoryLimitMB int64 // In MB
}

type SystemInfoData struct {
	SystemInfo *SystemInfo
	Errors     map[string]string
}

func DetectSystem() *SystemInfoData {
	sysInfoMu.Lock()
	if cachedSystemInfo != nil && time.Since(lastSystemCheck) < 10*time.Minute {
		defer sysInfoMu.Unlock()
		return cachedSystemInfo
	}
	sysInfoMu.Unlock()

	sysInfoMu.Lock()
	defer sysInfoMu.Unlock()

	info := SystemInfo{
		NodeType:     detectNodeType(),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	errors := make(map[string]string)

	cpu, err := detectCPULimit()
	if err != nil {
		errors["cpu"] = err.Error()
	} else {
		info.CPULimit = cpu
	}

	mem, err := detectMemoryLimit()
	if err != nil {
		errors["memory"] = err.Error()
	} else {
		info.MemoryLimitMB = mem
	}

	cachedSystemInfo = &SystemInfoData{
		SystemInfo: &info,
		Errors:     errors,
	}
	lastSystemCheck = time.Now()

	return cachedSystemInfo
}

func detectNodeType() NodeType {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return NodeKubernetes
	}
	if content, err := os.ReadFile("/proc/1/cgroup"); err == nil && strings.Contains(string(content), "docker") {
		return NodeDocker
	}
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		val := strings.ToLower(string(data))
		if strings.Contains(val, "kvm") || strings.Contains(val, "vmware") || strings.Contains(val, "virtualbox") {
			return NodeVM
		}
	}
	return NodePhysical
}

func detectCPULimit() (int64, error) {
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) == 2 && parts[0] != "max" {
			quota, _ := strconv.Atoi(parts[0])
			period, _ := strconv.Atoi(parts[1])
			if period > 0 && quota > 0 {
				return int64(quota / period), nil
			}
		}
	}

	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("sysctl", "-n", "hw.logicalcpu").Output(); err == nil {
			if cpu, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
				return int64(cpu), nil
			}
		}
	}

	return int64(runtime.NumCPU()), nil
}

func detectMemoryLimit() (int64, error) {
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			if bytes, err := strconv.ParseInt(val, 10, 64); err == nil {
				return int64(int(bytes / 1024 / 1024)), nil
			}
		}
	}

	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
			if mem, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
				return int64(int(mem / 1024 / 1024)), nil
			}
		}
	}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if kb, err := strconv.Atoi(parts[1]); err == nil {
						return int64(kb / 1024), nil
					}
				}
			}
		}
	}

	return -1, nil
}
