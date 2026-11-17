package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestContextHandler_ContextAttributesAndErrorNormalization(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, slog.LevelInfo)

	ctx := context.Background()
	ctx = outbox.WithTraceID(ctx, "req-test-uuid-123")
	ctx = auth.WithTenant(ctx, auth.TenantContext{
		TenantID:   "tenant-corp-456",
		SupplierID: "sup-global-789",
	})
	ctx = WithOrderID(ctx, "ord-checkout-999")

	logger.InfoContext(ctx, "payment processed successfully",
		"err", errors.New("simulated error string"),
		"amount_tiyin", int64(500000),
	)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal json log: %v", err)
	}

	if entry["msg"] != "payment processed successfully" {
		t.Errorf("expected msg 'payment processed successfully', got %v", entry["msg"])
	}

	// Verify contextual extractions
	if entry["request_id"] != "req-test-uuid-123" {
		t.Errorf("expected request_id req-test-uuid-123, got %v", entry["request_id"])
	}
	if entry["tenant_id"] != "tenant-corp-456" {
		t.Errorf("expected tenant_id tenant-corp-456, got %v", entry["tenant_id"])
	}
	if entry["supplier_id"] != "sup-global-789" {
		t.Errorf("expected supplier_id sup-global-789, got %v", entry["supplier_id"])
	}
	if entry["order_id"] != "ord-checkout-999" {
		t.Errorf("expected order_id ord-checkout-999, got %v", entry["order_id"])
	}

	// Verify normalization: "err" replaced by "error"
	if _, hasErr := entry["err"]; hasErr {
		t.Errorf("'err' attribute should have been normalized to 'error'")
	}
	if entry["error"] != "simulated error string" {
		t.Errorf("expected error attribute 'simulated error string', got %v", entry["error"])
	}
}

func TestContextHandler_PartnerContextExtraction(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, slog.LevelInfo)

	ctx := context.Background()
	ctx = auth.WithTenant(ctx, auth.TenantContext{
		TenantID: "partner-ret-101",
		KeyID:    "pxk_live_test",
		Source:   "partner",
	})

	logger.InfoContext(ctx, "partner order submitted")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal json log: %v", err)
	}

	if entry["tenant_id"] != "partner-ret-101" {
		t.Errorf("expected tenant_id partner-ret-101, got %v", entry["tenant_id"])
	}
}
