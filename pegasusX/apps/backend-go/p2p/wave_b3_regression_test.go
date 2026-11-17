package p2p_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
)

func TestBuildThreeWayMatchGLLines_ZeroReceiptNoPhantomGRNI(t *testing.T) {
	po := p2p.PurchaseOrder{PoID: "po-zero-1", VendorID: "vnd-1"}
	inv := p2p.VendorInvoice{VendorInvoiceID: "vi-1", SubtotalMinor: 100_000, VatMinor: 12_000, TotalMinor: 112_000}

	lines, err := p2p.BuildThreeWayMatchGLLines(po, inv, 0)
	if err != nil {
		t.Fatalf("BuildThreeWayMatchGLLines zero receipt: %v", err)
	}
	var drTotal, crTotal int64
	for _, l := range lines {
		if l.AccountCode == p2p.AccountGRNI {
			t.Fatalf("unexpected phantom GRNI leg on zero receipt: %+v", l)
		}
		if l.Direction == p2p.DirectionDebit {
			drTotal += l.AmountMinor
		} else {
			crTotal += l.AmountMinor
		}
	}
	if drTotal != 112_000 || crTotal != 112_000 {
		t.Fatalf("unbalanced GL lines: dr=%d cr=%d", drTotal, crTotal)
	}

	if _, err := p2p.BuildThreeWayMatchGLLines(po, inv, -10); !errors.Is(err, p2p.ErrInvalidInput) {
		t.Fatalf("negative receiptNetMinor err=%v, want ErrInvalidInput", err)
	}
}

func TestListPurchaseOrders_BatchedLinesParity_LiveSpanner(t *testing.T) {
	client := openP2PSpannerClient(t)
	ctx := context.Background()
	tenantID := "sup-p2p-batch-" + uuid.NewString()[:8]

	glSvc := gl.NewService(client)
	_ = glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS")
	svc := p2p.NewSpannerService(client, p2p.ServiceConfig{GLPoster: &nonTxnGLBridge{svc: glSvc}})

	po1, err := svc.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:  tenantID,
		VendorID:  "vnd-b1",
		Lines:     []p2p.PurchaseOrderLineInput{{Sku: "SKU-B1-A", QtyOrdered: 2, UnitCostMinor: 10_000, VatBps: 1200}, {Sku: "SKU-B1-B", QtyOrdered: 3, UnitCostMinor: 5_000, VatBps: 1200}},
		CreatedBy: "maker-1",
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder 1: %v", err)
	}
	po2, err := svc.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:  tenantID,
		VendorID:  "vnd-b2",
		Lines:     []p2p.PurchaseOrderLineInput{{Sku: "SKU-B2-A", QtyOrdered: 5, UnitCostMinor: 7_000, VatBps: 0}},
		CreatedBy: "maker-1",
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder 2: %v", err)
	}

	list, err := svc.ListPurchaseOrders(ctx, p2p.ListFilter{TenantID: tenantID, Limit: 10})
	if err != nil {
		t.Fatalf("ListPurchaseOrders: %v", err)
	}
	linesByPO := map[string]int{}
	for _, po := range list {
		linesByPO[po.PoID] = len(po.Lines)
	}
	if linesByPO[po1.PurchaseOrder.PoID] != 2 || linesByPO[po2.PurchaseOrder.PoID] != 1 {
		t.Fatalf("batched PO line counts=%v, want po1=2 po2=1", linesByPO)
	}
}
