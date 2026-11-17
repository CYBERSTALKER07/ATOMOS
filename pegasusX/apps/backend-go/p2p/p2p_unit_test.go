package p2p_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
)

type stubGLPoster struct {
	mu      sync.Mutex
	entries []p2p.GLPostInput
	nextID  int
}

func (g *stubGLPoster) PostJournal(_ context.Context, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	var sumDebit, sumCredit int64
	for _, l := range in.Lines {
		if l.Direction == p2p.DirectionDebit {
			sumDebit += l.AmountMinor
		} else {
			sumCredit += l.AmountMinor
		}
	}
	if sumDebit <= 0 || sumDebit != sumCredit {
		return p2p.GLPostResult{}, errors.New("unbalanced gl entry")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, ex := range g.entries {
		if ex.IdempotencyKey == in.IdempotencyKey {
			return p2p.GLPostResult{EntryID: "gle_idem_" + in.SourceID, PeriodID: in.PeriodID, AlreadyPosted: true}, nil
		}
	}
	g.nextID++
	g.entries = append(g.entries, in)
	return p2p.GLPostResult{EntryID: "gle_" + in.SourceID, PeriodID: in.PeriodID}, nil
}

type stubApprovalChecker struct {
	mu          sync.Mutex
	require     bool
	approved    bool
	approverIDs []string
	err         error
	calls       []p2p.ApprovalCheckInput
}

func (a *stubApprovalChecker) CheckOrSubmitApproval(_ context.Context, in p2p.ApprovalCheckInput) (p2p.ApprovalCheckResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, in)
	if a.err != nil {
		return p2p.ApprovalCheckResult{}, a.err
	}
	if in.ApprovalID != "" {
		return p2p.ApprovalCheckResult{
			Required:          true,
			Approved:          a.approved,
			ApprovalRequestID: in.ApprovalID,
			Status:            "APPROVED",
			ApproverIDs:       a.approverIDs,
		}, nil
	}
	if !a.require && !in.ForceCreate {
		return p2p.ApprovalCheckResult{Required: false}, nil
	}
	return p2p.ApprovalCheckResult{
		Required:          true,
		Approved:          false,
		ApprovalRequestID: "apr_" + in.ObjectID,
		Status:            "PENDING",
	}, nil
}

type stubAuditRecorder struct {
	mu     sync.Mutex
	events []p2p.AuditRecordInput
}

func (a *stubAuditRecorder) RecordEvent(_ context.Context, ev p2p.AuditRecordInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, ev)
	return nil
}

func (a *stubAuditRecorder) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev p2p.AuditRecordInput) error {
	return a.RecordEvent(context.Background(), ev)
}

func assertNoFloatFields(t *testing.T, v any) {
	t.Helper()
	var walk func(typ reflect.Type, path string)
	seen := make(map[reflect.Type]bool)
	walk = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			typ = typ.Elem()
		}
		if typ.Kind() == reflect.Float32 || typ.Kind() == reflect.Float64 {
			t.Fatalf("forbidden floating-point field %s (%s)", path, typ.Kind())
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		if typ == reflect.TypeOf(time.Time{}) {
			return
		}
		seen[typ] = true
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeOf(v), reflect.TypeOf(v).Name())
}

func TestP2PStructsHaveZeroFloatFields(t *testing.T) {
	assertNoFloatFields(t, p2p.ProcessTemplate{})
	assertNoFloatFields(t, p2p.ProcessInstance{})
	assertNoFloatFields(t, p2p.PurchaseOrder{})
	assertNoFloatFields(t, p2p.GoodsReceipt{})
	assertNoFloatFields(t, p2p.VendorInvoice{})
	assertNoFloatFields(t, p2p.ThreeWayMatchResult{})
	assertNoFloatFields(t, p2p.ProcessMiningReport{})
}

func TestShippedUZTemplatesAndInstances(t *testing.T) {
	ctx := context.Background()
	repo := p2p.NewMemoryRepository()
	auditRec := &stubAuditRecorder{}
	svc := p2p.NewService(repo, p2p.ServiceConfig{Audit: auditRec})

	templates, err := svc.ListTemplates(ctx, "sup-1", "")
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	wantCodes := map[string]bool{
		p2p.TemplateO2CDirectUZ:       false,
		p2p.TemplateO2CPreorderUZ:     false,
		p2p.TemplateP2PStdUZ:          false,
		p2p.TemplateReturnsSupplierUZ: false,
	}
	for _, tpl := range templates {
		if _, ok := wantCodes[tpl.Code]; ok {
			wantCodes[tpl.Code] = true
		}
		if len(tpl.Steps) == 0 {
			t.Fatalf("template %s has zero steps", tpl.Code)
		}
	}
	for code, found := range wantCodes {
		if !found {
			t.Fatalf("missing shipped template %s", code)
		}
	}

	custom, err := svc.CreateTemplate(ctx, p2p.CreateTemplateInput{
		TenantID: "sup-1",
		Code:     "P2P-COLDCHAIN-UZ",
		Name:     "Cold Chain P2P",
		Domain:   p2p.DomainP2P,
		Steps: []p2p.ProcessStep{
			{Code: "PO_CREATED", Name: "Created", SLASeconds: 1800},
			{Code: "QC_INSPECTED", Name: "QC Check", SLASeconds: 3600},
			{Code: "MATCHED_3WAY", Name: "Matched", SLASeconds: 7200},
		},
		ActorID: "owner-1",
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if _, err := svc.CreateTemplate(ctx, p2p.CreateTemplateInput{
		TenantID: "sup-1",
		Code:     "P2P-COLDCHAIN-UZ",
		Name:     "Duplicate",
		Domain:   p2p.DomainP2P,
		Steps:    []p2p.ProcessStep{{Code: "START", Name: "Start", SLASeconds: 60}},
	}); !errors.Is(err, p2p.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate template code, got %v", err)
	}

	inst, err := svc.StartOrAdvanceInstance(ctx, p2p.StartOrAdvanceInstanceInput{
		TenantID:     "sup-1",
		TemplateCode: custom.Code,
		ObjectType:   "ColdPO",
		ObjectID:     "cpo-1",
		ActorID:      "user-1",
	})
	if err != nil || inst.CurrentStep != "PO_CREATED" || inst.Status != p2p.InstanceStatusActive {
		t.Fatalf("start instance: err=%v inst=%+v", err, inst)
	}
	inst, err = svc.StartOrAdvanceInstance(ctx, p2p.StartOrAdvanceInstanceInput{
		TenantID:     "sup-1",
		InstanceID:   inst.InstanceID,
		TemplateCode: custom.Code,
		NextStep:     "MATCHED_3WAY",
		ActorID:      "user-2",
	})
	if err != nil || inst.Status != p2p.InstanceStatusCompleted || inst.CompletedAt == nil {
		t.Fatalf("advance instance to final step: err=%v inst=%+v", err, inst)
	}
}

func TestP2PLifecycle_ApprovalSoD_Receipt_GL_ThreeWayMatch_ESF_Tolerance(t *testing.T) {
	ctx := context.Background()
	repo := p2p.NewMemoryRepository()
	glStub := &stubGLPoster{}
	appStub := &stubApprovalChecker{approved: true, approverIDs: []string{"fin-approver"}}
	auditRec := &stubAuditRecorder{}
	baseNow := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	curNow := baseNow

	svc := p2p.NewService(repo, p2p.ServiceConfig{
		Repo:                     repo,
		GLPoster:                 glStub,
		Approvals:                appStub,
		Audit:                    auditRec,
		POApprovalThresholdMinor: 500_000,
		Now:                      func() time.Time { return curNow },
	})

	// 1. High-value PO triggers PENDING_APPROVAL.
	poRes, err := svc.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:       "sup-p2p",
		EntityID:       "ent-uz-main",
		VendorID:       "vnd-1",
		VendorName:     "Tashkent Flour LLC",
		VendorTin:      "301234567",
		Currency:       "UZS",
		IdempotencyKey: "po-idem-1",
		Lines: []p2p.PurchaseOrderLineInput{
			{Sku: "FLOUR-50KG", ProductName: "Flour 50kg", QtyOrdered: 10, UnitCostMinor: 100_000, VatBps: 1200},
		},
		CreatedBy: "acc-maker",
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder: %v", err)
	}
	if !poRes.ApprovalRequired || poRes.PurchaseOrder.Status != p2p.POStatusPendingApproval {
		t.Fatalf("expected PENDING_APPROVAL PO, got %+v", poRes)
	}
	if poRes.PurchaseOrder.SubtotalMinor != 1_000_000 || poRes.PurchaseOrder.VatMinor != 120_000 || poRes.PurchaseOrder.TotalMinor != 1_120_000 {
		t.Fatalf("unexpected PO totals: %+v", poRes.PurchaseOrder)
	}

	// Receipt against PENDING_APPROVAL PO must be rejected.
	if _, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:    "sup-p2p",
		PoID:        poRes.PurchaseOrder.PoID,
		WarehouseID: "wh-1",
		Lines:       []p2p.GoodsReceiptLineInput{{Sku: "FLOUR-50KG", QtyReceived: 5}},
		ReceivedBy:  "wh-1",
	}); !errors.Is(err, p2p.ErrPONotApproved) {
		t.Fatalf("expected ErrPONotApproved, got %v", err)
	}

	// Self-approval SoD must fail.
	if _, err := svc.ApprovePurchaseOrder(ctx, p2p.ApprovePurchaseOrderInput{
		TenantID:   "sup-p2p",
		PoID:       poRes.PurchaseOrder.PoID,
		ApprovalID: poRes.ApprovalRequestID,
		ActorID:    "acc-maker",
	}); !errors.Is(err, p2p.ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation on self-approval, got %v", err)
	}

	// Distinct approver succeeds.
	curNow = curNow.Add(30 * time.Minute)
	approvedPO, err := svc.ApprovePurchaseOrder(ctx, p2p.ApprovePurchaseOrderInput{
		TenantID:   "sup-p2p",
		PoID:       poRes.PurchaseOrder.PoID,
		ApprovalID: poRes.ApprovalRequestID,
		ActorID:    "fin-approver",
	})
	if err != nil || approvedPO.Status != p2p.POStatusApproved {
		t.Fatalf("ApprovePurchaseOrder: err=%v po=%+v", err, approvedPO)
	}

	// 2. Partial Receipt (6 of 10) -> PARTIAL_RECEIVED + balanced GL (INVENTORY / GRNI = 600,000).
	curNow = curNow.Add(2 * time.Hour)
	rec1, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       "sup-p2p",
		PoID:           approvedPO.PoID,
		WarehouseID:    "wh-1",
		PeriodID:       "2026-11",
		IdempotencyKey: "grn-idem-1",
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "FLOUR-50KG", QtyReceived: 6}},
		ReceivedBy:     "wh-user",
	})
	if err != nil {
		t.Fatalf("PostGoodsReceipt 1: %v", err)
	}
	if rec1.PurchaseOrder.Status != p2p.POStatusPartialReceived || rec1.Receipt.TotalReceivedMinor != 600_000 {
		t.Fatalf("unexpected partial receipt result: %+v", rec1)
	}
	if len(glStub.entries) != 1 || glStub.entries[0].Lines[0].AccountCode != p2p.AccountInventory || glStub.entries[0].Lines[1].AccountCode != p2p.AccountGRNI {
		t.Fatalf("expected INVENTORY/GRNI GL entry, got %+v", glStub.entries)
	}

	// Over-receipt (5 remaining when only 4 open) -> ErrQtyExceeded.
	if _, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:    "sup-p2p",
		PoID:        approvedPO.PoID,
		WarehouseID: "wh-1",
		Lines:       []p2p.GoodsReceiptLineInput{{Sku: "FLOUR-50KG", QtyReceived: 5}},
		ReceivedBy:  "wh-user",
	}); !errors.Is(err, p2p.ErrQtyExceeded) {
		t.Fatalf("expected ErrQtyExceeded, got %v", err)
	}

	// Complete receipt (4 of 4 remaining) -> RECEIVED, and idempotent replay returns same receipt without error.
	rec2, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       "sup-p2p",
		PoID:           approvedPO.PoID,
		WarehouseID:    "wh-1",
		PeriodID:       "2026-11",
		IdempotencyKey: "grn-idem-2",
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "FLOUR-50KG", QtyReceived: 4}},
		ReceivedBy:     "wh-user",
	})
	if err != nil || rec2.PurchaseOrder.Status != p2p.POStatusReceived {
		t.Fatalf("PostGoodsReceipt 2: err=%v status=%s", err, rec2.PurchaseOrder.Status)
	}
	rec2Replay, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       "sup-p2p",
		PoID:           approvedPO.PoID,
		WarehouseID:    "wh-1",
		PeriodID:       "2026-11",
		IdempotencyKey: "grn-idem-2",
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "FLOUR-50KG", QtyReceived: 4}},
		ReceivedBy:     "wh-user",
	})
	if err != nil || rec2Replay.Receipt.ReceiptID != rec2.Receipt.ReceiptID {
		t.Fatalf("idempotent receipt replay failed: err=%v replay=%+v", err, rec2Replay)
	}

	// 3. Discrepancy VendorInvoice (overbilled 1,050,000 + 126,000 VAT = 1,176,000 vs 1,120,000).
	badInv, err := svc.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:      "sup-p2p",
		PoID:          approvedPO.PoID,
		SubtotalMinor: 1_050_000,
		VatMinor:      126_000,
		TotalMinor:    1_176_000,
		CreatedBy:     "acc-maker",
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice bad: %v", err)
	}

	// SoD check on ThreeWayMatch: creator cannot verify own invoice.
	if _, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        "sup-p2p",
		VendorInvoiceID: badInv.VendorInvoiceID,
		VerifiedBy:      "acc-maker",
	}); !errors.Is(err, p2p.ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation on ThreeWayMatch, got %v", err)
	}

	// Mismatch marks DISCREPANCY with exact variance & zero GL post.
	glCountBefore := len(glStub.entries)
	discRes, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        "sup-p2p",
		VendorInvoiceID: badInv.VendorInvoiceID,
		VerifiedBy:      "fin-verifier",
	})
	if err != nil {
		t.Fatalf("ThreeWayMatch discrepancy: %v", err)
	}
	if discRes.Matched || discRes.MatchStatus != p2p.MatchStatusDiscrepancy || discRes.VarianceMinor != 56_000 || discRes.VarianceReason == "" {
		t.Fatalf("expected DISCREPANCY variance=56000, got %+v", discRes)
	}
	if len(glStub.entries) != glCountBefore {
		t.Fatalf("expected zero GL postings on discrepancy")
	}

	// Override without approval_id fails closed.
	if _, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:            "sup-p2p",
		VendorInvoiceID:     badInv.VendorInvoiceID,
		OverrideDiscrepancy: true,
		VerifiedBy:          "fin-verifier",
	}); !errors.Is(err, p2p.ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired for discrepancy override without approval_id, got %v", err)
	}

	// 4. Inbound ESF + VendorInvoice within 50 bps tolerance (1,001,000 + 120,120 = 1,121,120 vs 1,120,000 -> 10 bps).
	repo.SeedInboundESF(p2p.InboundESFDoc{
		EntityID:        "sup-p2p",
		EsfID:           "esf-in-1",
		Direction:       "INBOUND",
		PoID:            approvedPO.PoID,
		CounterpartyTin: "301234567",
		TotalMinor:      1_121_120,
		VatMinor:        120_120,
		VatRateBps:      1200,
		Status:          "ACCEPTED",
	})
	goodInv, err := svc.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:      "sup-p2p",
		PoID:          approvedPO.PoID,
		EsfDocID:      "esf-in-1",
		SubtotalMinor: 1_001_000,
		VatMinor:      120_120,
		TotalMinor:    1_121_120,
		CreatedBy:     "acc-maker",
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice good: %v", err)
	}

	curNow = curNow.Add(45 * time.Minute)
	matchRes, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        "sup-p2p",
		VendorInvoiceID: goodInv.VendorInvoiceID,
		PeriodID:        "2026-11",
		ToleranceBps:    50, // 0.5% tolerance allows +1000 minor on 1,000,000
		VerifiedBy:      "fin-verifier",
	})
	if err != nil {
		t.Fatalf("ThreeWayMatch good: %v", err)
	}
	if !matchRes.Matched || matchRes.MatchStatus != p2p.MatchStatusMatched3Way || matchRes.JournalEntryID == "" {
		t.Fatalf("expected MATCHED_3WAY with JournalEntryID, got %+v", matchRes)
	}
	if matchRes.ProcessInstance == nil || matchRes.ProcessInstance.Status != p2p.InstanceStatusCompleted {
		t.Fatalf("expected ProcessInstance COMPLETED, got %+v", matchRes.ProcessInstance)
	}

	// Verify GL lines cleared GRNI + TAX:VAT:INPUT + COGS variance -> AP:VENDOR.
	lastGL := glStub.entries[len(glStub.entries)-1]
	foundAccounts := make(map[string]int64)
	for _, l := range lastGL.Lines {
		foundAccounts[l.AccountCode] = l.AmountMinor
	}
	if foundAccounts[p2p.AccountGRNI] != 1_000_000 || foundAccounts[p2p.AccountVatInput] != 120_120 ||
		foundAccounts[p2p.AccountPriceVariance] != 1_000 || foundAccounts[p2p.AccountVendorPayable] != 1_121_120 {
		t.Fatalf("unexpected 3-way match GL legs: %+v", lastGL.Lines)
	}

	// Verify outbox emitted p2p.po.created, p2p.receipt.posted, p2p.three_way_match.completed.
	evSeen := make(map[string]bool)
	for _, ev := range repo.OutboxEvents() {
		raw := string(ev.Payload)
		for _, want := range []string{p2p.EventP2PPOCreated, p2p.EventP2PReceiptPosted, p2p.EventP2PThreeWayMatchComplete} {
			if strings.Contains(raw, want) {
				evSeen[want] = true
			}
		}
	}
	for _, want := range []string{p2p.EventP2PPOCreated, p2p.EventP2PReceiptPosted, p2p.EventP2PThreeWayMatchComplete} {
		if !evSeen[want] {
			t.Fatalf("missing outbox event %s", want)
		}
	}

	// Repeat 3-way match on matched invoice must return ErrConflict.
	if _, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        "sup-p2p",
		VendorInvoiceID: goodInv.VendorInvoiceID,
		VerifiedBy:      "fin-verifier",
	}); !errors.Is(err, p2p.ErrConflict) {
		t.Fatalf("expected ErrConflict on re-matching MATCHED_3WAY invoice, got %v", err)
	}
}

func TestProcessMining_EmptyAndRealAggregations(t *testing.T) {
	ctx := context.Background()
	repo := p2p.NewMemoryRepository()
	svc := p2p.NewService(repo, p2p.ServiceConfig{})

	// 1. Empty tenant must never fabricate metrics.
	emptyReport, err := svc.GetProcessMining(ctx, p2p.MiningFilter{TenantID: "sup-empty"})
	if err != nil {
		t.Fatalf("GetProcessMining empty: %v", err)
	}
	if !emptyReport.Empty || emptyReport.TotalCases != 0 || emptyReport.TotalTransitions != 0 ||
		emptyReport.BottleneckStage != nil || len(emptyReport.Edges) != 0 {
		t.Fatalf("expected strictly empty mining report, got %+v", emptyReport)
	}

	// 2. Seed real O2C OrderStatusTransitions (SLA for CONFIRMED=3600s, DISPATCHED=28800s).
	t0 := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	repo.SeedOrderCase(p2p.MemoryOrderCase{
		TenantID:  "sup-mine",
		OrderID:   "ord-1",
		CreatedAt: t0,
		Transitions: []p2p.MemoryOrderTransition{
			{PreviousStatus: "PENDING", NewStatus: "CONFIRMED", CreatedAt: t0.Add(1000 * time.Second)},
			{PreviousStatus: "CONFIRMED", NewStatus: "IN_TRANSIT", CreatedAt: t0.Add(35000 * time.Second)}, // 34,000s > 28,800s SLA breach!
		},
	})
	repo.SeedOrderCase(p2p.MemoryOrderCase{
		TenantID:  "sup-mine",
		OrderID:   "ord-2",
		CreatedAt: t0,
		Transitions: []p2p.MemoryOrderTransition{
			{PreviousStatus: "PENDING", NewStatus: "CONFIRMED", CreatedAt: t0.Add(2000 * time.Second)},
			{PreviousStatus: "CONFIRMED", NewStatus: "IN_TRANSIT", CreatedAt: t0.Add(6000 * time.Second)}, // 4,000s OK
		},
	})

	report, err := svc.GetProcessMining(ctx, p2p.MiningFilter{TenantID: "sup-mine", Domain: p2p.DomainO2C})
	if err != nil {
		t.Fatalf("GetProcessMining populated: %v", err)
	}
	if report.Empty || report.TotalCases != 2 || report.TotalTransitions != 4 || report.SLABreachCount != 1 {
		t.Fatalf("unexpected mining summary: %+v", report)
	}
	if report.BottleneckStage == nil || report.BottleneckStage.FromStep != "CONFIRMED" || report.BottleneckStage.ToStep != "DISPATCHED" {
		t.Fatalf("expected CONFIRMED->DISPATCHED bottleneck, got %+v", report.BottleneckStage)
	}
	if report.BottleneckStage.MedianDurationSec != 19000 || report.BottleneckStage.P95DurationSec != 34000 {
		t.Fatalf("expected median=19000 p95=34000, got %+v", report.BottleneckStage)
	}
}
