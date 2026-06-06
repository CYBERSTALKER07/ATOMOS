package ws

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

var (
	allowedOriginMu sync.RWMutex
	allowedOrigins  = map[string]struct{}{}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     websocketOriginAllowed,
}

// SetAllowedOrigins replaces the browser origin allowlist used during
// websocket upgrades. Native clients without an Origin header remain allowed.
func SetAllowedOrigins(origins []string) {
	normalized := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		value := normalizeOrigin(origin)
		if value == "" {
			continue
		}
		normalized[value] = struct{}{}
	}
	allowedOriginMu.Lock()
	allowedOrigins = normalized
	allowedOriginMu.Unlock()
}

func websocketOriginAllowed(r *http.Request) bool {
	origin := normalizeOrigin(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if isLocalDevelopmentOrigin(origin) {
		return true
	}
	allowedOriginMu.RLock()
	_, ok := allowedOrigins[origin]
	allowedOriginMu.RUnlock()
	return ok
}

func normalizeOrigin(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func isLocalDevelopmentOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	parsedIP := net.ParseIP(host)
	return parsedIP != nil && parsedIP.IsLoopback()
}

// gorillaConn wraps a gorilla/websocket connection to implement ws.Connection.
type gorillaConn struct {
	id    string
	ident auth.Claims
	conn  *websocket.Conn
	mu    sync.Mutex
}

func (c *gorillaConn) ID() string {
	return c.id
}

func (c *gorillaConn) Identity() auth.Claims {
	return c.ident
}

func (c *gorillaConn) Send(ctx context.Context, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	_ = c.conn.SetWriteDeadline(deadline)
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *gorillaConn) ping(timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(timeout))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *gorillaConn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
}
