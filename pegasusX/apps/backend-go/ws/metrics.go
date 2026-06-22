package ws

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	wsConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "connections",
		Help:      "Active WebSocket connections per hub",
	}, []string{"hub"})

	wsPubFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "pub_failures_total",
		Help:      "Cross-pod WS relay publish failures per hub",
	}, []string{"hub"})

	wsShedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "shed_total",
		Help:      "WS connections shed due to capacity limits per hub",
	}, []string{"hub"})
)

func (h *Hub) recordMetricsLocked() {
	if h == nil {
		return
	}
	conns := 0
	for _, room := range h.rooms {
		conns += len(room)
	}
	wsConnections.WithLabelValues(h.name).Set(float64(conns))
}

func (h *Hub) recordPubFailure() {
	wsPubFailures.WithLabelValues(h.name).Inc()
}

func (h *Hub) recordShed(count int) {
	if count <= 0 {
		return
	}
	wsShedTotal.WithLabelValues(h.name).Add(float64(count))
}
