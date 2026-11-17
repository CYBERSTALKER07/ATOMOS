package connectorroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
)

func buildTestRouter(svc *connectors.Service, secret string) chi.Router {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth(secret))
	r.Use(auth.AttachTenantFromClaims)
	RegisterRoutes(r, Deps{
		Service:   svc,
		JWTSecret: secret,
	})
	return r
}

func issueRoleToken(t *testing.T, secret, sub, supID string, supRole auth.Role, actorType string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:      sub,
		Role:         auth.RoleAdmin,
		SupplierID:   supID,
		SupplierRole: supRole,
		ActorType:    actorType,
	}, auth.IssueOptions{
		Secret: secret,
		Issuer: "pegasusx-test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

func TestConnectorRoutes_RBAC_FailClosed_And_ShadowCompareFlow(t *testing.T) {
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")
	t.Setenv("BILLZ_API_BASE_URL", "")
	t.Setenv("BILLZ_API_TOKEN", "")
	t.Setenv("PEGASUSX_ENV", "production")

	const jwtSecret = "wave3c-connector-routes-secret-123456"
	repo := connectors.NewMemoryRepository()
	repo.SeedInternalSnapshot("sup-r1", connectors.InternalSnapshot{
		Products: []connectors.InternalProduct{
			{ProductID: "SKU-1", Name: "Tea", Barcode: "460001", PriceMinor: 1200000, StockQuantity: 10, Currency: "UZS", IsActive: true},
		},
		Stocks: []connectors.InternalStock{
			{WarehouseID: "WH-1", ProductID: "SKU-1", QuantityOnHand: 10},
		},
	})

	svc := connectors.NewServiceWithRepository(repo)
	r := buildTestRouter(svc, jwtSecret)

	ownerTok := issueRoleToken(t, jwtSecret, "owner-1", "sup-r1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	auditorTok := issueRoleToken(t, jwtSecret, "aud-1", "sup-r1", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	agentTok := issueRoleToken(t, jwtSecret, "agent-1", "sup-r1", auth.SupplierRoleAgent, auth.ActorTypeAgent)
	acctTok := issueRoleToken(t, jwtSecret, "acct-1", "sup-r1", auth.SupplierRoleAccountant, auth.ActorTypeHuman)

	doReq := func(method, path, tok string, body any, idem string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. Fail-closed 503 when Billz is not configured in PEGASUSX_ENV=production.
	if rec := doReq(http.MethodPost, "/v1/connectors/billz/health", ownerTok, map[string]any{}, ""); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured billz health status=%d want=503 body=%s", rec.Code, rec.Body.String())
	}

	// 2. AUDITOR and AGENT and ACCOUNTANT blocked (403) on credential mutation.
	credBody := map[string]any{
		"provider":      "billz",
		"account_label": "default",
		"auth_type":     "bearer",
		"secret_ref":    "env://CONNECTOR_SECRET_ROUTE_BILLZ_TOKEN",
		"base_url":      "https://invalid.local",
	}
	if rec := doReq(http.MethodPut, "/v1/connectors/credentials", auditorTok, credBody, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor PUT credentials=%d want=403 body=%s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodPut, "/v1/connectors/credentials", agentTok, credBody, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("agent PUT credentials=%d want=403 body=%s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodPut, "/v1/connectors/credentials", acctTok, credBody, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("accountant PUT credentials=%d want=403 body=%s", rec.Code, rec.Body.String())
	}

	// 3. AUDITOR and AGENT blocked (403) on health/pull/shadow-compare triggers.
	for _, path := range []string{"/v1/connectors/billz/health", "/v1/connectors/billz/pull", "/v1/connectors/billz/shadow-compare"} {
		if rec := doReq(http.MethodPost, path, auditorTok, map[string]any{}, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("auditor POST %s=%d want=403", path, rec.Code)
		}
		if rec := doReq(http.MethodPost, path, agentTok, map[string]any{}, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("agent POST %s=%d want=403", path, rec.Code)
		}
	}

	// 4. Spin up live mock Billz API and upsert credential as OWNER.
	billzSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/v1/shops":
			_, _ = w.Write([]byte(`{"shops":[{"id":"WH-1","name":"Main"}]}`))
		case "/v1/products":
			_, _ = w.Write([]byte(`{"products":[{"id":"SKU-1","sku":"SKU-1","barcode":"460001","name":"Tea","price_minor":1200000,"currency":"UZS"}]}`))
		case "/v1/stocks":
			_, _ = w.Write([]byte(`{"stocks":[{"product_id":"SKU-1","warehouse_id":"WH-1","quantity_on_hand":22}]}`))
		case "/v1/orders":
			_, _ = w.Write([]byte(`{"orders":[]}`))
		case "/v1/clients":
			_, _ = w.Write([]byte(`{"clients":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer billzSrv.Close()

	t.Setenv("CONNECTOR_SECRET_ROUTE_BILLZ_TOKEN", "secret-route-token-7788")
	svc.SetHTTPClient(billzSrv.Client())
	credBody["base_url"] = billzSrv.URL
	putRec := doReq(http.MethodPut, "/v1/connectors/credentials", ownerTok, credBody, "")
	if putRec.Code != http.StatusOK {
		t.Fatalf("owner PUT credentials=%d body=%s", putRec.Code, putRec.Body.String())
	}
	if strings.Contains(putRec.Body.String(), "CONNECTOR_SECRET_ROUTE_BILLZ_TOKEN") || strings.Contains(putRec.Body.String(), "secret-route-token-7788") {
		t.Fatalf("response leaked secret ref/value: %s", putRec.Body.String())
	}

	// 5. Health, Pull, ShadowCompare as OWNER.
	if rec := doReq(http.MethodPost, "/v1/connectors/billz/health", ownerTok, map[string]any{}, ""); rec.Code != http.StatusOK {
		t.Fatalf("owner health=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodPost, "/v1/connectors/billz/pull", ownerTok, map[string]any{"stream": "products"}, "idem-pull-1"); rec.Code != http.StatusOK {
		t.Fatalf("owner pull=%d body=%s", rec.Code, rec.Body.String())
	}
	scRec := doReq(http.MethodPost, "/v1/connectors/billz/shadow-compare", ownerTok, map[string]any{}, "idem-sc-1")
	if scRec.Code != http.StatusOK {
		t.Fatalf("owner shadow-compare=%d body=%s", scRec.Code, scRec.Body.String())
	}
	var scPayload connectors.ShadowCompareResult
	if err := json.Unmarshal(scRec.Body.Bytes(), &scPayload); err != nil {
		t.Fatalf("decode shadow-compare: %v", err)
	}
	if scPayload.Summary.NetStockDelta != 12 || scPayload.Run.RunID == "" {
		t.Fatalf("unexpected shadow payload: %+v", scPayload)
	}

	// 6. AUDITOR CAN read GET /v1/connectors, GET /v1/connectors/runs, and GET /v1/connectors/runs/{runID}.
	if rec := doReq(http.MethodGet, "/v1/connectors", auditorTok, nil, ""); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/connectors=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodGet, "/v1/connectors/runs", auditorTok, nil, ""); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/connectors/runs=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	if rec := doReq(http.MethodGet, "/v1/connectors/runs/"+scPayload.Run.RunID, auditorTok, nil, ""); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/connectors/runs/{runID}=%d want=200 body=%s", rec.Code, rec.Body.String())
	}

	// 7. Reject float literals in JSON payloads.
	badFloatReq := httptest.NewRequest(http.MethodPut, "/v1/connectors/credentials", strings.NewReader(`{"provider":"billz","secret_ref":"env://CONNECTOR_SECRET_X","metadata":{"ratio":1.25}}`))
	badFloatReq.Header.Set("Authorization", "Bearer "+ownerTok)
	badFloatReq.Header.Set("Content-Type", "application/json")
	badFloatRec := httptest.NewRecorder()
	r.ServeHTTP(badFloatRec, badFloatReq)
	if badFloatRec.Code != http.StatusBadRequest {
		t.Fatalf("float literal status=%d want=400 body=%s", badFloatRec.Code, badFloatRec.Body.String())
	}
}

func TestConnectorRoutes_ExpiredAuditor_Returns401(t *testing.T) {
	const jwtSecret = "wave3c-connector-routes-expired-aud-123456"
	repo := connectors.NewMemoryRepository()
	svc := connectors.NewServiceWithRepository(repo)
	r := buildTestRouter(svc, jwtSecret)

	past := time.Now().UTC().Add(-2 * time.Minute)
	expiredTok, err := auth.Issue(auth.Claims{
		Subject:          "ext-aud-exp",
		Role:             auth.RoleAdmin,
		SupplierID:       "sup-r1",
		SupplierRole:     auth.Role(auth.SupplierRoleAuditor),
		ActorType:        auth.ActorTypeHuman,
		SupplierPerms:    []string{auth.PermAuditRead},
		AuditorExpiresAt: &past,
	}, auth.IssueOptions{
		Secret: jwtSecret,
		Issuer: "pegasusx-test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue expired auditor token: %v", err)
	}

	for _, path := range []string{"/v1/connectors", "/v1/connectors/runs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+expiredTok)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expired auditor GET %s=%d want=401 body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "auditor_access_expired") {
			t.Fatalf("expired auditor GET %s body=%s missing auditor_access_expired", path, rec.Body.String())
		}
	}
}

type testApprovalsBridge struct {
	svc *approvals.Service
}

func (b *testApprovalsBridge) VerifyPushApproval(ctx context.Context, in connectors.PushApprovalInput) error {
	if b == nil || b.svc == nil {
		return connectors.ErrApprovalRequired
	}
	_, hash, err := approvals.CanonicalizePayload(in.Payload)
	if err != nil {
		return fmt.Errorf("%w: %v", connectors.ErrInvalidInput, err)
	}
	_, _, err = b.svc.VerifyApprovedRequest(ctx, in.TenantID, in.ApprovalID, "ConnectorPush", "push", hash, 1)
	if err != nil {
		switch {
		case errors.Is(err, approvals.ErrPayloadHashMismatch):
			return fmt.Errorf("%w: %v", connectors.ErrApprovalPayloadMismatch, err)
		case errors.Is(err, approvals.ErrAgentApprovalForbidden):
			return fmt.Errorf("%w: %v", connectors.ErrAgentForbidden, err)
		default:
			return fmt.Errorf("%w: %v", connectors.ErrApprovalRequired, err)
		}
	}
	return nil
}

func TestConnectorRoutes_ApprovalGatedPush_RBAC_And_All5Providers(t *testing.T) {
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")
	t.Setenv("PEGASUSX_ENV", "production")

	const jwtSecret = "wave4a-connector-push-routes-secret-123456"
	repo := connectors.NewMemoryRepository()
	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	svc := connectors.NewServiceWithRepository(repo)
	svc.SetApprovalChecker(&testApprovalsBridge{svc: appSvc})
	r := buildTestRouter(svc, jwtSecret)

	ownerTok := issueRoleToken(t, jwtSecret, "owner-push-1", "sup-p1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	acctTok := issueRoleToken(t, jwtSecret, "acct-push-1", "sup-p1", auth.SupplierRoleAccountant, auth.ActorTypeHuman)
	auditorTok := issueRoleToken(t, jwtSecret, "aud-push-1", "sup-p1", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	agentTok := issueRoleToken(t, jwtSecret, "agt-push-1", "sup-p1", auth.SupplierRoleAgent, auth.ActorTypeAgent)
	dispTok := issueRoleToken(t, jwtSecret, "disp-push-1", "sup-p1", auth.SupplierRoleDispatcher, auth.ActorTypeHuman)

	pushSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"document_id":"ext-push-doc-1"}`))
	}))
	defer pushSrv.Close()
	svc.SetHTTPClient(pushSrv.Client())
	t.Setenv("CONNECTOR_SECRET_PUSH_ROUTE_TOK", "push-secret-val")

	for _, prov := range []string{"billz", "onec", "moysklad", "smartup", "iiko"} {
		_, err := svc.UpsertCredential(context.Background(), connectors.UpsertCredentialInput{
			TenantID: "sup-p1", Provider: prov, AccountLabel: "default",
			AuthType: connectors.AuthTypeBearer, SecretRef: "env://CONNECTOR_SECRET_PUSH_ROUTE_TOK", BaseURL: pushSrv.URL, CreatedBy: "owner-push-1",
		})
		if err != nil {
			t.Fatalf("UpsertCredential %s: %v", prov, err)
		}
	}

	doPush := func(provider, tok string, body map[string]any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/v1/connectors/"+provider+"/push", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. AUDITOR, AGENT, and DISPATCHER (missing PermConfigManage/PermJournalPost) rejected 403.
	for name, tok := range map[string]string{"AUDITOR": auditorTok, "AGENT": agentTok, "DISPATCHER": dispTok} {
		rec := doPush("billz", tok, map[string]any{"stream": "goods_receipt", "approval_id": "apr-1"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s POST /v1/connectors/billz/push=%d want=403 body=%s", name, rec.Code, rec.Body.String())
		}
	}

	// 2. OWNER without approval_id -> 403 approval_required.
	if rec := doPush("billz", ownerTok, map[string]any{
		"stream":  "goods_receipt",
		"records": []map[string]any{{"external_id": "SKU-1", "quantity_on_hand": 10}},
	}); rec.Code != http.StatusForbidden {
		t.Fatalf("missing approval_id status=%d want=403 body=%s", rec.Code, rec.Body.String())
	}

	// 3. Submit and approve real ConnectorPush approval via approvals.Service.
	for _, tc := range []struct {
		provider string
		stream   string
		tok      string
	}{
		{"billz", "goods_receipt", ownerTok},
		{"onec", "journals", acctTok},
		{"moysklad", "goods_receipt", ownerTok},
		{"smartup", "goods_receipt", acctTok},
		{"iiko", "goods_receipt", ownerTok},
	} {
		records := []map[string]any{{"object_type": "SupplierInventoryV2", "external_id": "SKU-1", "sku": "SKU-1", "quantity_on_hand": 10}}
		approvalPayload := map[string]any{
			"provider":      tc.provider,
			"account_label": "default",
			"stream":        tc.stream,
			"records":       records,
		}
		subRes, err := appSvc.SubmitRequest(context.Background(), approvals.SubmitRequestInput{
			TenantID: "sup-p1", ObjectType: "ConnectorPush", ObjectID: tc.provider, Action: "push",
			RequestedBy: "acct-push-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleAccountant,
			Payload: approvalPayload, ForceCreate: true,
		})
		if err != nil || subRes.Request == nil {
			t.Fatalf("SubmitRequest(%s): err=%v res=%+v", tc.provider, err, subRes)
		}
		// Unapproved approval_id -> 403.
		if rec := doPush(tc.provider, tc.tok, map[string]any{
			"account_label": "default", "stream": tc.stream, "approval_id": subRes.Request.RequestID, "records": records,
		}); rec.Code != http.StatusForbidden {
			t.Fatalf("unapproved %s push=%d want=403 body=%s", tc.provider, rec.Code, rec.Body.String())
		}
		// Approve by distinct human Finance Manager.
		if _, err := appSvc.Decide(context.Background(), approvals.DecideInput{
			TenantID: "sup-p1", RequestID: subRes.Request.RequestID,
			ApproverID: "fm-approver-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleFinanceManager,
			Decision: approvals.DecisionApprove,
		}); err != nil {
			t.Fatalf("Decide(%s): %v", tc.provider, err)
		}
		// Tampered quantity after approval -> 409 payload_hash_mismatch.
		if rec := doPush(tc.provider, tc.tok, map[string]any{
			"account_label": "default", "stream": tc.stream, "approval_id": subRes.Request.RequestID,
			"records": []map[string]any{{"object_type": "SupplierInventoryV2", "external_id": "SKU-1", "sku": "SKU-1", "quantity_on_hand": 999}},
		}); rec.Code != http.StatusConflict {
			t.Fatalf("tampered payload %s push=%d want=409 body=%s", tc.provider, rec.Code, rec.Body.String())
		}
		// Exact approved payload -> 200 OK.
		okRec := doPush(tc.provider, tc.tok, map[string]any{
			"account_label": "default", "stream": tc.stream, "approval_id": subRes.Request.RequestID, "records": records,
		})
		if okRec.Code != http.StatusOK {
			t.Fatalf("approved %s push=%d want=200 body=%s", tc.provider, okRec.Code, okRec.Body.String())
		}
		var pushResp connectors.PushResponse
		if err := json.Unmarshal(okRec.Body.Bytes(), &pushResp); err != nil || pushResp.PushResult.ExternalDocID != "ext-push-doc-1" {
			t.Fatalf("unexpected %s PushResponse: err=%v body=%s", tc.provider, err, okRec.Body.String())
		}
	}
}
