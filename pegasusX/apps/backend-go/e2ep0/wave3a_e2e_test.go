package e2ep0

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"google.golang.org/grpc/codes"
)

func TestWave3A_FullStack_Periods_Approvals_Valuation_ESF_LiveSpanner(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST unset; skipping Wave 3A E2E test")
	}

	client := openInventorySpannerClient(t)
	ctx := context.Background()
	ro := client.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "Periods", spanner.Key{"__w3a_probe__", "2026-01"}, []string{"Status"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		t.Fatalf("Periods table unavailable: %v", pErr)
	}

	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	entityID := "sup-w3a-" + uuid.NewString()[:8]
	whID := "wh-w3a-" + uuid.NewString()[:8]
	prodID := "sku-w3a-" + uuid.NewString()[:8]
	esfSliceID := "slice-w3a-" + uuid.NewString()[:8]
	today := time.Now().UTC().Format("2006-01-02")

	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	if h.PeriodsSvc == nil || h.ApprovalsSvc == nil || h.ValuationSvc == nil || h.ESFSvc == nil {
		t.Fatalf("expected all Wave 3A services wired on TestHarness")
	}
	snapSink, err := periods.NewLocalSnapshotSink(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalSnapshotSink: %v", err)
	}
	h.PeriodsSvc.SetSnapshotSink(snapSink)

	// Configure Telegram approval webhook on live ApprovalsSvc.
	const tgSecret = "w3a-tg-secret-123"
	h.ApprovalsSvc.SetTelegramWebhook(tgSecret, nil)
	t.Setenv("TELEGRAM_APPROVAL_CHAT_MAP", `{"555001":{"tenant_id":"`+entityID+`","actor_id":"tg-cfo","role":"OWNER","actor_type":"human"}}`)

	if err := h.GLSvc.SeedDefaults(ctx, entityID, "UZ", "UZS"); err != nil {
		t.Fatalf("GL SeedDefaults: %v", err)
	}

	issueUser := func(sub string, supRole auth.Role, actorType string, perms ...string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:       sub,
			Role:          auth.RoleAdmin,
			SupplierID:    entityID,
			SupplierRole:  supRole,
			SupplierPerms: perms,
			ActorType:     actorType,
		}, auth.IssueOptions{Secret: h.JWTSecret, Issuer: h.JWTIssuer, TTL: time.Hour})
		if err != nil {
			t.Fatalf("auth.Issue: %v", err)
		}
		return tok
	}

	makerTok := issueUser("fin-maker-"+entityID, auth.SupplierRoleFinanceManager, auth.ActorTypeHuman,
		auth.PermJournalPost, auth.PermPeriodClose, auth.PermPeriodReopen, auth.PermConfigManage, auth.PermStockAdjustApprove, auth.PermCreditNoteApprove)
	checker1Tok := issueUser("cfo-1-"+entityID, auth.SupplierRoleOwner, auth.ActorTypeHuman)
	checker2Tok := issueUser("cfo-2-"+entityID, auth.SupplierRoleFinanceManager, auth.ActorTypeHuman, auth.PermPeriodReopen)
	delegateTok := issueUser("delegate-1-"+entityID, auth.SupplierRoleAccountant, auth.ActorTypeHuman)
	agentTok := issueUser("bot-agent-"+entityID, auth.SupplierRoleAgent, auth.ActorTypeAgent)

	doReq := func(tok, method, path string, body any) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		rec := httptest.NewRecorder()
		h.Router.ServeHTTP(rec, req)
		return rec
	}

	// --- M1.1 Periods lifecycle + operational rollover + checklist + snapshot + reopen ---
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods", map[string]any{"period_id": "2026-10", "timezone": "Asia/Tashkent"}); rec.Code != http.StatusCreated {
		t.Fatalf("open 2026-10: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods", map[string]any{"period_id": "2026-11", "timezone": "Asia/Tashkent"}); rec.Code != http.StatusCreated {
		t.Fatalf("open 2026-11: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods/2026-10/soft-close", nil); rec.Code != http.StatusOK {
		t.Fatalf("soft-close 2026-10: %d %s", rec.Code, rec.Body.String())
	}

	// Operational event in Oct 2026 must roll forward to 2026-11 stamping OriginalPeriodId = 2026-10!
	octTS := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	rollRes, err := h.GLSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       entityID,
		SourceType:     "ORDER_DELIVERED",
		SourceID:       "ord-roll-" + entityID,
		Memo:           "late Oct event into soft-closed period",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "roll-w3a-" + entityID,
		PostedBy:       "worker",
		OccurredAt:     octTS,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, PartyType: "RETAILER", PartyID: "ret-1", Direction: gl.DirectionDebit, AmountMinor: 112000, Currency: "UZS"},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 100000, Currency: "UZS"},
			{AccountCode: gl.AccountVatOutput, Direction: gl.DirectionCredit, AmountMinor: 12000, Currency: "UZS"},
		},
	})
	if err != nil {
		t.Fatalf("operational rollover PostEntry: %v", err)
	}
	if rollRes.PeriodID != "2026-11" || rollRes.OriginalPeriodID != "2026-10" {
		t.Fatalf("expected rolled period 2026-11 with OriginalPeriodId=2026-10, got %+v", rollRes)
	}

	// Seed an unsettled payout slice blocker -> Close MUST return 409 period_checklist_failed!
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("InvoiceSettlementSlices", map[string]any{
			"SliceId": esfSliceID, "OrderId": "ord-sl-1", "SupplierId": entityID, "CapturedLegId": "leg-1",
			"GrossMinor": int64(50000), "CommissionMinor": int64(2000), "NetPayoutMinor": int64(48000),
			"Currency": "UZS", "Status": "UNSETTLED", "CreatedAt": octTS,
		}),
	}); err != nil {
		t.Fatalf("seed unsettled slice: %v", err)
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods/2026-10/close", nil); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "period_checklist_failed") {
		t.Fatalf("expected 409 period_checklist_failed, got %d %s", rec.Code, rec.Body.String())
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("InvoiceSettlementSlices", map[string]any{"SliceId": esfSliceID, "Status": "BATCHED"}),
	}); err != nil {
		t.Fatalf("resolve slice: %v", err)
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods/2026-10/close", nil); rec.Code != http.StatusOK {
		t.Fatalf("close 2026-10 after resolving blockers: %d %s", rec.Code, rec.Body.String())
	}

	// Manual journal in CLOSED 2026-10 -> 409 period_closed!
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/journal/manual", map[string]any{
		"period_id":       "2026-10",
		"source_id":       "man-closed-w3a",
		"memo":            "blocked in closed 2026-10",
		"currency":        "UZS",
		"idempotency_key": "man-closed-w3a",
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 10000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 10000, "currency": "UZS"},
		},
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 period_closed on manual into CLOSED period, got %d %s", rec.Code, rec.Body.String())
	}

	// Reopen 2026-10 -> 202 PENDING_APPROVAL (requires 2 approvers) -> both approve -> 200 REOPENED!
	reopenReq := doReq(makerTok, http.MethodPost, "/v1/finance/periods/2026-10/reopen", map[string]any{
		"reason": "VAT audit adjustment",
	})
	if reopenReq.Code != http.StatusAccepted {
		t.Fatalf("expected 202 PENDING_APPROVAL for reopen, got %d %s", reopenReq.Code, reopenReq.Body.String())
	}
	var reopenPend struct {
		ApprovalRequestID string `json:"approval_request_id"`
	}
	_ = json.Unmarshal(reopenReq.Body.Bytes(), &reopenPend)
	if rec := doReq(checker1Tok, http.MethodPost, "/v1/approvals/"+reopenPend.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusOK {
		t.Fatalf("reopen approve 1: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(checker2Tok, http.MethodPost, "/v1/approvals/"+reopenPend.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusOK {
		t.Fatalf("reopen approve 2: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/periods/2026-10/reopen", map[string]any{
		"reason":      "VAT audit adjustment",
		"approval_id": reopenPend.ApprovalRequestID,
	}); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), periods.StatusReopened) {
		t.Fatalf("expected 200 REOPENED, got %d %s", rec.Code, rec.Body.String())
	}

	// --- M1.2 Approvals: CEL policy + Delegation + SoD/Agent 403 + PayloadHash 409 + Telegram ---
	if rec := doReq(makerTok, http.MethodPost, "/v1/approvals/policies", map[string]any{
		"name":           "Manual Journal > 500k",
		"object_type":    "JournalEntry",
		"action":         "manual_post",
		"condition_expr": `amount_minor > 500000`,
		"steps": []map[string]any{
			{"step_index": 0, "name": "Controller", "allowed_roles": []string{auth.SupplierRoleOwner}, "required_approvers": 1},
		},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create CEL policy: %d %s", rec.Code, rec.Body.String())
	}

	// Active delegation from cfo-1 (OWNER) to delegate-1 (ACCOUNTANT).
	if rec := doReq(checker1Tok, http.MethodPost, "/v1/approvals/delegations", map[string]any{
		"delegate_id": "delegate-1-" + entityID,
		"role_scope":  auth.SupplierRoleOwner,
		"ends_at":     time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		"reason":      "vacation coverage",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create delegation: %d %s", rec.Code, rec.Body.String())
	}

	pastTS := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond).Format(time.RFC3339Nano)
	manPayload := map[string]any{
		"source_id":       "man-cel-e2e-" + entityID,
		"memo":            "high value adjustment",
		"currency":        "UZS",
		"idempotency_key": "man-cel-e2e-" + entityID,
		"occurred_at":     pastTS,
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 800000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 800000, "currency": "UZS"},
		},
	}
	pendRec := doReq(makerTok, http.MethodPost, "/v1/finance/journal/manual", manPayload)
	if pendRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 approval_required for CEL journal, got %d %s", pendRec.Code, pendRec.Body.String())
	}
	var journalPend struct {
		ApprovalRequestID string `json:"approval_request_id"`
	}
	_ = json.Unmarshal(pendRec.Body.Bytes(), &journalPend)

	// Agent & SoD self-approve blocked (403)!
	if rec := doReq(agentTok, http.MethodPost, "/v1/approvals/"+journalPend.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AGENT approve, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/approvals/"+journalPend.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for maker SoD self-approve, got %d %s", rec.Code, rec.Body.String())
	}

	// Delegate approves on behalf of OWNER delegation!
	if rec := doReq(delegateTok, http.MethodPost, "/v1/approvals/"+journalPend.ApprovalRequestID+"/approve", map[string]any{
		"comment": "delegated signoff",
	}); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), approvals.StatusApproved) {
		t.Fatalf("delegate approve failed: %d %s", rec.Code, rec.Body.String())
	}

	// Mutated payload with valid approval_id returns 409 payload_hash_mismatch!
	mutatedPayload := map[string]any{
		"source_id":       "man-cel-e2e-" + entityID,
		"memo":            "high value adjustment",
		"currency":        "UZS",
		"idempotency_key": "man-cel-e2e-" + entityID,
		"occurred_at":     pastTS,
		"approval_id":     journalPend.ApprovalRequestID,
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 850000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 850000, "currency": "UZS"},
		},
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/journal/manual", mutatedPayload); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 payload_hash_mismatch, got %d %s", rec.Code, rec.Body.String())
	}

	// Exact payload with approval_id posts 201!
	manPayload["approval_id"] = journalPend.ApprovalRequestID
	if rec := doReq(makerTok, http.MethodPost, "/v1/finance/journal/manual", manPayload); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Posted manual journal after approval, got %d %s", rec.Code, rec.Body.String())
	}

	// Telegram HMAC webhook inline approve!
	tgSub, err := h.ApprovalsSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
		TenantID: entityID, ObjectType: "PurchaseOrder", ObjectID: "po-tg-1", Action: "approve",
		RequestedBy: "buyer-1", ActorType: auth.ActorTypeHuman, ForceCreate: true,
	})
	if err != nil {
		t.Fatalf("SubmitRequest for TG webhook: %v", err)
	}
	tgBody, _ := json.Marshal(map[string]any{
		"telegram_chat_id": "555001",
		"callback_data":    "approve:" + tgSub.Request.RequestID,
	})
	mac := hmac.New(sha256.New, []byte(tgSecret))
	_, _ = mac.Write(tgBody)
	tgReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/telegram/approvals", bytes.NewReader(tgBody))
	tgReq.Header.Set("Content-Type", "application/json")
	tgReq.Header.Set("X-Telegram-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	tgRec := httptest.NewRecorder()
	h.Router.ServeHTTP(tgRec, tgReq)
	if tgRec.Code != http.StatusOK || !strings.Contains(tgRec.Body.String(), approvals.StatusApproved) {
		t.Fatalf("Telegram webhook approve failed: %d %s", tgRec.Code, tgRec.Body.String())
	}

	// --- M1.3 Valuation + M1.4 ESF mounted routes smoke ---
	if rec := doReq(makerTok, http.MethodPost, "/v1/inventory/cost-layers/receipt", map[string]any{
		"warehouse_id":    whID,
		"product_id":      prodID,
		"receipt_ref_id":  "po-rec-w3a",
		"quantity":        10,
		"unit_cost_minor": 5000,
		"currency":        "UZS",
		"idempotency_key": "val-rec-" + entityID,
	}); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("valuation receipt route failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(makerTok, http.MethodPost, "/v1/fiscal/esf/inbound", map[string]any{
		"po_id":            "po-esf-w3a",
		"counterparty_tin": "201999888",
		"total_minor":      56000,
		"vat_minor":        6000,
		"vat_rate_bps":     1200,
		"period_id":        "2026-11",
	}); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("esf inbound route failed: %d %s", rec.Code, rec.Body.String())
	}
	vatRec := doReq(makerTok, http.MethodGet, "/v1/fiscal/vat-summary?period_id=2026-11", nil)
	if vatRec.Code != http.StatusOK {
		t.Fatalf("esf vat-summary failed: %d %s", vatRec.Code, vatRec.Body.String())
	}
	var vatBody struct {
		PeriodID         string `json:"period_id"`
		EsfInputVatMinor int64  `json:"esf_input_vat_minor"`
	}
	if err := json.Unmarshal(vatRec.Body.Bytes(), &vatBody); err != nil {
		t.Fatalf("unmarshal esf vat-summary: %v", err)
	}
	if vatBody.PeriodID != "2026-11" || vatBody.EsfInputVatMinor != 6000 {
		t.Fatalf("unexpected esf vat-summary: %+v (body=%s)", vatBody, vatRec.Body.String())
	}

	// Final hash-chain verifications across GL AND immutable AuditEvents!
	if glChain, err := gl.VerifyEntityHashChain(ctx, client, entityID); err != nil || !glChain.Valid {
		t.Fatalf("GL hash chain invalid: %+v err=%v", glChain, err)
	}
	audChain, err := h.AuditSvc.VerifyDayChain(ctx, entityID, today)
	if err != nil || !audChain.Valid || audChain.EventCount < 5 {
		t.Fatalf("AuditEvents hash chain invalid: %+v err=%v", audChain, err)
	}
}
