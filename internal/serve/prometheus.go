package serve

import (
	"context"
	"encoding/json"
	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PrometheusCollector struct {
	cfg                     *config.Config
	upDesc                  *prometheus.Desc
	acceptedConnectionsDesc *prometheus.Desc
	startSinceDesc          *prometheus.Desc
	listenQueueDesc         *prometheus.Desc
	maxListenQueueDesc      *prometheus.Desc
	listenQueueLengthDesc   *prometheus.Desc
	idleProcessesDesc       *prometheus.Desc
	activeProcessesDesc     *prometheus.Desc
	totalProcessesDesc      *prometheus.Desc
	maxActiveProcessesDesc  *prometheus.Desc
	maxChildrenReachedDesc  *prometheus.Desc
	slowRequestsDesc        *prometheus.Desc
	processesCpuDesc        *prometheus.Desc
	processesMemoryDesc     *prometheus.Desc
	memoryPeakDesc          *prometheus.Desc

	// Opcache metrics
	opcacheEnabledDesc         *prometheus.Desc
	opcacheUsedMemoryDesc      *prometheus.Desc
	opcacheFreeMemoryDesc      *prometheus.Desc
	opcacheWastedMemoryDesc    *prometheus.Desc
	opcacheWastedPercentDesc   *prometheus.Desc
	opcacheCachedScriptsDesc   *prometheus.Desc
	opcacheHitsDesc            *prometheus.Desc
	opcacheMissesDesc          *prometheus.Desc
	opcacheBlacklistMissesDesc *prometheus.Desc
	opcacheOomRestartsDesc     *prometheus.Desc
	opcacheHashRestartsDesc    *prometheus.Desc
	opcacheManualRestartsDesc  *prometheus.Desc
	opcacheHitRateDesc         *prometheus.Desc

	// Pool config metrics
	// Maximum child processes, limits concurrency and memory use
	pmMaxChildrenConfigDesc *prometheus.Desc
	// Number of processes created on startup, affects cold start latency
	pmStartServersConfigDesc *prometheus.Desc
	// Minimum idle processes, for load spikes
	pmMinSpareServersConfigDesc *prometheus.Desc
	// Maximum idle processes, prevents resource waste
	pmMaxSpareServersConfigDesc *prometheus.Desc
	// Max requests per process before respawn, mitigates memory leaks
	pmMaxRequestsConfigDesc *prometheus.Desc
	// Max processes spawned per second, prevents fork bomb
	pmMaxSpawnRateConfigDesc *prometheus.Desc
	// Idle timeout for workers, helps tune recycling
	pmProcessIdleTimeoutConfigDesc *prometheus.Desc
	// Timeout for slowlog, helps find slow requests
	requestSlowlogTimeoutConfigDesc *prometheus.Desc
	// Max execution time for request
	requestTerminateTimeoutConfigDesc *prometheus.Desc
	// Core dump size limit
	rlimitCoreConfigDesc *prometheus.Desc
	// File descriptors limit per process
	rlimitFilesConfigDesc *prometheus.Desc

	// System metrics
	systemInfoDesc    *prometheus.Desc
	cpuLimitDesc      *prometheus.Desc
	memoryLimitMBDesc *prometheus.Desc

	// Laravel metrics
	laravelInfoDesc *prometheus.Desc
}

func NewPrometheusCollector(cfg *config.Config) *PrometheusCollector {
	labels := []string{"pool", "socket"}
	return &PrometheusCollector{
		cfg: cfg,
		// FPM Metrics
		upDesc:                  prometheus.NewDesc("phpfpm_up", "Shows whether scraping PHP-FPM's status was successful (1 for yes, 0 for no).", labels, nil),
		acceptedConnectionsDesc: prometheus.NewDesc("phpfpm_accepted_connections", "The number of accepted connections to the pool.", labels, nil),
		startSinceDesc:          prometheus.NewDesc("phpfpm_start_since", "Number of seconds since FPM has started.", labels, nil),
		listenQueueDesc:         prometheus.NewDesc("phpfpm_listen_queue", "The number of requests in the queue of pending connections.", labels, nil),
		maxListenQueueDesc:      prometheus.NewDesc("phpfpm_max_listen_queue", "The maximum number of requests in the queue of pending connections since FPM has started.", labels, nil),
		listenQueueLengthDesc:   prometheus.NewDesc("phpfpm_listen_queue_length", "The size of the socket queue of pending connections.", labels, nil),
		idleProcessesDesc:       prometheus.NewDesc("phpfpm_idle_processes", "The number of idle PHP-FPM processes.", labels, nil),
		activeProcessesDesc:     prometheus.NewDesc("phpfpm_active_processes", "The number of active PHP-FPM processes.", labels, nil),
		totalProcessesDesc:      prometheus.NewDesc("phpfpm_total_processes", "The number of total PHP-FPM processes.", labels, nil),
		maxActiveProcessesDesc:  prometheus.NewDesc("phpfpm_max_active_processes", "The maximum number of active PHP-FPM processes since FPM has started.", labels, nil),
		maxChildrenReachedDesc:  prometheus.NewDesc("phpfpm_max_children_reached", "Number of times the process limit has been reached, when pm.max_children is reached.", labels, nil),
		slowRequestsDesc:        prometheus.NewDesc("phpfpm_slow_requests", "The number of requests that exceeded request_slowlog_timeout.", labels, nil),
		processesCpuDesc:        prometheus.NewDesc("phpfpm_processes_cpu_avg", "Average CPU usage across all processes in the pool.", labels, nil),
		processesMemoryDesc:     prometheus.NewDesc("phpfpm_processes_memory_avg", "Average memory usage across all processes in the pool.", labels, nil),
		memoryPeakDesc:          prometheus.NewDesc("phpfpm_memory_peak", "Peak memory usage of the pool.", labels, nil),

		// Opcache metrics
		opcacheEnabledDesc:         prometheus.NewDesc("phpfpm_opcache_enabled", "Whether opcache is enabled.", labels, nil),
		opcacheUsedMemoryDesc:      prometheus.NewDesc("phpfpm_opcache_used_memory_bytes", "Amount of used opcache memory in bytes.", labels, nil),
		opcacheFreeMemoryDesc:      prometheus.NewDesc("phpfpm_opcache_free_memory_bytes", "Amount of free opcache memory in bytes.", labels, nil),
		opcacheWastedMemoryDesc:    prometheus.NewDesc("phpfpm_opcache_wasted_memory_bytes", "Amount of wasted opcache memory in bytes.", labels, nil),
		opcacheWastedPercentDesc:   prometheus.NewDesc("phpfpm_opcache_wasted_memory_percent", "Percentage of wasted opcache memory.", labels, nil),
		opcacheCachedScriptsDesc:   prometheus.NewDesc("phpfpm_opcache_cached_scripts", "Number of cached scripts in opcache.", labels, nil),
		opcacheHitsDesc:            prometheus.NewDesc("phpfpm_opcache_hits_total", "Total number of opcache hits.", labels, nil),
		opcacheMissesDesc:          prometheus.NewDesc("phpfpm_opcache_misses_total", "Total number of opcache misses.", labels, nil),
		opcacheBlacklistMissesDesc: prometheus.NewDesc("phpfpm_opcache_blacklist_misses_total", "Number of blacklist misses in opcache.", labels, nil),
		opcacheOomRestartsDesc:     prometheus.NewDesc("phpfpm_opcache_oom_restarts_total", "Number of out-of-memory restarts in opcache.", labels, nil),
		opcacheHashRestartsDesc:    prometheus.NewDesc("phpfpm_opcache_hash_restarts_total", "Number of hash restarts in opcache.", labels, nil),
		opcacheManualRestartsDesc:  prometheus.NewDesc("phpfpm_opcache_manual_restarts_total", "Number of manual restarts in opcache.", labels, nil),
		opcacheHitRateDesc:         prometheus.NewDesc("phpfpm_opcache_hit_rate", "Opcache hit rate.", labels, nil),

		// Pool config metrics
		pmMaxChildrenConfigDesc:           prometheus.NewDesc("phpfpm_pm_max_children_config", "PHP-FPM pool config: max children. Maximum child processes, limits concurrency and memory use.", labels, nil),
		pmStartServersConfigDesc:          prometheus.NewDesc("phpfpm_pm_start_servers_config", "PHP-FPM pool config: start servers. Number of processes created on startup, affects cold start latency.", labels, nil),
		pmMinSpareServersConfigDesc:       prometheus.NewDesc("phpfpm_pm_min_spare_servers_config", "PHP-FPM pool config: min spare servers. Minimum idle processes for load spikes.", labels, nil),
		pmMaxSpareServersConfigDesc:       prometheus.NewDesc("phpfpm_pm_max_spare_servers_config", "PHP-FPM pool config: max spare servers. Maximum idle processes, prevents resource waste.", labels, nil),
		pmMaxRequestsConfigDesc:           prometheus.NewDesc("phpfpm_pm_max_requests_config", "PHP-FPM pool config: max requests. Max requests per process before respawn, mitigates memory leaks.", labels, nil),
		pmMaxSpawnRateConfigDesc:          prometheus.NewDesc("phpfpm_pm_max_spawn_rate_config", "PHP-FPM pool config: max spawn rate. Max processes spawned per second, prevents fork bomb scenarios.", labels, nil),
		pmProcessIdleTimeoutConfigDesc:    prometheus.NewDesc("phpfpm_pm_process_idle_timeout_config", "PHP-FPM pool config: process idle timeout in seconds, helps tune process recycling.", labels, nil),
		requestSlowlogTimeoutConfigDesc:   prometheus.NewDesc("phpfpm_request_slowlog_timeout_config", "PHP-FPM pool config: slowlog timeout in seconds, helps identify slow requests.", labels, nil),
		requestTerminateTimeoutConfigDesc: prometheus.NewDesc("phpfpm_request_terminate_timeout_config", "PHP-FPM pool config: terminate timeout in seconds, max execution time for a single request.", labels, nil),
		rlimitCoreConfigDesc:              prometheus.NewDesc("phpfpm_rlimit_core_config", "PHP-FPM pool config: core dump size limit for processes.", labels, nil),
		rlimitFilesConfigDesc:             prometheus.NewDesc("phpfpm_rlimit_files_config", "PHP-FPM pool config: file descriptors limit per process.", labels, nil),

		// System metrics
		systemInfoDesc:    prometheus.NewDesc("system_info", "System information", []string{"type", "os", "arch"}, nil),
		cpuLimitDesc:      prometheus.NewDesc("system_cpu_limit", "Logical CPU limit", nil, nil),
		memoryLimitMBDesc: prometheus.NewDesc("system_memory_limit_mb", "Memory limit in MB", nil, nil),

		// Laravel Metrics
		laravelInfoDesc: prometheus.NewDesc("laravel_app_info", "Basic information about Laravel site", []string{"site", "version", "php_version", "environment", "debug_mode"}, nil),
	}
}

func (pc *PrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	// FPM Metrics
	ch <- pc.upDesc
	ch <- pc.acceptedConnectionsDesc
	ch <- pc.startSinceDesc
	ch <- pc.listenQueueDesc
	ch <- pc.maxListenQueueDesc
	ch <- pc.listenQueueLengthDesc
	ch <- pc.idleProcessesDesc
	ch <- pc.activeProcessesDesc
	ch <- pc.totalProcessesDesc
	ch <- pc.maxActiveProcessesDesc
	ch <- pc.maxChildrenReachedDesc
	ch <- pc.slowRequestsDesc
	ch <- pc.processesCpuDesc
	ch <- pc.processesMemoryDesc
	ch <- pc.memoryPeakDesc

	// Opcache metrics
	ch <- pc.opcacheEnabledDesc
	ch <- pc.opcacheUsedMemoryDesc
	ch <- pc.opcacheFreeMemoryDesc
	ch <- pc.opcacheWastedMemoryDesc
	ch <- pc.opcacheWastedPercentDesc
	ch <- pc.opcacheCachedScriptsDesc
	ch <- pc.opcacheHitsDesc
	ch <- pc.opcacheMissesDesc
	ch <- pc.opcacheBlacklistMissesDesc
	ch <- pc.opcacheOomRestartsDesc
	ch <- pc.opcacheHashRestartsDesc
	ch <- pc.opcacheManualRestartsDesc
	ch <- pc.opcacheHitRateDesc

	// FPM Config
	ch <- pc.pmMaxChildrenConfigDesc
	ch <- pc.pmStartServersConfigDesc
	ch <- pc.pmMinSpareServersConfigDesc
	ch <- pc.pmMaxSpareServersConfigDesc
	ch <- pc.pmMaxRequestsConfigDesc
	ch <- pc.pmMaxSpawnRateConfigDesc
	ch <- pc.pmProcessIdleTimeoutConfigDesc
	ch <- pc.requestSlowlogTimeoutConfigDesc
	ch <- pc.requestTerminateTimeoutConfigDesc
	ch <- pc.rlimitCoreConfigDesc
	ch <- pc.rlimitFilesConfigDesc

	// System metrics
	ch <- pc.systemInfoDesc
	ch <- pc.cpuLimitDesc
	ch <- pc.memoryLimitMBDesc

	ch <- pc.laravelInfoDesc
}

func parseConfigValue(val string) (float64, bool) {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "s") {
		val = strings.TrimSuffix(val, "s")
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func (pc *PrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m, err := metrics.GetMetrics(ctx, pc.cfg)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(pc.upDesc, prometheus.GaugeValue, 0, "unknown", "unknown")
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("phpfpm_scrape_failures", "The number of failures scraping from PHP-FPM.", nil, nil),
			prometheus.CounterValue, 1)
		return
	}

	if m.Server != nil {
		nodeType := string(m.Server.NodeType)
		ch <- prometheus.MustNewConstMetric(pc.systemInfoDesc, prometheus.GaugeValue, 1, nodeType, m.Server.OS, m.Server.Architecture)
		ch <- prometheus.MustNewConstMetric(pc.cpuLimitDesc, prometheus.GaugeValue, float64(m.Server.CPULimit))
		ch <- prometheus.MustNewConstMetric(pc.memoryLimitMBDesc, prometheus.GaugeValue, float64(m.Server.MemoryLimitMB))
	}

	for site, lm := range m.Laravel {
		if lm == nil {
			continue
		}

		info := lm

		if lm.Info != nil {

			debugMode := "false"
			if *info.Info.Environment.DebugMode {
				debugMode = "true"
			}

			ch <- prometheus.MustNewConstMetric(pc.laravelInfoDesc, prometheus.GaugeValue, 1,
				site,
				*info.Info.Environment.LaravelVersion,
				*info.Info.Environment.PHPVersion,
				*info.Info.Environment.Environment,
				debugMode,
			)

			if info.Info != nil {
				// Laravel cache status
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_cache_config", "Is config cache enabled", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(info.Info.Cache.Config)), site)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_cache_events", "Is events cache enabled", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(info.Info.Cache.Events)), site)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_cache_routes", "Is routes cache enabled", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(info.Info.Cache.Routes)), site)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_cache_views", "Is views cache enabled", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(info.Info.Cache.Views)), site)
				// Laravel environment
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_maintenance_mode", "Whether Laravel is in maintenance mode", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(*info.Info.Environment.MaintenanceMode)), site)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_debug_mode", "Whether Laravel debug mode is enabled", []string{"site"}, nil),
					prometheus.GaugeValue, boolToFloat(bool(*info.Info.Environment.DebugMode)), site)

				// Laravel drivers
				drivers := info.Info.Drivers
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "broadcasting", *drivers.Broadcasting)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "cache", *drivers.Cache)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "database", *drivers.Database)
				for _, logDrivers := range *drivers.Logs {
					ch <- prometheus.MustNewConstMetric(
						prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
						prometheus.GaugeValue, 1, site, "logs", logDrivers)
				}
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "mail", *drivers.Mail)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "queue", *drivers.Queue)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("laravel_driver_info", "Configured Laravel driver", []string{"site", "type", "value"}, nil),
					prometheus.GaugeValue, 1, site, "session", *drivers.Session)
			}

		}

		// Laravel queue sizes
		if info.Queues != nil {
			for conn, queues := range *info.Queues {
				for queue, qdata := range queues {
					if qdata.Size != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_size", "Number of jobs in queue", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Size), site, conn, queue)
					}

					if qdata.Pending != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_pending", "Number of pending jobs in queue", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Pending), site, conn, queue)
					}

					if qdata.Scheduled != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_scheduled", "Number of scheduled jobs in queue", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Scheduled), site, conn, queue)
					}

					if qdata.Reserved != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_reserved", "Number of jobs reserved by workers", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Reserved), site, conn, queue)
					}

					if qdata.OldestPending != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_oldest_pending", "The oldest pending job in queue in seconds", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.OldestPending), site, conn, queue)
					}

					if qdata.Failed != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed", "Number of failed jobs in queue", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Failed), site, conn, queue)
					}

					if qdata.Failed1Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_1m", "Number of failed jobs in queue last 1min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Failed1Min), site, conn, queue)
					}

					if qdata.Failed5Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_5m", "Number of failed jobs in queue last 5min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Failed5Min), site, conn, queue)
					}

					if qdata.Failed10Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_10m", "Number of failed jobs in queue last 10min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.Failed10Min), site, conn, queue)
					}

					if qdata.FailedRate1Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_rate_1m", "Number of failed jobs rate in queue last 1min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.FailedRate1Min), site, conn, queue)
					}

					if qdata.FailedRate5Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_rate_5m", "Number of failed jobs rate in queue last 5min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.FailedRate5Min), site, conn, queue)
					}

					if qdata.FailedRate10Min != nil {
						ch <- prometheus.MustNewConstMetric(
							prometheus.NewDesc("laravel_queue_failed_rate_10m", "Number of failed jobs rate in queue last 10min", []string{"site", "connection", "queue"}, nil),
							prometheus.GaugeValue, float64(*qdata.FailedRate10Min), site, conn, queue)
					}

				}
			}
		}
	}

	//if m.Fpm == nil {
	//	ch <- prometheus.MustNewConstMetric(pc.upDesc, prometheus.GaugeValue, 0, "unknown", "unknown")
	//	return
	//}
	//if len(m.Fpm) == 0 {
	//	ch <- prometheus.MustNewConstMetric(pc.upDesc, prometheus.GaugeValue, 0, "none", "none")
	//	return
	//}
	for socket, pools := range m.Fpm {
		if socket == "" {
			socket = "unknown"
		}

		for poolName, pool := range pools.Pools {
			up := 1.0

			ch <- prometheus.MustNewConstMetric(pc.upDesc, prometheus.GaugeValue, up, poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.acceptedConnectionsDesc, prometheus.CounterValue, float64(pool.AcceptedConnections), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.startSinceDesc, prometheus.GaugeValue, float64(pool.StartSince), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.listenQueueDesc, prometheus.GaugeValue, float64(pool.ListenQueue), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.maxListenQueueDesc, prometheus.GaugeValue, float64(pool.MaxListenQueue), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.listenQueueLengthDesc, prometheus.GaugeValue, float64(pool.ListenQueueLength), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.idleProcessesDesc, prometheus.GaugeValue, float64(pool.IdleProcesses), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.activeProcessesDesc, prometheus.GaugeValue, float64(pool.ActiveProcesses), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.totalProcessesDesc, prometheus.GaugeValue, float64(pool.TotalProcesses), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.maxActiveProcessesDesc, prometheus.GaugeValue, float64(pool.MaxActiveProcesses), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.maxChildrenReachedDesc, prometheus.CounterValue, float64(pool.MaxChildrenReached), poolName, socket)
			ch <- prometheus.MustNewConstMetric(pc.slowRequestsDesc, prometheus.CounterValue, float64(pool.SlowRequests), poolName, socket)

			// --- New pool metrics ---
			ch <- prometheus.MustNewConstMetric(pc.memoryPeakDesc, prometheus.GaugeValue, float64(pool.MemoryPeak), poolName, socket)
			if pool.ProcessesCpu != nil {
				ch <- prometheus.MustNewConstMetric(pc.processesCpuDesc, prometheus.GaugeValue, *pool.ProcessesCpu, poolName, socket)
			}
			if pool.ProcessesMemory != nil {
				ch <- prometheus.MustNewConstMetric(pc.processesMemoryDesc, prometheus.GaugeValue, *pool.ProcessesMemory, poolName, socket)
			}

			// --- Per-process metrics ---
			for _, proc := range pool.Processes {
				labels := []string{poolName, socket, strconv.Itoa(proc.PID)}

				// Process state as labeled metric (e.g. Idle, Running)
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("phpfpm_process_state", "The state of the process (Idle, Running, ...).", []string{"pool", "socket", "pid", "state"}, nil),
					prometheus.GaugeValue, 1, poolName, socket, strconv.Itoa(proc.PID), proc.State)

				// Process request count
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("phpfpm_process_requests", "The number of requests the process has served.", []string{"pool", "socket", "pid"}, nil),
					prometheus.CounterValue, float64(proc.Requests), labels...)

				// Last request duration
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("phpfpm_process_request_duration", "The duration in microseconds of the last request.", []string{"pool", "socket", "pid"}, nil),
					prometheus.GaugeValue, float64(proc.RequestDuration), labels...)

				// Last request memory
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("phpfpm_process_last_request_memory", "The max amount of memory the last request consumed.", []string{"pool", "socket", "pid"}, nil),
					prometheus.GaugeValue, float64(proc.LastRequestMemory), labels...)

				// Last request CPU
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("phpfpm_process_last_request_cpu", "The %cpu the last request consumed.", []string{"pool", "socket", "pid"}, nil),
					prometheus.GaugeValue, proc.LastRequestCPU, labels...)
			}

			// Opcache metrics
			ch <- prometheus.MustNewConstMetric(pc.opcacheEnabledDesc, prometheus.GaugeValue, boolToFloat(pool.OpcacheStatus.Enabled), poolName, socket)
			if pool.OpcacheStatus.Enabled {
				ch <- prometheus.MustNewConstMetric(pc.opcacheUsedMemoryDesc, prometheus.GaugeValue, float64(pool.OpcacheStatus.MemoryUsage.UsedMemory), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheFreeMemoryDesc, prometheus.GaugeValue, float64(pool.OpcacheStatus.MemoryUsage.FreeMemory), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheWastedMemoryDesc, prometheus.GaugeValue, float64(pool.OpcacheStatus.MemoryUsage.WastedMemory), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheWastedPercentDesc, prometheus.GaugeValue, pool.OpcacheStatus.MemoryUsage.CurrentWastedPct, poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheCachedScriptsDesc, prometheus.GaugeValue, float64(pool.OpcacheStatus.Statistics.NumCachedScripts), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheHitsDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.Hits), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheMissesDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.Misses), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheBlacklistMissesDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.BlacklistMisses), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheOomRestartsDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.OomRestarts), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheHashRestartsDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.HashRestarts), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheManualRestartsDesc, prometheus.CounterValue, float64(pool.OpcacheStatus.Statistics.ManualRestarts), poolName, socket)
				ch <- prometheus.MustNewConstMetric(pc.opcacheHitRateDesc, prometheus.GaugeValue, pool.OpcacheStatus.Statistics.HitRate, poolName, socket)
			}

			// Pool config metrics
			cfg := pool.Config

			if v, ok := parseConfigValue(cfg["pm.max_children"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmMaxChildrenConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.start_servers"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmStartServersConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.min_spare_servers"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmMinSpareServersConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.max_spare_servers"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmMaxSpareServersConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.max_requests"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmMaxRequestsConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.max_spawn_rate"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmMaxSpawnRateConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["pm.process_idle_timeout"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.pmProcessIdleTimeoutConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["request_slowlog_timeout"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.requestSlowlogTimeoutConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["request_terminate_timeout"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.requestTerminateTimeoutConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["rlimit_core"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.rlimitCoreConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
			if v, ok := parseConfigValue(cfg["rlimit_files"]); ok {
				ch <- prometheus.MustNewConstMetric(pc.rlimitFilesConfigDesc, prometheus.GaugeValue, v, poolName, socket)
			}
		}
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func StartPrometheusServer(cfg *config.Config) {
	mux := http.NewServeMux()

	registry := prometheus.NewRegistry()
	collector := NewPrometheusCollector(cfg)
	registry.MustRegister(collector)

	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	if cfg.Monitor.EnableJson {
		mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			m, err := metrics.GetMetrics(ctx, cfg)
			if err != nil {
				http.Error(w, "failed to get metrics: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(m)
		})
	}

	server := &http.Server{
		Addr:    cfg.Monitor.ListenAddr,
		Handler: mux,
	}

	log.Printf("Prometheus metrics server listening on %s", cfg.Monitor.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start Prometheus server: %v", err)
	}
}
