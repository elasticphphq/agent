package phpfpm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/fcgx"
	"os"
	"path/filepath"
)

type OpcacheStatus struct {
	Enabled     bool   `json:"opcache_enabled"`
	MemoryUsage Memory `json:"memory_usage"`
	Statistics  Stats  `json:"opcache_statistics"`
}

type Memory struct {
	UsedMemory       uint64  `json:"used_memory"`
	FreeMemory       uint64  `json:"free_memory"`
	WastedMemory     uint64  `json:"wasted_memory"`
	CurrentWastedPct float64 `json:"current_wasted_percentage"`
}

type Stats struct {
	NumCachedScripts uint64  `json:"num_cached_scripts"`
	Hits             uint64  `json:"hits"`
	Misses           uint64  `json:"misses"`
	BlacklistMisses  uint64  `json:"blacklist_misses"`
	OomRestarts      uint64  `json:"oom_restarts"`
	HashRestarts     uint64  `json:"hash_restarts"`
	ManualRestarts   uint64  `json:"manual_restarts"`
	HitRate          float64 `json:"opcache_hit_rate"`
}

func GetOpcacheStatus(ctx context.Context, cfg config.FPMPoolConfig) (*OpcacheStatus, error) {
	tmpPath := "/tmp/elasticphp-opcache-status.php"
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		scriptContent := `<?php
error_reporting(0);
ini_set('display_errors', 0);
header("Status: 200 OK");
header("Content-Type: application/json");
echo json_encode(opcache_get_status());
exit;`
		if err := os.WriteFile(tmpPath, []byte(scriptContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write PHP script: %w", err)
		}
	}

	scheme, address, _, err := ParseAddress(cfg.StatusSocket, "")
	if err != nil {
		return nil, fmt.Errorf("invalid socket: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FPM: %w", err)
	}
	defer client.Close()

	scriptPath := tmpPath
	env := map[string]string{
		"SCRIPT_FILENAME": scriptPath,
		"SCRIPT_NAME":     "/" + filepath.Base(scriptPath),
		"SERVER_SOFTWARE": "elasticphp-agent",
		"REMOTE_ADDR":     "127.0.0.1",
	}

	resp, err := client.Get(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("fcgi GET failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := fcgx.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read opcache response: %w", err)
	}

	var status OpcacheStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse opcache JSON: %w", err)
	}

	return &status, nil
}
