package p2p_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openP2PSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner P2P tests")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	for _, tbl := range []string{"ProcessTemplates", "ProcessInstances", "PurchaseOrders", "PurchaseOrderLines", "GoodsReceipts", "VendorInvoices"} {
		ro := client.ReadOnlyTransaction()
		var key spanner.Key
		var col string
		if tbl == "PurchaseOrderLines" {
			key = spanner.Key{"__probe__", "__probe__", int64(1)}
			col = "Sku"
		} else {
			key = spanner.Key{"__probe__", "__probe__"}
			col = "TenantId"
		}
		_, pErr := ro.ReadRow(ctx, tbl, key, []string{col})
		ro.Close()
		if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
			client.Close()
			t.Fatalf("Wave 4A table %s unavailable: %v", tbl, pErr)
		}
	}
	t.Cleanup(client.Close)
	return client
}

type spannerGLBridge struct {
	svc *gl.Service
}

func (b *spannerGLBridge) PostJournal(ctx context.Context, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	res, err := b.svc.PostEntry(ctx, toGLInput(in))
	if err != nil {
		return p2p.GLPostResult{}, err
	}
	return p2p.GLPostResult{EntryID: res.EntryID, PeriodID: res.PeriodID, AlreadyPosted: res.AlreadyPosted}, nil
}

func (b *spannerGLBridge) PostJournalInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	res, err := gl.BufferJournalEntryInTxn(ctx, txn, buf, toGLInput(in))
	if err != nil {
		return p2p.GLPostResult{}, err
	}
	return p2p.GLPostResult{EntryID: res.EntryID, PeriodID: res.PeriodID, AlreadyPosted: res.AlreadyPosted}, nil
}

func toGLInput(in p2p.GLPostInput) gl.JournalEntryInput {
	postings := make([]gl.PostingInput, 0, len(in.Lines))
	for _, l := range in.Lines {
		postings = append(postings, gl.PostingInput{
			AccountCode: l.AccountCode,
			Direction:   gl.Direction(l.Direction),
			AmountMinor: l.AmountMinor,
			Currency:    in.Currency,
			PartyType:   l.PartyType,
			PartyID:     l.PartyID,
			WarehouseID: l.WarehouseID,
			ProductID:   l.ProductID,
			Description: l.Description,
		})
	}
	return gl.JournalEntryInput{
		EntityID:       in.TenantID,
		LegalEntityID:  in.EntityID,
		PeriodID:       in.PeriodID,
		SourceType:     in.SourceType,
		SourceID:       in.SourceID,
		Memo:           in.Memo,
		Currency:       in.Currency,
		IdempotencyKey: in.IdempotencyKey,
		PostedBy:       in.PostedBy,
		OccurredAt:     in.OccurredAt,
		Postings:       postings,
	}
}

type spannerApprovalBridge struct {
	svc *approvals.Service
}

func (a *spannerApprovalBridge) CheckOrSubmitApproval(ctx context.Context, in p2p.ApprovalCheckInput) (p2p.ApprovalCheckResult, error) {
	if strings.TrimSpace(in.ApprovalID) == "" {
		res, err := a.svc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:       in.TenantID,
			ObjectType:     in.ObjectType,
			ObjectID:       in.ObjectID,
			Action:         in.Action,
			AmountMinor:    in.AmountMinor,
			RequestedBy:    in.RequestedBy,
			ActorType:      in.ActorType,
			ApproverRole:   in.ApproverRole,
			Payload:        in.Payload,
			Context:        in.Context,
			IdempotencyKey: in.IdempotencyKey,
			ForceCreate:    in.ForceCreate,
		})
		if err != nil {
			return p2p.ApprovalCheckResult{}, err
		}
		if !res.Required || res.Request == nil {
			return p2p.ApprovalCheckResult{Required: false}, nil
		}
		return p2p.ApprovalCheckResult{
			Required:          true,
			Approved:          res.Request.Status == approvals.StatusApproved,
			ApprovalRequestID: res.Request.RequestID,
			PolicyID:          res.Request.PolicyID,
			PolicyVersion:     res.Request.PolicyVersion,
			PayloadHash:       res.Request.PayloadHash,
			Status:            res.Request.Status,
		}, nil
	}
	_, hash, err := approvals.CanonicalizePayload(in.Payload)
	if err != nil {
		return p2p.ApprovalCheckResult{}, err
	}
	req, decs, err := a.svc.VerifyApprovedRequest(ctx, in.TenantID, in.ApprovalID, in.ObjectType, in.Action, hash, 1)
	if err != nil {
		return p2p.ApprovalCheckResult{}, err
	}
	var approverIDs []string
	for _, d := range decs {
		if d.Decision == approvals.DecisionApprove {
			approverIDs = append(approverIDs, d.ApproverID)
		}
	}
	return p2p.ApprovalCheckResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
		Status:            req.Status,
		ApproverIDs:       approverIDs,
	}, nil
}

type spannerAuditBridge struct {
	svc *audit.Service
}

func (a *spannerAuditBridge) RecordEvent(ctx context.Context, ev p2p.AuditRecordInput) error {
	_, err := a.svc.Record(ctx, toAuditInput(ev))
	return err
}

func (a *spannerAuditBridge) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev p2p.AuditRecordInput) error {
	_, err := a.svc.RecordInTxn(ctx, txn, toAuditInput(ev))
	return err
}

func toAuditInput(ev p2p.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

func TestP2P_LiveSpanner_Templates_ThreeWayMatch_GL_Outbox_ProcessMining(t *testing.T) {
	client := openP2PSpannerClient(t)
	ctx := context.Background()

	tenantID := "sup-p2p-live-" + uuid.NewString()[:8]
	entityID := "ent-uz-" + uuid.NewString()[:8]
	orderID := "ord-p2p-" + uuid.NewString()[:8]
	esfID := "esf-in-" + uuid.NewString()[:8]
	periodID := time.Now().UTC().Format("2006-01")
	t0 := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("gl SeedDefaults: %v", err)
	}
	auditSvc := audit.NewService(client)
	appSvc := approvals.NewService(client)

	svc := p2p.NewSpannerService(client, p2p.ServiceConfig{
		GLPoster:                 &spannerGLBridge{svc: glSvc},
		Approvals:                &spannerApprovalBridge{svc: appSvc},
		Audit:                    &spannerAuditBridge{svc: auditSvc},
		POApprovalThresholdMinor: 500_000,
	})

	// Seed real Orders + OrderStatusTransitions + Inbound EsfDocuments for live mining & 3-way match.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         tenantID,
			"RetailerId":         "ret-p2p-1",
			"Status":             "CONFIRMED",
			"OrderSource":        "DIRECT",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(250000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          t0,
			"UpdatedAt":          t0,
		}),
		spanner.InsertMap("OrderStatusTransitions", map[string]any{
			"OrderId":        orderID,
			"TransitionId":   "tr-" + uuid.NewString()[:8],
			"PreviousStatus": "PENDING",
			"NewStatus":      "CONFIRMED",
			"Reason":         "confirmed",
			"CreatedAt":      t0.Add(1500 * time.Second),
		}),
		spanner.InsertMap("EsfDocuments", map[string]any{
			"EntityId":        tenantID,
			"EsfId":           esfID,
			"Direction":       "INBOUND",
			"CounterpartyTin": "309876543",
			"TotalMinor":      int64(1_120_000),
			"VatMinor":        int64(120_000),
			"VatRateBps":      int64(1200),
			"Status":          "ACCEPTED",
			"Operator":        "didox",
			"PeriodId":        periodID,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Orders/OrderStatusTransitions/EsfDocuments: %v", err)
	}

	// Empty-check on another tenant must stay empty.
	emptyM, err := svc.GetProcessMining(ctx, p2p.MiningFilter{TenantID: "sup-never-" + uuid.NewString()[:8]})
	if err != nil || !emptyM.Empty || emptyM.BottleneckStage != nil {
		t.Fatalf("expected empty mining report on unseeded tenant, got err=%v report=%+v", err, emptyM)
	}

	// 1. Create PO (> threshold -> PENDING_APPROVAL), approve via Approvals + P2P.
	poRes, err := svc.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:       tenantID,
		EntityID:       entityID,
		VendorID:       "vnd-live-1",
		VendorName:     "Samarkand Dairy",
		VendorTin:      "309876543",
		Currency:       "UZS",
		IdempotencyKey: "po-live-" + tenantID,
		Lines: []p2p.PurchaseOrderLineInput{
			{Sku: "MILK-1L", ProductName: "Milk 1L", QtyOrdered: 100, UnitCostMinor: 10_000, VatBps: 1200},
		},
		CreatedBy: "maker-live",
		ActorType: auth.ActorTypeHuman,
		ActorRole: auth.SupplierRoleAccountant,
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder live: %v", err)
	}
	if !poRes.ApprovalRequired || poRes.ApprovalRequestID == "" {
		t.Fatalf("expected CEL/threshold approval request, got %+v", poRes)
	}

	if _, err := appSvc.Decide(ctx, approvals.DecideInput{
		TenantID:     tenantID,
		RequestID:    poRes.ApprovalRequestID,
		ApproverID:   "cfo-live",
		ActorType:    auth.ActorTypeHuman,
		ApproverRole: auth.SupplierRoleOwner,
		Decision:     approvals.DecisionApprove,
	}); err != nil {
		t.Fatalf("approvals.Decide: %v", err)
	}

	approvedPO, err := svc.ApprovePurchaseOrder(ctx, p2p.ApprovePurchaseOrderInput{
		TenantID:   tenantID,
		PoID:       poRes.PurchaseOrder.PoID,
		ApprovalID: poRes.ApprovalRequestID,
		ActorID:    "cfo-live",
		ActorType:  auth.ActorTypeHuman,
		ActorRole:  auth.SupplierRoleOwner,
	})
	if err != nil || approvedPO.Status != p2p.POStatusApproved {
		t.Fatalf("ApprovePurchaseOrder live: err=%v status=%s", err, approvedPO.Status)
	}

	// 2. Goods Receipt -> DR INVENTORY 1,000,000 / CR GRNI 1,000,000.
	recRes, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       tenantID,
		PoID:           approvedPO.PoID,
		EntityID:       entityID,
		WarehouseID:    "wh-live-1",
		PeriodID:       periodID,
		IdempotencyKey: "grn-live-" + tenantID,
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "MILK-1L", QtyReceived: 100}},
		ReceivedBy:     "wh-live",
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleWarehouseManager,
	})
	if err != nil {
		t.Fatalf("PostGoodsReceipt live: %v", err)
	}
	if recRes.PurchaseOrder.Status != p2p.POStatusReceived || recRes.JournalEntryID == "" {
		t.Fatalf("expected RECEIVED + GL entry, got %+v", recRes)
	}

	// 3. Vendor Invoice linked to Inbound ESF -> 3-Way Match -> MATCHED_3WAY + GRNI cleared -> AP:VENDOR.
	inv, err := svc.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:       tenantID,
		PoID:           approvedPO.PoID,
		ReceiptID:      recRes.Receipt.ReceiptID,
		EsfDocID:       esfID,
		EntityID:       entityID,
		SubtotalMinor:  1_000_000,
		VatMinor:       120_000,
		TotalMinor:     1_120_000,
		IdempotencyKey: "vinv-live-" + tenantID,
		CreatedBy:      "maker-live",
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleAccountant,
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice live: %v", err)
	}

	matchRes, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        tenantID,
		VendorInvoiceID: inv.VendorInvoiceID,
		PeriodID:        periodID,
		VerifiedBy:      "cfo-live",
		ActorType:       auth.ActorTypeHuman,
		ActorRole:       auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("ThreeWayMatch live: %v", err)
	}
	if !matchRes.Matched || matchRes.MatchStatus != p2p.MatchStatusMatched3Way || matchRes.JournalEntryID == "" {
		t.Fatalf("expected MATCHED_3WAY + GL entry, got %+v", matchRes)
	}
	if matchRes.ProcessInstance == nil || matchRes.ProcessInstance.Status != p2p.InstanceStatusCompleted {
		t.Fatalf("expected ProcessInstance COMPLETED, got %+v", matchRes.ProcessInstance)
	}

	// Check TrialBalance (both org-wide and filtered by LegalEntityId): INVENTORY debit=1M, GRNI net=0 (cleared), TAX:VAT:INPUT debit=120k, AP:VENDOR credit=1.12M.
	tb, err := glSvc.TrialBalance(ctx, tenantID, periodID)
	if err != nil {
		t.Fatalf("glSvc.TrialBalance: %v", err)
	}
	if !tb.Balanced {
		t.Fatalf("expected balanced trial balance, got %+v", tb)
	}
	entityTB, err := glSvc.TrialBalanceForEntity(ctx, tenantID, periodID, entityID)
	if err != nil || !entityTB.Balanced || len(entityTB.Accounts) == 0 {
		t.Fatalf("glSvc.TrialBalanceForEntity(%s): err=%v tb=%+v", entityID, err, entityTB)
	}
	byAcc := make(map[string]gl.TrialBalanceLine)
	for _, l := range tb.Accounts {
		byAcc[l.AccountCode] = l
	}
	if byAcc[gl.AccountInventory].DebitMinor != 1_000_000 ||
		byAcc[gl.AccountGRNI].DebitMinor != 1_000_000 || byAcc[gl.AccountGRNI].CreditMinor != 1_000_000 ||
		byAcc[gl.AccountVatInput].DebitMinor != 120_000 ||
		byAcc[gl.AccountVendorPayable].CreditMinor != 1_120_000 {
		t.Fatalf("unexpected GL trial balance after 3-way match: %+v", byAcc)
	}

	// Verify live Process Mining across OrderStatusTransitions + ProcessInstances.
	mining, err := svc.GetProcessMining(ctx, p2p.MiningFilter{TenantID: tenantID})
	if err != nil {
		t.Fatalf("GetProcessMining live: %v", err)
	}
	if mining.Empty || mining.TotalCases < 2 || len(mining.Edges) < 2 {
		t.Fatalf("expected live mining metrics across O2C and P2P, got %+v", mining)
	}

	// Verify OutboxEvents persisted all 3 P2P event types for tenantID.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT CAST(Payload AS STRING) FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	defer iter.Stop()
	foundEv := make(map[string]bool)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var payload string
		if err := row.Column(0, &payload); err == nil {
			for _, want := range []string{p2p.EventP2PPOCreated, p2p.EventP2PReceiptPosted, p2p.EventP2PThreeWayMatchComplete} {
				if strings.Contains(payload, want) {
					foundEv[want] = true
				}
			}
		}
	}
	for _, want := range []string{p2p.EventP2PPOCreated, p2p.EventP2PReceiptPosted, p2p.EventP2PThreeWayMatchComplete} {
		if !foundEv[want] {
			t.Fatalf("missing Spanner OutboxEvents row for %s", want)
		}
	}

	// Verify receipt idempotency replay after full receipt returns same receipt without ErrQtyExceeded.
	recReplay, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:       tenantID,
		PoID:           approvedPO.PoID,
		EntityID:       entityID,
		WarehouseID:    "wh-live-1",
		PeriodID:       periodID,
		IdempotencyKey: "grn-live-" + tenantID,
		Lines:          []p2p.GoodsReceiptLineInput{{Sku: "MILK-1L", QtyReceived: 100}},
		ReceivedBy:     "wh-live",
	})
	if err != nil || recReplay.Receipt.ReceiptID != recRes.Receipt.ReceiptID {
		t.Fatalf("live receipt idempotency replay failed: err=%v replay=%+v", err, recReplay)
	}

	// Verify Spanner Discrepancy persistence (no silent auto-approval, no GL post).
	badInv, err := svc.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:      tenantID,
		PoID:          approvedPO.PoID,
		EntityID:      entityID,
		SubtotalMinor: 1_050_000,
		VatMinor:      126_000,
		TotalMinor:    1_176_000,
		CreatedBy:     "maker-live",
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice live discrepancy: %v", err)
	}
	discRes, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        tenantID,
		VendorInvoiceID: badInv.VendorInvoiceID,
		PeriodID:        periodID,
		VerifiedBy:      "cfo-live",
	})
	if err != nil || discRes.Matched || discRes.MatchStatus != p2p.MatchStatusDiscrepancy || discRes.VarianceMinor != 56_000 || discRes.JournalEntryID != "" {
		t.Fatalf("expected live DISCREPANCY variance=56000 without GL, got err=%v res=%+v", err, discRes)
	}

	// Verify custom ProcessTemplate & List endpoints on live Spanner.
	if _, err := svc.CreateTemplate(ctx, p2p.CreateTemplateInput{
		TenantID: tenantID,
		Code:     "P2P-PHARMA-UZ",
		Name:     "Pharma P2P",
		Domain:   p2p.DomainP2P,
		Steps:    []p2p.ProcessStep{{Code: "PO_CREATED", Name: "Created", SLASeconds: 1800}, {Code: "MATCHED_3WAY", Name: "Matched", SLASeconds: 7200}},
		ActorID:  "cfo-live",
	}); err != nil {
		t.Fatalf("CreateTemplate live: %v", err)
	}
	if listTpls, err := svc.ListTemplates(ctx, tenantID, p2p.DomainP2P); err != nil || len(listTpls) < 2 {
		t.Fatalf("ListTemplates live: err=%v len=%d", err, len(listTpls))
	}
	if listPOs, err := svc.ListPurchaseOrders(ctx, p2p.ListFilter{TenantID: tenantID}); err != nil || len(listPOs) != 1 {
		t.Fatalf("ListPurchaseOrders live: err=%v len=%d", err, len(listPOs))
	}
	if listRecs, err := svc.ListGoodsReceipts(ctx, p2p.ListFilter{TenantID: tenantID, PoID: approvedPO.PoID}); err != nil || len(listRecs) != 1 {
		t.Fatalf("ListGoodsReceipts live: err=%v len=%d", err, len(listRecs))
	}
	if listInvs, err := svc.ListVendorInvoices(ctx, p2p.ListFilter{TenantID: tenantID, PoID: approvedPO.PoID}); err != nil || len(listInvs) != 2 {
		t.Fatalf("ListVendorInvoices live: err=%v len=%d", err, len(listInvs))
	}

	// Verify AuditEvents recorded P2P actions.
	auditRows, err := auditSvc.ListEvents(ctx, audit.Filter{TenantID: tenantID, ObjectType: p2p.AggregateVendorInvoice, Limit: 20})
	if err != nil || len(auditRows) == 0 {
		t.Fatalf("expected AuditEvents for VendorInvoice, got err=%v count=%d", err, len(auditRows))
	}
}

type nonTxnGLBridge struct {
	svc *gl.Service
}

func (b *nonTxnGLBridge) PostJournal(ctx context.Context, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	res, err := b.svc.PostEntry(ctx, toGLInput(in))
	if err != nil {
		return p2p.GLPostResult{}, err
	}
	return p2p.GLPostResult{EntryID: res.EntryID, PeriodID: res.PeriodID, AlreadyPosted: res.AlreadyPosted}, nil
}

func TestP2P_LiveSpanner_NonTransactionalGLPoster_NoDeadlock(t *testing.T) {
	client := openP2PSpannerClient(t)
	ctx := context.Background()
	tenantID := "sup-p2p-nontxn-" + uuid.NewString()[:8]
	periodID := time.Now().UTC().Format("2006-01")

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("gl SeedDefaults: %v", err)
	}
	svc := p2p.NewSpannerService(client, p2p.ServiceConfig{
		GLPoster: &nonTxnGLBridge{svc: glSvc},
	})
	poRes, err := svc.CreatePurchaseOrder(ctx, p2p.CreatePurchaseOrderInput{
		TenantID:  tenantID,
		VendorID:  "vnd-nt",
		Lines:     []p2p.PurchaseOrderLineInput{{Sku: "SUGAR-1KG", QtyOrdered: 10, UnitCostMinor: 15_000, VatBps: 1200}},
		CreatedBy: "maker-nt",
	})
	if err != nil {
		t.Fatalf("CreatePurchaseOrder: %v", err)
	}
	recRes, err := svc.PostGoodsReceipt(ctx, p2p.PostGoodsReceiptInput{
		TenantID:    tenantID,
		PoID:        poRes.PurchaseOrder.PoID,
		WarehouseID: "wh-nt",
		PeriodID:    periodID,
		Lines:       []p2p.GoodsReceiptLineInput{{Sku: "SUGAR-1KG", QtyReceived: 10}},
		ReceivedBy:  "wh-nt",
	})
	if err != nil || recRes.JournalEntryID == "" {
		t.Fatalf("non-txn PostGoodsReceipt: err=%v res=%+v", err, recRes)
	}
	inv, err := svc.CreateVendorInvoice(ctx, p2p.CreateVendorInvoiceInput{
		TenantID:      tenantID,
		PoID:          poRes.PurchaseOrder.PoID,
		SubtotalMinor: 150_000,
		VatMinor:      18_000,
		TotalMinor:    168_000,
		CreatedBy:     "maker-nt",
	})
	if err != nil {
		t.Fatalf("CreateVendorInvoice: %v", err)
	}
	mRes, err := svc.ThreeWayMatch(ctx, p2p.ThreeWayMatchInput{
		TenantID:        tenantID,
		VendorInvoiceID: inv.VendorInvoiceID,
		PeriodID:        periodID,
		VerifiedBy:      "checker-nt",
	})
	if err != nil || !mRes.Matched || mRes.JournalEntryID == "" {
		t.Fatalf("non-txn ThreeWayMatch: err=%v res=%+v", err, mRes)
	}
}
