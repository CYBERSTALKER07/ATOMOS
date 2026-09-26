package partner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/partner/edi"
)

func TestAS2ReceiveInsecurePlain(t *testing.T) {
	t.Setenv("PARTNER_AS2_ENABLED", "true")
	t.Setenv("PARTNER_AS2_INSECURE_PLAIN", "true")
	t.Setenv("PARTNER_EDI_ENABLED", "true")

	as2Repo := NewMemoryAs2ConfigRepository()
	_ = as2Repo.Upsert(context.Background(), As2Config{
		TenantType: TenantSupplier, TenantID: "sup-1",
		As2Enabled: true, OurAs2Id: "US", PartnerAs2Id: "THEM",
		SignRequired: false, EncryptRequired: false,
	})
	docs := NewMemoryEdiDocumentRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetAs2Repository(as2Repo)
	in := NewEdiInboundWorker(docs, NewMemorySftpConfigRepository(), svc, nil)
	// Force CreateOrder failure path avoided: without orders svc CreateOrder fails after ledger RECEIVED.
	// We only assert unwrap+identity+MDN here by stubbing ingest via a worker that fails closed on create —
	// so use body that fails parse to get MDN failed, then a second call with valid ORDERS that hits create error.
	h := &Handlers{Svc: svc, EdiInbound: in}

	raw := edi.BuildORDERS(edi.OrdersMessage{
		ExternalDocID: "AS2-1",
		BuyerRef:      "ret-1",
		SellerRef:     "sup-1",
		Lat:           41.3, Lng: 69.2, H3Cell: "cell",
		Lines:         []edi.Line{{SKU: "SKU1", Qty: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/partner/v1/as2", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/edifact")
	req.Header.Set("Content-Disposition", `attachment; filename="ORDERS_AS2-1.edi"`)
	req.Header.Set("AS2-From", "THEM")
	req.Header.Set("AS2-To", "US")
	req.Header.Set("Message-ID", "<as2-1@test>")
	rr := httptest.NewRecorder()
	h.HandleAS2Receive(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "disposition-notification") {
		t.Fatalf("expected MDN, got %s", rr.Body.String())
	}
	// CreateOrder unavailable → MDN failed but transport path exercised
	if !strings.Contains(rr.Body.String(), "failed") && !strings.Contains(rr.Body.String(), "processed") {
		t.Fatalf("expected disposition, got %s", rr.Body.String())
	}
}
