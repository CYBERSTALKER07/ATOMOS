package telemetry

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var OptimizerStatusCount = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "optimizer",
		Name:      "status_count",
		Help:      "Total optimizer solver responses by solver type and solver status.",
	},
	[]string{"solver_type", "status"},
)

var OptimizerLatencyMatrixRatio = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "void",
		Subsystem: "optimizer",
		Name:      "latency_matrix_ratio",
		Help:      "Optimizer solver latency normalized by matrix size.",
		Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
	},
	[]string{"solver_type"},
)

func RecordSolverOutcome(solverType, status string, elapsed time.Duration, matrixSize int32) {
	if solverType == "" {
		solverType = "UNKNOWN"
	}
	if status == "" {
		status = "UNKNOWN"
	}

	OptimizerStatusCount.WithLabelValues(solverType, status).Inc()

	divisor := float64(matrixSize)
	if divisor < 1 {
		divisor = 1
	}
	OptimizerLatencyMatrixRatio.WithLabelValues(solverType).Observe(elapsed.Seconds() / divisor)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
