package audit

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type mutationContextKey struct{}
type mutationCarrierKey struct{}

type mutationCarrier struct {
	mu sync.Mutex
	mc MutationContext
}

func (c *mutationCarrier) merge(in MutationContext) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if s := strings.TrimSpace(in.TenantID); s != "" {
		c.mc.TenantID = s
	}
	if s := strings.TrimSpace(in.Action); s != "" {
		c.mc.Action = s
	}
	if s := strings.TrimSpace(in.ObjectType); s != "" {
		c.mc.ObjectType = s
	}
	if s := strings.TrimSpace(in.ObjectID); s != "" {
		c.mc.ObjectID = s
	}
	if in.BeforeJSON != "" {
		c.mc.BeforeJSON = in.BeforeJSON
	}
	if in.AfterJSON != "" {
		c.mc.AfterJSON = in.AfterJSON
	}
	if s := strings.TrimSpace(in.ReasonCode); s != "" {
		c.mc.ReasonCode = s
	}
	if s := strings.TrimSpace(in.ApprovalID); s != "" {
		c.mc.ApprovalID = s
	}
}

func (c *mutationCarrier) snapshot() MutationContext {
	if c == nil {
		return MutationContext{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mc
}

// WithMutationContext attaches (and updates the request's active AuditMiddleware carrier with)
// handler-supplied object/diff metadata such as BeforeJson, AfterJson, ReasonCode, and ApprovalId.
func WithMutationContext(ctx context.Context, mc MutationContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if carrier, ok := ctx.Value(mutationCarrierKey{}).(*mutationCarrier); ok && carrier != nil {
		carrier.merge(mc)
	}
	return context.WithValue(ctx, mutationContextKey{}, mc)
}

// SetMutationContext updates the active AuditMiddleware carrier in ctx in place.
func SetMutationContext(ctx context.Context, mc MutationContext) {
	if ctx == nil {
		return
	}
	if carrier, ok := ctx.Value(mutationCarrierKey{}).(*mutationCarrier); ok && carrier != nil {
		carrier.merge(mc)
	}
}

// MutationContextFromContext returns any MutationContext attached to ctx.
func MutationContextFromContext(ctx context.Context) (MutationContext, bool) {
	if ctx == nil {
		return MutationContext{}, false
	}
	if carrier, ok := ctx.Value(mutationCarrierKey{}).(*mutationCarrier); ok && carrier != nil {
		return carrier.snapshot(), true
	}
	mc, ok := ctx.Value(mutationContextKey{}).(MutationContext)
	return mc, ok
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wroteHeader {
		sr.statusCode = code
		sr.wroteHeader = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(p []byte) (int, error) {
	if !sr.wroteHeader {
		sr.statusCode = http.StatusOK
		sr.wroteHeader = true
	}
	return sr.ResponseWriter.Write(p)
}

func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

func (sr *statusRecorder) Flush() {
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := sr.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijacker not supported")
}

// AuditMiddleware intercepts every mutating HTTP request (POST, PUT, PATCH, DELETE)
// after the downstream handler completes and appends a hash-chained AuditEvents row.
// Failures (>=400 or >=500) are still recorded with outcome metadata.
func AuditMiddleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutatingMethod(r.Method) || svc == nil || svc.Repo() == nil {
				next.ServeHTTP(w, r)
				return
			}

			carrier := &mutationCarrier{}
			ctx := context.WithValue(r.Context(), mutationCarrierKey{}, carrier)
			r = r.WithContext(ctx)

			rec := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			next.ServeHTTP(rec, r)

			mc := carrier.snapshot()
			status := rec.statusCode
			if status == 0 {
				status = http.StatusOK
			}

			routePattern := resolveRoutePattern(r)
			action := strings.TrimSpace(mc.Action)
			if action == "" {
				action = strings.ToUpper(r.Method) + " " + routePattern
			}

			tenantID := resolveMiddlewareTenantID(r, mc)
			actorID, actorType, role := resolveMiddlewareActor(r)

			objType := strings.TrimSpace(mc.ObjectType)
			objID := strings.TrimSpace(mc.ObjectID)
			if objType == "" || objID == "" {
				inferredType, inferredID := inferObjectFromRequest(r, routePattern)
				if objType == "" {
					objType = inferredType
				}
				if objID == "" {
					objID = inferredID
				}
			}

			reasonCode := strings.TrimSpace(mc.ReasonCode)
			if reasonCode == "" && status >= 400 {
				reasonCode = "HTTP_" + strconv.Itoa(status)
			}

			afterJSON := buildMiddlewareAfterJSON(r.Method, routePattern, status, mc.AfterJSON)
			traceID := strings.TrimSpace(outbox.TraceIDFromContext(r.Context()))
			if traceID == "" {
				traceID = strings.TrimSpace(r.Header.Get("X-Trace-Id"))
			}

			auditCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
			_, err := svc.Record(auditCtx, EventInput{
				TenantID:   tenantID,
				ActorID:    actorID,
				ActorType:  actorType,
				Role:       role,
				Action:     action,
				ObjectType: objType,
				ObjectID:   objID,
				BeforeJSON: mc.BeforeJSON,
				AfterJSON:  afterJSON,
				ReasonCode: reasonCode,
				ApprovalID: mc.ApprovalID,
				TraceID:    traceID,
				IP:         extractClientIP(r),
				DeviceID:   extractDeviceID(r),
			})
			cancel()
			if err != nil {
				slog.ErrorContext(r.Context(), "audit.middleware_record_failed",
					"err", err,
					"tenant_id", tenantID,
					"action", action,
					"status", status,
				)
			}
		})
	}
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func resolveRoutePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pat := strings.TrimSpace(rctx.RoutePattern()); pat != "" {
			return pat
		}
	}
	if r.URL != nil && strings.TrimSpace(r.URL.Path) != "" {
		return r.URL.Path
	}
	return "/"
}

func resolveMiddlewareTenantID(r *http.Request, mc MutationContext) string {
	if tid := strings.TrimSpace(mc.TenantID); tid != "" {
		return tid
	}
	ctx := r.Context()
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		return strings.TrimSpace(t.SupplierID)
	}
	if sid, ok := auth.ResolveSupplierID(ctx); ok && strings.TrimSpace(sid) != "" {
		return strings.TrimSpace(sid)
	}
	if r.URL != nil {
		q := r.URL.Query()
		for _, k := range []string{"entity", "entity_id", "tenant_id", "supplier_id"} {
			if v := strings.TrimSpace(q.Get(k)); v != "" {
				return v
			}
		}
	}
	if auth.IsPlatformAdmin(ctx) {
		return "platform"
	}
	if t, ok := auth.PartnerIdentityFromContext(ctx); ok && strings.TrimSpace(t.TenantID) != "" {
		return strings.TrimSpace(t.TenantID)
	}
	return "public"
}

func resolveMiddlewareActor(r *http.Request) (actorID string, actorType ActorType, role string) {
	ctx := r.Context()
	if claims, ok := auth.FromContext(ctx); ok {
		actorID = auth.ActorLabel(claims)
		actorType = NormalizeActorType(auth.EffectiveActorType(claims))
		role = auth.EffectiveSupplierRole(claims)
		if role == "" {
			role = string(claims.Role)
		}
		return actorID, actorType, role
	}
	if tctx, ok := auth.PartnerIdentityFromContext(ctx); ok && strings.EqualFold(tctx.Source, "partner") {
		actorID = tctx.PrincipalIdentifier()
		if actorID == "" {
			actorID = "partner"
		}
		return actorID, ActorTypeConnector, "PARTNER"
	}
	path := ""
	if r.URL != nil {
		path = r.URL.Path
	}
	if strings.HasPrefix(path, "/v1/webhooks/") {
		return "webhook", ActorTypeConnector, "CONNECTOR"
	}
	return "anonymous", ActorTypeHuman, ""
}

func inferObjectFromRequest(r *http.Request, routePattern string) (objType, objID string) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		preferredKeys := []string{
			"id", "orderID", "orderId", "entryId", "manifestID", "manifestId",
			"userId", "userID", "holdId", "policyId", "vehicleID", "truckID",
			"retailerId", "retailerID", "supplierId", "supplierID", "invoiceId",
			"claimId", "code", "periodId", "runId", "breakId", "objectId",
		}
		for _, key := range preferredKeys {
			if val := strings.TrimSpace(rctx.URLParam(key)); val != "" {
				objID = val
				break
			}
		}
		if objID == "" && len(rctx.URLParams.Values) > 0 {
			for i := len(rctx.URLParams.Values) - 1; i >= 0; i-- {
				if v := strings.TrimSpace(rctx.URLParams.Values[i]); v != "" {
					objID = v
					break
				}
			}
		}
		if ot := strings.TrimSpace(rctx.URLParam("objectType")); ot != "" {
			objType = ot
		}
	}

	if objType == "" {
		raw := routePattern
		if raw == "" && r.URL != nil {
			raw = r.URL.Path
		}
		objType = inferObjectTypeFromPath(raw)
	}
	return objType, objID
}

func inferObjectTypeFromPath(p string) string {
	trimmed := strings.Trim(p, "/")
	if trimmed == "" {
		return "http_route"
	}
	parts := strings.Split(trimmed, "/")
	skip := map[string]bool{
		"v1": true, "v2": true, "supplier": true, "admin": true, "finance": true,
		"retailer": true, "driver": true, "warehouse": true, "factory": true,
		"payload": true, "payloader": true, "auth": true, "partner": true,
		"platform": true, "audit": true, "iam": true, "sim": true,
	}
	var meaningful []string
	for _, seg := range parts {
		if seg == "" || strings.HasPrefix(seg, "{") {
			continue
		}
		low := strings.ToLower(seg)
		if skip[low] {
			continue
		}
		meaningful = append(meaningful, low)
	}
	if len(meaningful) == 0 {
		for _, seg := range parts {
			if seg != "" && !strings.HasPrefix(seg, "{") && seg != "v1" {
				return strings.ReplaceAll(strings.ToLower(seg), "-", "_")
			}
		}
		return "http_route"
	}
	return strings.ReplaceAll(meaningful[0], "-", "_")
}

func buildMiddlewareAfterJSON(method, route string, status int, explicitAfter string) string {
	outcome := "success"
	if status >= 500 {
		outcome = "server_error"
	} else if status >= 400 {
		outcome = "client_error"
	}

	if strings.TrimSpace(explicitAfter) == "" {
		b, _ := json.Marshal(struct {
			HTTPStatus int    `json:"http_status"`
			Method     string `json:"method"`
			Route      string `json:"route"`
			Outcome    string `json:"outcome"`
		}{
			HTTPStatus: status,
			Method:     method,
			Route:      route,
			Outcome:    outcome,
		})
		return string(b)
	}
	if status < 400 {
		return explicitAfter
	}

	envelope := struct {
		HTTPStatus int             `json:"http_status"`
		Method     string          `json:"method"`
		Route      string          `json:"route"`
		Outcome    string          `json:"outcome"`
		Data       json.RawMessage `json:"data,omitempty"`
		Raw        string          `json:"raw,omitempty"`
	}{
		HTTPStatus: status,
		Method:     method,
		Route:      route,
		Outcome:    outcome,
	}
	if json.Valid([]byte(explicitAfter)) {
		envelope.Data = json.RawMessage(explicitAfter)
	} else {
		envelope.Raw = explicitAfter
	}
	b, _ := json.Marshal(envelope)
	return string(b)
}

func extractClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			xff = xff[:idx]
		}
		if ip := strings.TrimSpace(xff); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xri != "" {
		return xri
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	return host
}

func extractDeviceID(r *http.Request) string {
	for _, h := range []string{"X-Device-Id", "X-Device-ID"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			return v
		}
	}
	if cert := strings.TrimSpace(r.Header.Get("X-Device-Cert")); cert != "" {
		sum := sha256.Sum256([]byte(cert))
		return "cert:" + hex.EncodeToString(sum[:8])
	}
	return ""
}

// StartNightlyAnchorWorker periodically writes daily WORM hash-chain anchors for
// yesterday's UTC DayBucket across all tenants with AuditEvents until ctx is cancelled.
func StartNightlyAnchorWorker(ctx context.Context, interval time.Duration, svc *Service) {
	if svc == nil || svc.Repo() == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	runAnchorSweepSafely(ctx, svc)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAnchorSweepSafely(ctx, svc)
		}
	}
}

func runAnchorSweepSafely(ctx context.Context, svc *Service) {
	if ctx.Err() != nil {
		return
	}
	dayBucket := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	tenants, err := svc.ListTenantsForDay(ctx, dayBucket)
	if err != nil {
		slog.Warn("audit nightly anchor tenant scan failed", "day_bucket", dayBucket, "err", err)
		return
	}
	for _, tid := range tenants {
		if ctx.Err() != nil {
			return
		}
		if _, err := svc.WriteDailyAnchor(ctx, tid, dayBucket, "worker:nightly_anchor", nil); err != nil {
			slog.Warn("audit nightly anchor write failed", "tenant_id", tid, "day_bucket", dayBucket, "err", err)
		}
	}
}
