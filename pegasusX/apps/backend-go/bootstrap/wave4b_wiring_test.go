package bootstrap

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
)

func TestSetupWave4B_MemoryFallbackAndFailClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	appFail := &App{}
	if err := SetupWave4B(ctx, &Config{AllowMemoryFallback: false}, nil, appFail, nil); err != nil {
		t.Fatalf("SetupWave4B without spanner returned unexpected err: %v", err)
	}
	if appFail.Anomalies != nil || appFail.WhatIf != nil {
		t.Fatal("expected SetupWave4B to leave Anomalies and WhatIf nil when spannerClient=nil and AllowMemoryFallback=false")
	}

	auditSvc := audit.NewServiceWithRepository(audit.NewMemoryRepository())
	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	propSvc := proposals.NewServiceWithRepository(proposals.NewMemoryRepository())
	propSvc.SetApprovals(appSvc)
	propSvc.SetAuditRecorder(NewProposalsAuditRecorder(auditSvc))

	app := &App{
		Audit:     auditSvc,
		Proposals: propSvc,
	}
	if err := SetupWave4B(ctx, &Config{AllowMemoryFallback: true}, nil, app, nil); err != nil {
		t.Fatalf("SetupWave4B memory fallback: %v", err)
	}
	if app.Anomalies == nil || app.WhatIf == nil {
		t.Fatalf("expected Anomalies and WhatIf wired, got anom=%v whatif=%v", app.Anomalies, app.WhatIf)
	}

	// Verify Anomalies -> ProposalCreator bridge creates real L1 PENDING proposal in proposals.Service + AuditEvents.
	created, err := NewAnomaliesProposalCreator(propSvc).CreateProposal(ctx, anomalies.ProposalCreateInput{
		TenantID:             "sup-w4b-wire",
		ProposalType:         proposals.TypeReorder,
		TargetObjectType:     "StockCount",
		TargetObjectID:       "cnt-1",
		Title:                "Variance review",
		EstimatedImpactMinor: 12000,
		ConfidenceBps:        8200,
		Currency:             "UZS",
		CreatedBy:            "owner-1",
		ActorType:            auth.ActorTypeHuman,
	})
	if err != nil {
		t.Fatalf("AnomaliesProposalCreator.CreateProposal: %v", err)
	}
	if created.ProposalID == "" || created.Status != proposals.StatusPending {
		t.Fatalf("unexpected anomaly proposal: %+v", created)
	}

	// Verify WhatIf -> ProposalCreator bridge & WhatIf Simulate -> CommitProposals end-to-end.
	sim, _, err := app.WhatIf.Simulate(ctx, whatif.SimulateInput{
		TenantID:       "sup-w4b-wire",
		Name:           "Wire Sim",
		DemandDeltaBps: 1500,
		HorizonDays:    14,
		ActorID:        "owner-1",
		ActorType:      auth.ActorTypeHuman,
	})
	if err != nil {
		t.Fatalf("WhatIf.Simulate: %v", err)
	}
	committed, err := app.WhatIf.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:     "sup-w4b-wire",
		SimulationID: sim.SimulationID,
		ActorID:      "owner-1",
		ActorType:    auth.ActorTypeHuman,
	})
	if err != nil || len(committed.CommittedProposalIDs) == 0 {
		t.Fatalf("WhatIf.CommitProposals = (%+v, %v)", committed, err)
	}

	// Verify Stocklots HACCP AuditRecorder bridge writes into AuditEvents.
	recFn := NewStocklotsAuditRecorder(auditSvc)
	if err := recFn(ctx, stocklots.HACCPAuditInput{
		TenantID:   "sup-w4b-wire",
		ActorID:    "aud-1",
		ActorType:  auth.ActorTypeHuman,
		Role:       auth.SupplierRoleAuditor,
		Action:     "compliance.haccp.evaluated",
		ObjectType: "ComplianceHACCPReport",
		ObjectID:   "sup-w4b-wire",
		AfterJSON:  `{"ok":true}`,
	}); err != nil {
		t.Fatalf("NewStocklotsAuditRecorder: %v", err)
	}
	eventsList, err := auditSvc.ListEvents(ctx, audit.Filter{TenantID: "sup-w4b-wire", Limit: 20})
	if err != nil || len(eventsList) < 3 {
		t.Fatalf("expected >= 3 audit events recorded by Wave 4B bridges, got (%d, %v)", len(eventsList), err)
	}
}
