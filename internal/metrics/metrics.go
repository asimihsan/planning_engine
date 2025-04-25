package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// FactCollectLatency tracks the latency of fact collection operations
	FactCollectLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "planning_engine",
			Subsystem: "fact_provider",
			Name:      "collect_latency_seconds",
			Help:      "Time spent in FactProvider.Collect()",
		},
		[]string{"provider"},
	)

	// FactCollectErrors tracks fact collection errors
	FactCollectErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "planning_engine",
			Subsystem: "fact_provider",
			Name:      "collect_errors_total",
			Help:      "Number of fact collection errors",
		},
		[]string{"provider", "error_type"},
	)

	// FactStaleness tracks facts that were marked as stale
	FactStaleness = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "planning_engine",
			Subsystem: "fact_provider",
			Name:      "stale_facts_total",
			Help:      "Number of facts that were marked as stale",
		},
		[]string{"provider"},
	)
)

// MustRegister registers all metrics with the default Prometheus registry
func MustRegister() {
	prometheus.MustRegister(
		FactCollectLatency,
		FactCollectErrors,
		FactStaleness,
	)
}
