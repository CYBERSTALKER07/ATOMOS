package credit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

func TestComputeRiskScore_HonestBaselineNoPlaceholder(t *testing.T) {
	now := time.Date(2026, 11, 17, 4, 0, 0, 0, time.UTC)

	// 1. Thin file (no existing limit, no starter limit, no paid/order history) -> 0 (NOT 1_000_000!)
	thin := ComputeRiskScore(Profile{
		RetailerID: "ret-thin",
		SupplierID: "sup-1",
		Status:     StatusActive,
	}, ScoreMetrics{}, now)
	if thin.SuggestedLimitMinor != 0 {
		t.Fatalf("expected honest 0 limit for thin file, got %d", thin.SuggestedLimitMinor)
	}
	if !hasReasonCode(thin.ReasonCodes, "THIN_FILE_NO_HISTORY_OR_PROGRAM_LIMIT") {
		t.Fatalf("expected THIN_FILE_NO_HISTORY_OR_PROGRAM_LIMIT reason code, got %+v", thin.ReasonCodes)
	}
	if thin.PdBps <= 0 {
		t.Fatalf("expected positive PdBps, got %d", thin.PdBps)
	}

	// 2. Program starter limit (450_000) -> baseline derives from starter limit cleanly.
	starter := ComputeRiskScore(Profile{
		RetailerID: "ret-starter",
		SupplierID: "sup-1",
		Status:     StatusActive,
	}, ScoreMetrics{StarterLimitMinor: 450_000}, now)
	if starter.SuggestedLimitMinor != 450_000 {
		t.Fatalf("expected suggested limit derived from starter 450000, got %d", starter.SuggestedLimitMinor)
	}
	if !hasReasonCode(starter.ReasonCodes, "PROGRAM_STARTER_LIMIT") {
		t.Fatalf("expected PROGRAM_STARTER_LIMIT, got %+v", starter.ReasonCodes)
	}

	// 3. Observed 90d paid volume (900_000) -> 1/3 monthly capacity = 300_000.
	observed := ComputeRiskScore(Profile{
		RetailerID: "ret-obs",
		SupplierID: "sup-1",
		Status:     StatusActive,
	}, ScoreMetrics{
		PaidVolume90dMinor: 900_000,
		PaymentsLast90d:    3,
		ExpectedPayments90: 3,
	}, now)
	if observed.SuggestedLimitMinor != 300_000 {
		t.Fatalf("expected 300000 observed limit baseline, got %d", observed.SuggestedLimitMinor)
	}
	if !hasReasonCode(observed.ReasonCodes, "OBSERVED_PAID_VOLUME") {
		t.Fatalf("expected OBSERVED_PAID_VOLUME, got %+v", observed.ReasonCodes)
	}
}

func hasReasonCode(list []ReasonCode, code string) bool {
	for _, rc := range list {
		if rc.Code == code {
			return true
		}
	}
	return false
}

func TestMLScorerClient_ConfiguredAndFailClosed(t *testing.T) {
	ctx := context.Background()

	// Unconfigured + Required=true fails closed.
	reqClient := NewHTTPMLScorerClient(MLScorerConfig{Required: true})
	if _, err := reqClient.Score(ctx, MLScorerRequest{RetailerID: "ret-1", Scope: ScoreScopeRelationship}); !errors.Is(err, ErrMLScorerNotConfigured) {
		t.Fatalf("expected ErrMLScorerNotConfigured, got %v", err)
	}

	// Unconfigured + Required=false degrades cleanly to deterministic scorecard.
	optClient := NewHTTPMLScorerClient(MLScorerConfig{Required: false})
	respOpt, err := optClient.Score(ctx, MLScorerRequest{
		RetailerID: "ret-1",
		Scope:      ScoreScopeRelationship,
		Features:   MLFeatures{RetailerID: "ret-1", CreditLimitMinor: 500_000},
	})
	if err != nil {
		t.Fatalf("expected graceful fallback when unconfigured and optional, got %v", err)
	}
	if respOpt.ScorerSource != ScorerSourceScorecard || respOpt.FeaturesHash == "" {
		t.Fatalf("unexpected optional response: %+v", respOpt)
	}

	// Configured ML sidecar server returns calibrated PD without float fields on client.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body MLScorerRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode ml request: %v", err)
		}
		if body.FeaturesHash == "" {
			t.Fatal("expected non-empty features_hash")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model_version":"xgb_credit_2026_11",
			"score":88,
			"pd_bps":215,
			"features_hash":"` + body.FeaturesHash + `",
			"reason_codes":[{"code":"ML_ON_TIME_STRONG","label":"On-time payment ratio > 95%","impact_bps":650,"detail":"90d settled","source":"ML_SIDECAR"}]
		}`))
	}))
	defer srv.Close()

	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:       "ret-ml",
			SupplierID:       "sup-ml",
			CreditLimitMinor: 800_000,
			Status:           StatusActive,
		},
	}
	svc := newTestService(repo)
	memScoreRepo := NewMemoryScoreHistoryRepository(repo, nil)
	svc.SetScoreHistoryRepo(memScoreRepo)
	svc.SetMLScorer(NewHTTPMLScorerClient(MLScorerConfig{URL: srv.URL, Required: true}))

	rec, err := svc.ComputeRelationshipScore(ctx, "ret-ml", "sup-ml")
	if err != nil {
		t.Fatalf("ComputeRelationshipScore with live ML server failed: %v", err)
	}
	if rec.ScorerSource != ScorerSourceMLSidecar || rec.ModelVersion != "xgb_credit_2026_11" {
		t.Fatalf("expected ML_SIDECAR score, got source=%s ver=%s", rec.ScorerSource, rec.ModelVersion)
	}
	if rec.Score != 88 || rec.PdBps != 215 || rec.FeaturesHash == "" {
		t.Fatalf("unexpected ML score record: %+v", rec)
	}
	if !hasReasonCode(rec.ReasonCodes, "ML_ON_TIME_STRONG") {
		t.Fatalf("expected ML reason code preserved, got %+v", rec.ReasonCodes)
	}

	// Fail closed if ML endpoint returns 500 and Required=true.
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badSrv.Close()
	svc.SetMLScorer(NewHTTPMLScorerClient(MLScorerConfig{URL: badSrv.URL, Required: true}))
	if _, err := svc.ComputeRelationshipScore(ctx, "ret-ml", "sup-ml"); !errors.Is(err, ErrMLScorerFailed) {
		t.Fatalf("expected ErrMLScorerFailed on 500 with Required=true, got %v", err)
	}
}

func TestNetworkScore_ConsentFilteringAndDisputeIsolation(t *testing.T) {
	ctx := context.Background()
	repo := &testCreditRepo{found: true, profile: Profile{RetailerID: "ret-net", SupplierID: "sup-a", Status: StatusActive}}
	svc := newTestService(repo)
	svc.SetMLScorer(NewHTTPMLScorerClient(MLScorerConfig{Required: false}))
	scoreRepo := NewMemoryScoreHistoryRepository(repo, nil)
	svc.SetScoreHistoryRepo(scoreRepo)

	// Without any ConsentVerified=true supplier, network score stays unconsented.
	unconsented, err := svc.ComputeNetworkScore(ctx, "ret-net")
	if err != nil {
		t.Fatalf("ComputeNetworkScore unconsented failed: %v", err)
	}
	if unconsented.ConsentVerified || unconsented.Score != 0 {
		t.Fatalf("expected unconsented network score=0, got %+v", unconsented)
	}
	if !hasReasonCode(unconsented.ReasonCodes, "NETWORK_CONSENT_UNVERIFIED") {
		t.Fatalf("expected NETWORK_CONSENT_UNVERIFIED, got %+v", unconsented.ReasonCodes)
	}

	// Supplier A & B consent (with 3 disputes isolated on B); Supplier C is unconsented and terrible.
	scoreRepo.SeedConsentingRelationship(Profile{
		RetailerID: "ret-net", SupplierID: "sup-a", CreditLimitMinor: 500_000, Status: StatusActive,
	}, RetailerPaymentTerms{RetailerID: "ret-net", SupplierID: "sup-a", ConsentVerified: true}, ScoreMetrics{
		PaidVolume90dMinor: 900_000, PaymentsLast90d: 4, ExpectedPayments90: 4,
	}, true)
	scoreRepo.SeedConsentingRelationship(Profile{
		RetailerID: "ret-net", SupplierID: "sup-b", CreditLimitMinor: 300_000, Status: StatusActive,
	}, RetailerPaymentTerms{RetailerID: "ret-net", SupplierID: "sup-b", ConsentVerified: true}, ScoreMetrics{
		PaidVolume90dMinor: 600_000, PaymentsLast90d: 3, ExpectedPayments90: 3,
	}, true)
	scoreRepo.SeedDisputes("ret-net", "sup-b", 3, 250_000)

	// Non-consenting supplier C must be completely ignored.
	scoreRepo.SeedConsentingRelationship(Profile{
		RetailerID: "ret-net", SupplierID: "sup-c", CreditLimitMinor: 100_000, DelinquencyCount: 8, Status: StatusBlacklisted,
	}, RetailerPaymentTerms{RetailerID: "ret-net", SupplierID: "sup-c", ConsentVerified: false}, ScoreMetrics{
		MaxDaysPastDue: 60,
	}, false)

	netRec, err := svc.ComputeNetworkScore(ctx, "ret-net")
	if err != nil {
		t.Fatalf("ComputeNetworkScore consenting failed: %v", err)
	}
	if !netRec.ConsentVerified || netRec.DisputesIsolatedCount != 3 {
		t.Fatalf("expected ConsentVerified=true & DisputesIsolatedCount=3, got %+v", netRec)
	}
	if netRec.Score < 80 {
		t.Fatalf("expected healthy score (>=80) since sup-c has no consent and sup-b disputes are isolated, got %d", netRec.Score)
	}
	if !hasReasonCode(netRec.ReasonCodes, "NETWORK_CONSENT_VERIFIED") || !hasReasonCode(netRec.ReasonCodes, "SUPPLIER_DISPUTES_ISOLATED") {
		t.Fatalf("expected consent + dispute isolation reason codes, got %+v", netRec.ReasonCodes)
	}
}

func TestCreditHoldEvents_FreezeUnfreezeAndPrepayLock(t *testing.T) {
	ctx := context.Background()
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:       "ret-hold",
			SupplierID:       "sup-hold",
			CreditLimitMinor: 300_000,
			Status:           StatusActive,
		},
	}
	svc := newTestService(repo)

	if err := svc.FreezeProfile(ctx, "ret-hold", "sup-hold", "overdue_hold"); err != nil {
		t.Fatalf("FreezeProfile failed: %v", err)
	}
	if !containsEventType(repo.lastEvents, events.EventCreditHoldApplied) {
		t.Fatalf("expected %s emitted on FreezeProfile", events.EventCreditHoldApplied)
	}

	repo.profile = *repo.stored
	if err := svc.UnfreezeProfile(ctx, "ret-hold", "sup-hold", "paid_up"); err != nil {
		t.Fatalf("UnfreezeProfile failed: %v", err)
	}
	if !containsEventType(repo.lastEvents, events.EventCreditHoldReleased) {
		t.Fatalf("expected %s emitted on UnfreezeProfile", events.EventCreditHoldReleased)
	}

	repo.profile = *repo.stored
	locked, err := svc.SetPrepayOnlyLocked(ctx, "sup-hold", "ret-hold", true, "usr-1", "risk_review")
	if err != nil || !locked.PrepayOnlyLocked {
		t.Fatalf("SetPrepayOnlyLocked(true) failed: locked=%+v err=%v", locked, err)
	}
	if !containsEventType(repo.lastEvents, events.EventCreditHoldApplied) {
		t.Fatalf("expected %s emitted on prepay lock", events.EventCreditHoldApplied)
	}
}

func containsEventType(evts []outbox.Event, target string) bool {
	for _, e := range evts {
		if bytes.Contains(e.Payload, []byte(`"`+target+`"`)) {
			return true
		}
	}
	return false
}

type stubLimitBridge struct {
	thresholdMinor int64
	requester      string
	approvedID     string
	approvedHash   string
}

func (b *stubLimitBridge) EvaluateOrVerifyLimitChange(_ context.Context, in LimitChangeApprovalInput) (LimitChangeApprovalResult, error) {
	hash := computeTestPayloadHash(in.Payload)
	if in.NewLimitMinor <= b.thresholdMinor {
		return LimitChangeApprovalResult{Required: false, Approved: true, PayloadHash: hash}, nil
	}
	if in.ApprovalID == "" {
		b.requester = in.ActorID
		b.approvedHash = hash
		return LimitChangeApprovalResult{
			Required:          true,
			Approved:          false,
			ApprovalRequestID: "apr-credit-1",
			PayloadHash:       hash,
			Status:            "PENDING",
		}, nil
	}
	if in.ApprovalID != b.approvedID {
		return LimitChangeApprovalResult{}, ErrApprovalRequired
	}
	if in.ActorID == b.requester {
		return LimitChangeApprovalResult{}, ErrSoDViolation
	}
	if hash != b.approvedHash {
		return LimitChangeApprovalResult{}, ErrApprovalPayloadMismatch
	}
	return LimitChangeApprovalResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: in.ApprovalID,
		PayloadHash:       hash,
		Status:            "APPROVED",
	}, nil
}

func computeTestPayloadHash(p map[string]any) string {
	b, _ := json.Marshal(p)
	return ComputeFeaturesHash(MLFeatures{RetailerID: string(b)})
}

func TestGovernedLimitChange_ApprovalGateSoDAndHashMismatch(t *testing.T) {
	ctx := context.Background()
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:       "ret-gov",
			SupplierID:       "sup-gov",
			CreditLimitMinor: 200_000,
			Status:           StatusActive,
		},
	}
	svc := newTestService(repo)
	svc.SetMLScorer(NewHTTPMLScorerClient(MLScorerConfig{Required: false}))
	scoreRepo := NewMemoryScoreHistoryRepository(repo, nil)
	svc.SetScoreHistoryRepo(scoreRepo)
	bridge := &stubLimitBridge{thresholdMinor: 500_000, approvedID: "apr-credit-1"}
	svc.SetLimitApprovalBridge(bridge)

	// 1. AGENT actor forbidden.
	if _, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 300_000,
		ActorID: "agent:credit_bot", ActorType: "agent",
	}); !errors.Is(err, ErrAgentApprovalForbidden) {
		t.Fatalf("expected ErrAgentApprovalForbidden, got %v", err)
	}

	// 2. AUDITOR forbidden.
	if _, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 300_000,
		ActorID: "usr-aud", ActorType: "human", IsAuditor: true,
	}); !errors.Is(err, ErrAuditorReadOnly) {
		t.Fatalf("expected ErrAuditorReadOnly, got %v", err)
	}

	// 3. Large increase -> returns ApprovalRequired (202 contract).
	consent := true
	pendingOut, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 1_500_000, Reason: "ramadan_stockup",
		ConsentVerified: &consent, ActorID: "human:mgr-1", ActorType: "human", ActorRole: "MANAGER",
	})
	if err != nil {
		t.Fatalf("RequestOrApplyLimitChange pending failed: %v", err)
	}
	if !pendingOut.ApprovalRequired || pendingOut.ApprovalRequestID != "apr-credit-1" {
		t.Fatalf("expected ApprovalRequired output, got %+v", pendingOut)
	}

	// 4. Same requester re-submitting ApprovalID -> SoD violation.
	if _, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 1_500_000, Reason: "ramadan_stockup",
		ConsentVerified: &consent, ActorID: "human:mgr-1", ActorType: "human", ActorRole: "FINANCE", ApprovalID: "apr-credit-1",
	}); !errors.Is(err, ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation, got %v", err)
	}

	// 5. Different actor mutating amount -> payload hash mismatch.
	if _, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 1_900_000, Reason: "ramadan_stockup",
		ConsentVerified: &consent, ActorID: "human:fin-2", ActorType: "human", ActorRole: "FINANCE", ApprovalID: "apr-credit-1",
	}); !errors.Is(err, ErrApprovalPayloadMismatch) {
		t.Fatalf("expected ErrApprovalPayloadMismatch, got %v", err)
	}

	// 6. Second approver with matching payload -> applies limit + writes history + emits credit.limit.changed.
	applied, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: "ret-gov", SupplierID: "sup-gov", NewLimitMinor: 1_500_000, Reason: "ramadan_stockup",
		ConsentVerified: &consent, ActorID: "human:fin-2", ActorType: "human", ActorRole: "FINANCE",
		ApprovalID: "apr-credit-1",
	})
	if err != nil {
		t.Fatalf("RequestOrApplyLimitChange apply failed: %v", err)
	}
	if applied.Status != "APPLIED" || applied.NewLimitMinor != 1_500_000 || !containsEventType(repo.lastEvents, events.EventCreditLimitChanged) {
		t.Fatalf("unexpected applied output or missing event: %+v", applied)
	}
}

func TestLiveSpanner_CreditV2HistoryConsentLimitOutbox(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	retID := "r-" + tag
	supA := "sa-" + tag
	supB := "sb-" + tag
	supC := "sc-" + tag
	now := time.Now().UTC()

	defer func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("RetailerCreditProfiles", spanner.KeyRange{Start: spanner.Key{retID, ""}, End: spanner.Key{retID, "\uffff"}}),
			spanner.Delete("RetailerPaymentTerms", spanner.KeyRange{Start: spanner.Key{retID, ""}, End: spanner.Key{retID, "\uffff"}}),
			spanner.Delete("CreditScoreHistory", spanner.KeyRange{Start: spanner.Key{retID, ""}, End: spanner.Key{retID, "\uffff"}}),
			spanner.Delete("RetailerCreditScores", spanner.Key{retID}),
			spanner.Delete("SupplierCreditPrograms", spanner.Key{supA}),
			spanner.Delete("SupplierCreditPrograms", spanner.Key{supB}),
			spanner.Delete("SupplierCreditPrograms", spanner.Key{supC}),
		})
	}()

	// Seed programs, terms (A & B ConsentVerified=true, C false), profiles, settled + disputed invoices.
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("SupplierCreditPrograms", map[string]any{
			"SupplierId": supA, "ProgramEnabled": true, "GlobalDefaultLimitMinor": int64(400_000),
			"GlobalTermsDays": int64(30), "GlobalGraceDays": int64(0), "Timezone": "Asia/Tashkent", "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": supA, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(400_000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": supB, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(300_000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": supC, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(250_000), "UseGlobalDefaults": false,
			"ConsentVerified": false, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": supA, "CreditLimitMinor": int64(400_000), "CurrentBalanceMinor": int64(0),
			"ReservedMinor": int64(0), "AvailableCreditMinor": int64(400_000), "RiskScore": int64(0), "DelinquencyCount": int64(0),
			"PrepayOnlyLocked": false, "Status": "ACTIVE", "LastEvaluatedAt": now, "Version": int64(1),
			"CreatedAt": now, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": supC, "CreditLimitMinor": int64(250_000), "CurrentBalanceMinor": int64(200_000),
			"ReservedMinor": int64(0), "AvailableCreditMinor": int64(50_000), "RiskScore": int64(0), "DelinquencyCount": int64(9),
			"PrepayOnlyLocked": true, "Status": "BLACKLISTED", "LastEvaluatedAt": now, "Version": int64(1),
			"CreatedAt": now, "UpdatedAt": spanner.CommitTimestamp,
		}),
		// Settled invoice on supA + overdue DISPUTED invoice on supB (must be isolated from DPD!).
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": "inv-ok-" + tag, "SupplierId": supA, "RetailerId": retID, "OrderId": "ord-ok-" + tag,
			"Status": "SETTLED", "PrincipalMinor": int64(350_000), "BalanceMinor": int64(0), "Currency": "UZS",
			"CreditLeaveAt": now.Add(-15 * 24 * time.Hour), "DueAt": now.Add(-5 * 24 * time.Hour), "TermsDays": int64(30),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": "inv-disp-" + tag, "SupplierId": supB, "RetailerId": retID, "OrderId": "ord-disp-" + tag,
			"Status": "DISPUTED", "PrincipalMinor": int64(180_000), "BalanceMinor": int64(180_000), "Currency": "UZS",
			"CreditLeaveAt": now.Add(-60 * 24 * time.Hour), "DueAt": now.Add(-30 * 24 * time.Hour), "TermsDays": int64(30),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed spanner fixtures: %v", err)
	}

	_ = os.Setenv("CREDIT_ML_SCORER_REQUIRED", "false")
	svc := NewService(NewSpannerRepository(client))

	relRec, err := svc.ComputeRelationshipScore(ctx, retID, supA)
	if err != nil {
		t.Fatalf("live ComputeRelationshipScore failed: %v", err)
	}
	if !relRec.ConsentVerified || relRec.Score < 75 {
		t.Fatalf("unexpected live relationship score: %+v", relRec)
	}

	netRec, err := svc.ComputeNetworkScore(ctx, retID)
	if err != nil {
		t.Fatalf("live ComputeNetworkScore failed: %v", err)
	}
	if !netRec.ConsentVerified || netRec.DisputesIsolatedCount != 1 || netRec.Score < 75 {
		t.Fatalf("expected consenting network score isolating supB dispute and excluding unconsented supC, got %+v", netRec)
	}

	// Apply governed limit change + hold/unhold and verify CreditScoreHistory + CreditPolicyAudit + OutboxEvents.
	out, err := svc.RequestOrApplyLimitChange(ctx, LimitChangeInput{
		RetailerID: retID, SupplierID: supA, NewLimitMinor: 650_000, Reason: "live_expansion",
		ActorID: "fin-user-1", ActorType: "human", ActorRole: "ADMIN",
	})
	if err != nil || out.Status != "APPLIED" || out.NewLimitMinor != 650_000 {
		t.Fatalf("live RequestOrApplyLimitChange failed: out=%+v err=%v", out, err)
	}
	if err := svc.FreezeProfile(ctx, retID, supA, "live_hold_test"); err != nil {
		t.Fatalf("live FreezeProfile failed: %v", err)
	}
	if err := svc.UnfreezeProfile(ctx, retID, supA, "live_unhold_test"); err != nil {
		t.Fatalf("live UnfreezeProfile failed: %v", err)
	}

	// Verify CreditScoreHistory rows in live Spanner.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	histIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(1) FROM CreditScoreHistory WHERE RetailerId = @rid`,
		Params: map[string]any{"rid": retID},
	})
	row, err := histIter.Next()
	histIter.Stop()
	if err != nil {
		t.Fatalf("query CreditScoreHistory: %v", err)
	}
	var histCount int64
	_ = row.Columns(&histCount)
	if histCount < 3 {
		t.Fatalf("expected >= 3 CreditScoreHistory rows in Spanner, got %d", histCount)
	}

	// Verify OutboxEvents emitted all expected Wave 3B credit event types.
	seen := map[string]bool{}
	obIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT CAST(Payload AS STRING) FROM OutboxEvents
		      WHERE AggregateId IN (@rid, @sidRel, @sidNet, @sidLim)`,
		Params: map[string]any{
			"rid":    retID,
			"sidRel": relRec.ScoreID,
			"sidNet": netRec.ScoreID,
			"sidLim": out.ScoreRecord.ScoreID,
		},
	})
	for {
		r, err := obIter.Next()
		if err == iterator.Done || err != nil {
			break
		}
		var payload string
		if r.Columns(&payload) == nil {
			for _, evType := range []string{
				events.EventCreditScoreComputed,
				events.EventCreditLimitChanged,
				events.EventCreditHoldApplied,
				events.EventCreditHoldReleased,
			} {
				if strings.Contains(payload, evType) {
					seen[evType] = true
				}
			}
		}
	}
	obIter.Stop()
	for _, evType := range []string{
		events.EventCreditScoreComputed,
		events.EventCreditLimitChanged,
		events.EventCreditHoldApplied,
		events.EventCreditHoldReleased,
	} {
		if !seen[evType] {
			t.Fatalf("missing outbox event %s in Spanner OutboxEvents (seen=%v)", evType, seen)
		}
	}
}
