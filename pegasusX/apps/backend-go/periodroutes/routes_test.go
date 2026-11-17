package periodroutes_test

import (
	"bytes"
	"context"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/periodroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type testPeriodApprovalBridge struct {
	appSvc *approvals.Service
}

func (b testPeriodApprovalBridge) EvaluateOrVerifyPeriodReopen(
	ctx context.Context,
	tenantID, periodID, reason, actorID, actorType, actorRole, approvalID string,
	canonPayload map[string]any,
) (periods.ReopenApprovalResult, error) {
	if strings.TrimSpace(approvalID) == "" {
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:     tenantID,
			ObjectType:   "Period",
			ObjectID:     periodID,
			Action:       "reopen",
			AmountMinor:  0,
			RequestedBy:  actorID,
			ActorType:    actorType,
			Payload:      canonPayload,
			MinApprovers: 2,
			ForceCreate:  true,
		})
		if err != nil {
			return periods.ReopenApprovalResult{}, err
		}
		if res.Request == nil {
			return periods.ReopenApprovalResult{}, fmt.Errorf("missing created approval request")
		}
		return periods.ReopenApprovalResult{
			Approved:          res.Request.Status == approvals.StatusApproved,
			ApprovalRequestID: res.Request.RequestID,
			RequiredApprovers: res.Request.RequiredApprovers,
			PayloadHash:       res.Request.PayloadHash,
		}, nil
	}

	_, hash, err := approvals.CanonicalizePayload(canonPayload)
	if err != nil {
		return periods.ReopenApprovalResult{}, err
	}
	req, decisions, err := b.appSvc.VerifyApprovedRequest(ctx, tenantID, approvalID, "Period", "reopen", hash, 2)
	if err != nil {
		if errors.Is(err, approvals.ErrPayloadHashMismatch) {
			return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrPayloadMismatch, err)
		}
		if errors.Is(err, approvals.ErrSoDViolation) {
			return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrSoDViolation, err)
		}
		return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrApprovalRequired, err)
	}
	approvers := make([]string, 0, len(decisions))
	for _, d := range decisions {
		if d.Decision == approvals.DecisionApprove {
			approvers = append(approvers, d.ApproverID)
		}
	}
	return periods.ReopenApprovalResult{
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		RequiredApprovers: req.RequiredApprovers,
		PayloadHash:       req.PayloadHash,
		ApproverIDs:       approvers,
	}, nil
}

func openSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping periodroutes live test")
	}
	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(
		ctx, dbPath,
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
	ro := client.ReadOnlyTransaction()
	_, probeErr := ro.ReadRow(ctx, "Periods", spanner.Key{"__pr_probe__", "2026-01"}, []string{"Status"})
	ro.Close()
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		t.Skipf("Periods unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func TestPeriodRoutes_EndToEnd_Close_409Manual_DualApproverReopen_SnapshotCSV(t *testing.T) {
	client := openSpannerClient(t)
	ctx := context.Background()

	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	jwtSecret := "test-periodroutes-jwt-secret-32chars"
	entityA := "sup-pr-" + uuid.NewString()[:8]
	entityB := "sup-pr-other-" + uuid.NewString()[:8]

	gl.SetPeriodLockChecker(periods.CheckPeriodInTxn)
	t.Cleanup(func() { gl.SetPeriodLockChecker(nil) })

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, entityA, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	appSvc := approvals.NewService(client)
	periodSvc := periods.NewService(client)
	sink, err := periods.NewLocalSnapshotSink(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalSnapshotSink: %v", err)
	}
	periodSvc.SetSnapshotSink(sink)
	periodSvc.SetApprovalBridge(testPeriodApprovalBridge{appSvc: appSvc})

	r := chi.NewRouter()
	glroutes.RegisterRoutes(r, glroutes.Deps{Service: glSvc, JWTSecret: jwtSecret})
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{Service: appSvc, JWTSecret: jwtSecret})
	periodroutes.RegisterRoutes(r, periodroutes.Deps{Service: periodSvc, JWTSecret: jwtSecret})

	issueCookie := func(userID, supID, supplierRole string, perms ...string) *http.Cookie {
		tok, err := auth.Issue(auth.Claims{
			Subject:       userID,
			Role:          auth.RoleAdmin,
			SupplierID:    supID,
			SupplierRole:  auth.Role(supplierRole),
			SupplierPerms: perms,
			ActorType:     auth.ActorTypeHuman,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatalf("auth.Issue: %v", err)
		}
		return &http.Cookie{Name: auth.CookieName, Value: tok}
	}

	tokAdmin1 := issueCookie("fin-maker", entityA, auth.SupplierRoleFinanceManager, auth.PermPeriodClose, auth.PermPeriodReopen, auth.PermJournalPost, auth.PermConfigManage)
	tokApprover1 := issueCookie("cfo-1", entityA, auth.SupplierRoleOwner)
	tokApprover2 := issueCookie("cfo-2", entityA, auth.SupplierRoleFinanceManager, auth.PermPeriodReopen)
	tokOtherTenant := issueCookie("other-fin", entityB, auth.SupplierRoleFinanceManager, auth.PermPeriodClose)

	doReq := func(ck *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if ck != nil {
			req.AddCookie(ck)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// Tenant isolation check.
	if rec := doReq(tokOtherTenant, http.MethodGet, "/v1/finance/periods?entity="+entityA, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 cross-tenant list, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Open 2026-10 & close it.
	if rec := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods", map[string]any{
		"period_id": "2026-10", "timezone": "Asia/Tashkent",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("open 2026-10 failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods/2026-10/soft-close", nil); rec.Code != http.StatusOK {
		t.Fatalf("soft-close failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(tokAdmin1, http.MethodGet, "/v1/finance/periods/2026-10/checklist", nil); rec.Code != http.StatusOK {
		t.Fatalf("checklist failed: %d %s", rec.Code, rec.Body.String())
	}
	closeRec := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods/2026-10/close", nil)
	if closeRec.Code != http.StatusOK {
		t.Fatalf("close failed: %d %s", closeRec.Code, closeRec.Body.String())
	}

	// Snapshot JSON + CSV endpoints.
	snapJSONRec := doReq(tokAdmin1, http.MethodGet, "/v1/finance/periods/2026-10/snapshot?format=json", nil)
	if snapJSONRec.Code != http.StatusOK {
		t.Fatalf("snapshot json failed: %d %s", snapJSONRec.Code, snapJSONRec.Body.String())
	}
	var snapBody struct {
		SnapshotHash string `json:"snapshot_hash"`
		SnapshotURI  string `json:"snapshot_uri"`
	}
	_ = json.Unmarshal(snapJSONRec.Body.Bytes(), &snapBody)
	fileRaw, err := os.ReadFile(strings.TrimPrefix(snapBody.SnapshotURI, "file://"))
	if err != nil {
		t.Fatalf("read snapshot file: %v", err)
	}
	fileSum := sha256.Sum256(fileRaw)
	if hex.EncodeToString(fileSum[:]) != snapBody.SnapshotHash {
		t.Fatalf("snapshot hash mismatch")
	}

	snapCSVRec := doReq(tokAdmin1, http.MethodGet, "/v1/finance/periods/2026-10/snapshot?format=csv", nil)
	if snapCSVRec.Code != http.StatusOK || !strings.Contains(snapCSVRec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("snapshot csv failed: %d %s", snapCSVRec.Code, snapCSVRec.Header().Get("Content-Type"))
	}

	// Manual journal in CLOSED 2026-10 -> 409 period_closed!
	manRec := doReq(tokAdmin1, http.MethodPost, "/v1/finance/journal/manual", map[string]any{
		"period_id":       "2026-10",
		"source_type":     "MANUAL",
		"source_id":       "man-locked-1",
		"memo":            "blocked in closed period",
		"currency":        "UZS",
		"idempotency_key": "man-locked-1",
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 10000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 10000, "currency": "UZS"},
		},
	})
	if manRec.Code != http.StatusConflict || !strings.Contains(manRec.Body.String(), "period_closed") {
		t.Fatalf("expected 409 period_closed, got %d %s", manRec.Code, manRec.Body.String())
	}

	// Dual-approver reopen via Approvals workflow.
	reopenReq := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods/2026-10/reopen", map[string]any{
		"reason": "Fix fiscal posting cutoff",
	})
	if reopenReq.Code != http.StatusAccepted {
		t.Fatalf("expected 202 approval_required, got %d %s", reopenReq.Code, reopenReq.Body.String())
	}
	var pendingBody struct {
		ApprovalRequestID string `json:"approval_request_id"`
		RequiredApprovers int64  `json:"required_approvers"`
		PayloadHash       string `json:"payload_hash"`
	}
	_ = json.Unmarshal(reopenReq.Body.Bytes(), &pendingBody)
	if pendingBody.ApprovalRequestID == "" || pendingBody.RequiredApprovers != 2 {
		t.Fatalf("unexpected pending reopen payload: %+v", pendingBody)
	}

	// Approver 1 approves -> still PENDING (1/2).
	app1 := doReq(tokApprover1, http.MethodPost, "/v1/approvals/"+pendingBody.ApprovalRequestID+"/approve", map[string]any{
		"comment": "CFO 1 approved",
	})
	if app1.Code != http.StatusOK || !strings.Contains(app1.Body.String(), approvals.StatusPending) {
		t.Fatalf("expected PENDING after 1/2 approvals, got %d %s", app1.Code, app1.Body.String())
	}

	// Approver 2 approves -> APPROVED (2/2)!
	app2 := doReq(tokApprover2, http.MethodPost, "/v1/approvals/"+pendingBody.ApprovalRequestID+"/approve", map[string]any{
		"comment": "Finance Manager 2 approved",
	})
	if app2.Code != http.StatusOK || !strings.Contains(app2.Body.String(), approvals.StatusApproved) {
		t.Fatalf("expected APPROVED after 2/2 approvals, got %d %s", app2.Code, app2.Body.String())
	}

	// Reopen with mutated reason MUST fail with 409/403 (payload hash mismatch)!
	mutatedReopen := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods/2026-10/reopen", map[string]any{
		"reason":      "Tampered reason after approvals",
		"approval_id": pendingBody.ApprovalRequestID,
	})
	if mutatedReopen.Code != http.StatusForbidden && mutatedReopen.Code != http.StatusConflict {
		t.Fatalf("expected error on mutated reopen reason, got %d %s", mutatedReopen.Code, mutatedReopen.Body.String())
	}

	// Reopen with exact reason + approved approval_id -> 200 REOPENED!
	okReopen := doReq(tokAdmin1, http.MethodPost, "/v1/finance/periods/2026-10/reopen", map[string]any{
		"reason":      "Fix fiscal posting cutoff",
		"approval_id": pendingBody.ApprovalRequestID,
	})
	if okReopen.Code != http.StatusOK || !strings.Contains(okReopen.Body.String(), periods.StatusReopened) {
		t.Fatalf("expected 200 REOPENED, got %d %s", okReopen.Code, okReopen.Body.String())
	}

	// Verify GET /v1/finance/periods lists 2026-10 as REOPENED.
	listRec := doReq(tokAdmin1, http.MethodGet, "/v1/finance/periods?status=REOPENED", nil)
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), "2026-10") {
		t.Fatalf("expected 2026-10 in REOPENED list, got %d %s", listRec.Code, listRec.Body.String())
	}
}
