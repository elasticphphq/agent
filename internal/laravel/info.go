package laravel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
)

type BoolString bool

func (b *BoolString) UnmarshalJSON(data []byte) error {
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*b = BoolString(asBool)
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		switch strings.ToLower(asString) {
		case "enabled", "true", "on", "yes", "cached":
			*b = true
		default:
			*b = false
		}
		return nil
	}
	return fmt.Errorf("invalid boolean value: %s", string(data))
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var slice []string
	if err := json.Unmarshal(data, &slice); err == nil {
		*s = slice
		return nil
	}
	return fmt.Errorf("invalid value for StringOrSlice: %s", string(data))
}

type AppInfo struct {
	Environment struct {
		ApplicationName string     `json:"application_name"`
		LaravelVersion  string     `json:"laravel_version"`
		PHPVersion      string     `json:"php_version"`
		ComposerVersion string     `json:"composer_version"`
		Environment     string     `json:"environment"`
		DebugMode       BoolString `json:"debug_mode"`
		URL             string     `json:"url"`
		MaintenanceMode BoolString `json:"maintenance_mode"`
		Timezone        string     `json:"timezone"`
		Locale          string     `json:"locale"`
	} `json:"environment"`

	Cache struct {
		Config BoolString `json:"config"`
		Events BoolString `json:"events"`
		Routes BoolString `json:"routes"`
		Views  BoolString `json:"views"`
	} `json:"cache"`

	Drivers struct {
		Broadcasting string        `json:"broadcasting"`
		Cache        string        `json:"cache"`
		Database     string        `json:"database"`
		Logs         StringOrSlice `json:"logs"`
		Mail         string        `json:"mail"`
		Queue        string        `json:"queue"`
		Session      string        `json:"session"`
	} `json:"drivers"`

	Livewire map[string]string `json:"livewire,omitempty"`
}

var (
	appInfoCache = make(map[string]*AppInfo)
	cacheMutex   sync.RWMutex
)

func GetAppInfo(site config.LaravelConfig, phpBinary string) (*AppInfo, error) {
	if !site.EnableAppInfo {
		return nil, nil
	}

	if phpBinary == "" || site.Path == "" {
		return nil, fmt.Errorf("invalid input: phpBinary and path are required")
	}

	cacheKey := filepath.Clean(site.Path)

	cacheMutex.RLock()
	info, ok := appInfoCache[cacheKey]
	cacheMutex.RUnlock()
	if ok {
		if info == nil {
			return nil, fmt.Errorf("app info was previously attempted but failed")
		}
		return info, nil
	}

	logging.L().Debug("Uncached app info. Calling artisan about", "path", site.Path)

	cmd := exec.Command(phpBinary, "-d", "error_reporting=E_ALL & ~E_DEPRECATED", "artisan", "about", "--json")
	cmd.Dir = cacheKey

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		cacheMutex.Lock()
		appInfoCache[cacheKey] = nil
		cacheMutex.Unlock()
		return nil, fmt.Errorf("artisan about failed: %w\nOutput: %s", err, out.String())
	}

	var parsed AppInfo
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		cacheMutex.Lock()
		appInfoCache[cacheKey] = nil
		cacheMutex.Unlock()
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	cacheMutex.Lock()
	appInfoCache[cacheKey] = &parsed
	cacheMutex.Unlock()

	return &parsed, nil
}
