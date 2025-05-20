package phpfpm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/fcgx"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	scriptContent := `<?php header("Content-Type: application/json"); echo json_encode(opcache_get_status());`
	tmpFile, err := os.CreateTemp("/tmp", "opcache-status-*.php")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp PHP script: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp PHP script: %w", err)
	}
	tmpFile.Close()

	scheme, address, _, err := ParseAddress(cfg.Socket, "")
	if err != nil {
		return nil, fmt.Errorf("invalid socket: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FPM: %w", err)
	}
	defer client.Close()

	scriptPath := tmpFile.Name()
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read fcgi response: %w", err)
	}

	if !strings.HasPrefix(string(body), "{") {
		return nil, fmt.Errorf("invalid JSON output: %s", body)
	}

	var status OpcacheStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse opcache JSON: %w", err)
	}
	return &status, nil
}
