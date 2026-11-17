package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type testOrderDraftApprovalBridge struct {
	appSvc *approvals.Service
}

func (b testOrderDraftApprovalBridge) EvaluateOrVerifyFinancialDraft(
	ctx context.Context,
	tenantID, draftID, draftType, action, actorID, actorType, _ string,
	approvalID string,
	amountMinor int64,
	canonPayload map[string]any,
) (DraftApprovalResult, error) {
	if strings.TrimSpace(approvalID) == "" {
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:       tenantID,
			ObjectType:     "FinancialDraft",
			ObjectID:       draftID,
			Action:         "approve",
			AmountMinor:    amountMinor,
			RequestedBy:    actorID,
			ActorType:      actorType,
			Payload:        canonPayload,
			Context:        map[string]any{"draft_type": draftType, "action": action},
			IdempotencyKey: "draft:" + draftID + ":" + action,
		})
		if err != nil {
			return DraftApprovalResult{}, err
		}
		if !res.Required || res.Request == nil {
			return DraftApprovalResult{Required: false}, nil
		}
		return DraftApprovalResult{
			Required:          true,
			Approved:          res.Request.Status == approvals.StatusApproved,
			ApprovalRequestID: res.Request.RequestID,
			PolicyID:          res.Request.PolicyID,
			PolicyVersion:     res.Request.PolicyVersion,
			PayloadHash:       res.Request.PayloadHash,
		}, nil
	}

	_, hash, err := approvals.CanonicalizePayload(canonPayload)
	if err != nil {
		return DraftApprovalResult{}, err
	}
	req, _, err := b.appSvc.VerifyApprovedRequest(ctx, tenantID, approvalID, "FinancialDraft", "approve", hash, 1)
	if err != nil {
		if errors.Is(err, approvals.ErrPayloadHashMismatch) {
			return DraftApprovalResult{}, fmt.Errorf("%w: %v", ErrDraftPayloadMismatch, err)
		}
		return DraftApprovalResult{}, fmt.Errorf("%w: %v", ErrDraftApprovalForbidden, err)
	}
	return DraftApprovalResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
	}, nil
}

func TestFinancialDrafts_CELApprovalBridge_EndToEnd(t *testing.T) {
	ctx := context.Background()
	client := requireDraftSpannerClient(t, ctx)

	supplierID := "sup-ord-cel-" + uuid.NewString()[:8]
	orderID := "ord-cel-" + uuid.NewString()[:8]
	svc := newDraftTestService(map[string]Order{
		orderID: {
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: "ret-1",
			Status:     StatusCompleted,
			Currency:   "UZS",
			TotalMinor: 2_000_000,
		},
	})
	svc.spannerClient = client
	appSvc := approvals.NewService(client)
	svc.SetDraftApprovalBridge(testOrderDraftApprovalBridge{appSvc: appSvc})

	// CEL policy: credit note drafts >= 1_000_000 require CFO approval.
	if _, err := appSvc.CreatePolicy(ctx, approvals.CreatePolicyInput{
		TenantID:      supplierID,
		Name:          "Large Credit Note Approval",
		ObjectType:    "FinancialDraft",
		Action:        "approve",
		ConditionExpr: `amount_minor >= 1000000`,
		Steps: []approvals.ApprovalStep{
			{StepIndex: 0, Name: "CFO Signoff", RequiredApprovers: 1},
		},
		CreatedBy: "maker-1",
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	opClaims := &auth.Claims{Subject: "op-1", Role: auth.RoleAdmin, SupplierID: supplierID, SupplierRole: auth.SupplierRoleOwner}
	tenantCtx := &auth.TenantContext{SupplierID: supplierID}
	created, err := svc.CreateDraftCreditNote(auth.WithTenant(ctx, *tenantCtx), *opClaims, CreateDraftRequest{
		OrderID: orderID, ProposedAmountMinor: 1_500_000, ReasonCode: "DAMAGED", Notes: "damaged pallet",
	})
	if err != nil {
		t.Fatalf("CreateDraftCreditNote: %v", err)
	}
	cleanupSpannerDraftRow(t, client, draftTableCreditNotes, created.DraftID)

	// Reviewer tries to approve without prior approval -> 202 PENDING_APPROVAL + ApprovalRequestID!
	revClaims := &auth.Claims{Subject: "rev-1", Role: auth.RoleAdmin, SupplierID: supplierID, SupplierRole: auth.SupplierRoleOwner}
	rec := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID, Action: "APPROVE",
	}, revClaims, tenantCtx)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 PENDING_APPROVAL for CEL-matched credit note, got %d %s", rec.Code, rec.Body.String())
	}
	var pendingResp DraftResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &pendingResp)
	if pendingResp.ApprovalRequestID == "" {
		t.Fatalf("missing approvalRequestId in 202 response: %s", rec.Body.String())
	}

	// Unapproved approvalId rejected with 403!
	recBad := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID, Action: "APPROVE", ApprovalID: pendingResp.ApprovalRequestID,
	}, revClaims, tenantCtx)
	if recBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on unapproved approvalId, got %d %s", recBad.Code, recBad.Body.String())
	}

	// CFO approves the ApprovalRequest!
	if _, err := appSvc.Decide(ctx, approvals.DecideInput{
		TenantID: supplierID, RequestID: pendingResp.ApprovalRequestID, ApproverID: "cfo-approver",
		ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner, Decision: approvals.DecisionApprove,
	}); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	// Now Review with approvalId returns 200 APPROVED!
	recOK := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID, Action: "APPROVE", ApprovalID: pendingResp.ApprovalRequestID,
	}, revClaims, tenantCtx)
	if recOK.Code != http.StatusOK {
		t.Fatalf("expected 200 APPROVED with valid approvalId, got %d %s", recOK.Code, recOK.Body.String())
	}
}
