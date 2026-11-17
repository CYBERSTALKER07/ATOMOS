package whatif

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service orchestrates §7.9 multi-stage What-If simulation and proposal commitment.
type Service struct {
	repo      Repository
	audit     AuditRecorder
	proposals ProposalCreator
	now       func() time.Time
}

// NewService constructs a What-If Service backed by a custom Repository (e.g. MemoryRepository).
func NewService(repo Repository, cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		repo:      repo,
		audit:     cfg.Audit,
		proposals: cfg.Proposals,
		now:       nowFn,
	}
}

// SetAuditRecorder attaches an immutable AuditRecorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

// SetProposalCreator attaches the governed L1 ProposalCreator hook.
func (s *Service) SetProposalCreator(pc ProposalCreator) {
	if s != nil {
		s.proposals = pc
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// Simulate executes the multi-stage What-If pipeline over live Spanner data in a
// ReadOnlyTransaction, persists WhatIfSimulations, emits whatif.simulation.completed,
// and records AuditEvents without ever mutating domain or ledger tables directly.
func (s *Service) Simulate(ctx context.Context, in SimulateInput) (WhatIfSimulation, bool, error) {
	if s == nil || s.repo == nil {
		return WhatIfSimulation{}, false, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	name := strings.TrimSpace(in.Name)
	if tid == "" || name == "" {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: tenant_id and name are required", ErrInvalidInput)
	}
	if len(name) > 160 {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: name exceeds 160 characters", ErrInvalidInput)
	}
	horizon := in.HorizonDays
	if horizon == 0 {
		horizon = DefaultHorizonDays
	}
	if horizon < MinHorizonDays || horizon > MaxHorizonDays {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: horizon_days must be between %d and %d", ErrInvalidInput, MinHorizonDays, MaxHorizonDays)
	}
	if in.FactoryDowntimeHours < 0 || in.FactoryDowntimeHours > horizon*24 {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: factory_downtime_hours out of bounds", ErrInvalidInput)
	}
	for label, v := range map[string]int64{
		"demand_delta_bps":       in.DemandDeltaBps,
		"fx_rate_shock_bps":      in.FxRateShockBps,
		"credit_limit_delta_bps": in.CreditLimitDeltaBps,
		"unit_price_delta_bps":   in.UnitPriceDeltaBps,
	} {
		if v < MinDeltaBps || v > MaxDeltaBps {
			return WhatIfSimulation{}, false, fmt.Errorf("%w: %s (%d) must be within [%d, %d]", ErrInvalidInput, label, v, MinDeltaBps, MaxDeltaBps)
		}
	}

	simID := "sim_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		sum := sha256.Sum256([]byte(tid + ":" + idem))
		simID = "sim_" + hex.EncodeToString(sum[:12])
		if existing, found, err := s.repo.GetSimulation(ctx, tid, simID); err != nil {
			return WhatIfSimulation{}, false, err
		} else if found {
			return existing, true, nil
		}
	}

	scenarioID := strings.TrimSpace(in.ScenarioID)
	entityID := strings.TrimSpace(in.EntityID)
	ds, err := s.repo.LoadSimulationDataset(ctx, tid, scenarioID, entityID)
	if err != nil {
		return WhatIfSimulation{}, false, err
	}
	in.HorizonDays = horizon
	outcome := EvaluatePipeline(ds, in)

	shocks := WhatIfShocks{
		DemandDeltaBps:       in.DemandDeltaBps,
		FactoryDowntimeHours: in.FactoryDowntimeHours,
		FxRateShockBps:       in.FxRateShockBps,
		CreditLimitDeltaBps:  in.CreditLimitDeltaBps,
		UnitPriceDeltaBps:    in.UnitPriceDeltaBps,
	}
	shocksRaw, err := json.Marshal(shocks)
	if err != nil {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: marshal shocks: %v", ErrInvalidInput, err)
	}
	breakdownRaw, err := json.Marshal(outcome.Breakdown)
	if err != nil {
		return WhatIfSimulation{}, false, fmt.Errorf("%w: marshal breakdown: %v", ErrInvalidInput, err)
	}

	now := s.currentNow()
	actorID := nonEmpty(in.ActorID, auth.ActorFromContext(ctx), "system:whatif")
	actorType := nonEmpty(in.ActorType, auth.ActorTypeHuman)
	sim := WhatIfSimulation{
		TenantID:                 tid,
		SimulationID:             simID,
		ScenarioID:               scenarioID,
		Name:                     name,
		EntityID:                 entityID,
		HorizonDays:              outcome.HorizonDays,
		ShocksJSON:               string(shocksRaw),
		Shocks:                   shocks,
		ForecastDeltaMinor:       outcome.ForecastDeltaMinor,
		ReplenishmentCostMinor:   outcome.ReplenishmentCostMinor,
		DispatchCostDeltaMinor:   outcome.DispatchCostDeltaMinor,
		CreditExposureDeltaMinor: outcome.CreditExposureDeltaMinor,
		NetPnLDeltaMinor:         outcome.NetPnLDeltaMinor,
		BreakdownJSON:            string(breakdownRaw),
		Breakdown:                &outcome.Breakdown,
		CommittedProposalIdsJSON: "[]",
		CommittedProposalIDs:     []string{},
		Status:                   StatusSimulated,
		CreatedBy:                actorID,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	auditIn := AuditRecordInput{
		TenantID:   tid,
		ActorID:    actorID,
		ActorType:  actorType,
		Role:       in.ActorRole,
		Action:     "whatif.simulate",
		ObjectType: "WhatIfSimulation",
		ObjectID:   simID,
		AfterJSON:  string(breakdownRaw),
		TraceID:    in.TraceID,
		IP:         in.IP,
		DeviceID:   in.DeviceID,
	}

	saved, replayed, err := s.repo.SaveSimulation(ctx, sim, s.audit, auditIn)
	if err != nil {
		return WhatIfSimulation{}, false, err
	}
	return saved, replayed, nil
}

// ListSimulations returns WhatIfSimulations scoped to tenantID.
func (s *Service) ListSimulations(ctx context.Context, f ListFilter) ([]WhatIfSimulation, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	f.TenantID = strings.TrimSpace(f.TenantID)
	if f.TenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if st := strings.ToUpper(strings.TrimSpace(f.Status)); st != "" {
		switch st {
		case StatusSimulated, StatusCommitted:
			f.Status = st
		default:
			return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidInput, f.Status)
		}
	}
	f.Limit = clampLimit(f.Limit)
	return s.repo.ListSimulations(ctx, f)
}

// GetSimulation loads one WhatIfSimulation by tenantID and simID.
func (s *Service) GetSimulation(ctx context.Context, tenantID, simID string) (WhatIfSimulation, error) {
	if s == nil || s.repo == nil {
		return WhatIfSimulation{}, ErrRepoUnavailable
	}
	sim, ok, err := s.repo.GetSimulation(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(simID))
	if err != nil {
		return WhatIfSimulation{}, err
	}
	if !ok {
		return WhatIfSimulation{}, ErrNotFound
	}
	return sim, nil
}

// CommitProposals turns recommended actions from a What-If simulation into governed L1
// PENDING Proposals via ProposalCreator (never self-approves or mutates domain tables).
func (s *Service) CommitProposals(ctx context.Context, in CommitProposalsInput) (CommitProposalsResult, error) {
	if s == nil || s.repo == nil {
		return CommitProposalsResult{}, ErrRepoUnavailable
	}
	if s.proposals == nil {
		return CommitProposalsResult{}, ErrProposalUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	sid := strings.TrimSpace(in.SimulationID)
	if tid == "" || sid == "" {
		return CommitProposalsResult{}, fmt.Errorf("%w: tenant_id and simulation_id are required", ErrInvalidInput)
	}
	if strings.EqualFold(in.ActorRole, auth.SupplierRoleAuditor) {
		return CommitProposalsResult{}, fmt.Errorf("%w: AUDITOR cannot commit simulation proposals", ErrForbidden)
	}
	if strings.EqualFold(in.ActorRole, auth.SupplierRoleAgent) || strings.EqualFold(in.ActorType, auth.ActorTypeAgent) {
		return CommitProposalsResult{}, fmt.Errorf("%w: AGENT cannot commit simulation proposals", ErrForbidden)
	}

	sim, found, err := s.repo.GetSimulation(ctx, tid, sid)
	if err != nil {
		return CommitProposalsResult{}, err
	}
	if !found {
		return CommitProposalsResult{}, ErrNotFound
	}

	idemKey := strings.TrimSpace(in.IdempotencyKey)
	if sim.Status == StatusCommitted {
		if idemKey != "" && sim.Breakdown != nil && sim.Breakdown.CommitIdempotencyKey == idemKey {
			return CommitProposalsResult{
				Simulation:           sim,
				Proposals:            []CreatedProposal{},
				CommittedProposalIDs: sim.CommittedProposalIDs,
			}, nil
		}
		return CommitProposalsResult{}, fmt.Errorf("%w: simulation %s proposals already committed", ErrConflict, sid)
	}

	var actions []RecommendedAction
	if sim.Breakdown != nil {
		actions = sim.Breakdown.RecommendedActions
	}
	if len(in.ProposalTypes) > 0 {
		allow := make(map[string]bool, len(in.ProposalTypes))
		for _, pt := range in.ProposalTypes {
			if norm := strings.ToUpper(strings.TrimSpace(pt)); norm != "" {
				allow[norm] = true
			}
		}
		filtered := make([]RecommendedAction, 0, len(actions))
		for _, a := range actions {
			if allow[strings.ToUpper(strings.TrimSpace(a.ProposalType))] {
				filtered = append(filtered, a)
			}
		}
		actions = filtered
	}
	if len(actions) == 0 {
		return CommitProposalsResult{}, fmt.Errorf("%w: no matching recommended actions to commit", ErrInvalidInput)
	}

	actorID := nonEmpty(in.ActorID, auth.ActorFromContext(ctx), "human")
	actorType := nonEmpty(in.ActorType, auth.ActorTypeHuman)
	currency := "UZS"
	if sim.Breakdown != nil && strings.TrimSpace(sim.Breakdown.Currency) != "" {
		currency = sim.Breakdown.Currency
	}

	createdProps := make([]CreatedProposal, 0, len(actions))
	propIDs := make([]string, 0, len(actions))
	overrideAgent := strings.ToUpper(strings.TrimSpace(in.AgentID))

	for idx, act := range actions {
		agentID := nonEmpty(overrideAgent, act.AgentID, "REORDER_AGENT")
		payload := make(map[string]any, len(act.Payload)+2)
		for k, v := range act.Payload {
			payload[k] = v
		}
		payload["simulation_id"] = sim.SimulationID
		payload["simulation_name"] = sim.Name
		if err := validateNoFloats(payload); err != nil {
			return CommitProposalsResult{}, err
		}

		prop, pErr := s.proposals.CreateProposal(ctx, ProposalCreateInput{
			TenantID:         tid,
			AgentID:          agentID,
			ProposalType:     act.ProposalType,
			TargetObjectType: act.TargetObjectType,
			TargetObjectID:   act.TargetObjectID,
			Title:            act.Title,
			Summary:          act.Summary,
			Rationale:        fmt.Sprintf("Committed from What-If simulation %s (%s): Net P&L delta %+d minor", sim.SimulationID, sim.Name, sim.NetPnLDeltaMinor),
			Payload:          payload,
			EvidenceRefs: []ProposalEvidenceRef{
				{Type: "WhatIfSimulation", ID: sim.SimulationID, Label: sim.Name},
				{Type: act.TargetObjectType, ID: act.TargetObjectID, Label: act.ProposalType},
			},
			ConfidenceBps:        clampBps(act.ConfidenceBps),
			EstimatedImpactMinor: max64(0, act.EstimatedImpactMinor),
			Currency:             currency,
			CreatedBy:            actorID,
			ActorType:            actorType,
			IdempotencyKey:       fmt.Sprintf("whatif-commit:%s:%d:%s", sim.SimulationID, idx, act.ProposalType),
		})
		if pErr != nil {
			return CommitProposalsResult{}, fmt.Errorf("create proposal for %s: %w", act.ProposalType, pErr)
		}
		createdProps = append(createdProps, prop)
		propIDs = append(propIDs, prop.ProposalID)
	}

	now := s.currentNow()
	beforeRaw, _ := json.Marshal(sim)
	idsRaw, _ := json.Marshal(propIDs)

	updated, err := s.repo.CommitSimulationProposals(ctx, tid, sid, func(cur WhatIfSimulation) (WhatIfSimulation, error) {
		if cur.Status == StatusCommitted {
			return WhatIfSimulation{}, fmt.Errorf("%w: simulation %s proposals already committed", ErrConflict, sid)
		}
		cur.Status = StatusCommitted
		cur.CommittedProposalIDs = propIDs
		cur.CommittedProposalIdsJSON = string(idsRaw)
		if cur.Breakdown != nil && idemKey != "" {
			cur.Breakdown.CommitIdempotencyKey = idemKey
			if bRaw, mErr := json.Marshal(cur.Breakdown); mErr == nil {
				cur.BreakdownJSON = string(bRaw)
			}
		}
		cur.UpdatedAt = now
		return cur, nil
	}, s.audit, func(u WhatIfSimulation) AuditRecordInput {
		afterRaw, _ := json.Marshal(u)
		return AuditRecordInput{
			TenantID:   tid,
			ActorID:    actorID,
			ActorType:  actorType,
			Role:       in.ActorRole,
			Action:     "whatif.commit_proposals",
			ObjectType: "WhatIfSimulation",
			ObjectID:   u.SimulationID,
			BeforeJSON: string(beforeRaw),
			AfterJSON:  string(afterRaw),
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
	}, now)
	if err != nil {
		return CommitProposalsResult{}, err
	}

	return CommitProposalsResult{
		Simulation:           updated,
		Proposals:            createdProps,
		CommittedProposalIDs: propIDs,
	}, nil
}
