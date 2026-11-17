package bootstrap

import (
	"context"
	"log/slog"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

// auditIAMRecorder adapts *audit.Service to iam.AuditRecorder without introducing package cycles.
type auditIAMRecorder struct {
	auditSvc *audit.Service
}

// NewIAMAuditRecorder returns an iam.AuditRecorder backed by auditSvc.
func NewIAMAuditRecorder(auditSvc *audit.Service) iam.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &auditIAMRecorder{auditSvc: auditSvc}
}

func (a *auditIAMRecorder) RecordEvent(ctx context.Context, ev iam.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, toAuditEventInput(ev))
	return err
}

func (a *auditIAMRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev iam.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, toAuditEventInput(ev))
	return err
}

func toAuditEventInput(ev iam.AuditRecordInput) audit.EventInput {
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

func setupAuditAndIAM(
	ctx context.Context,
	spannerClient *spanner.Client,
	cfg *Config,
	log *slog.Logger,
) (*audit.Service, *iam.Service) {
	var auditSvc *audit.Service
	var iamSvc *iam.Service

	if spannerClient != nil {
		auditSvc = audit.NewService(spannerClient)
		if sink, err := audit.NewDefaultWormSinkFromEnv(ctx); err != nil {
			if log != nil {
				log.Warn("audit worm anchor sink not configured at startup", "err", err)
			}
		} else if sink != nil {
			auditSvc.WithAnchorSink(sink)
		}
		iamSvc = iam.NewService(spannerClient)
	} else if cfg != nil && cfg.AllowMemoryFallback {
		auditSvc = audit.NewServiceWithRepository(audit.NewMemoryRepository())
		iamSvc = iam.NewServiceWithRepository(iam.NewMemoryRepository())
	}

	if iamSvc != nil && auditSvc != nil {
		iamSvc.SetAuditRecorder(NewIAMAuditRecorder(auditSvc))
	}
	return auditSvc, iamSvc
}

// CheckOrderDraftSoD adapts *iam.Service to order.Service.SetDraftSoDCheck.
func CheckOrderDraftSoD(iamSvc *iam.Service) func(ctx context.Context, in order.DraftSoDCheckInput) error {
	if iamSvc == nil || iamSvc.Repo() == nil {
		return nil
	}
	return func(ctx context.Context, in order.DraftSoDCheckInput) error {
		return iamSvc.CheckFourEyes(ctx, iam.CheckFourEyesInput{
			TenantID:    in.TenantID,
			ActorID:     in.ActorID,
			ActorType:   in.ActorType,
			Role:        in.Role,
			Claims:      in.Claims,
			Action:      in.Action,
			MakerAction: in.MakerAction,
			ObjectType:  in.ObjectType,
			ObjectID:    in.ObjectID,
			MakerID:     in.MakerID,
			RoutePath:   in.RoutePath,
			IsApproval:  in.IsApproval,
		})
	}
}

// CheckStockAdjustmentSoD adapts *iam.Service to stocklots.Handler.SoDCheck.
func CheckStockAdjustmentSoD(iamSvc *iam.Service) func(ctx context.Context, in stocklots.SoDCheckInput) error {
	if iamSvc == nil || iamSvc.Repo() == nil {
		return nil
	}
	return func(ctx context.Context, in stocklots.SoDCheckInput) error {
		return iamSvc.CheckFourEyes(ctx, iam.CheckFourEyesInput{
			TenantID:    in.TenantID,
			ActorID:     in.ActorID,
			ActorType:   in.ActorType,
			Role:        in.Role,
			Claims:      in.Claims,
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
}

// BuildFoundryAuditHook forwards Kinetic Action executions into immutable AuditEvents.
func BuildFoundryAuditHook(auditSvc *audit.Service, log *slog.Logger) func(ctx context.Context, rec *foundry.ActionExecutionRecord) error {
	if auditSvc == nil || auditSvc.Repo() == nil {
		return nil
	}
	return func(ctx context.Context, rec *foundry.ActionExecutionRecord) error {
		if rec == nil {
			return nil
		}
		tenantID := strings.TrimSpace(rec.SupplierID)
		if tenantID == "" {
			tenantID = strings.TrimSpace(rec.CellID)
		}
		if tenantID == "" {
			tenantID = "platform"
		}
		actorType := audit.ActorTypeHuman
		if strings.EqualFold(strings.TrimSpace(rec.ActorRole), "AGENT") {
			actorType = audit.ActorTypeAgent
		} else if strings.EqualFold(strings.TrimSpace(rec.ActorRole), "SYSTEM") {
			actorType = audit.ActorTypeSystem
		}
		_, err := auditSvc.Record(ctx, audit.EventInput{
			TenantID:   tenantID,
			ActorID:    strings.TrimSpace(rec.ActorID),
			ActorType:  actorType,
			Role:       strings.TrimSpace(rec.ActorRole),
			Action:     "kinetic:" + strings.TrimSpace(rec.ActionType),
			ObjectType: "KineticActionExecution",
			ObjectID:   strings.TrimSpace(rec.ActionExecutionID),
			AfterJSON:  strings.TrimSpace(rec.DiffPayloadJson),
			ReasonCode: strings.TrimSpace(rec.Status),
		})
		if err != nil && log != nil {
			log.WarnContext(ctx, "foundry audit hook failed", "err", err, "action_id", rec.ActionExecutionID)
		}
		return nil
	}
}
