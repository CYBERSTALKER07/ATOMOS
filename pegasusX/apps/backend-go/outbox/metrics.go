package outbox

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var unpublishedCount = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "void_outbox_unpublished_count",
	Help: "Number of outbox events awaiting Kafka publish",
})

// SetUnpublishedCount updates the backlog gauge (called by relay watchdog).
func SetUnpublishedCount(n int64) {
	unpublishedCount.Set(float64(n))
}

var deadLetteredTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "void_outbox_dead_lettered_total",
	Help: "Outbox events moved to the dead-letter sink after exhausting publish attempts",
})

// IncDeadLettered counts events moved to the dead-letter sink.
func IncDeadLettered(n int) {
	if n > 0 {
		deadLetteredTotal.Add(float64(n))
	}
}
