package laravel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elasticphphq/agent/internal/logging"
	"os/exec"
	"path/filepath"
)

// StringOrSlice is a custom type that unmarshals from either a string or a slice of strings.
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
		ApplicationName string `json:"application_name"`
		LaravelVersion  string `json:"laravel_version"`
		PHPVersion      string `json:"php_version"`
		ComposerVersion string `json:"composer_version"`
		Environment     string `json:"environment"`
		DebugMode       bool   `json:"debug_mode"`
		URL             string `json:"url"`
		MaintenanceMode bool   `json:"maintenance_mode"`
		Timezone        string `json:"timezone"`
		Locale          string `json:"locale"`
	} `json:"environment"`

	Cache struct {
		Config bool `json:"config"`
		Events bool `json:"events"`
		Routes bool `json:"routes"`
		Views  bool `json:"views"`
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

func GetAppInfo(appPath string, phpBinary string) (*AppInfo, error) {
	if phpBinary == "" || appPath == "" {
		return nil, fmt.Errorf("invalid input: phpBinary and appPath are required")
	}

	cmd := exec.Command(phpBinary, "artisan", "about", "--json")
	cmd.Dir = filepath.Clean(appPath)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("artisan tinker failed: %w\nOutput: %s", err, out.String())
	}

	logging.L().Debug("Raw Laravel info", "output", out.String(), "binary", phpBinary, "dir", cmd.Dir, "args", cmd.Args)

	var info AppInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	return &info, nil
}
