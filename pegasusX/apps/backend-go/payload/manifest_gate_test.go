package payload

import "testing"

func TestManifestGateSnapshot_PayloadDemoManifest(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	svc.mu.Unlock()

	state, stops, volume, ok := svc.ManifestGateSnapshot("mf_payload_1")
	if !ok {
		t.Fatal("expected demo manifest")
	}
	if state != payloadManifestStateDraft {
		t.Fatalf("expected DRAFT, got %q", state)
	}
	if stops != 2 || volume != 75 {
		t.Fatalf("unexpected gate snapshot stops=%d volume=%d", stops, volume)
	}
}
