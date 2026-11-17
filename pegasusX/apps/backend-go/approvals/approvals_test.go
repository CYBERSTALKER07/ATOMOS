package approvals

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type stubMFAVerifier struct {
	expectedCode string
}

func (m stubMFAVerifier) Verify(_ context.Context, _ string, code string) error {
	if code == m.expectedCode {
		return nil
	}
	return fmt.Errorf("invalid_totp")
}

type stubSoDChecker struct {
	calls []SoDCheckInput
}

func (s *stubSoDChecker) CheckFourEyes(_ context.Context, in SoDCheckInput) error {
	s.calls = append(s.calls, in)
	return nil
}

type stubTelegramResolver struct {
	bindings map[string]TelegramApproverBinding
}

func (r stubTelegramResolver) ResolveTelegramApprover(_ context.Context, chatID, _ string) (string, string, string, string, bool, error) {
	b, ok := r.bindings[chatID]
	if !ok {
		return "", "", "", "", false, nil
	}
	return b.TenantID, b.ActorID, b.Role, b.ActorType, true, nil
}

func TestApprovals_Memory_MultiStep_Delegation_MFA_Hash_SoD_BreakGlass_Telegram(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	svc.now = func() time.Time { return now }
	sod := &stubSoDChecker{}
	svc.SetSoDChecker(sod)
	svc.SetMFAVerifier(stubMFAVerifier{expectedCode: "654321"})
	svc.SetTelegramWebhook("tg-secret-123", stubTelegramResolver{
		bindings: map[string]TelegramApproverBinding{
			"999001": {
				TenantID:  "sup-app-1",
				ActorID:   "finmgr-tg",
				Role:      auth.SupplierRoleFinanceManager,
				ActorType: auth.ActorTypeHuman,
			},
		},
	})

	// 1. Create multi-step CEL policy (Step 0: FINANCE_MANAGER, Step 1: OWNER w/ MFA >= 50_000_000).
	pol, err := svc.CreatePolicy(ctx, CreatePolicyInput{
		TenantID:      "sup-app-1",
		Name:          "High Value Journal & High Risk Retailer",
		ObjectType:    "JournalEntry",
		Action:        "journal.post",
		ConditionExpr: `amount_minor > 50000000 && retailer.risk_tier == "HIGH"`,
		Steps: []ApprovalStep{
			{StepIndex: 0, Name: "Finance Review", AllowedRoles: []string{auth.SupplierRoleFinanceManager}, RequiredApprovers: 1, TimeoutHours: 24},
			{StepIndex: 1, Name: "Owner MFA Signoff", AllowedRoles: []string{auth.SupplierRoleOwner}, RequiredApprovers: 1, RequireMFA: true, MFAThresholdMinor: 50000000},
		},
		CreatedBy: "owner-1",
	})
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	if pol.Version != 1 {
		t.Fatalf("policy version=%d, want 1", pol.Version)
	}

	// Non-matching payload -> Required=false.
	noMatch, err := svc.SubmitRequest(ctx, SubmitRequestInput{
		TenantID:    "sup-app-1",
		ObjectType:  "JournalEntry",
		ObjectID:    "je-low",
		Action:      "journal.post",
		AmountMinor: 10000,
		RequestedBy: "acct-1",
		ActorType:   auth.ActorTypeHuman,
		Context:     map[string]any{"amount_minor": int64(10000), "retailer": map[string]any{"risk_tier": "LOW"}},
	})
	if err != nil || noMatch.Required {
		t.Fatalf("expected non-match, got required=%v err=%v", noMatch.Required, err)
	}

	// Matching payload -> creates request pinned to Version 1.
	subRes, err := svc.SubmitRequest(ctx, SubmitRequestInput{
		TenantID:       "sup-app-1",
		ObjectType:     "JournalEntry",
		ObjectID:       "je-high-1",
		Action:         "journal.post",
		AmountMinor:    75000000,
		RequestedBy:    "acct-1",
		ActorType:      auth.ActorTypeHuman,
		IdempotencyKey: "req-high-1",
		Context: map[string]any{
			"amount_minor": int64(75000000),
			"retailer":     map[string]any{"risk_tier": "HIGH"},
		},
		Payload: map[string]any{
			"amount_minor": int64(75000000),
			"currency":     "UZS",
			"memo":         "high value manual entry",
		},
	})
	if err != nil || !subRes.Required || subRes.Request == nil {
		t.Fatalf("SubmitRequest failed: res=%+v err=%v", subRes, err)
	}
	req := subRes.Request
	if req.PolicyVersion != 1 || req.Status != StatusPending {
		t.Fatalf("unexpected request state: %+v", req)
	}

	// Update policy -> bumps to Version 2; existing request stays pinned to Version 1!
	polV2, err := svc.UpdatePolicy(ctx, UpdatePolicyInput{
		TenantID: "sup-app-1",
		PolicyID: pol.PolicyID,
		Name:     strPtr("High Value v2"),
	})
	if err != nil || polV2.Version != 2 {
		t.Fatalf("UpdatePolicy v2 failed: pol=%+v err=%v", polV2, err)
	}
	detail, err := svc.GetRequest(ctx, "sup-app-1", req.RequestID)
	if err != nil || detail.Request.PolicyVersion != 1 {
		t.Fatalf("pinned version drifted: %+v err=%v", detail, err)
	}

	// Negative checks on Step 0:
	// a) AGENT forbidden -> ErrAgentApprovalForbidden
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "agent-1",
		ActorType: auth.ActorTypeAgent, ApproverRole: auth.SupplierRoleAgent, Decision: DecisionApprove,
	}); !errors.Is(err, ErrAgentApprovalForbidden) {
		t.Fatalf("agent decide err=%v, want ErrAgentApprovalForbidden", err)
	}
	// b) Maker self-approval SoD -> ErrSoDViolation
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "acct-1",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleFinanceManager, Decision: DecisionApprove,
	}); !errors.Is(err, ErrSoDViolation) {
		t.Fatalf("self-approve err=%v, want ErrSoDViolation", err)
	}
	// c) PayloadHash mismatch -> ErrPayloadHashMismatch
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "finmgr-1",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleFinanceManager,
		Decision: DecisionApprove, ExpectedHash: "0000deadbeef",
	}); !errors.Is(err, ErrPayloadHashMismatch) {
		t.Fatalf("hash mismatch err=%v, want ErrPayloadHashMismatch", err)
	}
	// d) Unauthorized role without delegation -> ErrRoleNotAuthorized
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "delegate-user",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleAccountant, Decision: DecisionApprove,
	}); !errors.Is(err, ErrRoleNotAuthorized) {
		t.Fatalf("unauthorized role err=%v, want ErrRoleNotAuthorized", err)
	}

	// Expired delegation must NOT authorize!
	if _, err := svc.CreateDelegation(ctx, CreateDelegationInput{
		TenantID:    "sup-app-1",
		DelegatorID: "finmgr-1",
		DelegateID:  "delegate-user",
		RoleScope:   auth.SupplierRoleFinanceManager,
		StartsAt:    now.Add(-48 * time.Hour),
		EndsAt:      now.Add(-1 * time.Hour),
		Reason:      "expired vacation",
	}); err != nil {
		t.Fatalf("create expired delegation: %v", err)
	}
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "delegate-user",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleAccountant, Decision: DecisionApprove,
	}); !errors.Is(err, ErrRoleNotAuthorized) {
		t.Fatalf("expired delegation err=%v, want ErrRoleNotAuthorized", err)
	}

	// Active delegation allows delegate-user to approve Step 0 and advances to Step 1!
	if _, err := svc.CreateDelegation(ctx, CreateDelegationInput{
		TenantID:    "sup-app-1",
		DelegatorID: "finmgr-1",
		DelegateID:  "delegate-user",
		RoleScope:   auth.SupplierRoleFinanceManager,
		StartsAt:    now.Add(-1 * time.Hour),
		EndsAt:      now.Add(24 * time.Hour),
		Reason:      "OOO cover",
	}); err != nil {
		t.Fatalf("create active delegation: %v", err)
	}
	step0Res, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "delegate-user",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleAccountant,
		Decision: DecisionApprove, ExpectedHash: req.PayloadHash, Comment: "step0 via delegation",
	})
	if err != nil {
		t.Fatalf("step 0 delegated approve failed: %v", err)
	}
	if step0Res.Request.CurrentStep != 1 || step0Res.Request.Status != StatusPending {
		t.Fatalf("expected CurrentStep=1 PENDING, got %+v", step0Res.Request)
	}
	if step0Res.Decision.DelegatedFrom != "finmgr-1" {
		t.Fatalf("expected DelegatedFrom=finmgr-1, got %q", step0Res.Decision.DelegatedFrom)
	}

	// Step 1 requires OWNER + MFA: without MFA -> ErrMFARequired.
	if _, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "owner-1",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner, Decision: DecisionApprove,
	}); !errors.Is(err, ErrMFARequired) {
		t.Fatalf("step 1 without MFA err=%v, want ErrMFARequired", err)
	}
	// With TOTP code -> APPROVED!
	step1Res, err := svc.Decide(ctx, DecideInput{
		TenantID: "sup-app-1", RequestID: req.RequestID, ApproverID: "owner-1",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
		Decision: DecisionApprove, TOTPCode: "654321", ExpectedHash: req.PayloadHash,
	})
	if err != nil {
		t.Fatalf("step 1 MFA approve failed: %v", err)
	}
	if step1Res.Request.Status != StatusApproved || !step1Res.Decision.MfaVerified {
		t.Fatalf("expected APPROVED with MfaVerified=true, got req=%+v dec=%+v", step1Res.Request, step1Res.Decision)
	}

	// Telegram inline webhook test: create another request and approve via HMAC-signed webhook.
	sub2, err := svc.SubmitRequest(ctx, SubmitRequestInput{
		TenantID:    "sup-app-1",
		ObjectType:  "JournalEntry",
		ObjectID:    "je-high-2",
		Action:      "journal.post",
		AmountMinor: 80000000,
		RequestedBy: "acct-1",
		ActorType:   auth.ActorTypeHuman,
		Context:     map[string]any{"amount_minor": int64(80000000), "retailer": map[string]any{"risk_tier": "HIGH"}},
	})
	if err != nil || sub2.Request == nil {
		t.Fatalf("submit 2 failed: %v", err)
	}

	tgBody := []byte(fmt.Sprintf(`{"callback_query":{"id":"cbq-1","from":{"id":999001,"username":"finmgr"},"data":"approve:%s"}}`, sub2.Request.RequestID))
	// Bad secret -> 401
	badReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/telegram/approvals", bytes.NewReader(tgBody))
	badReq.Header.Set(headerTelegramSecretToken, "wrong")
	badRec := httptest.NewRecorder()
	svc.HandleTelegramWebhook(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("telegram bad secret status=%d, want 401", badRec.Code)
	}
	// Valid HMAC signature -> 200
	mac := hmac.New(sha256.New, []byte("tg-secret-123"))
	_, _ = mac.Write(tgBody)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	goodReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/telegram/approvals", bytes.NewReader(tgBody))
	goodReq.Header.Set(headerTelegramSignature, sig)
	goodRec := httptest.NewRecorder()
	svc.HandleTelegramWebhook(goodRec, goodReq)
	if goodRec.Code != http.StatusOK {
		t.Fatalf("telegram valid HMAC status=%d body=%s", goodRec.Code, goodRec.Body.String())
	}
	afterTG, err := svc.GetRequest(ctx, "sup-app-1", sub2.Request.RequestID)
	if err != nil || afterTG.Request.CurrentStep != 1 {
		t.Fatalf("expected telegram approve to advance to step 1, got %+v err=%v", afterTG, err)
	}

	// Break-glass post-review flag check.
	bgSub, err := svc.SubmitRequest(ctx, SubmitRequestInput{
		TenantID:         "sup-app-1",
		ObjectType:       "Period",
		ObjectID:         "2026-10",
		Action:           "period.reopen",
		RequestedBy:      "owner-1",
		ActorType:        auth.ActorTypeHuman,
		ApproverRole:     auth.SupplierRoleOwner,
		ForceCreate:      true,
		BreakGlass:       true,
		BreakGlassReason: "urgent tax audit correction",
	})
	if err != nil || bgSub.Request == nil || !bgSub.Request.BreakGlass || bgSub.Request.Status != StatusApproved {
		t.Fatalf("break-glass failed: res=%+v err=%v", bgSub, err)
	}
}

func openApprovalSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner approvals test")
	}
	dbPath := "projects/pegasusx-local/instances/pegasusx-instance/databases/pegasusx-db"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ApprovalPolicies", spanner.Key{"__probe__", "__probe__"}, []string{"PolicyId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("ApprovalPolicies unavailable: %v", probeErr)
		}
		t.Skipf("ApprovalPolicies unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestApprovals_LiveSpanner_PolicyCEL_HashMismatch_Agent_SoD_Outbox(t *testing.T) {
	client := openApprovalSpannerClient(t)
	ctx := context.Background()
	tid := "sup-appr-" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalPolicies", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDelegations", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		}
		iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(cleanCtx, spanner.Statement{
			SQL: `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`, Params: map[string]any{"sid": tid},
		})
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		_, _ = client.Apply(cleanCtx, muts)
	})

	svc := NewService(client)
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("test-approval-secret-32bytes-minimum!"))
	r.Use(auth.AttachTenantFromClaims)
	r.Post("/v1/approvals/policies", svc.HandleCreatePolicy)
	r.Post("/v1/approvals/requests", svc.HandleSubmitRequest)
	r.Post("/v1/approvals/{id}/approve", svc.HandleApprove)
	r.Get("/v1/approvals/inbox", svc.HandleInbox)

	issue := func(sub, supRole, actorType string, mfa bool) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   tid,
			SupplierRole: auth.Role(supRole),
			ActorType:    actorType,
			MFAVerified:  mfa,
		}, auth.IssueOptions{Secret: "test-approval-secret-32bytes-minimum!", TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ownerTok := issue("owner-"+tid, auth.SupplierRoleOwner, auth.ActorTypeHuman, true)
	acctTok := issue("acct-"+tid, auth.SupplierRoleAccountant, auth.ActorTypeHuman, false)
	finMgrTok := issue("finmgr-"+tid, auth.SupplierRoleFinanceManager, auth.ActorTypeHuman, false)
	agentTok := issue("agent-"+tid, auth.SupplierRoleAgent, auth.ActorTypeAgent, false)

	call := func(method, path string, body any, tok string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// Malformed CEL -> 400
	if rec := call(http.MethodPost, "/v1/approvals/policies", map[string]any{
		"name": "bad", "object_type": "JournalEntry", "action": "journal.post",
		"condition_expr": "amount_minor > &&",
	}, ownerTok); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed CEL status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}

	// Valid policy -> 201
	if rec := call(http.MethodPost, "/v1/approvals/policies", map[string]any{
		"name": "Manual JE > 500k UZS", "object_type": "JournalEntry", "action": "journal.post",
		"condition_expr": `amount_minor > 50000000`,
		"steps":          []map[string]any{{"step_index": 0, "allowed_roles": []string{"FINANCE_MANAGER"}, "required_approvers": 1}},
	}, ownerTok); rec.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Submit matching request -> 201
	subRec := call(http.MethodPost, "/v1/approvals/requests", map[string]any{
		"object_type": "JournalEntry", "object_id": "je-sp-1", "action": "journal.post",
		"amount_minor": int64(60000000), "idempotency_key": "sp-idem-1",
		"payload": map[string]any{"amount_minor": int64(60000000), "memo": "live spanner"},
	}, acctTok)
	if subRec.Code != http.StatusCreated {
		t.Fatalf("submit request status=%d body=%s", subRec.Code, subRec.Body.String())
	}
	var subResp SubmitRequestResult
	_ = json.Unmarshal(subRec.Body.Bytes(), &subResp)

	// Agent -> 403
	if rec := call(http.MethodPost, "/v1/approvals/"+subResp.Request.RequestID+"/approve", map[string]any{}, agentTok); rec.Code != http.StatusForbidden {
		t.Fatalf("agent approve status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	// Self-approve -> 403
	if rec := call(http.MethodPost, "/v1/approvals/"+subResp.Request.RequestID+"/approve", map[string]any{}, acctTok); rec.Code != http.StatusForbidden {
		t.Fatalf("self approve status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	// Hash mismatch -> 409
	if rec := call(http.MethodPost, "/v1/approvals/"+subResp.Request.RequestID+"/approve", map[string]any{"payload_hash": "badhash"}, finMgrTok); rec.Code != http.StatusConflict {
		t.Fatalf("hash mismatch status=%d body=%s, want 409", rec.Code, rec.Body.String())
	}
	// Valid approve -> 200 APPROVED
	if rec := call(http.MethodPost, "/v1/approvals/"+subResp.Request.RequestID+"/approve", map[string]any{"payload_hash": subResp.Request.PayloadHash}, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("finMgr approve status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Verify OutboxEvents recorded both approval.requested and approval.decided on TopicMain.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @sid`, Params: map[string]any{"sid": tid},
	})
	defer iter.Stop()
	seenTypes := map[string]bool{}
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var topic string
		var payload []byte
		if row.Columns(&topic, &payload) == nil {
			var base events.BaseEvent
			_ = json.Unmarshal(payload, &base)
			seenTypes[base.Type] = true
			if topic != events.TopicMain {
				t.Fatalf("expected TopicMain, got %q for %s", topic, base.Type)
			}
		}
	}
	if !seenTypes[events.EventApprovalRequested] || !seenTypes[events.EventApprovalDecided] {
		t.Fatalf("expected approval.requested and approval.decided in OutboxEvents, got %v", seenTypes)
	}
}

func strPtr(s string) *string { return &s }
