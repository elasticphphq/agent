package cmd

import (
	"context"
	"github.com/elasticphphq/agent/internal/logging"
	"github.com/elasticphphq/agent/internal/metrics"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var once bool

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Collect and persist runtime metrics",
	Run: func(cmd *cobra.Command, args []string) {
		if !Config.PHPFpm.Enabled {
			logging.L().Error("PHP-FPM not enabled")
			os.Exit(1)
		}

		logging.L().Debug("Monitoring php-fpm", "interval", Config.PHPFpm.PollInterval)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		collector := metrics.NewCollector(Config, Config.PHPFpm.PollInterval)
		collector.RunPerPoolCollector(ctx)

		if once {
			result, err := collector.Collect(ctx)
			if err != nil {
				logging.L().Error("Collection failed", "error", err)
				os.Exit(1)
			}
			logging.L().Info("Collected metrics", "timestamp", result.Timestamp, "metrics", result)
			return
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		logging.L().Info("Shutting down monitor...")
	},
}

func init() {
	monitorCmd.Flags().BoolVar(&once, "once", false, "collect metrics once")
	rootCmd.AddCommand(monitorCmd)
}
