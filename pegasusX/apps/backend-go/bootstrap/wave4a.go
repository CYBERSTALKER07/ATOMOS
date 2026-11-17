// Package bootstrap wires ATOMOS Wave 4A services (§6.4 Multi-Entity/FX Reval/Consolidation,
// §6.6 P2P & Process Mining, GL/Approval/Audit bridges, and §7.10 2-way connector push approval gating).
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/p2p"
)

type connectorsPushApprovalChecker struct {
	appSvc *approvals.Service
}

// NewConnectorsPushApprovalChecker adapts *approvals.Service to connectors.PushApprovalChecker.
func NewConnectorsPushApprovalChecker(appSvc *approvals.Service) connectors.PushApprovalChecker {
	if appSvc == nil {
		return nil
	}
	return &connectorsPushApprovalChecker{appSvc: appSvc}
}

func (c *connectorsPushApprovalChecker) VerifyPushApproval(ctx context.Context, in connectors.PushApprovalInput) error {
	if c == nil || c.appSvc == nil {
		return fmt.Errorf("%w: approval service unavailable", connectors.ErrApprovalRequired)
	}
	if strings.EqualFold(strings.TrimSpace(in.ActorRole), auth.SupplierRoleAuditor) {
		return connectors.ErrAuditorReadOnly
	}
	if strings.EqualFold(strings.TrimSpace(in.ActorRole), auth.SupplierRoleAgent) ||
		strings.EqualFold(strings.TrimSpace(in.ActorType), auth.ActorTypeAgent) {
		return connectors.ErrAgentForbidden
	}
	if strings.TrimSpace(in.ApprovalID) == "" {
		return fmt.Errorf("%w: approval_id is required", connectors.ErrApprovalRequired)
	}
	_, payloadHash, err := approvals.CanonicalizePayload(in.Payload)
	if err != nil {
		if errors.Is(err, approvals.ErrFloatForbidden) {
			return fmt.Errorf("%w: %v", connectors.ErrFloatForbidden, err)
		}
		return fmt.Errorf("%w: %v", connectors.ErrInvalidInput, err)
	}
	if _, _, err := c.appSvc.VerifyApprovedRequest(
		ctx,
		in.TenantID,
		in.ApprovalID,
		"ConnectorPush",
		"push",
		payloadHash,
		1,
	); err != nil {
		switch {
		case errors.Is(err, approvals.ErrPayloadHashMismatch):
			return fmt.Errorf("%w: %v", connectors.ErrApprovalPayloadMismatch, err)
		case errors.Is(err, approvals.ErrAgentApprovalForbidden):
			return fmt.Errorf("%w: %v", connectors.ErrAgentForbidden, err)
		case errors.Is(err, approvals.ErrAuditorReadOnly):
			return fmt.Errorf("%w: %v", connectors.ErrAuditorReadOnly, err)
		default:
			return fmt.Errorf("%w: %v", connectors.ErrApprovalRequired, err)
		}
	}
	return nil
}

type multiEntityAuditRecorder struct {
	auditSvc *audit.Service
}

// NewMultiEntityAuditRecorder adapts *audit.Service to multientity.AuditRecorder.
func NewMultiEntityAuditRecorder(auditSvc *audit.Service) multientity.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &multiEntityAuditRecorder{auditSvc: auditSvc}
}

func toMultiEntityAuditEventInput(ev multientity.AuditRecordInput) audit.EventInput {
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

func (a *multiEntityAuditRecorder) RecordEvent(ctx context.Context, ev multientity.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, toMultiEntityAuditEventInput(ev))
	return err
}

func (a *multiEntityAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev multientity.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, toMultiEntityAuditEventInput(ev))
	return err
}

type p2pGLPoster struct {
	glSvc *gl.Service
}

// NewP2PGLPoster adapts *gl.Service to p2p.TransactionalGLPoster (and p2p.GLPoster).
func NewP2PGLPoster(glSvc *gl.Service) p2p.GLPoster {
	if glSvc == nil {
		return nil
	}
	return &p2pGLPoster{glSvc: glSvc}
}

func (g *p2pGLPoster) PostJournal(ctx context.Context, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	if g == nil || g.glSvc == nil {
		return p2p.GLPostResult{}, p2p.ErrGLPosterUnavailable
	}
	res, err := g.glSvc.PostEntry(ctx, p2pToGLInput(in))
	if err != nil {
		return p2p.GLPostResult{}, err
	}
	return p2p.GLPostResult{
		EntryID:       res.EntryID,
		PeriodID:      res.PeriodID,
		AlreadyPosted: res.AlreadyPosted,
	}, nil
}

func (g *p2pGLPoster) PostJournalInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, in p2p.GLPostInput) (p2p.GLPostResult, error) {
	if g == nil || g.glSvc == nil || txn == nil {
		return p2p.GLPostResult{}, p2p.ErrGLPosterUnavailable
	}
	res, err := gl.BufferJournalEntryInTxn(ctx, txn, buf, p2pToGLInput(in))
	if err != nil {
		return p2p.GLPostResult{}, err
	}
	return p2p.GLPostResult{
		EntryID:       res.EntryID,
		PeriodID:      res.PeriodID,
		AlreadyPosted: res.AlreadyPosted,
	}, nil
}

func p2pToGLInput(in p2p.GLPostInput) gl.JournalEntryInput {
	postings := make([]gl.PostingInput, 0, len(in.Lines))
	for _, l := range in.Lines {
		postings = append(postings, gl.PostingInput{
			AccountCode: l.AccountCode,
			Direction:   gl.Direction(l.Direction),
			AmountMinor: l.AmountMinor,
			Currency:    in.Currency,
			PartyType:   l.PartyType,
			PartyID:     l.PartyID,
			WarehouseID: l.WarehouseID,
			ProductID:   l.ProductID,
			Description: l.Description,
		})
	}
	return gl.JournalEntryInput{
		EntityID:       in.TenantID,
		LegalEntityID:  in.EntityID,
		PeriodID:       in.PeriodID,
		SourceType:     in.SourceType,
		SourceID:       in.SourceID,
		Memo:           in.Memo,
		Currency:       in.Currency,
		IdempotencyKey: in.IdempotencyKey,
		PostedBy:       in.PostedBy,
		OccurredAt:     in.OccurredAt,
		Postings:       postings,
	}
}

type p2pApprovalChecker struct {
	appSvc *approvals.Service
}

// NewP2PApprovalChecker adapts *approvals.Service to p2p.ApprovalChecker.
func NewP2PApprovalChecker(appSvc *approvals.Service) p2p.ApprovalChecker {
	if appSvc == nil {
		return nil
	}
	return &p2pApprovalChecker{appSvc: appSvc}
}

func (a *p2pApprovalChecker) CheckOrSubmitApproval(ctx context.Context, in p2p.ApprovalCheckInput) (p2p.ApprovalCheckResult, error) {
	if a == nil || a.appSvc == nil {
		return p2p.ApprovalCheckResult{}, fmt.Errorf("%w: approval service unavailable", p2p.ErrApprovalRequired)
	}
	if strings.TrimSpace(in.ApprovalID) == "" {
		res, err := a.appSvc.SubmitRequest(ctx, approvals.SubmitRequestInput{
			TenantID:       in.TenantID,
			ObjectType:     in.ObjectType,
			ObjectID:       in.ObjectID,
			Action:         in.Action,
			AmountMinor:    in.AmountMinor,
			RequestedBy:    in.RequestedBy,
			ActorType:      in.ActorType,
			ApproverRole:   in.ApproverRole,
			Payload:        in.Payload,
			Context:        in.Context,
			IdempotencyKey: in.IdempotencyKey,
			ForceCreate:    in.ForceCreate,
		})
		if err != nil {
			return p2p.ApprovalCheckResult{}, mapApprovalsErrorToP2P(err)
		}
		if !res.Required || res.Request == nil {
			return p2p.ApprovalCheckResult{Required: false}, nil
		}
		return p2p.ApprovalCheckResult{
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
		return p2p.ApprovalCheckResult{}, mapApprovalsErrorToP2P(err)
	}
	req, decs, err := a.appSvc.VerifyApprovedRequest(ctx, in.TenantID, in.ApprovalID, in.ObjectType, in.Action, hash, 1)
	if err != nil {
		return p2p.ApprovalCheckResult{}, mapApprovalsErrorToP2P(err)
	}
	approverIDs := make([]string, 0, len(decs))
	for _, d := range decs {
		if d.Decision == approvals.DecisionApprove {
			approverIDs = append(approverIDs, d.ApproverID)
		}
	}
	return p2p.ApprovalCheckResult{
		Required:          true,
		Approved:          true,
		ApprovalRequestID: req.RequestID,
		PolicyID:          req.PolicyID,
		PolicyVersion:     req.PolicyVersion,
		PayloadHash:       req.PayloadHash,
		Status:            req.Status,
		ApproverIDs:       approverIDs,
	}, nil
}

func mapApprovalsErrorToP2P(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, approvals.ErrAgentApprovalForbidden), errors.Is(err, approvals.ErrAuditorReadOnly):
		return fmt.Errorf("%w: %v", p2p.ErrForbidden, err)
	case errors.Is(err, approvals.ErrPayloadHashMismatch):
		return fmt.Errorf("%w: %v", p2p.ErrApprovalFailed, err)
	default:
		return err
	}
}

type p2pAuditRecorder struct {
	auditSvc *audit.Service
}

// NewP2PAuditRecorder adapts *audit.Service to p2p.AuditRecorder.
func NewP2PAuditRecorder(auditSvc *audit.Service) p2p.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &p2pAuditRecorder{auditSvc: auditSvc}
}

func toP2PAuditEventInput(ev p2p.AuditRecordInput) audit.EventInput {
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

func (a *p2pAuditRecorder) RecordEvent(ctx context.Context, ev p2p.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil {
		return nil
	}
	_, err := a.auditSvc.Record(ctx, toP2PAuditEventInput(ev))
	return err
}

func (a *p2pAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev p2p.AuditRecordInput) error {
	if a == nil || a.auditSvc == nil || txn == nil {
		return nil
	}
	_, err := a.auditSvc.RecordInTxn(ctx, txn, toP2PAuditEventInput(ev))
	return err
}

// SetupWave4A initializes Wave 4A Multi-Entity/FX Revaluation/Consolidation, P2P/Process Mining,
// GL/Audit/Approval bridges, and 2-way connector push approval gating on app.
func SetupWave4A(_ context.Context, cfg *Config, spannerClient *spanner.Client, app *App, log *slog.Logger) error {
	if app == nil {
		return nil
	}

	if spannerClient != nil {
		if app.MultiEntity == nil {
			app.MultiEntity = multientity.NewService(spannerClient, app.GL, app.FxRatesService, app.FxRatesRepo)
		}
		if app.P2P == nil {
			app.P2P = p2p.NewSpannerService(spannerClient, p2p.ServiceConfig{
				GLPoster:  NewP2PGLPoster(app.GL),
				Approvals: NewP2PApprovalChecker(app.Approvals),
				Audit:     NewP2PAuditRecorder(app.Audit),
			})
		}
	} else if cfg != nil && cfg.AllowMemoryFallback {
		if app.MultiEntity == nil {
			fxRepo := app.FxRatesRepo
			if fxRepo == nil {
				fxRepo = fxrates.NewMemoryRepository()
				app.FxRatesRepo = fxRepo
			}
			fxSvc := app.FxRatesService
			if fxSvc == nil {
				fxSvc = fxrates.NewService(fxRepo)
				app.FxRatesService = fxSvc
			}
			app.MultiEntity = multientity.NewServiceWithDeps(
				multientity.NewMemoryRepository(app.GL),
				app.GL,
				fxSvc,
				fxRepo,
			)
		}
		if app.P2P == nil {
			app.P2P = p2p.NewService(p2p.NewMemoryRepository(), p2p.ServiceConfig{
				GLPoster:  NewP2PGLPoster(app.GL),
				Approvals: NewP2PApprovalChecker(app.Approvals),
				Audit:     NewP2PAuditRecorder(app.Audit),
			})
		}
	}

	if app.MultiEntity != nil {
		if app.Audit != nil {
			app.MultiEntity.SetAuditRecorder(NewMultiEntityAuditRecorder(app.Audit))
		}
		if app.GL != nil {
			app.GL.SetLegalEntityOwnershipChecker(app.MultiEntity)
		}
	}

	if app.P2P != nil {
		if app.GL != nil {
			app.P2P.SetGLPoster(NewP2PGLPoster(app.GL))
		}
		if app.Approvals != nil {
			app.P2P.SetApprovalChecker(NewP2PApprovalChecker(app.Approvals))
		}
		if app.Audit != nil {
			app.P2P.SetAuditRecorder(NewP2PAuditRecorder(app.Audit))
		}
	}

	if app.Connectors != nil && app.Approvals != nil {
		app.Connectors.SetApprovalChecker(NewConnectorsPushApprovalChecker(app.Approvals))
	}

	if log != nil {
		log.Info("wave4a initialized",
			"multientity_enabled", app.MultiEntity != nil,
			"p2p_enabled", app.P2P != nil,
			"connectors_push_approval_wired", app.Connectors != nil && app.Approvals != nil,
		)
	}
	return nil
}
