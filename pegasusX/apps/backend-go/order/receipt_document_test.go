package order

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPegasusPayloadHasBranding(t *testing.T) {
	p := PegasusReceiptProvider{PublicBaseURL: "https://api.example.test"}
	res, err := p.CreateReceipt(t.Context(), FiscalCreateRequest{
		AttemptID:     "att-brand-1",
		OrderID:       "ord-1",
		SupplierID:    "sup-1",
		RetailerID:    "ret-1",
		AmountMinor:   125000,
		Currency:      "UZS",
		PaymentMethod: "CASH",
		LineItems: []LineItem{
			{SKU: "sku-a", Name: "Cola 1L", Quantity: 2, UnitPrice: 62500},
		},
	})
	if err != nil {
		t.Fatalf("CreateReceipt: %v", err)
	}
	if !strings.HasPrefix(res.FiscalReceiptID, "PX-RCPT-") {
		t.Fatalf("receipt id = %q", res.FiscalReceiptID)
	}
	if !strings.Contains(res.FiscalQR, "format=html") {
		t.Fatalf("qr should target HTML: %s", res.FiscalQR)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.RawPayload, &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload["legal_class"] != "platform_receipt" {
		t.Fatalf("legal_class = %v", payload["legal_class"])
	}
	if payload["tax_ofd"] != false {
		t.Fatalf("tax_ofd should be false")
	}
	if payload["country_code"] != "UZ" {
		t.Fatalf("country_code = %v", payload["country_code"])
	}
	if payload["company_name"] == "" {
		t.Fatal("company_name missing")
	}
	branding, _ := payload["branding"].(map[string]any)
	if branding == nil || branding["style"] != "pegasus_settlement_v1" {
		t.Fatalf("branding = %#v", payload["branding"])
	}
}

func TestPegasusPayloadKZCountry(t *testing.T) {
	p := PegasusReceiptProvider{PublicBaseURL: "https://api.example.test"}
	res, err := p.CreateReceipt(t.Context(), FiscalCreateRequest{
		AttemptID:   "att-kz-1",
		OrderID:     "ord-kz",
		AmountMinor: 100,
		Currency:    "KZT",
	})
	if err != nil {
		t.Fatalf("CreateReceipt: %v", err)
	}
	var payload map[string]any
	_ = json.Unmarshal(res.RawPayload, &payload)
	if payload["country_code"] != "KZ" {
		t.Fatalf("country_code = %v, want KZ", payload["country_code"])
	}
}

func TestBuildAndRenderReceiptHTMLPDF(t *testing.T) {
	p := PegasusReceiptProvider{PublicBaseURL: "https://api.example.test"}
	res, err := p.CreateReceipt(t.Context(), FiscalCreateRequest{
		AttemptID:     "att-render-1",
		OrderID:       "ord-render",
		SupplierID:    "sup-1",
		RetailerID:    "ret-1",
		AmountMinor:   99900,
		Currency:      "UZS",
		PaymentMethod: "CARD",
		LineItems: []LineItem{
			{SKU: "sku-1", Name: "Water", Quantity: 3, UnitPrice: 33300},
		},
	})
	if err != nil {
		t.Fatalf("CreateReceipt: %v", err)
	}
	fr := FiscalReceiptRow{
		OrderID:         "ord-render",
		AttemptID:       "att-render-1",
		SupplierID:      "sup-1",
		RetailerID:      "ret-1",
		Provider:        FiscalProviderPegasus,
		Status:          FiscalAttemptSuccess,
		FiscalReceiptID: res.FiscalReceiptID,
		FiscalQR:        res.FiscalQR,
		AmountMinor:     99900,
		Currency:        "UZS",
		PaymentMethod:   "CARD",
		ProviderPayload: res.RawPayload,
		CreatedAt:       time.Now().UTC(),
	}
	doc := BuildReceiptDocument(fr, nil, PartyCopyRetailer)
	if doc.PartyLabel == "" || doc.CompanyName == "" {
		t.Fatalf("doc incomplete: %+v", doc)
	}
	if len(doc.LineItems) != 1 {
		t.Fatalf("line items = %d", len(doc.LineItems))
	}
	html, err := RenderReceiptHTML(doc)
	if err != nil {
		t.Fatalf("html: %v", err)
	}
	hs := string(html)
	if !strings.Contains(hs, "PEGASUS") && !strings.Contains(hs, doc.CompanyName) {
		t.Fatalf("html missing brand")
	}
	if !strings.Contains(hs, "ord-render") {
		t.Fatalf("html missing order id")
	}
	if !strings.Contains(hs, "Water") {
		t.Fatalf("html missing line item")
	}
	if !strings.Contains(hs, "Buyer / Retailer copy") {
		t.Fatalf("html missing party copy")
	}
	pdf, err := RenderReceiptPDF(doc)
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	if len(pdf) < 100 || !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("pdf invalid: len=%d prefix=%q", len(pdf), string(pdf[:min(8, len(pdf))]))
	}
}

func TestAuthorizeReceiptParty(t *testing.T) {
	o := Order{OrderID: "o1", RetailerID: "ret-1", SupplierID: "sup-1"}
	if !authorizeReceiptParty(auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer, SupplierID: "sup-1"}, o, PartyCopyRetailer) {
		t.Fatal("retailer should access own receipt")
	}
	if authorizeReceiptParty(auth.Claims{Subject: "ret-2", Role: auth.RoleRetailer, SupplierID: "sup-1"}, o, PartyCopyRetailer) {
		t.Fatal("other retailer forbidden")
	}
	if !authorizeReceiptParty(auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}, o, PartyCopySupplier) {
		t.Fatal("supplier admin should access")
	}
	if !authorizeReceiptParty(auth.Claims{Subject: "wh-1", Role: auth.RoleWarehouse, SupplierID: "sup-1"}, o, PartyCopyWarehouse) {
		t.Fatal("warehouse should access")
	}
	if authorizeReceiptParty(auth.Claims{Subject: "wh-1", Role: auth.RoleWarehouse, SupplierID: "sup-other"}, o, PartyCopyWarehouse) {
		t.Fatal("cross-supplier warehouse forbidden")
	}
}
