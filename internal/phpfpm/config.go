package phpfpm

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)
import (
	"sync"
)

var (
	fpmConfigCache     = make(map[string]*FPMConfig)
	fpmConfigCacheLock sync.Mutex
)

type FPMConfig struct {
	Global map[string]string
	Pools  map[string]map[string]string
}

func ParseFPMConfig(FPMBinaryPath string, FPMConfigPath string) (*FPMConfig, error) {
	key := FPMBinaryPath + "::" + FPMConfigPath

	fpmConfigCacheLock.Lock()
	cached, ok := fpmConfigCache[key]
	fpmConfigCacheLock.Unlock()

	if ok {
		return cached, nil
	}

	cmd := exec.Command(FPMBinaryPath, "-tt", "--fpm-config", FPMConfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run php-fpm -tt: %w\nOutput: %s", err, output)
	}

	fpmconfig := &FPMConfig{
		Global: make(map[string]string),
		Pools:  make(map[string]map[string]string),
	}
	currentSection := "global"

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if idx := strings.Index(line, "] NOTICE:"); idx != -1 {
			line = strings.TrimSpace(line[idx+len("] NOTICE:"):])
		}

		line = strings.ReplaceAll(line, "\\t", "")
		line = strings.ReplaceAll(line, "\t", "")
		line = strings.TrimSpace(strings.Trim(line, `"`))

		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if section == "global" {
				currentSection = "global"
				continue
			}
			currentSection = section
			if _, ok := fpmconfig.Pools[currentSection]; !ok {
				fpmconfig.Pools[currentSection] = make(map[string]string)
			}
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(strings.Trim(parts[0], `"`))
			val := strings.TrimSpace(strings.Trim(parts[1], `"`))
			if val == "undefined" {
				val = ""
			}

			if currentSection != "global" {
				if _, ok := fpmconfig.Pools[currentSection]; !ok {
					fpmconfig.Pools[currentSection] = make(map[string]string)
				}
				fpmconfig.Pools[currentSection][key] = val
			} else {
				fpmconfig.Global[key] = val
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan php-fpm config output: %w", err)
	}

	fpmConfigCacheLock.Lock()
	fpmConfigCache[key] = fpmconfig
	fpmConfigCacheLock.Unlock()

	return fpmconfig, nil
}
