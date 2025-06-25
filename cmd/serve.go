package cmd

import (
	"github.com/elasticphphq/agent/internal/logging"

	"github.com/elasticphphq/agent/internal/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start agent HTTP server with metrics and control endpoints",
	Run: func(cmd *cobra.Command, args []string) {
		logging.L().Info("ElasticPHP-agent Starting")
		serve.StartPrometheusServer(Config)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
