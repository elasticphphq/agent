package metrics

import (
	"github.com/elasticphphq/agent/internal/laravel"
	"github.com/elasticphphq/agent/internal/phpfpm"
	"github.com/elasticphphq/agent/internal/server"
	"time"
)

type Metrics struct {
	Timestamp time.Time
	Server    *server.SystemInfo
	Fpm       map[string]*phpfpm.Result
	Laravel   map[string]*laravel.LaravelMetrics `json:"laravel,omitempty"`
	Errors    map[string]string
}
