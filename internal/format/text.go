package format

import (
	"fmt"
	"strings"

	"github.com/elasticphphq/agent/internal/metrics"
)

func MetricsToText(m *metrics.Metrics) string {
	var sb strings.Builder

	phpVersion := "unknown"
	if m.PHP != nil {
		phpVersion = m.PHP.Version
	}

	numPools := 0
	if m.Runtime != nil {
		numPools = len(m.Runtime.Pools)
	}

	sb.WriteString(fmt.Sprintf("📦 elasticphp-agent — %s │ FPM: %d pool(s)\n", phpVersion, numPools))
	sb.WriteString("──────────────────────────────────────────────────────\n")

	if m.Runtime != nil {
		for _, p := range m.Runtime.Pools {
			sb.WriteString(fmt.Sprintf("  %-8s active=%-3d idle=%-3d total=%-3d max=%-3d slow=%-3d\n",
				p.Name, p.ActiveProcesses, p.IdleProcesses, p.TotalProcesses, p.MaxChildrenReached, p.SlowRequests))
		}
	}

	sb.WriteString("──────────────────────────────────────────────────────\n")

	if m.PHP != nil && len(m.PHP.Extensions) > 0 {
		sb.WriteString(fmt.Sprintf("Extensions: %s\n", strings.Join(m.PHP.Extensions, ", ")))
	}

	if len(m.Errors) > 0 {
		sb.WriteString("Errors:\n")
		for k, v := range m.Errors {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	return sb.String()
}
