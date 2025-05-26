package config

import (
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	Debug   bool            `mapstructure:"debug"`
	Logging LoggingBlock    `mapstructure:"logging"`
	PHPFpm  FPMConfig       `mapstructure:"phpfpm"`
	PHP     PHPConfig       `mapstructure:"php"`
	Monitor MonitorConfig   `mapstructure:"monitor"`
	Laravel []LaravelConfig `mapstructure:"laravel"`
}

type LoggingBlock struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // text, json
	Color  bool   `mapstructure:"color"`  // enable ANSI colors in text format
}

type PHPConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Binary  string `mapstructure:"binary"`
	IniPath string `mapstructure:"ini_path"`
}

type FPMConfig struct {
	Enabled      bool            `mapstructure:"enabled"`
	Autodiscover bool            `mapstructure:"autodiscover"`
	Retries      int             `mapstructure:"retries"`
	RetryDelay   int             `mapstructure:"retry_delay"`
	Pools        []FPMPoolConfig `mapstructure:"pools"`
	PollInterval time.Duration   `mapstructure:"poll_interval"`
}

type FPMPoolConfig struct {
	Socket            string        `mapstructure:"socket"`
	StatusSocket      string        `mapstructure:"status_socket"`
	StatusPath        string        `mapstructure:"status_path"`
	StatusPathEnabled bool          `mapstructure:"status_path_enabled"`
	ConfigPath        string        `mapstructure:"config_path"`
	Binary            string        `mapstructure:"binary"`
	CliBinary         string        `mapstructure:"cli_binary"`
	PollInterval      time.Duration `mapstructure:"poll_interval"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

type LaravelConfig struct {
	Name      string              `mapstructure:"name"`       // Optional name for identification
	Path      string              `mapstructure:"path"`       // Root path to Laravel app
	PHPConfig *PHPConfig          `mapstructure:"php_config"` // Optional override of global PHP config
	Queues    map[string][]string `mapstructure:"queues"`     // Map of connection name to list of queue names
}

type MonitorConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
	EnableJson bool   `mapstructure:"enable_json"`
}

func Load() (*Config, error) {
	viper.SetDefault("debug", false)

	viper.SetDefault("phpfpm.enabled", true)
	viper.SetDefault("phpfpm.autodiscover", true)
	viper.SetDefault("phpfpm.retries", 5)
	viper.SetDefault("phpfpm.retry_delay", 2)
	viper.SetDefault("phpfpm.poll_interval", "1s")
	viper.SetDefault("phpfpm.pools", []FPMPoolConfig{})

	viper.SetDefault("php.enabled", true)
	viper.SetDefault("php.binary", "php")

	viper.SetDefault("monitor.listen_addr", ":9114")
	viper.SetDefault("monitor.enable_json", true)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.color", true)

	viper.SetDefault("laravel", []LaravelConfig{})
	// No default queue config, expected to be provided per site if needed

	viper.SetEnvPrefix("ELASTICPHP")
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
