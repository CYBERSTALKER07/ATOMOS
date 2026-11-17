package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"

	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

type proposalsAuditRecorder struct {
	auditSvc *audit.Service
}

// NewProposalsAuditRecorder adapts *audit.Service to proposals.AuditRecorder.
func NewProposalsAuditRecorder(auditSvc *audit.Service) proposals.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &proposalsAuditRecorder{auditSvc: auditSvc}
}

func (a *proposalsAuditRecorder) RecordEvent(ctx context.Context, ev proposals.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, audit.EventInput{
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
	})
	return err
}

type roiAuditRecorder struct {
	auditSvc *audit.Service
}

// NewROIAuditRecorder adapts *audit.Service to roi.AuditRecorder.
func NewROIAuditRecorder(auditSvc *audit.Service) roi.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &roiAuditRecorder{auditSvc: auditSvc}
}

func (a *roiAuditRecorder) RecordEvent(ctx context.Context, ev roi.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, audit.EventInput{
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
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	})
	return err
}

type creditLimitApprovalBridge struct {
	appSvc *approvals.Service
}

// NewCreditLimitApprovalBridge connects governed credit limit changes to M1.2 CEL Approvals.
func NewCreditLimitApprovalBridge(appSvc *approvals.Service) credit.LimitApprovalBridge {
	if appSvc == nil || appSvc.Repo() == nil {
		return nil
	}
	return &creditLimitApprovalBridge{appSvc: appSvc}
}

func (b *creditLimitApprovalBridge) EvaluateOrVerifyLimitChange(ctx context.Context, in credit.LimitChangeApprovalInput) (credit.LimitChangeApprovalResult, error) {
	if b == nil || b.appSvc == nil {
		return credit.LimitChangeApprovalResult{Required: false}, nil
	}
	tenantID := strings.TrimSpace(in.TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(in.SupplierID)
	}
	approvalID := strings.TrimSpace(in.ApprovalID)

	if approvalID == "" {
		absDelta := in.NewLimitMinor - in.OldLimitMinor
		if absDelta < 0 {
			absDelta = -absDelta
		}
		idemKey := strings.TrimSpace(in.IdempotencyKey)
		if idemKey != "" {
			idemKey = "credit_limit:" + strings.TrimSpace(in.RetailerID) + ":" + idemKey
		}
		res, err := b.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:       tenantID,
			ObjectType:     "CreditLimit",
			ObjectID:       strings.TrimSpace(in.RetailerID),
			Action:         "change",
			AmountMinor:    absDelta,
			RequestedBy:    in.ActorID,
			ActorType:      in.ActorType,
			ApproverRole:   in.ActorRole,
			Payload:        in.Payload,
			Context:        in.Context,
			IdempotencyKey: idemKey,
		})
		if err != nil {
			if errors.Is(err, approvals.ErrAgentApprovalForbidden) {
				return credit.LimitChangeApprovalResult{}, fmt.Errorf("%w: %v", credit.ErrAgentApprovalForbidden, err)
			}
			return credit.LimitChangeApprovalResult{}, err
		}
		if !res.Required || res.Request == nil {
			return credit.LimitChangeApprovalResult{Required: false}, nil
		}
		return credit.LimitChangeApprovalResult{
			Required:          true,
			Approved:          res.Request.Status == approvals.StatusApproved,
			ApprovalRequestID: res.Request.RequestID,
			PolicyID:          res.Request.PolicyID,
			PolicyVersion:     res.Request.PolicyVersion,
			PayloadHash:       res.Request.PayloadHash,
			Status:            res.Request.Status,
		}, nil
	}

	_, hash, err := approvals.CanonicalizePayload(in.Payload)
	if err != nil {
		return credit.LimitChangeApprovalResult{}, fmt.Errorf("%w: %v", credit.ErrInvalidInput, err)
	}
	req, _, err := b.appSvc.VerifyApprovedRequest(ctx, tenantID, approvalID, "CreditLimit", "change", hash, 1)
	if err != nil {
		switch {
		case errors.Is(err, approvals.ErrPayloadHashMismatch):
			return credit.LimitChangeApprovalResult{}, fmt.Errorf("%w: %v", credit.ErrApprovalPayloadMismatch, err)
		case errors.Is(err, approvals.ErrAgentApprovalForbidden):
			return credit.LimitChangeApprovalResult{}, fmt.Errorf("%w: %v", credit.ErrAgentApprovalForbidden, err)
		case errors.Is(err, approvals.ErrRequestNotFound),
			errors.Is(err, approvals.ErrRequestAlreadyResolved),
			errors.Is(err, approvals.ErrRequestExpired):
			return credit.LimitChangeApprovalResult{}, fmt.Errorf("%w: %v", credit.ErrApprovalRequired, err)
		default:
			return credit.LimitChangeApprovalResult{}, err
		}
	}
	return credit.LimitChangeApprovalResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
		Status:            req.Status,
	}, nil
}

// SetupWave3B initializes and wires M1.5 Forecast v2, M1.6 Credit v2 & CEL approval bridge,
// M1.7 Proposals Inbox & L1 Agents, and M1.8 ROI v1 services.
func SetupWave3B(
	_ context.Context,
	cfg *Config,
	spannerClient *spanner.Client,
	app *App,
	log *slog.Logger,
) error {
	if app == nil {
		return nil
	}

	// 1. M1.5 Forecast v2 & Replenishment 1-tap suggestions API.
	if app.ForecastV2 == nil {
		if spannerClient != nil {
			app.ForecastV2 = forecastv2.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.ForecastV2 = forecastv2.NewServiceWithRepository(forecastv2.NewMemoryRepository())
		}
	}
	if app.ForecastV2 != nil {
		app.ForecastV2.SetGeminiConfig(forecastv2.GeminiConfigFromEnv())
	}
	if app.ReplenishmentSuggestions == nil && (spannerClient != nil || app.OrderService != nil || (cfg != nil && cfg.AllowMemoryFallback)) {
		supplierIDFn := func(r *http.Request) string {
			if app.SupplierService != nil {
				if id := strings.TrimSpace(app.SupplierService.ScopedSupplierID(r)); id != "" {
					return id
				}
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				if id := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID)); id != "" {
					return id
				}
				return strings.TrimSpace(claims.SupplierID)
			}
			return ""
		}
		app.ReplenishmentSuggestions = replenishment.NewSuggestionsAPI(spannerClient, app.OrderService, supplierIDFn)
	}

	// 2. M1.6 Credit v2 score history, policy link, ML scorer, and CEL LimitApprovalBridge.
	if app.CreditService != nil {
		if spannerClient != nil {
			app.CreditService.SetScoreHistoryRepo(credit.NewSpannerScoreHistoryRepo(spannerClient))
		}
		if app.CreditPolicyService != nil {
			app.CreditService.SetPolicyService(app.CreditPolicyService)
		}
		app.CreditService.SetMLScorer(credit.NewHTTPMLScorerClientFromEnv())
		if app.Approvals != nil {
			if bridge := NewCreditLimitApprovalBridge(app.Approvals); bridge != nil {
				app.CreditService.SetLimitApprovalBridge(bridge)
			}
		}
		if app.Audit != nil {
			app.CreditService.SetMutationContextHook(func(mCtx context.Context, tenantID, action, objectType, objectID, reasonCode, approvalID string) {
				audit.SetMutationContext(mCtx, audit.MutationContext{
					TenantID:   tenantID,
					Action:     action,
					ObjectType: objectType,
					ObjectID:   objectID,
					ReasonCode: reasonCode,
					ApprovalID: approvalID,
				})
			})
		}
	}

	// 3. M1.8 ROI Attribution v1.
	if app.ROI == nil {
		if spannerClient != nil {
			app.ROI = roi.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.ROI = roi.NewServiceWithRepository(roi.NewMemoryRepository())
		}
	}
	if app.ROI != nil && app.Audit != nil {
		app.ROI.SetAuditRecorder(NewROIAuditRecorder(app.Audit))
	}

	// 4. M1.7 Proposals Inbox & 3 Governed L1 Agents.
	if app.Proposals == nil {
		if spannerClient != nil {
			app.Proposals = proposals.NewService(spannerClient)
		} else if cfg != nil && cfg.AllowMemoryFallback {
			app.Proposals = proposals.NewServiceWithRepository(proposals.NewMemoryRepository())
		}
	}
	if app.Proposals != nil {
		if app.Approvals != nil {
			app.Proposals.SetApprovals(app.Approvals)
		}
		if app.ROI != nil {
			app.Proposals.SetROI(app.ROI)
		}
		if spannerClient != nil {
			app.Proposals.SetPlanningExecutor(planning.NewExecutor(spannerClient))
			app.Proposals.SetDomainExecutor(&proposals.SpannerDomainExecutor{
				Client:        spannerClient,
				Notifications: app.NotificationService,
			})
		}
		if app.ControlTowerService != nil {
			if exec := app.ControlTowerService.ActionExecutor(); exec != nil {
				app.Proposals.SetControlTowerExecutor(exec)
			}
		}
		if app.Audit != nil {
			app.Proposals.SetAuditRecorder(NewProposalsAuditRecorder(app.Audit))
		}
	}

	if log != nil {
		log.Info("wave3b subsystems wired",
			"forecast_v2", app.ForecastV2 != nil,
			"replenishment_suggestions", app.ReplenishmentSuggestions != nil,
			"credit_v2", app.CreditService != nil,
			"roi", app.ROI != nil,
			"proposals", app.Proposals != nil,
		)
	}
	return nil
}
