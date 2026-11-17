package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
)

// NewStocklotsAuditRecorder bridges stocklots HACCP compliance evaluations and exports into hash-chained AuditEvents.
func NewStocklotsAuditRecorder(auditSvc *audit.Service) func(context.Context, stocklots.HACCPAuditInput) error {
	if auditSvc == nil {
		return nil
	}
	return func(ctx context.Context, in stocklots.HACCPAuditInput) error {
		_, err := auditSvc.Record(ctx, audit.EventInput{
			TenantID:   in.TenantID,
			ActorID:    in.ActorID,
			ActorType:  audit.NormalizeActorType(in.ActorType),
			Role:       in.Role,
			Action:     in.Action,
			ObjectType: in.ObjectType,
			ObjectID:   in.ObjectID,
			AfterJSON:  in.AfterJSON,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		})
		return err
	}
}

type anomaliesAuditRecorder struct {
	audit *audit.Service
}

// NewAnomaliesAuditRecorder bridges anomalies operations into immutable hash-chained AuditEvents.
func NewAnomaliesAuditRecorder(auditSvc *audit.Service) anomalies.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &anomaliesAuditRecorder{audit: auditSvc}
}

func (a *anomaliesAuditRecorder) RecordEvent(ctx context.Context, ev anomalies.AuditRecordInput) error {
	if a == nil || a.audit == nil {
		return nil
	}
	_, err := a.audit.Record(ctx, toAnomaliesAuditEventInput(ev))
	return err
}

func (a *anomaliesAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev anomalies.AuditRecordInput) error {
	if a == nil || a.audit == nil || txn == nil {
		return nil
	}
	_, err := a.audit.RecordInTxn(ctx, txn, toAnomaliesAuditEventInput(ev))
	return err
}

func toAnomaliesAuditEventInput(ev anomalies.AuditRecordInput) audit.EventInput {
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

type whatifAuditRecorder struct {
	audit *audit.Service
}

// NewWhatIfAuditRecorder bridges What-If simulations and proposal commitments into immutable AuditEvents.
func NewWhatIfAuditRecorder(auditSvc *audit.Service) whatif.AuditRecorder {
	if auditSvc == nil {
		return nil
	}
	return &whatifAuditRecorder{audit: auditSvc}
}

func (w *whatifAuditRecorder) RecordEvent(ctx context.Context, ev whatif.AuditRecordInput) error {
	if w == nil || w.audit == nil {
		return nil
	}
	_, err := w.audit.Record(ctx, toWhatIfAuditEventInput(ev))
	return err
}

func (w *whatifAuditRecorder) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev whatif.AuditRecordInput) error {
	if w == nil || w.audit == nil || txn == nil {
		return nil
	}
	_, err := w.audit.RecordInTxn(ctx, txn, toWhatIfAuditEventInput(ev))
	return err
}

func toWhatIfAuditEventInput(ev whatif.AuditRecordInput) audit.EventInput {
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

type anomaliesProposalCreator struct {
	proposals *proposals.Service
}

// NewAnomaliesProposalCreator adapts proposals.Service as an anomalies.ProposalCreator.
func NewAnomaliesProposalCreator(propSvc *proposals.Service) anomalies.ProposalCreator {
	if propSvc == nil {
		return nil
	}
	return &anomaliesProposalCreator{proposals: propSvc}
}

func (c *anomaliesProposalCreator) CreateProposal(ctx context.Context, in anomalies.ProposalCreateInput) (anomalies.CreatedProposal, error) {
	if c == nil || c.proposals == nil {
		return anomalies.CreatedProposal{}, fmt.Errorf("proposals service unavailable")
	}
	refs := make([]proposals.EvidenceRef, 0, len(in.EvidenceRefs))
	for _, r := range in.EvidenceRefs {
		refs = append(refs, proposals.EvidenceRef{
			Type:  r.Type,
			ID:    r.ID,
			Label: r.Label,
		})
	}
	p, err := c.proposals.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             in.TenantID,
		AgentID:              in.AgentID,
		ProposalType:         in.ProposalType,
		TargetObjectType:     in.TargetObjectType,
		TargetObjectID:       in.TargetObjectID,
		Title:                in.Title,
		Summary:              in.Summary,
		Rationale:            in.Rationale,
		Payload:              in.Payload,
		EvidenceRefs:         refs,
		ConfidenceBps:        in.ConfidenceBps,
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		Currency:             in.Currency,
		RoiCategory:          in.RoiCategory,
		CreatedBy:            in.CreatedBy,
		ActorType:            in.ActorType,
		IdempotencyKey:       in.IdempotencyKey,
		ExpiresInHours:       in.ExpiresInHours,
	})
	if err != nil {
		return anomalies.CreatedProposal{}, err
	}
	return anomalies.CreatedProposal{
		TenantID:             p.TenantID,
		ProposalID:           p.ProposalID,
		AgentID:              p.AgentID,
		ProposalType:         p.ProposalType,
		TargetObjectType:     p.TargetObjectType,
		TargetObjectID:       p.TargetObjectID,
		Title:                p.Title,
		Summary:              p.Summary,
		Status:               p.Status,
		ApprovalRequestID:    p.ApprovalRequestID,
		PayloadHash:          p.PayloadHash,
		EstimatedImpactMinor: p.EstimatedImpactMinor,
		ConfidenceBps:        p.ConfidenceBps,
		Currency:             p.Currency,
		CreatedBy:            p.CreatedBy,
	}, nil
}

type whatifProposalCreator struct {
	proposals *proposals.Service
}

// NewWhatIfProposalCreator adapts proposals.Service as a whatif.ProposalCreator.
func NewWhatIfProposalCreator(propSvc *proposals.Service) whatif.ProposalCreator {
	if propSvc == nil {
		return nil
	}
	return &whatifProposalCreator{proposals: propSvc}
}

func (c *whatifProposalCreator) CreateProposal(ctx context.Context, in whatif.ProposalCreateInput) (whatif.CreatedProposal, error) {
	if c == nil || c.proposals == nil {
		return whatif.CreatedProposal{}, fmt.Errorf("proposals service unavailable")
	}
	refs := make([]proposals.EvidenceRef, 0, len(in.EvidenceRefs))
	for _, r := range in.EvidenceRefs {
		refs = append(refs, proposals.EvidenceRef{
			Type:  r.Type,
			ID:    r.ID,
			Label: r.Label,
		})
	}
	p, err := c.proposals.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             in.TenantID,
		AgentID:              in.AgentID,
		ProposalType:         in.ProposalType,
		TargetObjectType:     in.TargetObjectType,
		TargetObjectID:       in.TargetObjectID,
		Title:                in.Title,
		Summary:              in.Summary,
		Rationale:            in.Rationale,
		Payload:              in.Payload,
		EvidenceRefs:         refs,
		ConfidenceBps:        in.ConfidenceBps,
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		Currency:             in.Currency,
		RoiCategory:          in.RoiCategory,
		CreatedBy:            in.CreatedBy,
		ActorType:            in.ActorType,
		IdempotencyKey:       in.IdempotencyKey,
		ExpiresInHours:       in.ExpiresInHours,
	})
	if err != nil {
		return whatif.CreatedProposal{}, err
	}
	return whatif.CreatedProposal{
		TenantID:             p.TenantID,
		ProposalID:           p.ProposalID,
		AgentID:              p.AgentID,
		ProposalType:         p.ProposalType,
		TargetObjectType:     p.TargetObjectType,
		TargetObjectID:       p.TargetObjectID,
		Title:                p.Title,
		Summary:              p.Summary,
		Status:               p.Status,
		ApprovalRequestID:    p.ApprovalRequestID,
		PayloadHash:          p.PayloadHash,
		EstimatedImpactMinor: p.EstimatedImpactMinor,
		ConfidenceBps:        p.ConfidenceBps,
		Currency:             p.Currency,
		CreatedBy:            p.CreatedBy,
	}, nil
}

// SetupWave4B wires the Wave 4B Anomaly & Fraud Detector Catalogue and Full What-If
// Simulator services onto app with immutable AuditEvents and governed L1 Proposals bridges.
func SetupWave4B(_ context.Context, cfg *Config, spannerClient *spanner.Client, app *App, log *slog.Logger) error {
	if app == nil {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}

	anomAudit := NewAnomaliesAuditRecorder(app.Audit)
	anomProps := NewAnomaliesProposalCreator(app.Proposals)
	whatifAudit := NewWhatIfAuditRecorder(app.Audit)
	whatifProps := NewWhatIfProposalCreator(app.Proposals)

	if spannerClient != nil {
		if app.Anomalies == nil {
			app.Anomalies = anomalies.NewSpannerService(spannerClient, anomalies.ServiceConfig{
				Audit:     anomAudit,
				Proposals: anomProps,
			})
		}
		if app.WhatIf == nil {
			app.WhatIf = whatif.NewSpannerService(spannerClient, whatif.ServiceConfig{
				Audit:     whatifAudit,
				Proposals: whatifProps,
			})
		}
	} else if cfg != nil && cfg.AllowMemoryFallback {
		if app.Anomalies == nil {
			app.Anomalies = anomalies.NewService(anomalies.NewMemoryRepository(), anomalies.ServiceConfig{
				Audit:     anomAudit,
				Proposals: anomProps,
			})
		}
		if app.WhatIf == nil {
			app.WhatIf = whatif.NewService(whatif.NewMemoryRepository(), whatif.ServiceConfig{
				Audit:     whatifAudit,
				Proposals: whatifProps,
			})
		}
	}

	if app.Anomalies != nil {
		if anomAudit != nil {
			app.Anomalies.SetAuditRecorder(anomAudit)
		}
		if anomProps != nil {
			app.Anomalies.SetProposalCreator(anomProps)
		}
	}
	if app.WhatIf != nil {
		if whatifAudit != nil {
			app.WhatIf.SetAuditRecorder(whatifAudit)
		}
		if whatifProps != nil {
			app.WhatIf.SetProposalCreator(whatifProps)
		}
	}

	log.Info("wave4b initialized",
		"anomalies_enabled", app.Anomalies != nil,
		"whatif_enabled", app.WhatIf != nil,
		"mode", map[bool]string{true: "spanner", false: "memory_or_disabled"}[spannerClient != nil],
	)
	return nil
}
