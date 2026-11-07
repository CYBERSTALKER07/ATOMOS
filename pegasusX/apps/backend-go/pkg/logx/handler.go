package logx

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type contextKey string

const (
	orderIDKey    contextKey = "logx.order_id"
	tenantIDKey   contextKey = "logx.tenant_id"
	supplierIDKey contextKey = "logx.supplier_id"
)

// WithOrderID attaches order_id to context for automated slog extraction.
func WithOrderID(ctx context.Context, orderID string) context.Context {
	return context.WithValue(ctx, orderIDKey, strings.TrimSpace(orderID))
}

// WithTenantID attaches tenant_id to context for automated slog extraction.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, strings.TrimSpace(tenantID))
}

// WithSupplierID attaches supplier_id to context for automated slog extraction.
func WithSupplierID(ctx context.Context, supplierID string) context.Context {
	return context.WithValue(ctx, supplierIDKey, strings.TrimSpace(supplierID))
}

// ContextHandler wraps an inner slog.Handler and extracts standard contextual attributes.
type ContextHandler struct {
	inner slog.Handler
}

// NewHandler creates a ContextHandler wrapping an inner handler.
func NewHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		// 1. Correlate Request / Trace ID
		traceID := outbox.TraceIDFromContext(ctx)
		if traceID != "" {
			r.AddAttrs(slog.String("request_id", traceID))
		}

		// 2. Correlate Tenant ID
		tenantID := ""
		if v, ok := ctx.Value(tenantIDKey).(string); ok && v != "" {
			tenantID = v
		} else if tc, ok := auth.TenantFromContext(ctx); ok && tc.TenantID != "" {
			tenantID = tc.TenantID
		} else if pi, ok := auth.PartnerIdentityFromContext(ctx); ok && pi.TenantID != "" {
			tenantID = pi.TenantID
		}
		if tenantID != "" {
			r.AddAttrs(slog.String("tenant_id", tenantID))
		}

		// 3. Correlate Supplier ID
		supplierID := ""
		if v, ok := ctx.Value(supplierIDKey).(string); ok && v != "" {
			supplierID = v
		} else if tc, ok := auth.TenantFromContext(ctx); ok && tc.SupplierID != "" {
			supplierID = tc.SupplierID
		} else if sid, ok := auth.ResolveSupplierID(ctx); ok && sid != "" {
			supplierID = sid
		}
		if supplierID != "" {
			r.AddAttrs(slog.String("supplier_id", supplierID))
		}

		// 4. Correlate Order ID
		if v, ok := ctx.Value(orderIDKey).(string); ok && v != "" {
			r.AddAttrs(slog.String("order_id", v))
		}
	}
	return h.inner.Handle(ctx, r)
}

// NewLogger constructs a standardized enterprise JSON logger.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Normalize "err" -> "error"
			if a.Key == "err" {
				a.Key = "error"
			}
			return a
		},
	}
	jsonHandler := slog.NewJSONHandler(w, opts)
	return slog.New(NewHandler(jsonHandler))
}
