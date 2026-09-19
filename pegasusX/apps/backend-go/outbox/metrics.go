package outbox

import (
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	lagMu        sync.RWMutex
	recentLags   []float64
	maxLagSample = 1000
)

// RecordPublishLag records the publish lag (in seconds) for a published event.
func RecordPublishLag(lagSeconds float64) {
	if lagSeconds < 0 {
		lagSeconds = 0
	}
	lagMu.Lock()
	defer lagMu.Unlock()
	recentLags = append(recentLags, lagSeconds)
	if len(recentLags) > maxLagSample {
		recentLags = recentLags[len(recentLags)-maxLagSample:]
	}
}

// GetOutboxLagP99Seconds calculates the p99 lag in seconds from in-memory samples.
func GetOutboxLagP99Seconds() float64 {
	lagMu.RLock()
	defer lagMu.RUnlock()
	if len(recentLags) == 0 {
		return 0
	}
	sorted := append([]float64(nil), recentLags...)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * 0.99)
	return sorted[idx]
}

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

var relayRestartsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "void_outbox_relay_restarts_total",
	Help: "Outbox relay Start() invocations (process / loop restarts; SLO < 1/hour)",
})

// IncRelayRestart increments when the outbox relay loop starts.
func IncRelayRestart() {
	relayRestartsTotal.Inc()
}

var stuckEventsDetected = promauto.NewCounter(prometheus.CounterOpts{
	Name: "void_outbox_relay_stuck_events_total",
	Help: "Times the outbox watchdog observed stuck unpublished events",
})

// IncStuckEventsDetected increments when watchdog finds stuck outbox rows.
func IncStuckEventsDetected() {
	stuckEventsDetected.Inc()
}
