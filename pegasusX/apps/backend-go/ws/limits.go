package ws

import (
	"os"
	"strconv"
	"strings"
)

// HubLimits caps live sockets per hub instance. Per-room shedding reaps the
// oldest connection when a newer client joins (Desert Protocol: mobile clients
// in dead zones reconnect aggressively; shedding prevents duplicate ghost sessions).
type HubLimits struct {
	// MaxPerRoom is the maximum simultaneous connections in one room. Zero disables.
	MaxPerRoom int
	// MaxTotal is the maximum connections across all rooms on this pod. Zero disables.
	MaxTotal int
}

// DefaultHubLimits returns role-aware defaults. WS_MAX_CONNECTIONS_PER_POD
// (default 25000) caps total sockets per process; per-room limits shed stale
// tabs/devices when a fresher session arrives.
func DefaultHubLimits(hubName string) HubLimits {
	total := envInt("WS_MAX_CONNECTIONS_PER_POD", 25000)
	switch strings.ToLower(strings.TrimSpace(hubName)) {
	case "driver":
		return HubLimits{MaxPerRoom: envInt("WS_DRIVER_MAX_PER_ROOM", 2), MaxTotal: total}
	case "retailer":
		return HubLimits{MaxPerRoom: envInt("WS_RETAILER_MAX_PER_ROOM", 5), MaxTotal: total}
	case "telemetry":
		return HubLimits{MaxPerRoom: envInt("WS_TELEMETRY_MAX_PER_ROOM", 2), MaxTotal: total}
	case "supplier", "warehouse", "factory", "payload":
		return HubLimits{MaxPerRoom: envInt("WS_PORTAL_MAX_PER_ROOM", 4), MaxTotal: total}
	default:
		return HubLimits{MaxPerRoom: envInt("WS_DEFAULT_MAX_PER_ROOM", 3), MaxTotal: total}
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// CapacityRetryAfterSeconds is the Retry-After hint returned when a pod refuses
// new WebSocket upgrades at MaxTotal capacity (Desert Protocol).
func CapacityRetryAfterSeconds() int {
	return envInt("WS_RETRY_AFTER_SECONDS", 30)
}
