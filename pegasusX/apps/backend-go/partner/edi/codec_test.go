package edi

import (
	"strings"
	"testing"
	"time"
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

func TestBuildDESADV_SSCCPacking(t *testing.T) {
	o := OrderSnapshot{
		OrderID: "o1", RetailerID: "ret-1", SupplierID: "sup-1",
		ManifestID: "m1", Status: "LOADED", Currency: "UZS", TotalMinor: 150000,
		Lines: []Line{{SKU: "A", Qty: 2}},
		ShipUnits: []ShipUnit{
			{ManifestID: "m1", SSCC: "003761234567890123", OrderID: "o1", Sequence: 0, GTIN: "01234567890128"},
		},
	}
	body := BuildDESADV(o, "desadv-1")
	_, segs, err := ParseUNAFile(body)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"RFF": false, "CPS": false, "PAC": false, "GIN": false}
	ginBJ, ginBN := false, false
	for _, s := range segs {
		switch s.Tag {
		case "RFF":
			if s.Elem(0) == "PK:m1" {
				want["RFF"] = true
			}
		case "CPS":
			want["CPS"] = true
		case "PAC":
			want["PAC"] = true
		case "GIN":
			want["GIN"] = true
			if s.Elem(0) == "BJ" && s.Elem(1) == "003761234567890123" {
				ginBJ = true
			}
			if s.Elem(0) == "BN" && s.Elem(1) == "01234567890128" {
				ginBN = true
			}
		}
	}
	for tag, ok := range want {
		if !ok {
			t.Fatalf("missing %s in DESADV:\n%s", tag, body)
		}
	}
	if !ginBJ {
		t.Fatalf("missing GIN+BJ+SSCC:\n%s", body)
	}
	if !ginBN {
		t.Fatalf("missing GIN+BN+GTIN:\n%s", body)
	}
}

func TestBuildDESADV_NoShipUnitsOmitsPacking(t *testing.T) {
	o := OrderSnapshot{
		OrderID: "o1", RetailerID: "ret-1", SupplierID: "sup-1",
		Status: "LOADED", Lines: []Line{{SKU: "A", Qty: 1}},
	}
	body := BuildDESADV(o, "")
	if strings.Contains(body, "GIN+") || strings.Contains(body, "CPS+") {
		t.Fatalf("unexpected packing segments:\n%s", body)
	}
}

func TestBuildCONTRLAndAPERAK(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	contrl, err := BuildCONTRL("our-gln", "their-gln", "PO-1", true, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contrl, "CONTRL:D:96A:UN") || !strings.Contains(contrl, "UCI+") {
		t.Fatalf("contrl=%s", contrl)
	}
	aperak, err := BuildAPERAK("our-gln", "their-gln", "PO-1", "bad_sku", false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(aperak, "APERAK:D:96A:UN") || !strings.Contains(aperak, "FTX+") {
		t.Fatalf("aperak=%s", aperak)
	}
}

func TestParseORDRSPAndINVOIC(t *testing.T) {
	o := OrderSnapshot{
		OrderID: "o1", RetailerID: "ret-1", SupplierID: "sup-1",
		Status: "CONFIRMED", Currency: "UZS", TotalMinor: 1000,
		Lines: []Line{{SKU: "A", Qty: 1}},
	}
	raw := BuildORDRSP(o, "rsp-1")
	msg, err := ParseORDRSP(raw)
	if err != nil {
		t.Fatal(err)
	}
	if msg.ExternalDocID == "" || !msg.Accepted {
		t.Fatalf("msg=%+v", msg)
	}
	inv := BuildINVOIC(o, &InvoiceSnapshot{InvoiceID: "inv-1", PrincipalMinor: 1000, Currency: "UZS"}, "inv-ext")
	im, err := ParseINVOIC(inv)
	if err != nil {
		t.Fatal(err)
	}
	if im.ExternalDocID == "" {
		t.Fatalf("invoic=%+v", im)
	}
	dt, err := DetectDocType(raw)
	if err != nil || dt != DocTypeORDRSP {
		t.Fatalf("detect=%s err=%v", dt, err)
	}
}

func TestEDIBreadthRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"PRICAT", BuildPRICAT(PricatMessage{ExternalDocID: "PC-1", SellerRef: "sup-1", Lines: []CatalogLine{{SKU: "A", Name: "Item", PriceMinor: 100, Currency: "UZS"}}}), DocTypePRICAT},
		{"INVRPT", BuildINVRPT(InvrptMessage{ExternalDocID: "IV-1", SellerRef: "sup-1", Lines: []StockLine{{SKU: "A", QtyOnHand: 5, Warehouse: "wh-1"}}}), DocTypeINVRPT},
		{"SLSRPT", BuildSLSRPT(SlsrptMessage{ExternalDocID: "SL-1", BuyerRef: "ret-1", SellerRef: "sup-1", ReportDate: "2026-08-01", Lines: []SalesLine{{SKU: "A", Qty: 2}}}), DocTypeSLSRPT},
		{"RECADV", BuildRECADV(RecadvMessage{ExternalDocID: "RA-1", RefOrderID: "o1", BuyerRef: "ret-1", AcceptedQty: 3}), DocTypeRECADV},
		{"ORDCHG", BuildORDCHG(OrdchgMessage{ExternalDocID: "OC-1", RefOrderID: "o1", BuyerRef: "ret-1", Lines: []Line{{SKU: "A", Qty: 4}}}), DocTypeORDCHG},
		{"DELFOR", BuildDELFOR(DelforMessage{ExternalDocID: "DF-1", BuyerRef: "ret-1", DeliveryDate: "2026-09-01", Lines: []Line{{SKU: "A", Qty: 10}}}), DocTypeDELFOR},
		{"REMADV", BuildREMADV(RemadvMessage{ExternalDocID: "RM-1", RefInvoiceID: "inv-1", PaidMinor: 500, Currency: "UZS"}), DocTypeREMADV},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dt, err := DetectDocType(tc.body)
			if err != nil || dt != tc.want {
				t.Fatalf("detect=%s err=%v", dt, err)
			}
			switch tc.want {
			case DocTypePRICAT:
				if _, err := ParsePRICAT(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeINVRPT:
				if _, err := ParseINVRPT(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeSLSRPT:
				if _, err := ParseSLSRPT(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeRECADV:
				if _, err := ParseRECADV(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeORDCHG:
				if _, err := ParseORDCHG(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeDELFOR:
				if _, err := ParseDELFOR(tc.body); err != nil {
					t.Fatal(err)
				}
			case DocTypeREMADV:
				if _, err := ParseREMADV(tc.body); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
