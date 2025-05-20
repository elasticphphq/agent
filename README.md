# ElasticPHP Agent

ElasticPHP Agent is a lightweight, Go-based monitoring agent for PHP-FPM and Laravel applications.  
It is designed to run locally, in Docker/Kubernetes, as a sidecar, or in VMs or shared hosting environments.

## Features

- 📊 Exposes PHP-FPM metrics via FastCGI (using [fcgx](https://github.com/elasticphphq/fcgx))
- ⚙️ Automatically discovers PHP-FPM pools and extracts config using `php-fpm -tt`
- 🚦 Tracks Laravel queue sizes via `php artisan queue:size`
- 🧠 Provides Laravel application info (`about --json`)
- 🔌 Prometheus metrics endpoint at `/metrics`
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
- Host system info

See full example below:

```text
# HELP laravel_app_info Basic information about Laravel site
# TYPE laravel_app_info gauge
laravel_app_info{debug_mode="false",env="production",php_version="8.3.14",site="App",version="11.41.3"} 1

# HELP laravel_cache_config Is config cache enabled
# TYPE laravel_cache_config gauge
laravel_cache_config{site="App"} 0
laravel_cache_events{site="App"} 0
laravel_cache_routes{site="App"} 0
laravel_cache_views{site="App"} 0

# HELP laravel_debug_mode Whether Laravel debug mode is enabled
# TYPE laravel_debug_mode gauge
laravel_debug_mode{site="App"} 0

# HELP laravel_driver_info Configured Laravel driver
# TYPE laravel_driver_info gauge
laravel_driver_info{site="App",type="broadcasting",value="null"} 1
laravel_driver_info{site="App",type="cache",value="database"} 1
laravel_driver_info{site="App",type="database",value="sqlite"} 1
laravel_driver_info{site="App",type="logs",value="laravel-cloud-socket"} 1
laravel_driver_info{site="App",type="mail",value="log"} 1
laravel_driver_info{site="App",type="queue",value="database"} 1
laravel_driver_info{site="App",type="session",value="cookie"} 1

# HELP laravel_maintenance_mode Whether Laravel is in maintenance mode
# TYPE laravel_maintenance_mode gauge
laravel_maintenance_mode{site="App"} 0

# HELP phpfpm_active_processes The number of active PHP-FPM processes.
# TYPE phpfpm_active_processes gauge
phpfpm_active_processes{pool="www",socket="tcp://127.0.0.1:9000"} 1
# HELP phpfpm_idle_processes The number of idle PHP-FPM processes.
# TYPE phpfpm_idle_processes gauge
phpfpm_idle_processes{pool="www",socket="tcp://127.0.0.1:9000"} 1
# HELP phpfpm_accepted_connections Total accepted connections.
# TYPE phpfpm_accepted_connections counter
phpfpm_accepted_connections{pool="www",socket="tcp://127.0.0.1:9000"} 437
# HELP phpfpm_listen_queue Number of requests in queue.
# TYPE phpfpm_listen_queue gauge
phpfpm_listen_queue{pool="www",socket="tcp://127.0.0.1:9000"} 0
# HELP phpfpm_max_children_reached Process limit reached count.
# TYPE phpfpm_max_children_reached counter
phpfpm_max_children_reached{pool="www",socket="tcp://127.0.0.1:9000"} 0
# HELP phpfpm_total_processes Total number of FPM processes.
# TYPE phpfpm_total_processes gauge
phpfpm_total_processes{pool="www",socket="tcp://127.0.0.1:9000"} 2
# HELP phpfpm_up Indicates if scrape succeeded.
# TYPE phpfpm_up gauge
phpfpm_up{pool="www",socket="tcp://127.0.0.1:9000"} 1

# HELP phpfpm_pm_max_children_config Pool config: max children
# TYPE phpfpm_pm_max_children_config gauge
phpfpm_pm_max_children_config{pool="www",socket="tcp://127.0.0.1:9000"} 17
# HELP phpfpm_pm_start_servers_config Pool config: start servers
# TYPE phpfpm_pm_start_servers_config gauge
phpfpm_pm_start_servers_config{pool="www",socket="tcp://127.0.0.1:9000"} 2
# HELP phpfpm_pm_process_idle_timeout_config Pool config: idle timeout
# TYPE phpfpm_pm_process_idle_timeout_config gauge
phpfpm_pm_process_idle_timeout_config{pool="www",socket="tcp://127.0.0.1:9000"} 10

# HELP system_cpu_limit Logical CPU limit
# TYPE system_cpu_limit gauge
system_cpu_limit 1

# HELP system_info System information
# TYPE system_info gauge
system_info{arch="arm64",os="linux",type="kubernetes"} 1

# HELP system_memory_limit_mb Memory limit in MB
# TYPE system_memory_limit_mb gauge
system_memory_limit_mb 512
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