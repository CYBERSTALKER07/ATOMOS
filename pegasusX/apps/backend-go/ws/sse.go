package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

const (
	// SSERetryTimeoutMS is the default client reconnection backoff directive (3 seconds).
	SSERetryTimeoutMS = 3000

	// SSEPingInterval is the periodic keep-alive comment interval (15 seconds).
	SSEPingInterval = 15 * time.Second

	// SSEContentType is the standard MIME type for Server-Sent Events.
	SSEContentType = "text/event-stream; charset=utf-8"

	// SSECacheControl disables downstream caching and proxy transformation.
	SSECacheControl = "no-cache, no-transform"
)

// sseConn wraps an HTTP response stream to implement ws.Connection and ws.Reapable.
type sseConn struct {
	id        string
	ident     auth.Claims
	w         http.ResponseWriter
	flusher   http.Flusher
	mu        sync.Mutex
	closed    chan struct{}
	closeOnce sync.Once
}

// NewSSEConn constructs a new sseConn instance.
func NewSSEConn(ident auth.Claims, w http.ResponseWriter, flusher http.Flusher) *sseConn {
	return &sseConn{
		id:      uuid.New().String(),
		ident:   ident,
		w:       w,
		flusher: flusher,
		closed:  make(chan struct{}),
	}
}

func (c *sseConn) ID() string {
	return c.id
}

func (c *sseConn) Identity() auth.Claims {
	return c.ident
}

func (c *sseConn) Done() <-chan struct{} {
	return c.closed
}

func (c *sseConn) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
}

// Reap force-closes the connection when the hub sheds stale connections.
func (c *sseConn) Reap() {
	c.Close()
}

// Ping sends an SSE comment line (: ping\n\n) to keep the stream alive.
func (c *sseConn) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.closed:
		return errors.New("sse connection closed")
	default:
	}

	if _, err := fmt.Fprintf(c.w, ": ping\n\n"); err != nil {
		c.Close()
		return err
	}
	if c.flusher != nil {
		c.flusher.Flush()
	}
	return nil
}

// Send formats the payload into an SSE frame (id, event, data) and flushes it to the client.
func (c *sseConn) Send(ctx context.Context, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.closed:
		return errors.New("sse connection closed")
	default:
	}

	eventType, eventID := parseSSEEventMetadata(payload)
	if eventID == "" {
		eventID = strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "id: %s\n", eventID)
	if eventType != "" {
		fmt.Fprintf(&buf, "event: %s\n", eventType)
	}
	lines := bytes.Split(payload, []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			fmt.Fprintf(&buf, "data: %s\n", line)
		}
	}
	if len(lines) == 0 || len(payload) == 0 {
		buf.WriteString("data: {}\n")
	}
	buf.WriteString("\n")

	if _, err := c.w.Write(buf.Bytes()); err != nil {
		c.Close()
		return err
	}
	if c.flusher != nil {
		c.flusher.Flush()
	}
	return nil
}

func parseSSEEventMetadata(payload []byte) (string, string) {
	if len(payload) == 0 {
		return "", ""
	}
	var partial struct {
		Type    string `json:"type"`
		EventID string `json:"event_id"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(payload, &partial); err == nil {
		id := strings.TrimSpace(partial.EventID)
		if id == "" {
			id = strings.TrimSpace(partial.ID)
		}
		return strings.TrimSpace(partial.Type), id
	}
	return "", ""
}

func claimsFromSSERequest(req *http.Request, jwtSecret string) (auth.Claims, bool) {
	if ident, ok := auth.FromContext(req.Context()); ok {
		if !auth.IsPendingOrgSelect(ident) {
			return ident, true
		}
	}
	if strings.TrimSpace(jwtSecret) == "" {
		return auth.Claims{}, false
	}
	if ident, ok := auth.ParseBearerClaims(req, jwtSecret); ok {
		if !auth.IsPendingOrgSelect(ident) {
			return ident, true
		}
	}
	token := strings.TrimSpace(req.URL.Query().Get("token"))
	if token != "" {
		ident, err := auth.Parse(token, jwtSecret)
		if err == nil && !auth.IsPendingOrgSelect(ident) {
			jti := strings.TrimSpace(ident.JTI)
			if jti != "" {
				revoked, err := auth.GetRevocationStore().IsRevoked(req.Context(), jti)
				if err != nil || revoked {
					return auth.Claims{}, false
				}
			}
			return ident, true
		}
	}
	return auth.Claims{}, false
}

// HandleSSEEvents handles general multi-role Server-Sent Events mounted at /v1/events.
func HandleSSEEvents(log *slog.Logger, jwtSecret string, platformSvc *platform.Service, hubs roleHubs, cfg RegisterConfig, roleFilter ...auth.Role) http.HandlerFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(w http.ResponseWriter, req *http.Request) {
		ident, ok := claimsFromSSERequest(req, jwtSecret)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if len(roleFilter) > 0 {
			allowed := false
			for _, r := range roleFilter {
				if ident.Role == r {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
		}

		if !roleHubsHaveCapacity(ident, hubs) {
			log.Warn("sse upgrade rejected: hub at capacity", "role", ident.Role)
			w.Header().Set("Retry-After", strconv.Itoa(CapacityRetryAfterSeconds()))
			http.Error(w, `{"error":"service_unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming_unsupported"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", SSEContentType)
		w.Header().Set("Cache-Control", SSECacheControl)
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Initial handshake
		if _, err := fmt.Fprintf(w, "retry: %d\n\n: connected\n\n", SSERetryTimeoutMS); err != nil {
			return
		}
		flusher.Flush()

		conn := NewSSEConn(ident, w, flusher)
		unsubscribes, ok := subscribeIdentityRooms(ident, conn, hubs, cfg)
		if !ok {
			log.Warn("sse: unrecognized role", "role", ident.Role)
			conn.Close()
			return
		}
		defer func() {
			for _, unsub := range unsubscribes {
				unsub()
			}
			conn.Close()
		}()

		replayMissedEvents(req, ident, conn, hubs, cfg)

		ticker := time.NewTicker(SSEPingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-req.Context().Done():
				return
			case <-conn.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(); err != nil {
					return
				}
			}
		}
	}
}

// HandleSupplierEvents handles supplier-scoped Server-Sent Events mounted at /v1/supplier/events.
func HandleSupplierEvents(log *slog.Logger, jwtSecret string, supplierHub *Hub, warehouseHub *Hub, telemetryHub *Hub) http.HandlerFunc {
	if log == nil {
		log = slog.Default()
	}
	hubs := roleHubs{
		supplier:  supplierHub,
		warehouse: warehouseHub,
		telemetry: telemetryHub,
	}
	return func(w http.ResponseWriter, req *http.Request) {
		ident, ok := claimsFromSSERequest(req, jwtSecret)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Supplier events allow ADMIN, WAREHOUSE_ADMIN, FACTORY_ADMIN, PLATFORM_ADMIN, or any caller with SupplierID
		isSupplierRole := ident.Role == auth.RoleAdmin ||
			ident.Role == auth.RoleWarehouseAdmin ||
			ident.Role == auth.RoleWarehouse ||
			ident.Role == auth.RoleFactoryAdmin ||
			ident.Role == auth.RoleFactory ||
			ident.Role == auth.RolePlatformAdmin ||
			ident.SupplierID != ""
		if !isSupplierRole {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		if !roleHubsHaveCapacity(ident, hubs) {
			log.Warn("sse supplier stream rejected: hub at capacity", "role", ident.Role)
			w.Header().Set("Retry-After", strconv.Itoa(CapacityRetryAfterSeconds()))
			http.Error(w, `{"error":"service_unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming_unsupported"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", SSEContentType)
		w.Header().Set("Cache-Control", SSECacheControl)
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Initial handshake
		if _, err := fmt.Fprintf(w, "retry: %d\n\n: connected\n\n", SSERetryTimeoutMS); err != nil {
			return
		}
		flusher.Flush()

		conn := NewSSEConn(ident, w, flusher)
		var unsubscribes []func()

		if supplierHub != nil && ident.SupplierID != "" {
			unsubscribes = append(unsubscribes, supplierHub.Subscribe("supplier:"+ident.SupplierID, conn))
		}
		if warehouseHub != nil && ident.HomeNodeID != "" {
			unsubscribes = append(unsubscribes, warehouseHub.Subscribe("warehouse:"+ident.HomeNodeID, conn))
		}
		if telemetryHub != nil && ident.SupplierID != "" {
			unsubscribes = append(unsubscribes, telemetryHub.Subscribe("telemetry:supplier:"+ident.SupplierID, conn))
		}

		defer func() {
			for _, unsub := range unsubscribes {
				unsub()
			}
			conn.Close()
		}()

		sinceSeq, lastEventID := parseReconnectParams(req)
		if sinceSeq > 0 || lastEventID != "" {
			if supplierHub != nil && ident.SupplierID != "" {
				supplierHub.ReplaySince(req.Context(), "supplier:"+ident.SupplierID, sinceSeq, lastEventID, conn)
			}
			if warehouseHub != nil && ident.HomeNodeID != "" {
				warehouseHub.ReplaySince(req.Context(), "warehouse:"+ident.HomeNodeID, sinceSeq, lastEventID, conn)
			}
			if telemetryHub != nil && ident.SupplierID != "" {
				telemetryHub.ReplaySince(req.Context(), "telemetry:supplier:"+ident.SupplierID, sinceSeq, lastEventID, conn)
			}
		}

		ticker := time.NewTicker(SSEPingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-req.Context().Done():
				return
			case <-conn.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(); err != nil {
					return
				}
			}
		}
	}
}
