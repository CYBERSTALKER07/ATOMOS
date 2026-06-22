package plan

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var optimizerSourceTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "void_optimizer_source_total",
	Help: "Dispatch optimizer attribution counts by source (optimizer, fallback_phase1, fallback_validation_rejected).",
}, []string{"source"})

// RecordPrometheus increments the Prometheus counter matching source.
// nil-safe; complements SourceCounters for Cloud Monitoring alerts.
func RecordPrometheus(source string) {
	if source == "" {
		return
	}
	optimizerSourceTotal.WithLabelValues(source).Inc()
}
