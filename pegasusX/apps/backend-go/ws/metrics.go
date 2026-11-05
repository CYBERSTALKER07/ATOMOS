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

	wsMessagesBroadcastTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "messages_broadcast_total",
		Help:      "Total WebSocket messages broadcast per hub",
	}, []string{"hub"})

	wsMessagesDeliveredTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "messages_delivered_total",
		Help:      "Total WebSocket messages delivered to local subscribers per hub",
	}, []string{"hub"})

	wsMessagesAckedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "messages_acked_total",
		Help:      "Total WebSocket message acknowledgments received from clients per role",
	}, []string{"role"})

	wsRateLimitedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "rate_limited_total",
		Help:      "Total WebSocket connections or frames rejected due to ingress rate limiting",
	})

	wsUnauthorizedRoomTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "unauthorized_room_total",
		Help:      "Total unauthorized room join or replay attempts rejected per role",
	}, []string{"role"})

	wsReplayedEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "ws",
		Name:      "replayed_events_total",
		Help:      "Total buffered WebSocket events replayed upon client reconnection per hub",
	}, []string{"hub"})
)

func (h *Hub) recordMetricsLocked() {
	if h == nil {
		return
	}
	wsConnections.WithLabelValues(h.name).Set(float64(h.connectionCountLocked()))
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

func (h *Hub) recordBroadcast() {
	if h == nil {
		return
	}
	wsMessagesBroadcastTotal.WithLabelValues(h.name).Inc()
}

func (h *Hub) recordDelivered(count int) {
	if h == nil || count <= 0 {
		return
	}
	wsMessagesDeliveredTotal.WithLabelValues(h.name).Add(float64(count))
}

func (h *Hub) recordReplayed(count int) {
	if h == nil || count <= 0 {
		return
	}
	wsReplayedEventsTotal.WithLabelValues(h.name).Add(float64(count))
}

func recordClientAck(role string) {
	if role == "" {
		role = "unknown"
	}
	wsMessagesAckedTotal.WithLabelValues(role).Inc()
}

func recordRateLimited() {
	wsRateLimitedTotal.Inc()
}

func recordUnauthorizedRoom(role string) {
	if role == "" {
		role = "unknown"
	}
	wsUnauthorizedRoomTotal.WithLabelValues(role).Inc()
}
