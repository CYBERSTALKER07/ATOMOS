package edi

import (
	"strings"
	"testing"
)

func TestORDERSRoundTrip(t *testing.T) {
	raw := BuildORDERS(OrdersMessage{
		ExternalDocID: "PO-1001",
		BuyerRef:      "ret-1",
		SellerRef:     "sup-1",
		Lat:           41.3,
		Lng:           69.2,
		H3Cell:        "8b2945c0c2cffff",
		DeliveryDate:  "2026-08-10",
		Lines:         []Line{{SKU: "SSMR-SKU-1", Qty: 3}},
	})
	msg, err := ParseORDERS(raw)
	if err != nil {
		t.Fatal(err)
	}
	if msg.ExternalDocID != "PO-1001" || msg.BuyerRef != "ret-1" || msg.SellerRef != "sup-1" {
		t.Fatalf("ids=%+v", msg)
	}
	if len(msg.Lines) != 1 || msg.Lines[0].SKU != "SSMR-SKU-1" || msg.Lines[0].Qty != 3 {
		t.Fatalf("lines=%+v", msg.Lines)
	}
	if msg.H3Cell != "8b2945c0c2cffff" {
		t.Fatalf("h3=%s", msg.H3Cell)
	}
}

func TestOutboundBuilders(t *testing.T) {
	o := OrderSnapshot{
		OrderID: "o1", RetailerID: "ret-1", SupplierID: "sup-1",
		Status: "LOADED", Currency: "UZS", TotalMinor: 150000,
		Lines: []Line{{SKU: "A", Qty: 2}},
	}
	for _, body := range []string{
		BuildORDRSP(o, ""),
		BuildDESADV(o, ""),
		BuildINVOIC(o, &InvoiceSnapshot{InvoiceID: "inv-1", PrincipalMinor: 150000, Currency: "UZS", DueDate: "2026-09-01"}, ""),
	} {
		if !strings.Contains(body, "UNA:+.? '") {
			t.Fatalf("missing UNA: %s", body[:40])
		}
		_, segs, err := ParseUNAFile(body)
		if err != nil || len(segs) < 4 {
			t.Fatalf("parse err=%v segs=%d", err, len(segs))
		}
	}
}
