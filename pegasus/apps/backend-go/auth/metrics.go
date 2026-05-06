package auth

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var wsQueryTokenTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "auth",
		Name:      "ws_query_token_total",
		Help:      "WebSocket query-token authentication attempts by endpoint class and result.",
	},
	[]string{"endpoint", "result"},
)

var wsGraceActivationsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "auth",
		Name:      "ws_grace_activations_total",
		Help:      "WebSocket token-grace activations by endpoint class and role.",
	},
	[]string{"endpoint", "role"},
)

func wsEndpointClass(path string) string {
	normalized := strings.TrimSuffix(path, "/")
	if normalized == "" {
		normalized = "/"
	}

	switch normalized {
	case "/ws/telemetry":
		return "telemetry"
	case "/ws/fleet":
		return "fleet"
	case "/ws/warehouse":
		return "warehouse"
	case "/v1/ws/factory":
		return "factory"
	case "/v1/ws/retailer":
		return "retailer"
	case "/v1/ws/driver":
		return "driver"
	case "/v1/ws/payloader":
		return "payloader"
	default:
		return "unknown"
	}
}

func allowsWSQueryToken(path string) bool {
	switch wsEndpointClass(path) {
	case "telemetry", "fleet", "warehouse", "factory", "retailer", "driver", "payloader":
		return true
	default:
		return false
	}
}

func recordWSQueryToken(endpoint, result string) {
	wsQueryTokenTotal.WithLabelValues(endpoint, result).Inc()
}

func recordWSGraceActivation(endpoint, role string) {
	if role == "" {
		role = "unknown"
	}
	wsGraceActivationsTotal.WithLabelValues(endpoint, role).Inc()
}
