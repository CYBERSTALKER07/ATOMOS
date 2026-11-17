package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/featureflags"
	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

type approvalsAuditRecorder struct {
	auditSvc *audit.Service
}

// NewApprovalsAuditRecorder adapts *audit.Service to approvals.AuditRecorder.
func NewApprovalsAuditRecorder(auditSvc *audit.Service) approvals.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &approvalsAuditRecorder{auditSvc: auditSvc}
}

func (a *approvalsAuditRecorder) RecordEvent(ctx context.Context, ev approvals.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, approvalsAuditEventInput(ev))
	return err
}

func (a *approvalsAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev approvals.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, approvalsAuditEventInput(ev))
	return err
}

func approvalsAuditEventInput(ev approvals.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

type periodsAuditRecorder struct {
	auditSvc *audit.Service
}

// NewPeriodsAuditRecorder adapts *audit.Service to periods.AuditRecorder.
func NewPeriodsAuditRecorder(auditSvc *audit.Service) periods.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &periodsAuditRecorder{auditSvc: auditSvc}
}

func (a *periodsAuditRecorder) RecordEvent(ctx context.Context, ev periods.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, periodsAuditEventInput(ev))
	return err
}

func (a *periodsAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev periods.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, periodsAuditEventInput(ev))
	return err
}

func periodsAuditEventInput(ev periods.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

type approvalsSoDAdapter struct {
	iamSvc *iam.Service
}

// NewApprovalsSoDChecker adapts *iam.Service to approvals.SoDChecker.
func NewApprovalsSoDChecker(iamSvc *iam.Service) approvals.SoDChecker {
	if iamSvc == nil || iamSvc.Repo() == nil {
		return nil
	}
	return &approvalsSoDAdapter{iamSvc: iamSvc}
}

func (a *approvalsSoDAdapter) CheckFourEyes(ctx context.Context, in approvals.SoDCheckInput) error {
	if a == nil || a.iamSvc == nil {
		return nil
	}
	return a.iamSvc.CheckFourEyes(ctx, iam.CheckFourEyesInput{
		TenantID:    in.TenantID,
		ActorID:     in.ActorID,
		ActorType:   in.ActorType,
		Role:        in.Role,
		Action:      in.Action,
		MakerAction: in.MakerAction,
		ObjectType:  in.ObjectType,
		ObjectID:    in.ObjectID,
		MakerID:     in.MakerID,
		RoutePath:   in.RoutePath,
		TraceID:     in.TraceID,
		IP:          in.IP,
		DeviceID:    in.DeviceID,
		IsApproval:  in.IsApproval,
	})
}

type glApprovalBridge struct {
	appSvc *approvals.Service
}

// NewGLApprovalBridge connects gl.HandleManualJournal to the M1.2 CEL Approval Engine.
func NewGLApprovalBridge(appSvc *approvals.Service) gl.ManualApprovalBridge {
	if appSvc == nil || appSvc.Repo() == nil {
		return nil
	}
	return &glApprovalBridge{appSvc: appSvc}
}

func (b *glApprovalBridge) EvaluateOrVerifyManualJournal(
	ctx context.Context,
	entityID, actorID, actorType, actorRole, approvalID string,
	amountMinor int64,
	in gl.JournalEntryInput,
	canonicalPayload map[string]any,
) (gl.ManualApprovalResult, error) {
	if strings.TrimSpace(approvalID) == "" {
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:     entityID,
			ObjectType:   "JournalEntry",
			ObjectID:     in.SourceID,
			Action:       "manual_post",
			AmountMinor:  amountMinor,
			RequestedBy:  actorID,
			ActorType:    actorType,
			ApproverRole: actorRole,
			Payload:      canonicalPayload,
			Context: map[string]any{
				"source_type": in.SourceType,
				"currency":    in.Currency,
				"period_id":   in.PeriodID,
			},
			IdempotencyKey: "glman:" + entityID + ":" + in.IdempotencyKey,
		})
		if err != nil {
			return gl.ManualApprovalResult{}, err
		}
		if !res.Required || res.Request == nil {
			return gl.ManualApprovalResult{Required: false}, nil
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
		switch {
		case errors.Is(err, approvals.ErrPayloadHashMismatch):
			return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalPayloadMismatch, err)
		case errors.Is(err, approvals.ErrRequestNotFound):
			return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalNotFound, err)
		default:
			return gl.ManualApprovalResult{}, fmt.Errorf("%w: %v", gl.ErrApprovalPending, err)
		}
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

type periodApprovalBridge struct {
	appSvc *approvals.Service
}

// NewPeriodApprovalBridge connects M1.1 period reopen workflows to M1.2 dual-approver approvals.
func NewPeriodApprovalBridge(appSvc *approvals.Service) periods.ApprovalBridge {
	if appSvc == nil || appSvc.Repo() == nil {
		return nil
	}
	return &periodApprovalBridge{appSvc: appSvc}
}

func (b *periodApprovalBridge) EvaluateOrVerifyPeriodReopen(
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
			ApproverRole: actorRole,
			Payload:      canonPayload,
			Context: map[string]any{
				"period_id": periodID,
				"reason":    reason,
			},
			MinApprovers: 2,
			ForceCreate:  true,
		})
		if err != nil {
			return periods.ReopenApprovalResult{}, err
		}
		if res.Request == nil {
			return periods.ReopenApprovalResult{}, periods.ErrApprovalRequired
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
		return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrInvalidInput, err)
	}
	req, decisions, err := b.appSvc.VerifyApprovedRequest(ctx, tenantID, approvalID, "Period", "reopen", hash, 2)
	if err != nil {
		switch {
		case errors.Is(err, approvals.ErrPayloadHashMismatch):
			return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrPayloadMismatch, err)
		case errors.Is(err, approvals.ErrSoDViolation), errors.Is(err, approvals.ErrAgentApprovalForbidden):
			return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrSoDViolation, err)
		default:
			return periods.ReopenApprovalResult{}, fmt.Errorf("%w: %v", periods.ErrApprovalRequired, err)
		}
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

type featureFlagApprovalBridge struct {
	appSvc *approvals.Service
}

// NewFeatureFlagApprovalBridge connects money-affecting feature flags to M1.2 CEL approvals.
func NewFeatureFlagApprovalBridge(appSvc *approvals.Service) featureflags.FlagApprovalBridge {
	if appSvc == nil || appSvc.Repo() == nil {
		return nil
	}
	return &featureFlagApprovalBridge{appSvc: appSvc}
}

func (b *featureFlagApprovalBridge) EnsureMoneyFlagRequest(
	ctx context.Context,
	tenantType, tenantID, flagKey string,
	enabled bool,
	reason, actorID, actorType, actorRole string,
) (featureflags.FlagApprovalDecision, error) {
	scopeTenant := strings.TrimSpace(tenantID)
	if scopeTenant == "" {
		scopeTenant = strings.TrimSpace(tenantType)
	}
	res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
		TenantID:     scopeTenant,
		ObjectType:   "FeatureFlag",
		ObjectID:     strings.ToUpper(flagKey) + ":" + strings.ToUpper(tenantType) + ":" + tenantID,
		Action:       "approve",
		RequestedBy:  actorID,
		ActorType:    actorType,
		ApproverRole: actorRole,
		ForceCreate:  true,
		Payload: map[string]any{
			"flag_key":    strings.ToUpper(flagKey),
			"tenant_type": strings.ToUpper(tenantType),
			"tenant_id":   tenantID,
			"enabled":     enabled,
			"reason":      reason,
		},
		Context: map[string]any{
			"flag_key":    strings.ToUpper(flagKey),
			"tenant_type": strings.ToUpper(tenantType),
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

func (b *featureFlagApprovalBridge) ApproveMoneyFlagRequest(
	ctx context.Context,
	tenantType, tenantID, flagKey, actorID, actorType, actorRole string,
	approvalID, comment, totpCode string,
	mfaVerified, breakGlass bool,
	breakGlassReason string,
) (featureflags.FlagApprovalDecision, error) {
	scopeTenant := strings.TrimSpace(tenantID)
	if scopeTenant == "" {
		scopeTenant = strings.TrimSpace(tenantType)
	}
	reqID := strings.TrimSpace(approvalID)
	if reqID == "" {
		list, err := b.appSvc.ListInbox(ctx, approvals.InboxFilter{
			TenantID:   scopeTenant,
			Status:     approvals.StatusPending,
			ObjectType: "FeatureFlag",
			Action:     "approve",
			Limit:      50,
		})
		if err != nil {
			return featureflags.FlagApprovalDecision{}, err
		}
		wantObj := strings.ToUpper(flagKey) + ":" + strings.ToUpper(tenantType) + ":" + tenantID
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
		TenantID:         scopeTenant,
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

type orderDraftApprovalBridge struct {
	appSvc *approvals.Service
}

// NewOrderDraftApprovalBridge connects high-value financial drafts/credit notes to M1.2 CEL approvals.
func NewOrderDraftApprovalBridge(appSvc *approvals.Service) order.DraftApprovalBridge {
	if appSvc == nil || appSvc.Repo() == nil {
		return nil
	}
	return &orderDraftApprovalBridge{appSvc: appSvc}
}

func (b *orderDraftApprovalBridge) EvaluateOrVerifyFinancialDraft(
	ctx context.Context,
	tenantID, draftID, draftType, action, actorID, actorType, actorRole, approvalID string,
	amountMinor int64,
	canonPayload map[string]any,
) (order.DraftApprovalResult, error) {
	if strings.TrimSpace(approvalID) == "" {
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:     tenantID,
			ObjectType:   "FinancialDraft",
			ObjectID:     draftID,
			Action:       "approve",
			AmountMinor:  amountMinor,
			RequestedBy:  actorID,
			ActorType:    actorType,
			ApproverRole: actorRole,
			Payload:      canonPayload,
			Context: map[string]any{
				"draft_type": draftType,
				"action":     action,
			},
			IdempotencyKey: "draft:" + draftID + ":" + action,
		})
		if err != nil {
			return order.DraftApprovalResult{}, err
		}
		if !res.Required || res.Request == nil {
			return order.DraftApprovalResult{Required: false}, nil
		}
		return order.DraftApprovalResult{
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
		return order.DraftApprovalResult{}, fmt.Errorf("%w: %v", order.ErrDraftValidation, err)
	}
	req, _, err := b.appSvc.VerifyApprovedRequest(ctx, tenantID, approvalID, "FinancialDraft", "approve", hash, 1)
	if err != nil {
		if errors.Is(err, approvals.ErrPayloadHashMismatch) {
			return order.DraftApprovalResult{}, fmt.Errorf("%w: %v", order.ErrDraftPayloadMismatch, err)
		}
		return order.DraftApprovalResult{}, fmt.Errorf("%w: %v", order.ErrDraftApprovalForbidden, err)
	}
	return order.DraftApprovalResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
	}, nil
}

// SetupWave3A initializes and wires M1.1 Periods, M1.2 Approvals, M1.3 Inventory Valuation,
// and M1.4 ESF Fiscal Services along with all cross-module transactional hooks and bridges.
func SetupWave3A(
	ctx context.Context,
	cfg *Config,
	spannerClient *spanner.Client,
	fiscalProvider order.FiscalProvider,
	app *App,
	log *slog.Logger,
) error {
	if app == nil {
		return nil
	}

	// 1. Construct Approvals, Periods, Valuation, and ESF services if not already injected.
	if app.Approvals == nil {
		if spannerClient != nil {
			app.Approvals = approvals.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.Approvals = approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
		}
	}
	if app.Periods == nil {
		if spannerClient != nil {
			app.Periods = periods.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.Periods = periods.NewServiceWithRepository(periods.NewMemoryRepository())
		}
	}
	if app.Valuation == nil {
		if spannerClient != nil {
			app.Valuation = valuation.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.Valuation = valuation.NewServiceWithRepository(valuation.NewMemoryRepository())
		}
	}
	if app.ESF == nil {
		var esfRepo esf.Repository
		if spannerClient != nil {
			esfRepo = esf.NewSpannerRepository(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			esfRepo = esf.NewMemoryRepository()
		}
		if esfRepo != nil {
			var soliqCli soliq.SoliqClient
			if fiscalProvider != nil {
				soliqCli = fiscalProvider.GetSoliqClient()
			}
			var signer fiscal.EDSSigner
			if s, sErr := fiscal.SignerFromEnv(os.Getenv("PEGASUSX_ENV")); sErr == nil {
				signer = s
			} else if log != nil {
				log.Debug("esf signer not preconfigured from env; live submit will require EDS config", "reason", sErr)
			}
			app.ESF = esf.NewService(esf.ServiceConfig{
				Repo:          esfRepo,
				SoliqClient:   soliqCli,
				Signer:        signer,
				CreditNoteSvc: app.CreditNoteService,
			})
		}
	}

	// 2. Wire M1.1 Periods: GL period lock checker, audit recorder, GCS snapshot sink.
	if app.Periods != nil {
		gl.SetPeriodLockChecker(periods.CheckPeriodInTxn)
		if app.Audit != nil {
			app.Periods.SetAuditRecorder(NewPeriodsAuditRecorder(app.Audit))
		}
		if bucket := strings.TrimSpace(os.Getenv("PERIOD_SNAPSHOT_GCS_BUCKET")); bucket != "" {
			gcsClient, err := storage.NewClient(ctx)
			if err != nil {
				if auth.IsProduction() {
					return fmt.Errorf("period snapshot GCS client init failed in production: %w", err)
				}
				if log != nil {
					log.Warn("period snapshot GCS client unavailable; falling back in non-prod", "err", err)
				}
			} else {
				_ = app.Periods.ConfigureGCSSnapshotSink(gcsClient, bucket, os.Getenv("PERIOD_SNAPSHOT_GCS_PREFIX"))
			}
		} else if auth.IsProduction() {
			app.Periods.SetSnapshotSink(periods.FailClosedSnapshotSink{})
		}
	}

	// 3. Wire M1.2 Approvals: audit, SoD, MFA step-up, GL/Periods/FeatureFlags/Orders bridges, OnApprovedHooks.
	if app.Approvals != nil {
		if app.Audit != nil {
			app.Approvals.SetAuditRecorder(NewApprovalsAuditRecorder(app.Audit))
		}
		if app.IAM != nil {
			app.Approvals.SetSoDChecker(NewApprovalsSoDChecker(app.IAM))
		}
		if app.MFAService != nil {
			app.Approvals.SetMFAVerifier(app.MFAService)
		}
		if app.GL != nil {
			app.GL.SetApprovalBridge(NewGLApprovalBridge(app.Approvals))
			if app.Audit != nil {
				app.GL.SetMutationContextHook(func(mCtx context.Context, tenantID, action, objectType, objectID, approvalID string) {
					audit.SetMutationContext(mCtx, audit.MutationContext{
						TenantID:   tenantID,
						Action:     action,
						ObjectType: objectType,
						ObjectID:   objectID,
						ApprovalID: approvalID,
					})
				})
			}
		}
		if app.Periods != nil {
			app.Periods.SetApprovalBridge(NewPeriodApprovalBridge(app.Approvals))
			app.Approvals.RegisterOnApprovedHook("Period", "reopen", func(hCtx context.Context, req approvals.ApprovalRequest, decisions []approvals.ApprovalDecision) error {
				if len(decisions) < 2 {
					return nil
				}
				var p map[string]string
				_ = json.Unmarshal([]byte(req.PayloadJSON), &p)
				reason := strings.TrimSpace(p["reason"])
				if reason == "" {
					reason = "approved via dual-control workflow"
				}
				periodID := strings.TrimSpace(p["period_id"])
				if periodID == "" {
					periodID = req.ObjectID
				}
				actorID := req.RequestedBy
				if len(decisions) > 0 && strings.TrimSpace(decisions[len(decisions)-1].ApproverID) != "" {
					actorID = decisions[len(decisions)-1].ApproverID
				}
				_, err := app.Periods.CompleteApprovedReopen(hCtx, req.TenantID, periodID, reason, actorID, req.RequestID)
				if errors.Is(err, periods.ErrInvalidStatusTransition) {
					return nil
				}
				return err
			})
		}
		if app.FeatureFlagHandlers != nil {
			app.FeatureFlagHandlers.ApprovalBridge = NewFeatureFlagApprovalBridge(app.Approvals)
			if app.FeatureFlagService != nil {
				app.Approvals.RegisterOnApprovedHook("FeatureFlag", "approve", func(hCtx context.Context, req approvals.ApprovalRequest, decisions []approvals.ApprovalDecision) error {
					var p struct {
						FlagKey    string `json:"flag_key"`
						TenantType string `json:"tenant_type"`
						TenantID   string `json:"tenant_id"`
					}
					if err := json.Unmarshal([]byte(req.PayloadJSON), &p); err != nil || p.FlagKey == "" {
						return nil
					}
					approver := req.RequestedBy
					if len(decisions) > 0 && strings.TrimSpace(decisions[len(decisions)-1].ApproverID) != "" {
						approver = decisions[len(decisions)-1].ApproverID
					}
					if err := app.FeatureFlagService.ApproveOverride(hCtx, p.FlagKey, p.TenantType, p.TenantID, approver); err != nil {
						if err.Error() == "override_not_pending" || err.Error() == "override_not_found" {
							return nil
						}
						return err
					}
					if app.FeatureFlagHandlers.Audit != nil {
						detail, _ := json.Marshal(map[string]any{
							"flag_key":            p.FlagKey,
							"tenant_type":         p.TenantType,
							"tenant_id":           p.TenantID,
							"approver":            approver,
							"approval_request_id": req.RequestID,
						})
						_ = app.FeatureFlagHandlers.Audit.RecordFlagAudit(hCtx, approver, "FLAG_OVERRIDE_APPROVE", p.TenantType, p.TenantID, string(detail))
					}
					return nil
				})
			}
		}
		if app.OrderService != nil {
			app.OrderService.SetDraftApprovalBridge(NewOrderDraftApprovalBridge(app.Approvals))
		}
	}

	// 4. Wire M1.3 Inventory Valuation transactional hooks onto Warehouse & StockLots.
	if app.Valuation != nil {
		if app.WarehouseService != nil {
			valSvc := app.Valuation
			app.WarehouseService.SetValuationReceiptHook(func(txCtx context.Context, txn *spanner.ReadWriteTransaction, items []warehouse.ReceiptValuationArgs) error {
				var batch []valuation.ReceiptInput
				for _, it := range items {
					if it.Quantity <= 0 || it.UnitCostMinor <= 0 {
						continue
					}
					batch = append(batch, valuation.ReceiptInput{
						EntityID:       it.EntityID,
						WarehouseID:    it.WarehouseID,
						ProductID:      it.ProductID,
						LotID:          it.LotID,
						ReceiptRefType: it.ReceiptRefType,
						ReceiptRefID:   it.ReceiptRefID,
						Quantity:       it.Quantity,
						UnitCostMinor:  it.UnitCostMinor,
						VatMinor:       it.VatMinor,
						CreditAccount:  it.CreditAccount,
						VendorID:       it.VendorID,
						Currency:       it.Currency,
						ExpiryDate:     it.ExpiryDate,
						IdempotencyKey: it.IdempotencyKey,
						ActorID:        it.ActorID,
					})
				}
				if len(batch) == 0 {
					return nil
				}
				_, err := valSvc.RecordReceiptsBatchInTxn(txCtx, txn, nil, batch)
				return err
			})
		}
		valSvc := app.Valuation
		stocklots.SetPickValuationHook(func(txCtx context.Context, txn *spanner.ReadWriteTransaction, args stocklots.PickValuationArgs) error {
			_, _, err := valSvc.IssueStockFEFOIfPresentInTxn(txCtx, txn, nil, valuation.IssueInput{
				EntityID:       args.EntityID,
				WarehouseID:    args.WarehouseID,
				ProductID:      args.ProductID,
				LotID:          args.LotID,
				ReferenceType:  "PICK",
				ReferenceID:    args.WaveID + ":" + args.TaskID,
				OrderID:        args.OrderID,
				Quantity:       args.Quantity,
				IdempotencyKey: args.IdempotencyKey,
				ActorID:        args.PickerID,
			})
			return err
		})
		stocklots.SetAdjustmentValuationHook(func(txCtx context.Context, txn *spanner.ReadWriteTransaction, args stocklots.AdjustmentValuationArgs) error {
			_, _, err := valSvc.RecordAdjustmentCostIfPresentInTxn(txCtx, txn, nil, valuation.AdjustmentCostInput{
				EntityID:       args.EntityID,
				WarehouseID:    args.WarehouseID,
				ProductID:      args.ProductID,
				LotID:          args.LotID,
				AdjustmentID:   args.AdjustmentID,
				DeltaQty:       args.DeltaQty,
				ReasonCode:     args.ReasonCode,
				IdempotencyKey: args.IdempotencyKey,
				ActorID:        args.ApprovedBy,
			})
			return err
		})
	}

	// 5. Wire M1.4 ESF buyer acceptance syncer onto BuyerAcceptancePoller.
	if app.BuyerAcceptancePoller != nil && app.ESF != nil {
		app.BuyerAcceptancePoller.SetESFSyncer(app.ESF)
	}

	return nil
}
