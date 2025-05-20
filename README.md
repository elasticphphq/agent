# ElasticPHP Agent

ElasticPHP Agent is a lightweight, Go-based monitoring agent for PHP-FPM and Laravel applications.  
It is designed to run locally, in Docker/Kubernetes, as a sidecar, or in VMs or shared hosting environments.

## Features

- 📊 Exposes PHP-FPM metrics via FastCGI (using [fcgx](https://github.com/elasticphphq/fcgx))
- ⚙️ Automatically discovers PHP-FPM pools and extracts config using `php-fpm -tt`
- 🧠 Collects and exposes detailed Opcache statistics per FPM pool
- 🚦 Tracks Laravel queue sizes via `php artisan queue:size`
- 🧠 Provides Laravel application info (`about --json`)
- 🔌 Prometheus metrics endpoint at `/metrics`, and full JSON snapshot available at `/json`
- ⚙️ Structured configuration via CLI flags, environment variables, or config files (YAML)
- 🐘 Multi-site support for Laravel applications

---

## Quickstart

```bash
# Build the binary
make build

# Run once with debugging
./elasticphp-agent monitor --once --debug \
  --laravel "name=App,path=/var/www/html,connection=redis,queues=default|emails"
```

---

## Configuration

### CLI flags

```bash
./elasticphp-agent monitor \
  --laravel "name=Site1,path=/var/www/site1,connection=redis,queues=default|emails" \
  --laravel "name=Site2,path=/var/www/site2"
```

### Environment variables

| ENV                         | Description                             |
|----------------------------|-----------------------------------------|
| ELASTICPHP_DEBUG           | Enable debug mode                       |
| ELASTICPHP_MONITOR_LISTEN  | Prometheus listen address (e.g. :9114)  |
| ELASTICPHP_PHP_BINARY      | Path to default PHP binary              |
| ELASTICPHP_PHPFPM_ENABLED  | Enable PHP-FPM monitoring (default: true) |

### YAML config

Example `config.yaml`:

```yaml
debug: true
monitor:
  listen_addr: ":9114"
php:
  enabled: true
  binary: /usr/bin/php
phpfpm:
  enabled: true
  autodiscover: true
  poll_interval: 1s
laravel:
  - name: App
    path: /var/www/html
    queues:
      redis: ["default", "emails"]
      database: ["urgent", "slow"]
```

---

## Prometheus Metrics

This agent exposes metrics for:

- Laravel app info, cache and driver state
- Laravel queue size per connection/queue
- PHP-FPM process stats and pool configuration
- Prometheus metrics endpoint at `/metrics`
- Host system info

See full example below:

```text
# HELP laravel_app_info Basic information about Laravel site
# TYPE laravel_app_info gauge
laravel_app_info{debug_mode="true",env="local",php_version="8.4.7",site="App",version="11.44.7"} 1
# HELP laravel_cache_config Is config cache enabled
# TYPE laravel_cache_config gauge
laravel_cache_config{site="App"} 0
# HELP laravel_cache_events Is events cache enabled
# TYPE laravel_cache_events gauge
laravel_cache_events{site="App"} 0
# HELP laravel_cache_routes Is routes cache enabled
# TYPE laravel_cache_routes gauge
laravel_cache_routes{site="App"} 0
# HELP laravel_cache_views Is views cache enabled
# TYPE laravel_cache_views gauge
laravel_cache_views{site="App"} 1
# HELP laravel_debug_mode Whether Laravel debug mode is enabled
# TYPE laravel_debug_mode gauge
laravel_debug_mode{site="App"} 1
# HELP laravel_driver_info Configured Laravel driver
# TYPE laravel_driver_info gauge
laravel_driver_info{site="App",type="broadcasting",value="log"} 1
laravel_driver_info{site="App",type="cache",value="database"} 1
laravel_driver_info{site="App",type="database",value="mysql"} 1
laravel_driver_info{site="App",type="logs",value="single"} 1
laravel_driver_info{site="App",type="mail",value="smtp"} 1
laravel_driver_info{site="App",type="queue",value="database"} 1
laravel_driver_info{site="App",type="session",value="database"} 1
# HELP laravel_maintenance_mode Whether Laravel is in maintenance mode
# TYPE laravel_maintenance_mode gauge
laravel_maintenance_mode{site="App"} 0
# HELP laravel_queue_size Number of jobs in queue
# TYPE laravel_queue_size gauge
laravel_queue_size{connection="database",queue="slow",site="App"} 0
laravel_queue_size{connection="database",queue="urgent",site="App"} 0
laravel_queue_size{connection="redis",queue="default",site="App"} 0
laravel_queue_size{connection="redis",queue="emails",site="App"} 0
# HELP phpfpm_accepted_connections The number of accepted connections to the pool.
# TYPE phpfpm_accepted_connections counter
phpfpm_accepted_connections{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 155
phpfpm_accepted_connections{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 155
phpfpm_accepted_connections{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 155
# HELP phpfpm_active_processes The number of active PHP-FPM processes.
# TYPE phpfpm_active_processes gauge
phpfpm_active_processes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 2
phpfpm_active_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_active_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 2
# HELP phpfpm_idle_processes The number of idle PHP-FPM processes.
# TYPE phpfpm_idle_processes gauge
phpfpm_idle_processes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 1
phpfpm_idle_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_idle_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 1
# HELP phpfpm_listen_queue The number of requests in the queue of pending connections.
# TYPE phpfpm_listen_queue gauge
phpfpm_listen_queue{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_listen_queue{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_listen_queue{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_listen_queue_length The size of the socket queue of pending connections.
# TYPE phpfpm_listen_queue_length gauge
phpfpm_listen_queue_length{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_listen_queue_length{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_listen_queue_length{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_max_active_processes The maximum number of active PHP-FPM processes since FPM has started.
# TYPE phpfpm_max_active_processes gauge
phpfpm_max_active_processes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 3
phpfpm_max_active_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2
phpfpm_max_active_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 3
# HELP phpfpm_max_children_reached Number of times the process limit has been reached, when pm.max_children is reached.
# TYPE phpfpm_max_children_reached counter
phpfpm_max_children_reached{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_max_children_reached{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_max_children_reached{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_max_listen_queue The maximum number of requests in the queue of pending connections since FPM has started.
# TYPE phpfpm_max_listen_queue gauge
phpfpm_max_listen_queue{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_max_listen_queue{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_max_listen_queue{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_memory_peak Peak memory usage of the pool.
# TYPE phpfpm_memory_peak gauge
phpfpm_memory_peak{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 2.097152e+06
phpfpm_memory_peak{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2.097152e+06
phpfpm_memory_peak{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 2.097152e+06
# HELP phpfpm_opcache_blacklist_misses_total Number of blacklist misses in opcache.
# TYPE phpfpm_opcache_blacklist_misses_total counter
phpfpm_opcache_blacklist_misses_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_blacklist_misses_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_blacklist_misses_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_opcache_cached_scripts Number of cached scripts in opcache.
# TYPE phpfpm_opcache_cached_scripts gauge
phpfpm_opcache_cached_scripts{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 16
phpfpm_opcache_cached_scripts{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 17
phpfpm_opcache_cached_scripts{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 16
# HELP phpfpm_opcache_enabled Whether opcache is enabled.
# TYPE phpfpm_opcache_enabled gauge
phpfpm_opcache_enabled{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 1
phpfpm_opcache_enabled{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_opcache_enabled{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 1
# HELP phpfpm_opcache_free_memory_bytes Amount of free opcache memory in bytes.
# TYPE phpfpm_opcache_free_memory_bytes gauge
phpfpm_opcache_free_memory_bytes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 1.24857512e+08
phpfpm_opcache_free_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1.247834e+08
phpfpm_opcache_free_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 1.24857512e+08
# HELP phpfpm_opcache_hash_restarts_total Number of hash restarts in opcache.
# TYPE phpfpm_opcache_hash_restarts_total counter
phpfpm_opcache_hash_restarts_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_hash_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_hash_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_opcache_hit_rate Opcache hit rate.
# TYPE phpfpm_opcache_hit_rate gauge
phpfpm_opcache_hit_rate{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 93.51432880844645
phpfpm_opcache_hit_rate{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 92.89533995416348
phpfpm_opcache_hit_rate{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 93.51043643263756
# HELP phpfpm_opcache_hits_total Total number of opcache hits.
# TYPE phpfpm_opcache_hits_total counter
phpfpm_opcache_hits_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 2480
phpfpm_opcache_hits_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1216
phpfpm_opcache_hits_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 2464
# HELP phpfpm_opcache_manual_restarts_total Number of manual restarts in opcache.
# TYPE phpfpm_opcache_manual_restarts_total counter
phpfpm_opcache_manual_restarts_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_manual_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_manual_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_opcache_misses_total Total number of opcache misses.
# TYPE phpfpm_opcache_misses_total counter
phpfpm_opcache_misses_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 172
phpfpm_opcache_misses_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 93
phpfpm_opcache_misses_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 171
# HELP phpfpm_opcache_oom_restarts_total Number of out-of-memory restarts in opcache.
# TYPE phpfpm_opcache_oom_restarts_total counter
phpfpm_opcache_oom_restarts_total{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_oom_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_oom_restarts_total{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_opcache_used_memory_bytes Amount of used opcache memory in bytes.
# TYPE phpfpm_opcache_used_memory_bytes gauge
phpfpm_opcache_used_memory_bytes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 9.360216e+06
phpfpm_opcache_used_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 9.434328e+06
phpfpm_opcache_used_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 9.360216e+06
# HELP phpfpm_opcache_wasted_memory_bytes Amount of wasted opcache memory in bytes.
# TYPE phpfpm_opcache_wasted_memory_bytes gauge
phpfpm_opcache_wasted_memory_bytes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_wasted_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_wasted_memory_bytes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_opcache_wasted_memory_percent Percentage of wasted opcache memory.
# TYPE phpfpm_opcache_wasted_memory_percent gauge
phpfpm_opcache_wasted_memory_percent{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_opcache_wasted_memory_percent{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_opcache_wasted_memory_percent{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_pm_max_children_config PHP-FPM pool config: max children. Maximum child processes, limits concurrency and memory use.
# TYPE phpfpm_pm_max_children_config gauge
phpfpm_pm_max_children_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 10
phpfpm_pm_max_children_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2
phpfpm_pm_max_children_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 10
# HELP phpfpm_pm_max_requests_config PHP-FPM pool config: max requests. Max requests per process before respawn, mitigates memory leaks.
# TYPE phpfpm_pm_max_requests_config gauge
phpfpm_pm_max_requests_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_pm_max_requests_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_pm_max_requests_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_pm_max_spare_servers_config PHP-FPM pool config: max spare servers. Maximum idle processes, prevents resource waste.
# TYPE phpfpm_pm_max_spare_servers_config gauge
phpfpm_pm_max_spare_servers_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 4
phpfpm_pm_max_spare_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2
phpfpm_pm_max_spare_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 4
# HELP phpfpm_pm_max_spawn_rate_config PHP-FPM pool config: max spawn rate. Max processes spawned per second, prevents fork bomb scenarios.
# TYPE phpfpm_pm_max_spawn_rate_config gauge
phpfpm_pm_max_spawn_rate_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 32
phpfpm_pm_max_spawn_rate_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 32
phpfpm_pm_max_spawn_rate_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 32
# HELP phpfpm_pm_min_spare_servers_config PHP-FPM pool config: min spare servers. Minimum idle processes for load spikes.
# TYPE phpfpm_pm_min_spare_servers_config gauge
phpfpm_pm_min_spare_servers_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 1
phpfpm_pm_min_spare_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_pm_min_spare_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 1
# HELP phpfpm_pm_process_idle_timeout_config PHP-FPM pool config: process idle timeout in seconds, helps tune process recycling.
# TYPE phpfpm_pm_process_idle_timeout_config gauge
phpfpm_pm_process_idle_timeout_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 10
phpfpm_pm_process_idle_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 10
phpfpm_pm_process_idle_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 10
# HELP phpfpm_pm_start_servers_config PHP-FPM pool config: start servers. Number of processes created on startup, affects cold start latency.
# TYPE phpfpm_pm_start_servers_config gauge
phpfpm_pm_start_servers_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 3
phpfpm_pm_start_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_pm_start_servers_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 3
# HELP phpfpm_process_current_rss Resident set size (RSS) of the current process.
# TYPE phpfpm_process_current_rss gauge
phpfpm_process_current_rss{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_process_current_rss{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_process_current_rss{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_current_rss{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_current_rss{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_current_rss{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_current_rss{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_current_rss{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
# HELP phpfpm_process_last_request_cpu The %cpu the last request consumed.
# TYPE phpfpm_process_last_request_cpu gauge
phpfpm_process_last_request_cpu{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_process_last_request_cpu{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_process_last_request_cpu{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_last_request_cpu{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 110.94
phpfpm_process_last_request_cpu{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_last_request_cpu{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_last_request_cpu{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_last_request_cpu{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 68.76
# HELP phpfpm_process_last_request_memory The max amount of memory the last request consumed.
# TYPE phpfpm_process_last_request_memory gauge
phpfpm_process_last_request_memory{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_process_last_request_memory{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2.097152e+06
phpfpm_process_last_request_memory{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_last_request_memory{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 2.097152e+06
phpfpm_process_last_request_memory{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
phpfpm_process_last_request_memory{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_last_request_memory{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_process_last_request_memory{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 2.097152e+06
# HELP phpfpm_process_request_duration The duration in microseconds of the last request.
# TYPE phpfpm_process_request_duration gauge
phpfpm_process_request_duration{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 642
phpfpm_process_request_duration{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1520
phpfpm_process_request_duration{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 254575
phpfpm_process_request_duration{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 18027
phpfpm_process_request_duration{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 813
phpfpm_process_request_duration{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 775
phpfpm_process_request_duration{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 233489
phpfpm_process_request_duration{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 14543
# HELP phpfpm_process_requests The number of requests the process has served.
# TYPE phpfpm_process_requests counter
phpfpm_process_requests{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 77
phpfpm_process_requests{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 78
phpfpm_process_requests{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 41
phpfpm_process_requests{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 75
phpfpm_process_requests{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 39
phpfpm_process_requests{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 42
phpfpm_process_requests{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 55
phpfpm_process_requests{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 58
# HELP phpfpm_process_state The state of the process (Idle, Running, ...).
# TYPE phpfpm_process_state gauge
phpfpm_process_state{pid="19125",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock",state="Running"} 1
phpfpm_process_state{pid="3552",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock",state="Idle"} 1
phpfpm_process_state{pid="3553",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock",state="Finishing"} 1
phpfpm_process_state{pid="3554",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock",state="Idle"} 1
phpfpm_process_state{pid="3555",pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock",state="Running"} 1
phpfpm_process_state{pid="3556",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock",state="Running"} 1
phpfpm_process_state{pid="3557",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock",state="Finishing"} 1
phpfpm_process_state{pid="3558",pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock",state="Idle"} 1
# HELP phpfpm_request_slowlog_timeout_config PHP-FPM pool config: slowlog timeout in seconds, helps identify slow requests.
# TYPE phpfpm_request_slowlog_timeout_config gauge
phpfpm_request_slowlog_timeout_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_request_slowlog_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_request_slowlog_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_request_terminate_timeout_config PHP-FPM pool config: terminate timeout in seconds, max execution time for a single request.
# TYPE phpfpm_request_terminate_timeout_config gauge
phpfpm_request_terminate_timeout_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_request_terminate_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_request_terminate_timeout_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_rlimit_core_config PHP-FPM pool config: core dump size limit for processes.
# TYPE phpfpm_rlimit_core_config gauge
phpfpm_rlimit_core_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_rlimit_core_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_rlimit_core_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_rlimit_files_config PHP-FPM pool config: file descriptors limit per process.
# TYPE phpfpm_rlimit_files_config gauge
phpfpm_rlimit_files_config{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_rlimit_files_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_rlimit_files_config{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_slow_requests The number of requests that exceeded request_slowlog_timeout.
# TYPE phpfpm_slow_requests counter
phpfpm_slow_requests{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 0
phpfpm_slow_requests{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 0
phpfpm_slow_requests{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 0
# HELP phpfpm_start_since Number of seconds since FPM has started.
# TYPE phpfpm_start_since gauge
phpfpm_start_since{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 36827
phpfpm_start_since{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 36827
phpfpm_start_since{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 36827
# HELP phpfpm_total_processes The number of total PHP-FPM processes.
# TYPE phpfpm_total_processes gauge
phpfpm_total_processes{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 3
phpfpm_total_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 2
phpfpm_total_processes{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 3
# HELP phpfpm_up Shows whether scraping PHP-FPM's status was successful (1 for yes, 0 for no).
# TYPE phpfpm_up gauge
phpfpm_up{pool="elasticphp",socket="unix:///Users/syda/Library/Application Support/Herd/herd84-elasticphp.sock"} 1
phpfpm_up{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd-debug84.sock"} 1
phpfpm_up{pool="herd",socket="unix:///Users/syda/Library/Application Support/Herd/herd84.sock"} 1
# HELP system_cpu_limit Logical CPU limit
# TYPE system_cpu_limit gauge
system_cpu_limit 12
# HELP system_info System information
# TYPE system_info gauge
system_info{arch="arm64",os="darwin",type="physical"} 1
# HELP system_memory_limit_mb Memory limit in MB
# TYPE system_memory_limit_mb gauge
system_memory_limit_mb 36864
```

---

## Development

```bash
# Run tests
make test

# Run linter
make lint
```

---

## License

MIT License — © 2024–2025 [ElasticPHP HQ](https://github.com/elasticphphq)