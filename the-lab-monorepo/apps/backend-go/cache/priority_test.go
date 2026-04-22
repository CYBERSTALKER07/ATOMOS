package cache

import (
	"testing"
	"time"
)

// ── Priority Classification Tests ───────────────────────────────────────────

func TestClassifyRequest_P0Critical(t *testing.T) {
	cases := []string{
		"/v1/checkout/b2b",
		"/v1/checkout/unified",
		"/v1/order/cash-checkout",
		"/v1/order/card-checkout",
		"/v1/orders/request-cancel",
		"/v1/admin/orders/approve-cancel",
		"/v1/order/cancel",
		"/v1/webhooks/click",
		"/v1/webhooks/payme",
		"/v1/payment/chargeback",
	}
	for _, path := range cases {
		if got := ClassifyRequest(path); got != PriorityCritical {
			t.Errorf("ClassifyRequest(%q) = %s, want P0_CRITICAL", path, got)
		}
	}
}

func TestClassifyRequest_P2Telemetry(t *testing.T) {
	cases := []string{
		"/ws/telemetry",
		"/ws/fleet",
		"/v1/sync/batch",
		"/v1/analytics/revenue",
		"/v1/supplier/dashboard",
	}
	for _, path := range cases {
		if got := ClassifyRequest(path); got != PriorityTelemetry {
			t.Errorf("ClassifyRequest(%q) = %s, want P2_TELEMETRY", path, got)
		}
	}
}

func TestClassifyRequest_P1Operational(t *testing.T) {
	cases := []string{
		"/v1/orders/123/state",
		"/v1/fleet/drivers",
		"/v1/supplier/profile",
		"/v1/delivery/confirm",
	}
	for _, path := range cases {
		if got := ClassifyRequest(path); got != PriorityOperational {
			t.Errorf("ClassifyRequest(%q) = %s, want P1_OPERATIONAL", path, got)
		}
	}
}

// ── Backpressure Engine Tests ───────────────────────────────────────────────

func TestBackpressureEngine_ShouldShed(t *testing.T) {
	eng := &BackpressureEngine{
		config: DefaultBackpressureConfig(),
		done:   make(chan struct{}),
	}

	// Healthy — no shedding
	eng.latencyNanos.Store(int64(10 * time.Millisecond))
	if eng.ShouldShed(PriorityCritical) {
		t.Error("P0 should never be shed")
	}
	if eng.ShouldShed(PriorityOperational) {
		t.Error("P1 should not be shed at 10ms")
	}
	if eng.ShouldShed(PriorityTelemetry) {
		t.Error("P2 should not be shed at 10ms")
	}

	// Moderate — P2 shed, P1 ok
	eng.latencyNanos.Store(int64(60 * time.Millisecond))
	if eng.ShouldShed(PriorityCritical) {
		t.Error("P0 should never be shed")
	}
	if eng.ShouldShed(PriorityOperational) {
		t.Error("P1 should not be shed at 60ms")
	}
	if !eng.ShouldShed(PriorityTelemetry) {
		t.Error("P2 should be shed at 60ms (threshold 50ms)")
	}

	// Severe — both P1 and P2 shed
	eng.latencyNanos.Store(int64(200 * time.Millisecond))
	if eng.ShouldShed(PriorityCritical) {
		t.Error("P0 should never be shed")
	}
	if !eng.ShouldShed(PriorityOperational) {
		t.Error("P1 should be shed at 200ms (threshold 150ms)")
	}
	if !eng.ShouldShed(PriorityTelemetry) {
		t.Error("P2 should be shed at 200ms")
	}
}

func TestBackpressureEngine_BackpressureInterval(t *testing.T) {
	eng := &BackpressureEngine{
		config: DefaultBackpressureConfig(),
		done:   make(chan struct{}),
	}

	eng.latencyNanos.Store(int64(10 * time.Millisecond))
	if got := eng.BackpressureInterval(); got != 0 {
		t.Errorf("interval at 10ms = %d, want 0", got)
	}

	eng.latencyNanos.Store(int64(60 * time.Millisecond))
	if got := eng.BackpressureInterval(); got != 10 {
		t.Errorf("interval at 60ms = %d, want 10", got)
	}

	eng.latencyNanos.Store(int64(200 * time.Millisecond))
	if got := eng.BackpressureInterval(); got != 30 {
		t.Errorf("interval at 200ms = %d, want 30", got)
	}
}

// ── Token Bucket Tests (no Redis — fail-open behavior) ─────────────────────

func TestCheckTokenBucket_NilRedis(t *testing.T) {
	// When Redis is nil, token bucket should fail open
	origClient := Client
	Client = nil
	defer func() { Client = origClient }()

	result := CheckTokenBucket(nil, "tb:test", 10, 60)
	if !result.Allowed {
		t.Error("expected fail-open when Redis is nil")
	}
	if result.Remaining != 10 {
		t.Errorf("remaining = %d, want 10 (full capacity)", result.Remaining)
	}
}

func TestPriority_String(t *testing.T) {
	cases := []struct {
		p    Priority
		want string
	}{
		{PriorityCritical, "P0_CRITICAL"},
		{PriorityOperational, "P1_OPERATIONAL"},
		{PriorityTelemetry, "P2_TELEMETRY"},
		{Priority(99), "UNKNOWN"},
	}
	for _, tc := range cases {
		if got := tc.p.String(); got != tc.want {
			t.Errorf("Priority(%d).String() = %q, want %q", tc.p, got, tc.want)
		}
	}
}
