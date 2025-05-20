package metrics

import (
	"context"
	"github.com/elasticphphq/agent/internal/laravel"
	"github.com/elasticphphq/agent/internal/server"
	"sync"
	"time"

	"github.com/elasticphphq/agent/internal/config"
	"github.com/elasticphphq/agent/internal/phpfpm"
)

type Listener func(*Metrics)

type Collector struct {
	cfg       *config.Config
	interval  time.Duration
	listeners []Listener
	mu        sync.Mutex
	results   map[string]*phpfpm.Result
}

func NewCollector(cfg *config.Config, interval time.Duration) *Collector {
	return &Collector{
		cfg:       cfg,
		interval:  interval,
		listeners: make([]Listener, 0),
		results:   make(map[string]*phpfpm.Result),
	}
}

func (c *Collector) AddListener(fn Listener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, fn)
}

func (c *Collector) notify(m *Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, fn := range c.listeners {
		fn(m)
	}
}

func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m, _ := GetMetrics(ctx, c.cfg)
			c.notify(m)
		}
	}
}

func (c *Collector) Collect(ctx context.Context) (*Metrics, error) {
	return GetMetrics(ctx, c.cfg)
}

func (c *Collector) RunPerPoolCollector(ctx context.Context) {
	for _, pool := range c.cfg.PHPFpm.Pools {
		go func(poolCfg config.FPMPoolConfig) {
			interval := poolCfg.PollInterval
			if interval == 0 {
				interval = c.cfg.PHPFpm.PollInterval
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					timeout := poolCfg.Timeout
					if timeout == 0 {
						timeout = 2 * time.Second
					}

					poolCtx, cancel := context.WithTimeout(ctx, timeout)
					result, err := phpfpm.GetMetricsForPool(poolCtx, poolCfg)
					cancel()

					c.mu.Lock()
					if err == nil {
						c.results[poolCfg.Socket] = result
					} else {
						c.results[poolCfg.Socket] = &phpfpm.Result{
							Timestamp: time.Now(),
							Pools:     nil,
							Global:    nil,
						}
					}
					c.mu.Unlock()
				}
			}
		}(pool)
	}
}

func GetMetrics(ctx context.Context, cfg *config.Config) (*Metrics, error) {
	out := &Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	systemInfoData := server.DetectSystem()
	out.Server = systemInfoData.SystemInfo
	for k, v := range systemInfoData.Errors {
		out.Errors[k] = v
	}

	if cfg.PHPFpm.Enabled {
		fpmResults, err := phpfpm.GetMetrics(ctx, cfg)
		if err != nil {
			out.Errors["fpm"] = err.Error()
		} else {
			out.Fpm = fpmResults
		}
	}

	if len(cfg.Laravel) > 0 {
		data, errs := laravel.Collect(ctx, cfg)
		for key, msg := range errs {
			out.Errors[key] = msg
		}
		out.Laravel = make(map[string]*laravel.LaravelMetrics)
		for name, metrics := range data {
			m := metrics // capture loop variable
			out.Laravel[name] = &m
		}
	}

	return out, nil
}
