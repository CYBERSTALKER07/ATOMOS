package creditroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type memProfileRepo struct {
	profiles map[string]credit.Profile
}

func (m *memProfileRepo) GetProfile(_ context.Context, rid, sid string) (credit.Profile, bool, error) {
	p, ok := m.profiles[rid+"|"+sid]
	return p, ok, nil
}

func (m *memProfileRepo) ListBySupplier(_ context.Context, sid, status string, limit int) ([]credit.Profile, error) {
	var out []credit.Profile
	for _, p := range m.profiles {
		if p.SupplierID != sid {
			continue
		}
		if status != "" && string(p.Status) != status {
			continue
		}
		out = append(out, p)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memProfileRepo) UpsertProfile(_ context.Context, p credit.Profile, emit func(outbox.TxnBuffer) error) error {
	if emit != nil {
		_ = emit(nil)
	}
	if m.profiles == nil {
		m.profiles = map[string]credit.Profile{}
	}
	m.profiles[p.RetailerID+"|"+p.SupplierID] = p
	return nil
}

func (m *memProfileRepo) AdjustBalance(_ context.Context, _, _ string, _ int64, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *memProfileRepo) GetScoresForRetailers(_ context.Context, _ []string) (map[string]credit.RetailerCreditScore, error) {
	return nil, nil
}

func (m *memProfileRepo) ReserveOrder(_ context.Context, _ credit.OrderReservation, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *memProfileRepo) ReleaseOrderReservation(_ context.Context, _ string, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *memProfileRepo) ConvertOrderReservation(_ context.Context, _ string, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (m *memProfileRepo) GetOrderReservation(_ context.Context, _ string) (credit.OrderReservation, bool, error) {
	return credit.OrderReservation{}, false, nil
}

func (m *memProfileRepo) ListDelinquentCandidates(_ context.Context, _ int) ([]credit.Profile, error) {
	return nil, nil
}

type routeLimitBridge struct{}

func (b *routeLimitBridge) EvaluateOrVerifyLimitChange(_ context.Context, in credit.LimitChangeApprovalInput) (credit.LimitChangeApprovalResult, error) {
	if in.NewLimitMinor <= 500_000 {
		return credit.LimitChangeApprovalResult{Required: false, Approved: true}, nil
	}
	if in.ApprovalID == "" {
		return credit.LimitChangeApprovalResult{
			Required:          true,
			Approved:          false,
			ApprovalRequestID: "apr-route-1",
			PayloadHash:       "sha256-route-1",
			Status:            "PENDING",
		}, nil
	}
	if in.ApprovalID == "apr-sod" {
		return credit.LimitChangeApprovalResult{}, credit.ErrSoDViolation
	}
	if in.ApprovalID == "apr-mismatch" {
		return credit.LimitChangeApprovalResult{}, credit.ErrApprovalPayloadMismatch
	}
	return credit.LimitChangeApprovalResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: in.ApprovalID,
		PayloadHash:       "sha256-route-1",
		Status:            "APPROVED",
	}, nil
}

func issueV2Token(t *testing.T, sub string, role auth.Role, supplierID, retailerOrgID, supRole, actorType string, perms ...string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:       sub,
		Role:          role,
		SupplierID:    supplierID,
		RetailerOrgID: retailerOrgID,
		SupplierRole:  auth.Role(supRole),
		ActorType:     actorType,
		SupplierPerms: perms,
	}, auth.IssueOptions{Secret: "credit-v2-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func setupV2CreditRouter() http.Handler {
	repo := &memProfileRepo{
		profiles: map[string]credit.Profile{
			"ret-1|sup-1": {
				RetailerID:          "ret-1",
				SupplierID:          "sup-1",
				CreditLimitMinor:    400_000,
				CurrentBalanceMinor: 50_000,
				Status:              credit.StatusActive,
			},
		},
	}
	svc := credit.NewService(repo)
	svc.SetMLScorer(credit.NewHTTPMLScorerClient(credit.MLScorerConfig{Required: false}))
	sr := credit.NewMemoryScoreHistoryRepository(repo, nil)
	sr.SeedConsentingRelationship(
		repo.profiles["ret-1|sup-1"],
		credit.RetailerPaymentTerms{RetailerID: "ret-1", SupplierID: "sup-1", ConsentVerified: true},
		credit.ScoreMetrics{PaidVolume90dMinor: 900_000, PaymentsLast90d: 4, ExpectedPayments90: 4},
		true,
	)
	svc.SetScoreHistoryRepo(sr)
	svc.SetLimitApprovalBridge(&routeLimitBridge{})

	r := chi.NewRouter()
	r.Use(auth.SessionAuth("credit-v2-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func doCreditV2Req(r http.Handler, method, path, tok, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestCreditV2Routes_ScorePortfolioAndGovernedLimits(t *testing.T) {
	r := setupV2CreditRouter()

	adminTok := issueV2Token(t, "usr-admin", auth.RoleAdmin, "sup-1", "", "FINANCE", "human", auth.PermCreditLimitChange)
	retailer1Tok := issueV2Token(t, "usr-ret1", auth.RoleRetailer, "", "ret-1", "", "human")
	retailer2Tok := issueV2Token(t, "usr-ret2", auth.RoleRetailer, "", "ret-2", "", "human")
	agentTok := issueV2Token(t, "agent-1", auth.RoleAdmin, "sup-1", "", "FINANCE", "agent", auth.PermCreditLimitChange)
	auditorTok := issueV2Token(t, "aud-1", auth.RoleAdmin, "sup-1", "", "AUDITOR", "human")
	driverTok := issueV2Token(t, "drv-1", auth.RoleDriver, "sup-1", "", "", "human")

	// 1. Unauthenticated -> 401.
	if rr := doCreditV2Req(r, http.MethodGet, "/v1/credit/retailers/ret-1/score", "", ""); rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauth, got %d", rr.Code)
	}

	// 2. Driver -> 403; cross-tenant retailer -> 403; own retailer -> 200; supplier -> 200.
	if rr := doCreditV2Req(r, http.MethodGet, "/v1/credit/retailers/ret-1/score", driverTok, ""); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 driver score read, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodGet, "/v1/credit/retailers/ret-1/score", retailer2Tok, ""); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 cross-tenant retailer score read, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodGet, "/v1/credit/retailers/ret-1/score", retailer1Tok, ""); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 own retailer score read, got %d body=%s", rr.Code, rr.Body.String())
	}

	rrScore := doCreditV2Req(r, http.MethodGet, "/v1/credit/retailers/ret-1/score?scope=BOTH", adminTok, "")
	if rrScore.Code != http.StatusOK {
		t.Fatalf("expected 200 supplier score read, got %d body=%s", rrScore.Code, rrScore.Body.String())
	}
	var scoreView credit.RetailerScoreView
	if err := json.Unmarshal(rrScore.Body.Bytes(), &scoreView); err != nil {
		t.Fatalf("unmarshal scoreView: %v", err)
	}
	if scoreView.RelationshipScore == nil || scoreView.NetworkScore == nil {
		t.Fatalf("expected both relationship & network scores populated: %+v", scoreView)
	}

	// 3. Portfolio endpoint: retailer denied 403, admin gets 200 summary + items.
	if rr := doCreditV2Req(r, http.MethodGet, "/v1/credit/portfolio", retailer1Tok, ""); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 retailer portfolio, got %d", rr.Code)
	}
	rrPort := doCreditV2Req(r, http.MethodGet, "/v1/credit/portfolio", adminTok, "")
	if rrPort.Code != http.StatusOK {
		t.Fatalf("expected 200 portfolio, got %d body=%s", rrPort.Code, rrPort.Body.String())
	}
	if !bytes.Contains(rrPort.Body.Bytes(), []byte(`"total_retailers":1`)) {
		t.Fatalf("expected summary total_retailers=1, got %s", rrPort.Body.String())
	}

	// 4. POST /v1/credit/limits/ret-1: rejects float JSON, AGENT, AUDITOR; handles 202 / 403 SoD / 409 hash mismatch / 200 approved.
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", adminTok, `{"new_limit_minor":500000.5,"reason":"test"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on float literal, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", agentTok, `{"new_limit_minor":450000,"reason":"bot"}`); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AGENT limit mutation, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", auditorTok, `{"new_limit_minor":450000,"reason":"audit"}`); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AUDITOR limit mutation, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", adminTok, `{"new_limit_minor":1200000,"reason":"cel_trigger"}`); rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 APPROVAL_REQUIRED, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", adminTok, `{"new_limit_minor":1200000,"reason":"cel_trigger","approval_id":"apr-sod"}`); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 SoD, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", adminTok, `{"new_limit_minor":1200000,"reason":"cel_trigger","approval_id":"apr-mismatch"}`); rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 payload_hash_mismatch, got %d", rr.Code)
	}
	if rr := doCreditV2Req(r, http.MethodPost, "/v1/credit/limits/ret-1", adminTok, `{"new_limit_minor":1200000,"reason":"cel_trigger","approval_id":"apr-route-1"}`); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 approved limit change, got %d body=%s", rr.Code, rr.Body.String())
	}
}
