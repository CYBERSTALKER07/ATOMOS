package creditroutes

import (
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

func creditRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("credit-test-secret"))
	RegisterRoutes(r, d)
	return r
}

func issueCreditToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "credit-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "credit-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCreditRoutes_AllNil(t *testing.T) {
	r := creditRouter(Deps{})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/credit-profile", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestCreditRoutes_Unauthenticated(t *testing.T) {
	r := creditRouter(Deps{
		Service:       &credit.Service{},
		PolicyService: &credit.PolicyService{},
	})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/retailer/credit-profile"},
		{http.MethodGet, "/v1/supplier/credit-profiles"},
		{http.MethodGet, "/v1/supplier/credit-scores"},
		{http.MethodGet, "/v1/supplier/credit-program"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestCreditRoutes_RoleGuard(t *testing.T) {
	r := creditRouter(Deps{
		Service:       &credit.Service{},
		PolicyService: &credit.PolicyService{},
	})

	retailerToken := issueCreditToken(t, auth.RoleRetailer)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-scores", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("retailer token returned %d, want 403 Forbidden", rr.Code)
	}
}

type routeCreditRepo struct {
	profiles map[string]credit.Profile
}

func (r *routeCreditRepo) GetProfile(_ context.Context, rid, sid string) (credit.Profile, bool, error) {
	p, ok := r.profiles[rid+"|"+sid]
	return p, ok, nil
}
func (r *routeCreditRepo) ListBySupplier(_ context.Context, sid, status string, _ int) ([]credit.Profile, error) {
	var out []credit.Profile
	for _, p := range r.profiles {
		if p.SupplierID == sid && (status == "" || string(p.Status) == status) {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *routeCreditRepo) UpsertProfile(_ context.Context, p credit.Profile, _ func(outbox.TxnBuffer) error) error {
	r.profiles[p.RetailerID+"|"+p.SupplierID] = p
	return nil
}
func (r *routeCreditRepo) AdjustBalance(_ context.Context, _, _ string, _ int64, _ func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *routeCreditRepo) GetScoresForRetailers(_ context.Context, _ []string) (map[string]credit.RetailerCreditScore, error) {
	return map[string]credit.RetailerCreditScore{}, nil
}
func (r *routeCreditRepo) ReserveOrder(_ context.Context, _ credit.OrderReservation, _ func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *routeCreditRepo) ReleaseOrderReservation(_ context.Context, _ string, _ func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *routeCreditRepo) ConvertOrderReservation(_ context.Context, _ string, _ func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *routeCreditRepo) GetOrderReservation(_ context.Context, _ string) (credit.OrderReservation, bool, error) {
	return credit.OrderReservation{}, false, nil
}
func (r *routeCreditRepo) ListDelinquentCandidates(_ context.Context, _ int) ([]credit.Profile, error) {
	return nil, nil
}

func TestCreditRoutes_PrepayLockEndpoint(t *testing.T) {
	repo := &routeCreditRepo{
		profiles: map[string]credit.Profile{
			"ret-1|sup-1": {
				RetailerID:       "ret-1",
				SupplierID:       "sup-1",
				CreditLimitMinor: 250000,
				Status:           credit.StatusActive,
			},
		},
	}
	svc := credit.NewService(repo)
	r := creditRouter(Deps{Service: svc})

	retailerToken := issueCreditToken(t, auth.RoleRetailer)
	adminToken := issueCreditToken(t, auth.RoleAdmin)

	forbidReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/credit-relationships/ret-1/prepay-lock", strings.NewReader(`{"locked":true}`))
	forbidReq.Header.Set("Authorization", "Bearer "+retailerToken)
	forbidRR := httptest.NewRecorder()
	r.ServeHTTP(forbidRR, forbidReq)
	if forbidRR.Code != http.StatusForbidden {
		t.Fatalf("retailer prepay-lock status=%d want 403", forbidRR.Code)
	}

	lockReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/credit-relationships/ret-1/prepay-lock", strings.NewReader(`{"locked":true,"reason":"collections_hold"}`))
	lockReq.Header.Set("Authorization", "Bearer "+adminToken)
	lockRR := httptest.NewRecorder()
	r.ServeHTTP(lockRR, lockReq)
	if lockRR.Code != http.StatusOK {
		t.Fatalf("admin prepay-lock status=%d body=%s", lockRR.Code, lockRR.Body.String())
	}
	var lockBody map[string]any
	if err := json.Unmarshal(lockRR.Body.Bytes(), &lockBody); err != nil {
		t.Fatal(err)
	}
	if locked, _ := lockBody["prepay_only_locked"].(bool); !locked {
		t.Fatalf("expected prepay_only_locked=true in response, got %+v", lockBody)
	}

	chkLocked, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 10000)
	if err != nil {
		t.Fatal(err)
	}
	if chkLocked.Allowed || chkLocked.Reason != "prepay_only_locked" {
		t.Fatalf("expected locked credit check rejection, got %+v", chkLocked)
	}

	unlockReq := httptest.NewRequest(http.MethodPatch, "/v1/supplier/retailer-credit-profile", strings.NewReader(`{"retailer_id":"ret-1","prepay_only_locked":false}`))
	unlockReq.Header.Set("Authorization", "Bearer "+adminToken)
	unlockRR := httptest.NewRecorder()
	r.ServeHTTP(unlockRR, unlockReq)
	if unlockRR.Code != http.StatusOK {
		t.Fatalf("admin unlock status=%d body=%s", unlockRR.Code, unlockRR.Body.String())
	}

	chkUnlocked, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 10000)
	if err != nil {
		t.Fatal(err)
	}
	if !chkUnlocked.Allowed {
		t.Fatalf("expected allowed after unlock, got %+v", chkUnlocked)
	}

	// AUDITOR and AGENT cannot modify supplier credit profile or prepay lock.
	issueSup := func(sub, sr, at string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   "sup-1",
			SupplierRole: auth.Role(sr),
			ActorType:    at,
		}, auth.IssueOptions{Secret: "credit-test-secret", TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	audTok := issueSup("aud-cred", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	agentTok := issueSup("agent-cred", auth.SupplierRoleAgent, auth.ActorTypeAgent)

	reqAudPatch := httptest.NewRequest(http.MethodPatch, "/v1/supplier/retailer-credit-profile", strings.NewReader(`{"retailer_id":"ret-1","credit_limit_minor":5000}`))
	reqAudPatch.Header.Set("Authorization", "Bearer "+audTok)
	rrAudPatch := httptest.NewRecorder()
	r.ServeHTTP(rrAudPatch, reqAudPatch)
	if rrAudPatch.Code != http.StatusForbidden {
		t.Fatalf("auditor credit patch status=%d want 403", rrAudPatch.Code)
	}

	reqAgentPatch := httptest.NewRequest(http.MethodPatch, "/v1/supplier/retailer-credit-profile", strings.NewReader(`{"retailer_id":"ret-1","credit_limit_minor":5000}`))
	reqAgentPatch.Header.Set("Authorization", "Bearer "+agentTok)
	rrAgentPatch := httptest.NewRecorder()
	r.ServeHTTP(rrAgentPatch, reqAgentPatch)
	if rrAgentPatch.Code != http.StatusForbidden {
		t.Fatalf("agent credit patch status=%d want 403", rrAgentPatch.Code)
	}
}
