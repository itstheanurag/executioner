package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "executioner_executions_total",
			Help: "Total number of code executions",
		},
		[]string{"language", "status"},
	)

	ExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "executioner_execution_duration_ms",
			Help:    "Execution duration in milliseconds",
			Buckets: []float64{50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"language", "phase"}, // phase: "compile", "run", "total"
	)

	QueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "executioner_queue_depth",
			Help: "Current number of jobs in the queue",
		},
	)

	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "executioner_active_workers",
			Help: "Number of workers currently processing jobs",
		},
	)

	MemoryUsage = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "executioner_memory_usage_kb",
			Help:    "Peak memory usage per execution in KB",
			Buckets: []float64{1024, 4096, 16384, 65536, 131072, 262144},
		},
		[]string{"language"},
	)

	ContainerCreationTime = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "executioner_container_creation_ms",
			Help:    "Time to create and start a container",
			Buckets: []float64{50, 100, 200, 500, 1000, 2000},
		},
	)

	RateLimitHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "executioner_rate_limit_hits_total",
			Help: "Total number of requests rejected by rate limiter",
		},
	)
)
