package payload

import "testing"

func TestManifestDetailSnapshotForDriver_PayloadDemo(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	svc.mu.Unlock()

	snapshot, ok := svc.ManifestDetailSnapshotForDriver("drv_payload_1", "mf_payload_1", "")
	if !ok {
		t.Fatal("expected demo manifest detail")
	}
	if snapshot.Manifest.ManifestID != "mf_payload_1" {
		t.Fatalf("unexpected manifest %#v", snapshot.Manifest)
	}
	if snapshot.StopCount != 2 || len(snapshot.Transfers) != 2 {
		t.Fatalf("expected 2 stops/transfers, got stop=%d transfers=%d", snapshot.StopCount, len(snapshot.Transfers))
	}
	if snapshot.RouteID != "route_veh_payload_1" {
		t.Fatalf("unexpected route_id %q", snapshot.RouteID)
	}
}

func TestManifestDetailSnapshotForDriver_RejectsDriverMismatch(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	if _, ok := svc.ManifestDetailSnapshotForDriver("drv_other", "mf_payload_1", ""); ok {
		t.Fatal("expected driver mismatch rejection")
	}
}
