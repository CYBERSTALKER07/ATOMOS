package ws

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"backend-go/cache"
)

const (
	defaultWSMaxPerUser     = 5
	defaultWSConnectPerIP   = 30
	defaultWSConnectWindow  = time.Minute
	envWSMaxPerUser         = "WS_MAX_CONNECTIONS_PER_USER"
	envWSConnectPerIP       = "WS_CONNECT_RATE_PER_IP"
	envWSConnectWindowSec   = "WS_CONNECT_WINDOW_SEC"
)

// EnforceWSConnectionLimits rejects WebSocket upgrades when per-IP connect rate
// or per-user concurrent connection caps are exceeded.
// activeConnections is the hub-local count for userID before registering the new conn.
func EnforceWSConnectionLimits(w http.ResponseWriter, r *http.Request, hub, userID string, activeConnections int) bool {
	maxPerUser := envInt(envWSMaxPerUser, defaultWSMaxPerUser)
	if userID != "" && activeConnections >= maxPerUser {
		http.Error(w, fmt.Sprintf(`{"error":"ws_connection_limit","message":"maximum %d concurrent connections per user"}`, maxPerUser), http.StatusTooManyRequests)
		return false
	}

	if cache.GetClient() == nil {
		return true
	}

	windowSec := int64(envInt(envWSConnectWindowSec, int(defaultWSConnectWindow/time.Second)))
	if windowSec <= 0 {
		windowSec = 60
	}
	maxPerIP := int64(envInt(envWSConnectPerIP, defaultWSConnectPerIP))
	ip := clientIP(r)
	key := fmt.Sprintf("%sws:connect:%s:%s", cache.PrefixRateLimit, hub, ip)

	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()
	result := cache.CheckTokenBucket(ctx, key, maxPerIP, windowSec)
	if !result.Allowed {
		if result.TTL > 0 {
			w.Header().Set("Retry-After", strconv.FormatInt(result.TTL, 10))
		}
		http.Error(w, `{"error":"ws_connect_rate_limit","message":"too many websocket connection attempts"}`, http.StatusTooManyRequests)
		return false
	}
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
