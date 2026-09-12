package partner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestG5_PartiesConflictAndDLQ(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	p := Principal{TenantType: TenantSupplier, TenantID: "sup-g5", Scopes: []string{ScopeCatalogWrite}}
	r1, err := svc.UpsertParties(context.Background(), p, []PartyUpsertItem{
		{ExternalID: "R1", Role: "RETAILER", LegalName: "Shop", Version: 2},
	})
	if err != nil || r1[0].Action != "created" {
		t.Fatalf("create: %+v err=%v", r1, err)
	}
	r2, err := svc.UpsertParties(context.Background(), p, []PartyUpsertItem{
		{ExternalID: "R1", Role: "RETAILER", LegalName: "Shop2", Version: 1},
	})
	if err != nil || r2[0].Action != "conflict" {
		t.Fatalf("conflict: %+v err=%v", r2, err)
	}
	dlq, err := svc.ListMasterDataDLQ(context.Background(), p, 10)
	if err != nil || len(dlq) == 0 {
		t.Fatalf("dlq: %+v err=%v", dlq, err)
	}
}

func TestG5_ASNIdempotent(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	p := Principal{TenantType: TenantSupplier, TenantID: "sup-asn", Scopes: []string{ScopeInventoryWrite}}
	req := ASNRequest{
		ExternalASNID: "ASN-1",
		WarehouseID:   "wh-1",
		Lines:         []ASNLine{{SKU: "sku1", Qty: 5}},
	}
	a, err := svc.ProcessInboundASN(context.Background(), p, req)
	if err != nil || a.Status != "accepted" {
		t.Fatalf("first: %+v err=%v", a, err)
	}
	b, err := svc.ProcessInboundASN(context.Background(), p, req)
	if err != nil || b.Status != "duplicate" {
		t.Fatalf("dup: %+v err=%v", b, err)
	}
}

func TestG5_EdiProfileOutboundGate(t *testing.T) {
	docs := NewMemoryEdiDocumentRepository()
	sftp := NewMemorySftpConfigRepository()
	w := NewEdiOutboundWorker(docs, sftp, nil, nil, nil)
	prof := NewMemoryEdiProfiles()
	_ = prof.Upsert(context.Background(), EdiProfile{
		TenantType: TenantSupplier, TenantID: "sup-out",
		EnabledDocTypes: []string{EdiDocORDRSP},
		PackName:        EdiPackEdifactLiteV1,
	})
	w.SetEdiProfiles(prof)
	// DESADV disabled — enqueue is no-op success
	if err := w.EnqueueOutbound(context.Background(), TenantSupplier, "sup-out", EdiDocDESADV, "o1:DESADV", "o1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := docs.GetByExternal(context.Background(), TenantSupplier, "sup-out", EdiDirectionOut, EdiDocDESADV, "o1:DESADV"); ok {
		t.Fatal("DESADV should not be queued")
	}
	if err := w.EnqueueOutbound(context.Background(), TenantSupplier, "sup-out", EdiDocORDRSP, "o1:ORDRSP", "o1"); err != nil {
		t.Fatal(err)
	}
}

func TestG5_ProfileHandlers(t *testing.T) {
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	h := &Handlers{Svc: svc}
	body, _ := json.Marshal(EdiProfile{
		EnabledDocTypes: []string{EdiDocORDERS},
		PackName:        EdiPackEdifactLiteV1,
		RequireCONTRL:   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/partner/v1/edi/profile", bytes.NewReader(body))
	req = req.WithContext(WithPrincipal(req.Context(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-p", Scopes: []string{ScopeExportsRead},
	}))
	rr := httptest.NewRecorder()
	h.HandlePutEdiProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rr.Code, rr.Body.String())
	}
	req2 := httptest.NewRequest(http.MethodGet, "/partner/v1/edi/profile", nil)
	req2 = req2.WithContext(WithPrincipal(context.Background(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-p", Scopes: []string{ScopeExportsRead},
	}))
	rr2 := httptest.NewRecorder()
	h.HandleGetEdiProfile(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("get status=%d", rr2.Code)
	}
}
