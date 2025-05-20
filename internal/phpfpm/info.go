package phpfpm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/fcgx"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Info struct {
	Version    string
	Extensions []string
	Opcache    *OpcacheStatus
}

func GetPHPStats(ctx context.Context, cfg config.FPMPoolConfig) (*Info, error) {
	version, err := getPHPVersion(cfg.Binary)
	if err != nil {
		return nil, err
	}
	ext, err := getPHPExtensions(cfg.Binary)
	if err != nil {
		return nil, err
	}

	info := &Info{
		Version:    version,
		Extensions: ext,
	}

	return info, nil
}

func getPHPVersion(bin string) (string, error) {
	out, err := exec.Command(bin, "-v").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "unknown", nil
}

func getPHPExtensions(bin string) ([]string, error) {
	out, err := exec.Command(bin, "-m").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var exts []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "[") {
			exts = append(exts, line)
		}
	}
	return exts, nil
}

func getPHPConfig(ctx context.Context, cfg config.FPMPoolConfig) (map[string]interface{}, error) {
	scheme, address, _, err := ParseAddress(cfg.Socket, cfg.StatusPath)
	if err != nil {
		return nil, fmt.Errorf("invalid FPM socket address: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FastCGI: %w", err)
	}
	defer client.Close()

	confScript := `<?php header("Content-Type: application/json"); echo json_encode(ini_get_all());`
	tmpConfFile, err := os.CreateTemp("/tmp", "fpm-config-*.php")
	defer os.Remove(tmpConfFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to create temp PHP config script: %w", err)
	}
	if _, err := tmpConfFile.WriteString(confScript); err != nil {
		tmpConfFile.Close()
		return nil, fmt.Errorf("failed to write config PHP script: %w", err)
	}
	tmpConfFile.Close()

	scriptPath := tmpConfFile.Name()
	confEnv := map[string]string{
		"SCRIPT_FILENAME": scriptPath,
		"SCRIPT_NAME":     "/" + filepath.Base(scriptPath),
		"SERVER_SOFTWARE": "elasticphp-agent",
		"REMOTE_ADDR":     "127.0.0.1",
	}

	resp, err := client.Get(ctx, confEnv)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read FastCGI config response: %w", err)
	}
	var conf map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &conf); err != nil {
		return nil, fmt.Errorf("FPM Config JSON parse failed: %w", err)
	}
	return conf, nil
}
