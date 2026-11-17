package glroutes_test

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
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
)

type testGLApprovalBridge struct {
	appSvc *approvals.Service
}

func (b testGLApprovalBridge) EvaluateOrVerifyManualJournal(
	ctx context.Context,
	entityID, actorID, actorType, _ string,
	approvalID string,
	amountMinor int64,
	in gl.JournalEntryInput,
	canonicalPayload map[string]any,
) (gl.ManualApprovalResult, error) {
	if strings.TrimSpace(approvalID) == "" {
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:    entityID,
			ObjectType:  "JournalEntry",
			ObjectID:    in.SourceID,
			Action:      "manual_post",
			AmountMinor: amountMinor,
			RequestedBy: actorID,
			ActorType:   actorType,
			Payload:     canonicalPayload,
			Context: map[string]any{
				"source_type": in.SourceType,
				"currency":    in.Currency,
			},
			IdempotencyKey: "glman:" + entityID + ":" + in.IdempotencyKey,
		})
		if err != nil {
			return gl.ManualApprovalResult{}, err
		}
		if !res.Required || res.Request == nil {
			return gl.ManualApprovalResult{Required: false}, nil
		}
		if res.Request.Status == approvals.StatusApproved {
			return gl.ManualApprovalResult{
				Required:          true,
				ApprovalRequestID: res.Request.RequestID,
				PolicyID:          res.Request.PolicyID,
				PolicyVersion:     res.Request.PolicyVersion,
				PayloadHash:       res.Request.PayloadHash,
				Status:            res.Request.Status,
			}, nil
		}
		return gl.ManualApprovalResult{
			Required:          true,
			ApprovalRequestID: res.Request.RequestID,
			PolicyID:          res.Request.PolicyID,
			PolicyVersion:     res.Request.PolicyVersion,
			PayloadHash:       res.Request.PayloadHash,
			Status:            res.Request.Status,
		}, nil
	}

	_, hash, err := approvals.CanonicalizePayload(canonicalPayload)
	if err != nil {
		return gl.ManualApprovalResult{}, err
	}
	req, _, err := b.appSvc.VerifyApprovedRequest(ctx, entityID, approvalID, "JournalEntry", "manual_post", hash, 1)
	if err != nil {
		if errors.Is(err, approvals.ErrPayloadHashMismatch) {
			return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalPayloadMismatch, err)
		}
		if errors.Is(err, approvals.ErrRequestNotFound) {
			return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalNotFound, err)
		}
		return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalPending, err)
	}
	return gl.ManualApprovalResult{
		Required:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
		Status:            req.Status,
	}, nil
}

func TestGLManualJournal_CELApprovalBridge_HashTamper_And_Posting(t *testing.T) {
	client := openRoutesSpannerClient(t)
	ctx := context.Background()

	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	entityID := "sup-gl-cel-" + uuid.NewString()[:8]
	const jwtSecret = "test-gl-cel-secret-0123456789abcdef"

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, entityID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	appSvc := approvals.NewService(client)
	glSvc.SetApprovalBridge(testGLApprovalBridge{appSvc: appSvc})

	// Configure CEL policy requiring CFO approval above 500_000 minor.
	if _, err := appSvc.CreatePolicy(ctx, approvals.CreatePolicyInput{
		TenantID:      entityID,
		Name:          "High Value Manual Journal",
		ObjectType:    "JournalEntry",
		Action:        "manual_post",
		ConditionExpr: `amount_minor > 500000`,
		Steps: []approvals.ApprovalStep{
			{StepIndex: 0, Name: "CFO Review", AllowedRoles: []string{auth.SupplierRoleOwner, auth.SupplierRoleFinanceManager}, RequiredApprovers: 1},
		},
		CreatedBy: "admin-maker",
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	r := chi.NewRouter()
	glroutes.RegisterRoutes(r, glroutes.Deps{Service: glSvc, JWTSecret: jwtSecret})
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{Service: appSvc, JWTSecret: jwtSecret})

	makerCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:       "acct-maker",
		Role:          auth.RoleAdmin,
		SupplierID:    entityID,
		SupplierRole:  auth.SupplierRoleFinanceManager,
		SupplierPerms: []string{auth.PermJournalPost},
		ActorType:     auth.ActorTypeHuman,
	})
	checkerCookie := issueTestCookie(t, jwtSecret, auth.Claims{
		Subject:       "cfo-checker",
		Role:          auth.RoleAdmin,
		SupplierID:    entityID,
		SupplierRole:  auth.SupplierRoleOwner,
		SupplierPerms: []string{auth.PermJournalApprove},
		ActorType:     auth.ActorTypeHuman,
	})

	doJSON := func(ck *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.AddCookie(ck)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// 1. Low-value manual journal (<= 500_000) does NOT match CEL -> posts immediately (201)!
	lowRec := doJSON(makerCookie, http.MethodPost, "/v1/finance/journal/manual", map[string]any{
		"source_id":       "man-low-1",
		"memo":            "small rounding adjustment",
		"currency":        "UZS",
		"idempotency_key": "man-low-1",
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 100000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 100000, "currency": "UZS"},
		},
	})
	if lowRec.Code != http.StatusCreated {
		t.Fatalf("low-value manual entry expected 201, got %d %s", lowRec.Code, lowRec.Body.String())
	}

	// 2. High-value manual journal (> 500_000) matches CEL -> 202 PENDING_APPROVAL!
	occurredTS := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond).Format(time.RFC3339Nano)
	highPayload := map[string]any{
		"source_id":       "man-high-1",
		"memo":            "major accrual adjustment",
		"currency":        "UZS",
		"idempotency_key": "man-high-1",
		"occurred_at":     occurredTS,
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 900000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 900000, "currency": "UZS"},
		},
	}
	highRec := doJSON(makerCookie, http.MethodPost, "/v1/finance/journal/manual", highPayload)
	if highRec.Code != http.StatusAccepted {
		t.Fatalf("high-value manual entry expected 202 approval_required, got %d %s", highRec.Code, highRec.Body.String())
	}
	var pend struct {
		ApprovalRequestID string `json:"approval_request_id"`
		PayloadHash       string `json:"payload_hash"`
	}
	_ = json.Unmarshal(highRec.Body.Bytes(), &pend)
	if pend.ApprovalRequestID == "" || pend.PayloadHash == "" {
		t.Fatalf("missing approval_request_id/payload_hash in 202: %s", highRec.Body.String())
	}

	// Checker approves via /v1/approvals/{id}/approve.
	appRec := doJSON(checkerCookie, http.MethodPost, "/v1/approvals/"+pend.ApprovalRequestID+"/approve", map[string]any{
		"comment": "verified accrual schedule",
	})
	if appRec.Code != http.StatusOK || !strings.Contains(appRec.Body.String(), approvals.StatusApproved) {
		t.Fatalf("approve failed: %d %s", appRec.Code, appRec.Body.String())
	}

	// 3. Re-submit with tampered amount -> 409 payload_hash_mismatch!
	tampered := map[string]any{
		"source_id":       "man-high-1",
		"memo":            "major accrual adjustment",
		"currency":        "UZS",
		"idempotency_key": "man-high-1",
		"occurred_at":     occurredTS,
		"approval_id":     pend.ApprovalRequestID,
		"postings": []map[string]any{
			{"account_code": gl.AccountCashOffice, "direction": "DEBIT", "amount_minor": 950000, "currency": "UZS"},
			{"account_code": gl.AccountSalesRevenue, "direction": "CREDIT", "amount_minor": 950000, "currency": "UZS"},
		},
	}
	tamperRec := doJSON(makerCookie, http.MethodPost, "/v1/finance/journal/manual", tampered)
	if tamperRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 payload_hash_mismatch on tampered manual journal, got %d %s", tamperRec.Code, tamperRec.Body.String())
	}

	// 4. Re-submit exact payload with approval_id -> 201 Created!
	highPayload["approval_id"] = pend.ApprovalRequestID
	postRec := doJSON(makerCookie, http.MethodPost, "/v1/finance/journal/manual", highPayload)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created with approved approval_id, got %d %s", postRec.Code, postRec.Body.String())
	}
	if time.Now().IsZero() {
		t.Fatal("unreachable")
	}
}
