package laravel

import (
	"context"
	"github.com/elasticphphq/agent/internal/config"
)

type LaravelMetrics struct {
	Queues QueueSizes `json:"queues"`
	Info   AppInfo    `json:"app_info"`
}

// Collect gathers Laravel queue metrics for all configured sites.
func Collect(ctx context.Context, cfg *config.Config) (map[string]LaravelMetrics, map[string]string) {
	result := make(map[string]LaravelMetrics)
	errors := make(map[string]string)

	for _, site := range cfg.Laravel {
		php := cfg.PHP.Binary
		if site.PHPConfig != nil && site.PHPConfig.Binary != "" {
			php = site.PHPConfig.Binary
		}

		queues, err := GetQueueSizes(site.Path, php, site.Queues)
		if err != nil {
			errors["laravel:"+site.Name] = err.Error()
			continue
		}

		info, err := GetAppInfo(site.Path, php)
		if err != nil {
			errors["laravel:"+site.Name+":info"] = err.Error()
		}

		if info != nil {
			result[site.Name] = LaravelMetrics{
				Queues: queues,
				Info:   *info,
			}
		} else {
			result[site.Name] = LaravelMetrics{
				Queues: queues,
			}
		}
	}

	return result, errors
}
