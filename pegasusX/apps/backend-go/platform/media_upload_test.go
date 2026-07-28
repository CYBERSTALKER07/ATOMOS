package platform

import "testing"

func TestEvidenceObjectPrefix(t *testing.T) {
	p, err := evidenceObjectPrefix("claim_evidence", "ret-1", "ord-1")
	if err != nil || p != "evidence/claims/ret-1/ord-1" {
		t.Fatalf("got %q %v", p, err)
	}
	p, err = evidenceObjectPrefix("driver_exception", "drv-1", "ord-2")
	if err != nil || p != "evidence/driver/drv-1/ord-2" {
		t.Fatalf("got %q %v", p, err)
	}
	if _, err := evidenceObjectPrefix("nope", "x", ""); err == nil {
		t.Fatal("expected error")
	}
}
