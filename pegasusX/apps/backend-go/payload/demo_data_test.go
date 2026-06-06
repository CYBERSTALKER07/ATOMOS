package payload

import "testing"

func TestEnsureDemoDataLocked_IncludesSiblingManifest(t *testing.T) {
	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	svc.mu.Unlock()

	if _, _, _, ok := svc.ManifestGateSnapshot("mf_payload_2"); !ok {
		t.Fatal("expected mf_payload_2 in demo data")
	}
}
