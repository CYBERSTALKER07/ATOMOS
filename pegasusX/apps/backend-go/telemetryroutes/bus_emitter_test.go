package telemetryroutes

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

type fakeBusEmitter struct{ calls atomic.Int32 }

func (f *fakeBusEmitter) EmitDriverLocation(_ context.Context, _, _, _ string, _ []byte) error {
	f.calls.Add(1)
	return nil
}

func TestEmitLocationToBus_ThrottlesPerDriver(t *testing.T) {
	em := &fakeBusEmitter{}
	d := Deps{LocationBusEmitter: em, LocationBusInterval: 50 * time.Millisecond}
	id := locationIdentity{DriverID: "drv-throttle-test", SupplierID: "sup-1"}
	raw := []byte(`{"type":"DRIVER_LOCATION_UPDATED"}`)

	d.emitLocationToBus(context.Background(), id, raw)
	d.emitLocationToBus(context.Background(), id, raw) // within interval -> suppressed
	if got := em.calls.Load(); got != 1 {
		t.Fatalf("calls after 2 rapid pings = %d, want 1 (throttled)", got)
	}
	time.Sleep(60 * time.Millisecond)
	d.emitLocationToBus(context.Background(), id, raw) // past interval -> emitted
	if got := em.calls.Load(); got != 2 {
		t.Fatalf("calls after interval elapsed = %d, want 2", got)
	}
}

func TestEmitLocationToBus_NilEmitterNoop(t *testing.T) {
	d := Deps{}
	d.emitLocationToBus(context.Background(), locationIdentity{DriverID: "drv-nil"}, []byte("{}"))
}

func TestInjectRouteID(t *testing.T) {
	out := injectRouteID([]byte(`{"type":"DRIVER_LOCATION_UPDATED","data":{"driver_id":"d1"}}`), "route-9")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["route_id"] != "route-9" {
		t.Fatalf("route_id = %v, want route-9", got["route_id"])
	}
	data, _ := got["data"].(map[string]any)
	if data["route_id"] != "route-9" {
		t.Fatalf("data.route_id = %v, want route-9", data["route_id"])
	}
}
