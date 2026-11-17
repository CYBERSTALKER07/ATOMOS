package auditroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

const testJWTSecret = "audit-routes-test-secret-0123456789"

func mintToken(t *testing.T, claims auth.Claims) string {
	t.Helper()
	tok, err := auth.Issue(claims, auth.IssueOptions{
		Secret: testJWTSecret,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func newTestRouter(svc *audit.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth(testJWTSecret))
	r.Use(auth.AttachTenantFromClaims)
	r.Use(audit.AuditMiddleware(svc))
	RegisterRoutes(r, Deps{
		Service:   svc,
		JWTSecret: testJWTSecret,
	})
	return r
}

func doReq(r chi.Router, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestAuditRoutes_RBAC_PIIMasking_TenantIsolation_AndChainVerify(t *testing.T) {
	ctx := context.Background()
	memRepo := audit.NewMemoryRepository()
	svc := audit.NewServiceWithRepository(memRepo)
	t.Setenv("WORM_ANCHOR_LOCAL_DIR", t.TempDir())
	t.Setenv("WORM_ANCHOR_GCS_BUCKET", "")
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")

	runLifecycle := func(t *testing.T, router chi.Router, svc *audit.Service, supA, supB string) {
		t.Helper()
		today := time.Now().UTC().Format("2006-01-02")

		if _, err := svc.Record(ctx, audit.EventInput{
			TenantID:   supA,
			DayBucket:  today,
			ActorID:    "usr-fin-a",
			ActorType:  audit.ActorTypeHuman,
			Role:       "FINANCE_MANAGER",
			Action:     "credit.profile.updated",
			ObjectType: "retailer",
			ObjectID:   "ret-pii-1",
			AfterJSON:  `{"phone":"+998901234567","PhoneNumber":"+998937654321","inn":"301234567","limit_minor":5000000}`,
		}); err != nil {
			t.Fatalf("seed event A: %v", err)
		}
		if _, err := svc.Record(ctx, audit.EventInput{
			TenantID:   supB,
			DayBucket:  today,
			ActorID:    "usr-owner-b",
			ActorType:  audit.ActorTypeHuman,
			Role:       "OWNER",
			Action:     "order.create",
			ObjectType: "order",
			ObjectID:   "ord-b-1",
		}); err != nil {
			t.Fatalf("seed event B: %v", err)
		}

		tokOwnerA := mintToken(t, auth.Claims{Subject: "usr-owner-a", Role: auth.RoleAdmin, SupplierID: supA})
		tokFinanceA := mintToken(t, auth.Claims{Subject: "usr-fin-a", Role: auth.RoleAdmin, SupplierID: supA, SupplierRole: auth.Role("FINANCE_MANAGER")})
		tokAuditorA := mintToken(t, auth.Claims{Subject: "usr-aud-a", Role: auth.RoleAdmin, SupplierID: supA, SupplierRole: auth.Role("AUDITOR")})
		tokAgentA := mintToken(t, auth.Claims{Subject: "agent-a", Role: auth.RoleAdmin, SupplierID: supA, ActorType: auth.ActorTypeAgent})
		tokDispatcherA := mintToken(t, auth.Claims{Subject: "disp-a", Role: auth.RoleAdmin, SupplierID: supA, SupplierRole: auth.Role("DISPATCHER")})
		tokPlatform := mintToken(t, auth.Claims{Subject: "plat-admin", Role: auth.RolePlatformAdmin})

		// 1. Unauthenticated -> 401
		if rr := doReq(router, http.MethodGet, "/v1/audit/events", "", nil); rr.Code != http.StatusUnauthorized {
			t.Fatalf("unauth status = %d want 401 body=%s", rr.Code, rr.Body.String())
		}

		// 2. DISPATCHER lacks audit.read -> 403
		if rr := doReq(router, http.MethodGet, "/v1/audit/events", tokDispatcherA, nil); rr.Code != http.StatusForbidden {
			t.Fatalf("dispatcher audit.read status = %d want 403 body=%s", rr.Code, rr.Body.String())
		}

		// 3. AUDITOR can read history -> 200 AND PII MUST BE MASKED!
		rrAudHist := doReq(router, http.MethodGet, "/v1/audit/history/retailer/ret-pii-1", tokAuditorA, nil)
		if rrAudHist.Code != http.StatusOK {
			t.Fatalf("auditor history status = %d body=%s", rrAudHist.Code, rrAudHist.Body.String())
		}
		if !strings.Contains(rrAudHist.Body.String(), `"pii_masked":true`) {
			t.Fatalf("expected pii_masked=true for AUDITOR, got %s", rrAudHist.Body.String())
		}
		if strings.Contains(rrAudHist.Body.String(), "998901234567") || strings.Contains(rrAudHist.Body.String(), "301234567") {
			t.Fatalf("AUDITOR leaked PII: %s", rrAudHist.Body.String())
		}

		// 4. FINANCE_MANAGER sees unmasked PII -> 200 with pii_masked=false
		rrFinHist := doReq(router, http.MethodGet, "/v1/audit/history/retailer/ret-pii-1", tokFinanceA, nil)
		if rrFinHist.Code != http.StatusOK {
			t.Fatalf("finance history status = %d body=%s", rrFinHist.Code, rrFinHist.Body.String())
		}
		if !strings.Contains(rrFinHist.Body.String(), `"pii_masked":false`) || !strings.Contains(rrFinHist.Body.String(), "+998901234567") {
			t.Fatalf("expected unmasked PII for FINANCE_MANAGER, got %s", rrFinHist.Body.String())
		}

		// 5. AUDITOR, FINANCE_MANAGER, and AGENT cannot mutate audit anchors or legal holds -> 403
		for name, tok := range map[string]string{"AUDITOR": tokAuditorA, "FINANCE_MANAGER": tokFinanceA, "AGENT": tokAgentA} {
			if rr := doReq(router, http.MethodPost, "/v1/audit/anchors", tok, map[string]any{"date": today}); rr.Code != http.StatusForbidden {
				t.Fatalf("%s POST /v1/audit/anchors status = %d want 403 body=%s", name, rr.Code, rr.Body.String())
			}
			if rr := doReq(router, http.MethodPost, "/v1/audit/legal-hold", tok, map[string]any{"object_type": "order", "object_id": "ord-1", "reason": "hold"}); rr.Code != http.StatusForbidden {
				t.Fatalf("%s POST /v1/audit/legal-hold status = %d want 403 body=%s", name, rr.Code, rr.Body.String())
			}
		}

		// 6. Cross-tenant ?entity=supB blocked for supA OWNER -> 403
		if rr := doReq(router, http.MethodGet, "/v1/audit/events?entity="+supB, tokOwnerA, nil); rr.Code != http.StatusForbidden {
			t.Fatalf("cross-tenant audit read status = %d want 403 body=%s", rr.Code, rr.Body.String())
		}

		// 7. OWNER creates legal hold -> 201, lists legal holds -> 200
		rrHold := doReq(router, http.MethodPost, "/v1/audit/legal-hold", tokOwnerA, map[string]any{
			"object_type": "retailer",
			"object_id":   "ret-pii-1",
			"reason":      "litigation hold",
		})
		if rrHold.Code != http.StatusCreated {
			t.Fatalf("create legal hold status = %d want 201 body=%s", rrHold.Code, rrHold.Body.String())
		}
		rrListHolds := doReq(router, http.MethodGet, "/v1/audit/legal-holds?object=retailer&object_id=ret-pii-1", tokAuditorA, nil)
		if rrListHolds.Code != http.StatusOK || !strings.Contains(rrListHolds.Body.String(), "litigation hold") {
			t.Fatalf("list legal holds failed status=%d body=%s", rrListHolds.Code, rrListHolds.Body.String())
		}

		// 8. OWNER writes daily anchor -> 201, lists anchors -> 200, verifies day chain -> 200 valid+matched
		rrAnchor := doReq(router, http.MethodPost, "/v1/audit/anchors", tokOwnerA, map[string]any{"date": today})
		if rrAnchor.Code != http.StatusCreated {
			t.Fatalf("write anchor status = %d want 201 body=%s", rrAnchor.Code, rrAnchor.Body.String())
		}
		rrListAnchors := doReq(router, http.MethodGet, "/v1/audit/anchors?date="+today, tokAuditorA, nil)
		if rrListAnchors.Code != http.StatusOK || !strings.Contains(rrListAnchors.Body.String(), today) {
			t.Fatalf("list anchors failed status=%d body=%s", rrListAnchors.Code, rrListAnchors.Body.String())
		}
		rrVerify := doReq(router, http.MethodGet, "/v1/audit/verify?date="+today, tokAuditorA, nil)
		if rrVerify.Code != http.StatusOK {
			t.Fatalf("verify chain status = %d body=%s", rrVerify.Code, rrVerify.Body.String())
		}
		var verifyRes audit.ChainVerifyResult
		if err := json.Unmarshal(rrVerify.Body.Bytes(), &verifyRes); err != nil {
			t.Fatalf("decode verify response: %v", err)
		}
		if !verifyRes.Valid || verifyRes.EventCount < 3 {
			t.Fatalf("expected valid chain with >=3 events, got %+v", verifyRes)
		}

		// 9. Bad date -> 400
		if rr := doReq(router, http.MethodGet, "/v1/audit/verify?date=bad-date", tokOwnerA, nil); rr.Code != http.StatusBadRequest {
			t.Fatalf("bad date status = %d want 400 body=%s", rr.Code, rr.Body.String())
		}

		// 10. PLATFORM_ADMIN can view cross-tenant events (empty entity) and verify specific ?entity=supB
		rrPlatCross := doReq(router, http.MethodGet, "/v1/audit/events?limit=50", tokPlatform, nil)
		if rrPlatCross.Code != http.StatusOK || !strings.Contains(rrPlatCross.Body.String(), supB) {
			t.Fatalf("platform cross-tenant events failed status=%d body=%s", rrPlatCross.Code, rrPlatCross.Body.String())
		}
		rrPlatVerifyB := doReq(router, http.MethodGet, "/v1/audit/verify?date="+today+"&entity="+supB, tokPlatform, nil)
		if rrPlatVerifyB.Code != http.StatusOK || !strings.Contains(rrPlatVerifyB.Body.String(), `"valid":true`) {
			t.Fatalf("platform verify supB failed status=%d body=%s", rrPlatVerifyB.Code, rrPlatVerifyB.Body.String())
		}
	}

	t.Run("InMemory", func(t *testing.T) {
		runLifecycle(t, newTestRouter(svc), svc, "sup-route-a", "sup-route-b")
	})

	t.Run("LiveSpanner", func(t *testing.T) {
		requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
		host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
		if host == "" {
			if requireSpanner {
				t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
			}
			t.Skip("SPANNER_EMULATOR_HOST not set")
		}
		envOr := func(k, def string) string {
			if v := strings.TrimSpace(os.Getenv(k)); v != "" {
				return v
			}
			return def
		}
		dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
			envOr("SPANNER_PROJECT", "pegasusx-local"),
			envOr("SPANNER_INSTANCE", "pegasusx-instance"),
			envOr("SPANNER_DATABASE", "pegasusx-db"))
		cCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		client, err := spanner.NewClient(cCtx, dbPath,
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
		defer client.Close()
		if _, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(cCtx, "AuditEvents", spanner.Key{"__probe__", "2026-01-01", "__probe__"}, []string{"EventId"}); err != nil && spanner.ErrCode(err) != codes.NotFound {
			if requireSpanner {
				t.Fatalf("AuditEvents unavailable: %v", err)
			}
			t.Skipf("AuditEvents unavailable: %v", err)
		}

		suffix := time.Now().UnixNano()
		supA := fmt.Sprintf("sup-rt-a-%d", suffix)
		supB := fmt.Sprintf("sup-rt-b-%d", suffix)
		today := time.Now().UTC().Format("2006-01-02")
		t.Cleanup(func() {
			_, _ = client.Apply(context.Background(), []*spanner.Mutation{
				spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{supA}, End: spanner.Key{supA}, Kind: spanner.ClosedClosed}),
				spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{supB}, End: spanner.Key{supB}, Kind: spanner.ClosedClosed}),
				spanner.Delete("AuditChainAnchors", spanner.Key{supA, today}),
				spanner.Delete("AuditChainAnchors", spanner.Key{supB, today}),
				spanner.Delete("AuditLegalHolds", spanner.KeyRange{Start: spanner.Key{supA}, End: spanner.Key{supA}, Kind: spanner.ClosedClosed}),
			})
		})
		spSvc := audit.NewService(client)
		runLifecycle(t, newTestRouter(spSvc), spSvc, supA, supB)
	})
}
