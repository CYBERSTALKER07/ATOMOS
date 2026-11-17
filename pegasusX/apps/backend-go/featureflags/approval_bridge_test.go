package featureflags_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
)

type testFFApprovalBridge struct {
	appSvc *approvals.Service
}

func (b testFFApprovalBridge) EnsureMoneyFlagRequest(
	ctx context.Context,
	tenantType, tenantID, flagKey string,
	enabled bool,
	reason, actorID, actorType, _ string,
) (featureflags.FlagApprovalDecision, error) {
	res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
		TenantID:    tenantID,
		ObjectType:  "FeatureFlag",
		ObjectID:    flagKey + ":" + tenantType + ":" + tenantID,
		Action:      "approve",
		RequestedBy: actorID,
		ActorType:   actorType,
		ForceCreate: true,
		Payload: map[string]any{
			"flag_key":    flagKey,
			"tenant_type": tenantType,
			"tenant_id":   tenantID,
			"enabled":     enabled,
			"reason":      reason,
		},
		Context: map[string]any{
			"flag_key":    flagKey,
			"tenant_type": tenantType,
			"enabled":     enabled,
		},
	})
	if err != nil {
		return featureflags.FlagApprovalDecision{}, err
	}
	if res.Request == nil {
		return featureflags.FlagApprovalDecision{}, nil
	}
	return featureflags.FlagApprovalDecision{
		Required:          res.Required,
		Approved:          res.Request.Status == approvals.StatusApproved,
		ApprovalRequestID: res.Request.RequestID,
		PayloadHash:       res.Request.PayloadHash,
	}, nil
}

func (b testFFApprovalBridge) ApproveMoneyFlagRequest(
	ctx context.Context,
	tenantType, tenantID, flagKey, actorID, actorType, actorRole string,
	approvalID, comment, totpCode string,
	mfaVerified, breakGlass bool,
	breakGlassReason string,
) (featureflags.FlagApprovalDecision, error) {
	reqID := approvalID
	if reqID == "" {
		list, err := b.appSvc.ListInbox(ctx, approvals.InboxFilter{
			TenantID:   tenantID,
			Status:     approvals.StatusPending,
			ObjectType: "FeatureFlag",
			Action:     "approve",
			Limit:      50,
		})
		if err != nil {
			return featureflags.FlagApprovalDecision{}, err
		}
		wantObj := flagKey + ":" + tenantType + ":" + tenantID
		for _, item := range list {
			if item.ObjectID == wantObj {
				reqID = item.RequestID
				break
			}
		}
	}
	if reqID == "" {
		return featureflags.FlagApprovalDecision{Required: false, Approved: true}, nil
	}
	decRes, err := b.appSvc.Decide(ctx, approvals.DecideInput{
		TenantID:         tenantID,
		RequestID:        reqID,
		ApproverID:       actorID,
		ActorType:        actorType,
		ApproverRole:     actorRole,
		Decision:         approvals.DecisionApprove,
		Comment:          comment,
		MFAVerified:      mfaVerified,
		TOTPCode:         totpCode,
		BreakGlass:       breakGlass,
		BreakGlassReason: breakGlassReason,
	})
	if err != nil {
		return featureflags.FlagApprovalDecision{}, err
	}
	return featureflags.FlagApprovalDecision{
		Required:          true,
		Approved:          decRes.Request.Status == approvals.StatusApproved,
		ApprovalRequestID: decRes.Request.RequestID,
		PayloadHash:       decRes.Request.PayloadHash,
	}, nil
}

func TestFeatureFlags_ApprovalBridge_MFA_And_Activation(t *testing.T) {
	ctx := context.Background()
	flagRepo := featureflags.NewMemoryRepository()
	flagSvc := featureflags.NewService(flagRepo)

	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	if _, err := appSvc.CreatePolicy(ctx, approvals.CreatePolicyInput{
		TenantID:      "sup-ff-1",
		Name:          "Require MFA for Money Flags",
		ObjectType:    "FeatureFlag",
		Action:        "approve",
		ConditionExpr: `flag_key == "AUTO_ORDER_PLACE_ENABLED"`,
		Steps: []approvals.ApprovalStep{
			{StepIndex: 0, Name: "Platform Admin + MFA", RequiredApprovers: 1, RequireMFA: true},
		},
		CreatedBy: "admin-setter",
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	h := &featureflags.Handlers{
		Svc:            flagSvc,
		ApprovalBridge: testFFApprovalBridge{appSvc: appSvc},
	}
	r := chi.NewRouter()
	featureflags.RegisterRoutes(r, h)

	// 1. Setter sets AUTO_ORDER_PLACE_ENABLED -> PENDING + creates ApprovalRequest.
	putBody, _ := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER", "tenant_id": "sup-ff-1", "enabled": true, "reason": "enable pilot",
	})
	reqPut := httptest.NewRequest(http.MethodPut, "/v1/platform-admin/flags/AUTO_ORDER_PLACE_ENABLED", bytes.NewReader(putBody))
	reqPut = reqPut.WithContext(auth.WithClaims(reqPut.Context(), auth.Claims{
		Subject: "admin-setter", Role: auth.RolePlatformAdmin, ActorType: auth.ActorTypeHuman,
	}))
	recPut := httptest.NewRecorder()
	r.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", recPut.Code, recPut.Body.String())
	}
	var putResp struct {
		Status            string `json:"status"`
		ApprovalRequestID string `json:"approval_request_id"`
	}
	_ = json.Unmarshal(recPut.Body.Bytes(), &putResp)
	if putResp.Status != featureflags.StatusPending || putResp.ApprovalRequestID == "" {
		t.Fatalf("expected PENDING + approval_request_id, got %+v", putResp)
	}

	// 2. Checker without MFA is blocked (403) and flag MUST revert to PENDING (fail-closed)!
	appBody, _ := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER", "tenant_id": "sup-ff-1", "approval_id": putResp.ApprovalRequestID,
	})
	reqFail := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/flags/AUTO_ORDER_PLACE_ENABLED/approve", bytes.NewReader(appBody))
	reqFail = reqFail.WithContext(auth.WithClaims(reqFail.Context(), auth.Claims{
		Subject: "admin-checker", Role: auth.RolePlatformAdmin, ActorType: auth.ActorTypeHuman, MFAVerified: false,
	}))
	recFail := httptest.NewRecorder()
	r.ServeHTTP(recFail, reqFail)
	if recFail.Code != http.StatusForbidden {
		t.Fatalf("expected 403 MFA required, got %d %s", recFail.Code, recFail.Body.String())
	}
	ov, _, _ := flagRepo.Get(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "sup-ff-1")
	if ov.Status != featureflags.StatusPending {
		t.Fatalf("expected fail-closed revert to PENDING, got %q", ov.Status)
	}

	// 3. Checker with MFAVerified=true succeeds -> ACTIVE and ApprovalRequest APPROVED!
	reqOK := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/flags/AUTO_ORDER_PLACE_ENABLED/approve", bytes.NewReader(appBody))
	reqOK = reqOK.WithContext(auth.WithClaims(reqOK.Context(), auth.Claims{
		Subject: "admin-checker", Role: auth.RolePlatformAdmin, ActorType: auth.ActorTypeHuman, MFAVerified: true,
	}))
	recOK := httptest.NewRecorder()
	r.ServeHTTP(recOK, reqOK)
	if recOK.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with MFAVerified, got %d %s", recOK.Code, recOK.Body.String())
	}
	if enabled, _, err := flagSvc.Evaluate(ctx, "AUTO_ORDER_PLACE_ENABLED", "SUPPLIER", "sup-ff-1"); err != nil || !enabled {
		t.Fatalf("expected flag active & true, got enabled=%v err=%v", enabled, err)
	}
	if det, err := appSvc.GetRequest(ctx, "sup-ff-1", putResp.ApprovalRequestID); err != nil || det.Request.Status != approvals.StatusApproved {
		t.Fatalf("expected linked ApprovalRequest APPROVED, got status=%v err=%v", det.Request.Status, err)
	}
	_ = errors.New
}
