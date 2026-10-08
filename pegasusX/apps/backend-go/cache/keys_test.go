package cache

import (
	"testing"
	"time"
)

func TestSupplierScopedKey(t *testing.T) {
	t.Parallel()
	got := SupplierScopedKey("sup-1", "dispatch:plan")
	if got != "{sup:sup-1}:dispatch:plan" {
		t.Fatalf("key = %q", got)
	}
}

func TestKeyWithHashTags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		entity string
		id     string
		attrs  []string
		want   string
	}{
		{entity: "Order", id: "1001", attrs: []string{"items"}, want: "{order:1001}:items"},
		{entity: "User", id: "usr-42", attrs: []string{"profile", "settings"}, want: "{user:usr-42}:profile:settings"},
		{entity: "Supplier", id: "sup-99", attrs: nil, want: "{supplier:sup-99}"},
	}

	for _, tc := range cases {
		got := Key(tc.entity, tc.id, tc.attrs...)
		if got != tc.want {
			t.Errorf("Key(%s, %s, %v) = %q, want %q", tc.entity, tc.id, tc.attrs, got, tc.want)
		}
	}
}

func TestScopedKeys(t *testing.T) {
	t.Parallel()
	if got := RetailerScopedKey("ret-1", "profile"); got != "{ret:ret-1}:profile" {
		t.Errorf("RetailerScopedKey = %q", got)
	}
	if got := OrderScopedKey("ord-1", "events"); got != "{ord:ord-1}:events" {
		t.Errorf("OrderScopedKey = %q", got)
	}
	if got := DriverScopedKey("drv-1", "telemetry"); got != "{drv:drv-1}:telemetry" {
		t.Errorf("DriverScopedKey = %q", got)
	}
	if got := SimpleKey("ratelimit", "ip", "127.0.0.1"); got != "ratelimit:ip:127.0.0.1" {
		t.Errorf("SimpleKey = %q", got)
	}
}

func TestNamedTTLConstants(t *testing.T) {
	t.Parallel()
	if TTLDefault <= 0 || TTLRateLimitDefault <= 0 || TTLIdempotencyAPI <= 0 {
		t.Errorf("TTL constants must be positive durations")
	}
	if TTLIdempotencyWebhook < 24*time.Hour {
		t.Errorf("Webhook idempotency must retain for at least 24h")
	}
}
