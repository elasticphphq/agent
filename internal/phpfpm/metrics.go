package phpfpm

import (
	"bufio"
	"context"
	"fmt"
	"github.com/elasticphphq/agent/internal/logging"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/fcgx"
)

type PoolProcess struct {
	PID               int     `json:"pid"`
	State             string  `json:"state"`
	StartTime         int64   `json:"start time"`
	StartSince        int64   `json:"start since"`
	Requests          int64   `json:"requests"`
	RequestDuration   int64   `json:"request duration"`
	RequestMethod     string  `json:"request method"`
	RequestURI        string  `json:"request uri"`
	ContentLength     int64   `json:"content length"`
	User              string  `json:"user"`
	Script            string  `json:"script"`
	LastRequestCPU    float64 `json:"last request cpu"`
	LastRequestMemory float64 `json:"last request memory"`
	CurrentRSS        int64   `json:"current_rss"`
}

type Pool struct {
	Address             string            `json:"address"`
	Path                string            `json:"path"`
	Name                string            `json:"pool"`
	ProcessManager      string            `json:"process manager"`
	StartTime           int64             `json:"start time"`
	StartSince          int64             `json:"start since"`
	AcceptedConnections int64             `json:"accepted conn"`
	ListenQueue         int64             `json:"listen queue"`
	MaxListenQueue      int64             `json:"max listen queue"`
	ListenQueueLength   int64             `json:"listen queue len"`
	IdleProcesses       int64             `json:"idle processes"`
	ActiveProcesses     int64             `json:"active processes"`
	TotalProcesses      int64             `json:"total processes"`
	MaxActiveProcesses  int64             `json:"max active processes"`
	MaxChildrenReached  int64             `json:"max children reached"`
	SlowRequests        int64             `json:"slow requests"`
	MemoryPeak          int64             `json:"memory peak"`
	Processes           []PoolProcess     `json:"processes"`
	ProcessesCpu        *float64          `json:"processes_cpu"`
	ProcessesMemory     *float64          `json:"processes_memory"`
	ProcessesMemoryRss  *float64          `json:"processes_memory_rss"`
	Config              map[string]string `json:"config,omitempty"`
	OpcacheStatus       OpcacheStatus     `json:"opcache_status,omitempty"`
	PhpInfo             Info              `json:"php_info,omitempty"`
}

type Result struct {
	Timestamp time.Time
	Pools     map[string]Pool
	Global    map[string]string `json:"global_config,omitempty"`
}

func GetMetrics(ctx context.Context, cfg *config.Config) (map[string]*Result, error) {
	results := map[string]*Result{}

	for _, poolCfg := range cfg.PHPFpm.Pools {
		result := &Result{
			Timestamp: time.Now(),
			Pools:     make(map[string]Pool),
			Global:    make(map[string]string),
		}

		scheme, address, path, err := ParseAddress(poolCfg.Socket, poolCfg.StatusPath)
		if err != nil {
			log.Printf("invalid FPM socket address: %v", err)
			continue
		}

		dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		logging.L().Debug("Dialing FastCGI", "scheme", scheme, "address", address, "status_path", path)
		client, err := fcgx.DialContext(dialCtx, scheme, address)
		cancel()
		if err != nil {
			logging.L().Debug("failed to dial FastCGI", "error", err)
			continue
		}
		defer client.Close()

		env := map[string]string{
			"SCRIPT_FILENAME": path,
			"SCRIPT_NAME":     path,
			"SERVER_SOFTWARE": "elasticphp-agent",
			"REMOTE_ADDR":     "127.0.0.1",
			"QUERY_STRING":    "json&full",
		}
		logging.L().Debug("Sending FCGI request", "env", env)

		resp, err := client.Get(ctx, env)
		if err != nil {
			logging.L().Debug("fcgi GET failed", "error", err)
			continue
		}
		defer resp.Body.Close()

		var pool Pool
		err = fcgx.ReadJSON(resp, &pool)

		if err != nil {
			log.Printf("failed to parse FPM JSON: %v", err)
			continue
		}

		pool.Address = address
		pool.Path = path

		if conf, err := ParseFPMConfig(poolCfg.Binary, poolCfg.ConfigPath); err == nil {
			for section, values := range conf.Pools {
				if strings.EqualFold(section, pool.Name) {
					pool.Config = values
				}
			}
			for k, v := range conf.Global {
				result.Global[k] = v
			}
		}

		// CPU/mem parsing
		var totalCPU, totalMem, totalMemRss float64
		var count int
		for i, proc := range pool.Processes {
			if !strings.HasPrefix(proc.RequestURI, poolCfg.StatusPath) &&
				!strings.HasPrefix(proc.RequestURI, "/opcache-status-") {

				if runtime.GOOS == "darwin" {
					cmd := exec.Command("ps", "-o", "rss=,vsz=", "-p", fmt.Sprint(proc.PID))
					out, err := cmd.Output()
					if err == nil {
						fields := strings.Fields(string(out))
						if len(fields) >= 2 {
							rss, _ := strconv.ParseInt(fields[0], 10, 64)
							pool.Processes[i].CurrentRSS = rss * 1024
						}
					}
				} else if runtime.GOOS == "linux" {
					if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", proc.PID)); err == nil {
						scanner := bufio.NewScanner(strings.NewReader(string(data)))
						for scanner.Scan() {
							line := scanner.Text()
							if strings.HasPrefix(line, "VmRSS:") {
								fields := strings.Fields(line)
								if len(fields) >= 2 {
									val, _ := strconv.ParseInt(fields[1], 10, 64)
									pool.Processes[i].CurrentRSS = val * 1024
								}
							}
						}
					}
				}

				totalCPU += float64(proc.LastRequestCPU)
				totalMem += float64(proc.LastRequestMemory)
				totalMemRss += float64(proc.CurrentRSS)
				count++
			}
		}
		if count > 0 {
			pool.ProcessesCpu = ptr(totalCPU / float64(count))
			pool.ProcessesMemory = ptr(totalMem / float64(count))
			pool.ProcessesMemoryRss = ptr(totalMemRss / float64(count))
		}

		phpStatus, err := GetPHPStats(ctx, poolCfg)
		if err == nil && phpStatus != nil {
			pool.PhpInfo = *phpStatus
		} else {
			logging.L().Info("failed to get PHP info", "error", err)
		}

		opcacheStatus, err := GetOpcacheStatus(ctx, poolCfg)
		if err == nil && opcacheStatus != nil {
			pool.OpcacheStatus = *opcacheStatus
		}

		result.Pools[pool.Name] = pool
		results[poolCfg.Socket] = result
	}

	return results, nil
}

func GetMetricsForPool(ctx context.Context, pool config.FPMPoolConfig) (*Result, error) {
	scheme, address, path, err := ParseAddress(pool.Socket, pool.StatusPath)
	if err != nil {
		return nil, fmt.Errorf("invalid FPM socket address: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FastCGI: %w", err)
	}
	defer client.Close()

	env := map[string]string{
		"SCRIPT_FILENAME": path,
		"SCRIPT_NAME":     path,
		"SERVER_SOFTWARE": "elasticphp-agent",
		"REMOTE_ADDR":     "127.0.0.1",
		"QUERY_STRING":    "json&full",
	}

	resp, err := client.Get(ctx, env)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var poolData Pool
	err = fcgx.ReadJSON(resp, &poolData)

	if err != nil {
		return nil, fmt.Errorf("failed to parse FPM JSON: %w", err)
	}

	return &Result{
		Timestamp: time.Now(),
		Pools:     map[string]Pool{poolData.Name: poolData},
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}

func ParseAddress(addr string, path string) (scheme, address, scriptPath string, err error) {
	if strings.HasPrefix(addr, "unix://") {
		return "unix", strings.TrimPrefix(addr, "unix://"), path, nil
	}
	if strings.HasPrefix(addr, "tcp://") {
		return "tcp", strings.TrimPrefix(addr, "tcp://"), path, nil
	}
	if strings.HasPrefix(addr, "/") {
		return "unix", addr, path, nil
	}
	return "", "", "", fmt.Errorf("unsupported socket format: %s", addr)
}
