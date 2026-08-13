package partner

import (
	"context"
	"testing"
)

func TestEdiProfile_DocEnabled(t *testing.T) {
	p := DefaultEdiProfile(TenantSupplier, "sup-1")
	if !p.DocEnabled(EdiDocORDERS) || !p.DocEnabled(EdiDocDESADV) {
		t.Fatal("default should enable ORDERS/DESADV")
	}
	p.EnabledDocTypes = []string{EdiDocORDERS}
	if p.DocEnabled(EdiDocDESADV) {
		t.Fatal("DESADV should be disabled")
	}
	if !p.DocEnabled("orders") {
		t.Fatal("case insensitive")
	}
}

func TestMemoryEdiProfiles_RoundTrip(t *testing.T) {
	repo := NewMemoryEdiProfiles()
	p := DefaultEdiProfile(TenantSupplier, "sup-a")
	p.EnabledDocTypes = []string{EdiDocORDERS, EdiDocORDRSP}
	p.OurGLN = "123"
	if err := repo.Upsert(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got, ok, err := repo.Get(context.Background(), TenantSupplier, "sup-a")
	if err != nil || !ok {
		t.Fatalf("get ok=%v err=%v", ok, err)
	}
	if !got.DocEnabled(EdiDocORDERS) || got.DocEnabled(EdiDocDESADV) {
		t.Fatalf("enabled=%v", got.EnabledDocTypes)
	}
	if got.OurGLN != "123" {
		t.Fatalf("gln=%s", got.OurGLN)
	}
}

func TestResolveEdiProfile_Default(t *testing.T) {
	p := ResolveEdiProfile(context.Background(), nil, TenantSupplier, "x")
	if p.PackName != EdiPackEdifactLiteV1 {
		t.Fatalf("pack=%s", p.PackName)
	}
}
