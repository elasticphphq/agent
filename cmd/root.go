package cmd

import (
	"fmt"
	"github.com/elasticphphq/agent/internal/phpfpm"
	"os"
	"strings"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version that is being reported by the CLI
var Version string

var Config *config.Config

var laravelFlags []string

var rootCmd = &cobra.Command{
	Use:   "elasticphp-agent",
	Short: "ElasticPHP Agent for monitoring PHP",
	Long:  `elasticphp-agent is a lightweight php metrics collector`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Read config file if specified
		if path := viper.GetString("config"); path != "" {
			viper.SetConfigFile(path)
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
		}

		loaded, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Parse Laravel sites defined through CLI
		if len(laravelFlags) > 0 {
			var sites []config.LaravelConfig
			seenNames := map[string]bool{}

			for _, entry := range laravelFlags {
				parts := strings.Split(entry, ",")
				site := config.LaravelConfig{
					Queues: map[string][]string{},
				}
				var lastConnection string
				for _, part := range parts {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) != 2 {
						continue
					}
					key, val := kv[0], kv[1]
					switch key {
					case "name":
						site.Name = val
					case "path":
						site.Path = val
					case "connection":
						lastConnection = val
						if _, ok := site.Queues[lastConnection]; !ok {
							site.Queues[lastConnection] = []string{}
						}
					case "queues":
						if lastConnection == "" {
							continue
						}
						qnames := strings.Split(val, "|")
						site.Queues[lastConnection] = append(site.Queues[lastConnection], qnames...)
					}
				}

				if site.Path == "" {
					return fmt.Errorf("missing path for Laravel site: %v", entry)
				}
				if site.Name == "" {
					site.Name = "App"
				}
				if seenNames[site.Name] {
					return fmt.Errorf("duplicate Laravel site name: %s", site.Name)
				}
				seenNames[site.Name] = true
				sites = append(sites, site)
			}
			loaded.Laravel = sites
		}

		// Handle log level (priority: flag > config > debug)
		if lvl, _ := cmd.Flags().GetString("log-level"); lvl != "" {
			loaded.Logging.Level = lvl
		} else if viper.GetBool("debug") || loaded.Debug {
			loaded.Logging.Level = "debug"
		}

		Config = loaded

		logging.Init(Config.Logging)
		logging.L().Debug("Logging initialized", "level", Config.Logging.Level)
		logging.L().Debug("Loaded config", "config", Config)

		// phpfpm autodiscover
		if Config.PHPFpm.Enabled && Config.PHPFpm.Autodiscover {
			discovered, err := phpfpm.DiscoverFPMProcesses()
			if err != nil {
				logging.L().Error("Autodiscover failed", "error", err)
			} else {
				logging.L().Debug("Discovered FPM Processes", "pools", discovered)
				for _, d := range discovered {
					Config.PHPFpm.Pools = append(Config.PHPFpm.Pools, config.FPMPoolConfig{
						Socket:     d.Socket,
						StatusPath: d.StatusPath,
						ConfigPath: d.ConfigPath,
						Binary:     d.Binary,
						CliBinary:  d.CliBinary,
					})
				}
			}
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Command execution failed:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("debug", false, "Debug mode")
	rootCmd.PersistentFlags().String("config", "", "config file path")
	rootCmd.PersistentFlags().String("log-level", "", "Override log level (e.g. debug, info, warn)")
	rootCmd.PersistentFlags().StringArrayVar(&laravelFlags, "laravel", nil, "Laravel site config: name=...,path=...")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))

	viper.SetEnvPrefix("ELASTICPHP")
	viper.AutomaticEnv()
}
