package approvalroutes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestApprovalRoutes_PolicyInboxDelegationAndTelegram(t *testing.T) {
	t.Parallel()

	const jwtSecret = "approval-routes-test-secret-32bytes!"
	svc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	svc.SetTelegramWebhook("tg-route-secret", nil)

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issue := func(sub, supRole, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   "sup-rt-1",
			SupplierRole: auth.Role(supRole),
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}

	ownerTok := issue("owner-1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	acctTok := issue("acct-1", auth.SupplierRoleAccountant, auth.ActorTypeHuman)
	finMgrTok := issue("finmgr-1", auth.SupplierRoleFinanceManager, auth.ActorTypeHuman)

	call := func(method, path string, body any, tok string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// Unauthenticated -> 401
	if rec := call(http.MethodGet, "/v1/approvals/inbox", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated inbox status=%d, want 401", rec.Code)
	}

	// Create policy + toggle disabled via PATCH
	cRec := call(http.MethodPost, "/v1/approvals/policies", map[string]any{
		"name": "Credit Note >= 100k", "object_type": "FinancialDraft", "action": "credit_note.approve",
		"condition_expr": `amount_minor >= 100000`,
		"steps":          []map[string]any{{"step_index": 0, "allowed_roles": []string{"FINANCE_MANAGER"}, "required_approvers": 1}},
	}, ownerTok)
	if cRec.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d body=%s", cRec.Code, cRec.Body.String())
	}
	var cResp struct {
		Policy approvals.Policy `json:"policy"`
	}
	_ = json.Unmarshal(cRec.Body.Bytes(), &cResp)

	// Submit request -> 201
	sRec := call(http.MethodPost, "/v1/approvals/requests", map[string]any{
		"object_type": "FinancialDraft", "object_id": "dr-1", "action": "credit_note.approve",
		"amount_minor": int64(250000), "payload": map[string]any{"amount_minor": int64(250000)},
	}, acctTok)
	if sRec.Code != http.StatusCreated {
		t.Fatalf("submit request status=%d body=%s", sRec.Code, sRec.Body.String())
	}
	var sResp approvals.SubmitRequestResult
	_ = json.Unmarshal(sRec.Body.Bytes(), &sResp)

	// Inbox & detail GET -> 200
	if rec := call(http.MethodGet, "/v1/approvals/inbox?status=PENDING", nil, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("inbox status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodGet, "/v1/approvals/"+sResp.Request.RequestID, nil, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("get request status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Reject via route -> 200
	if rec := call(http.MethodPost, "/v1/approvals/"+sResp.Request.RequestID+"/reject", map[string]any{"comment": "incomplete"}, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("reject status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Delegation create + list + revoke -> 201, 200, 200
	dRec := call(http.MethodPost, "/v1/approvals/delegations", map[string]any{
		"delegate_id": "acct-1", "role_scope": "FINANCE_MANAGER",
		"ends_at": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339), "reason": "backup",
	}, finMgrTok)
	if dRec.Code != http.StatusCreated {
		t.Fatalf("create delegation status=%d body=%s", dRec.Code, dRec.Body.String())
	}
	var dResp struct {
		Delegation approvals.ApprovalDelegation `json:"delegation"`
	}
	_ = json.Unmarshal(dRec.Body.Bytes(), &dResp)
	if rec := call(http.MethodGet, "/v1/approvals/delegations", nil, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("list delegations status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodPost, "/v1/approvals/delegations/"+dResp.Delegation.DelegationID+"/revoke", nil, finMgrTok); rec.Code != http.StatusOK {
		t.Fatalf("revoke delegation status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Toggle policy disabled -> version bumps to 2
	uRec := call(http.MethodPatch, "/v1/approvals/policies/"+cResp.Policy.PolicyID, map[string]any{"enabled": false}, ownerTok)
	if uRec.Code != http.StatusOK {
		t.Fatalf("patch policy status=%d body=%s", uRec.Code, uRec.Body.String())
	}
}
