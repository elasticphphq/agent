package phpfpm

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/elasticphphq/agent/internal/logging"
	"github.com/shirou/gopsutil/v3/process"
)

type DiscoveredFPM struct {
	ConfigPath   string
	StatusPath   string
	Binary       string
	Socket       string
	StatusSocket string
	CliBinary    string
}

var fpmNamePattern = regexp.MustCompile(`^php[0-9]{0,2}.*fpm.*$`)

func DiscoverFPMProcesses() ([]DiscoveredFPM, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	var found []DiscoveredFPM

	for _, p := range procs {
		name, err := p.Name()
		if err != nil || !fpmNamePattern.MatchString(filepath.Base(name)) {
			continue
		}

		cmdlineStr, err := p.Cmdline()
		if err != nil || !strings.Contains(cmdlineStr, "master process") {
			continue
		}

		config := extractConfigFromMaster(cmdlineStr)
		if config == "" {
			continue
		}

		exe, err := p.Exe()
		if err != nil {
			logging.L().Debug("Cannot determine binary path", "pid", p.Pid, "error", err)
			continue
		}

		parsed, err := ParseFPMConfig(exe, config)
		if err != nil {
			logging.L().Error("Failed to parse FPM config", "config", config, "error", err)
			continue
		}

		for poolName, poolConfig := range parsed.Pools {
			socket := parseSocket(poolConfig["listen"])
			if socket == "" {
				continue
			}

			statusSocket := parseSocket(poolConfig["status_listen"])
			if statusSocket == "" {
				statusSocket = socket
			}

			status := poolConfig["pm.status_path"]
			if status == "" {
				status = parsed.Global["pm.status_path"]
			}
			if status == "" {
				logging.L().Debug("Skipping pool with no status path", "pool", poolName, "config", config)
				continue
			}

			cliBinary, _ := findMatchingCliBinary(exe)

			found = append(found, DiscoveredFPM{
				ConfigPath:   config,
				StatusPath:   status,
				Binary:       exe,
				Socket:       socket,
				StatusSocket: statusSocket,
				CliBinary:    cliBinary,
			})

			logging.L().Debug("Discovered php-fpm pool",
				"config", config,
				"pool", poolName,
				"socket", socket,
				"status_socket", statusSocket,
				"status_path", status,
				"cli_binary", cliBinary,
			)
		}
	}

	return found, nil
}

func parseSocket(socket string) string {
	if socket == "" {
		return ""
	}
	if strings.HasPrefix(socket, "/") {
		return "unix://" + socket
	} else if strings.Contains(socket, ":") {
		return "tcp://" + socket
	} else {
		// fallback if only a port is specified
		try := []string{"127.0.0.1:" + socket, "[::1]:" + socket}
		resolved := ""
		for _, candidate := range try {
			conn, err := net.DialTimeout("tcp", candidate, 500*time.Millisecond)
			if err == nil {
				conn.Close()
				resolved = candidate
				break
			} else {
				logging.L().Warn("Failed to connect to socket", "socket", candidate, "error", err)
			}
		}
		if resolved != "" {
			return "tcp://" + resolved
		}
	}
	return ""
}

func extractConfigFromMaster(cmdline string) string {
	start := strings.Index(cmdline, "(")
	end := strings.Index(cmdline, ")")
	if start != -1 && end != -1 && end > start {
		return cmdline[start+1 : end]
	}
	return ""
}

// findMatchingCliBinary attempts to find the php-cli binary that matches the version of the FPM binary.
func findMatchingCliBinary(fpmBinary string) (string, error) {
	out, err := exec.Command(fpmBinary, "-v").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version from fpm binary: %w", err)
	}
	re := regexp.MustCompile(`PHP (\d+\.\d+)`)
	matches := re.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse PHP version from output: %s", string(out))
	}
	version := string(matches[1]) // e.g. "8.2"

	candidates := []string{
		filepath.Join("/usr/bin", "php"+version),
		filepath.Join("/usr/local/bin", "php"+version),
		"php" + version, // Fallback til PATH
		"php",           // Sidste fallback
	}

	for _, cli := range candidates {
		out, err := exec.Command(cli, "-v").Output()
		if err != nil {
			continue
		}
		if bytes.Contains(out, []byte(version)) && bytes.Contains(out, []byte("cli")) {
			return cli, nil
		}
	}
	return "", fmt.Errorf("matching php-cli binary for version %s not found", version)
}
