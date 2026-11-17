package credit

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// LimitChangeInput captures inputs for governed credit limit mutations on POST /v1/credit/limits/{id}.
type LimitChangeInput struct {
	RetailerID      string
	SupplierID      string
	NewLimitMinor   int64
	Reason          string
	ConsentVerified *bool
	ActorID         string
	ActorType       string
	ActorRole       string
	IsAuditor       bool
	ApprovalID      string
	IdempotencyKey  string
}

// LimitChangeOutput captures either immediate/approved limit application or pending CEL approval.
type LimitChangeOutput struct {
	Status            string             `json:"status"` // APPLIED | APPROVAL_REQUIRED
	ApprovalRequired  bool               `json:"approval_required"`
	ApprovalRequestID string             `json:"approval_request_id,omitempty"`
	PayloadHash       string             `json:"payload_hash,omitempty"`
	OldLimitMinor     int64              `json:"old_limit_minor"`
	NewLimitMinor     int64              `json:"new_limit_minor"`
	Profile           *Profile           `json:"profile,omitempty"`
	ScoreRecord       *CreditScoreRecord `json:"score_record,omitempty"`
}

func (s *Service) getScoreRepo() ScoreHistoryRepository {
	if s == nil {
		return nil
	}
	s.mutMu.Lock()
	r := s.scoreRepo
	s.mutMu.Unlock()
	return r
}

func (s *Service) getMLScorer() MLScorer {
	if s == nil {
		return nil
	}
	s.mutMu.Lock()
	m := s.mlScorer
	s.mutMu.Unlock()
	return m
}

func (s *Service) getLimitApprovalBridge() LimitApprovalBridge {
	if s == nil {
		return nil
	}
	s.mutMu.Lock()
	b := s.limitApprovalBridge
	s.mutMu.Unlock()
	return b
}

func (s *Service) getPolicyService() *PolicyService {
	if s == nil {
		return nil
	}
	s.mutMu.Lock()
	ps := s.policySvc
	s.mutMu.Unlock()
	return ps
}

func (s *Service) loadRelMetricsAndConsent(ctx context.Context, retailerID, supplierID string) (ScoreMetrics, bool) {
	var m ScoreMetrics
	var consent bool
	if sr := s.getScoreRepo(); sr != nil {
		if loaded, cv, err := sr.LoadRelationshipMetrics(ctx, retailerID, supplierID); err == nil {
			m = loaded
			consent = cv
		}
	}
	if s.scoreMetrics != nil && m.MaxDaysPastDue == 0 && m.ExpectedPayments90 == 0 && m.PaymentsLast90d == 0 {
		if fallback, err := s.scoreMetrics.LoadScoreMetrics(ctx, retailerID, supplierID); err == nil {
			if fallback.MaxDaysPastDue > m.MaxDaysPastDue {
				m.MaxDaysPastDue = fallback.MaxDaysPastDue
			}
			if fallback.PaymentsLast90d > m.PaymentsLast90d {
				m.PaymentsLast90d = fallback.PaymentsLast90d
			}
			if fallback.ExpectedPayments90 > m.ExpectedPayments90 {
				m.ExpectedPayments90 = fallback.ExpectedPayments90
			}
		}
	}
	if ps := s.getPolicyService(); ps != nil && ps.repo != nil {
		if terms, ok, err := ps.repo.GetTerms(ctx, retailerID, supplierID); err == nil && ok {
			if terms.ConsentVerified {
				consent = true
			}
			if m.StarterLimitMinor <= 0 && terms.CreditLimitMinor > 0 {
				m.StarterLimitMinor = terms.CreditLimitMinor
			}
		}
		if m.StarterLimitMinor <= 0 {
			if prog, ok, err := ps.repo.GetProgram(ctx, supplierID); err == nil && ok && prog.GlobalDefaultLimitMinor > 0 {
				m.StarterLimitMinor = prog.GlobalDefaultLimitMinor
			}
		}
	}
	return m, consent
}

func buildMLFeatures(p Profile, m ScoreMetrics, scope string, consentingSuppliers int64) MLFeatures {
	exposure := p.CurrentBalanceMinor + p.ReservedMinor
	var utilBps int64
	if p.CreditLimitMinor > 0 && exposure > 0 {
		utilBps = (exposure * 10000) / p.CreditLimitMinor
	}
	return MLFeatures{
		RetailerID:            p.RetailerID,
		SupplierID:            p.SupplierID,
		Scope:                 scope,
		CreditLimitMinor:      p.CreditLimitMinor,
		CurrentBalanceMinor:   p.CurrentBalanceMinor,
		ReservedMinor:         p.ReservedMinor,
		UtilizationBps:        utilBps,
		DelinquencyCount:      p.DelinquencyCount,
		MaxDaysPastDue:        m.MaxDaysPastDue,
		PaymentsLast90d:       m.PaymentsLast90d,
		ExpectedPayments90d:   m.ExpectedPayments90,
		PaidVolume90dMinor:    m.PaidVolume90dMinor,
		AvgOrderMinor:         m.AvgOrderMinor,
		OrderCount90d:         m.OrderCount90d,
		DisputedInvoicesCount: m.DisputedInvoicesCount,
		TenureDays:            m.TenureDays,
		ConsentingSuppliers:   consentingSuppliers,
		ProfileStatus:         string(p.Status),
	}
}

func appendUniqueReasonCodes(base, extra []ReasonCode) []ReasonCode {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]ReasonCode, 0, len(base)+len(extra))
	for _, rc := range base {
		if rc.Code == "" {
			continue
		}
		if _, ok := seen[rc.Code]; ok {
			continue
		}
		seen[rc.Code] = struct{}{}
		out = append(out, rc)
	}
	for _, rc := range extra {
		if rc.Code == "" {
			continue
		}
		if _, ok := seen[rc.Code]; ok {
			continue
		}
		seen[rc.Code] = struct{}{}
		out = append(out, rc)
	}
	return out
}

func (s *Service) evaluateScoreWithML(
	ctx context.Context,
	p Profile,
	m ScoreMetrics,
	scope string,
	consentVerified bool,
	consentScope string,
	consentingSuppliers int64,
	isolatedDisputes int64,
	isolatedAmount int64,
	now time.Time,
) (CreditScoreRecord, error) {
	sc := ComputeRiskScore(p, m, now)
	features := buildMLFeatures(p, m, scope, consentingSuppliers)
	featuresHash := ComputeFeaturesHash(features)

	reasons := sc.ReasonCodes
	if reasons == nil {
		reasons = []ReasonCode{}
	}
	score := sc.Score
	pdBps := sc.PdBps
	riskTier := sc.RiskTier
	suggested := sc.SuggestedLimitMinor
	scorerSource := ScorerSourceScorecard
	modelVersion := "scorecard_v2"

	if scorer := s.getMLScorer(); scorer != nil {
		mlResp, err := scorer.Score(ctx, MLScorerRequest{
			RetailerID:   p.RetailerID,
			SupplierID:   p.SupplierID,
			Scope:        scope,
			Features:     features,
			FeaturesHash: featuresHash,
		})
		if err != nil {
			return CreditScoreRecord{}, err
		}
		if mlResp.FeaturesHash != "" {
			featuresHash = mlResp.FeaturesHash
		}
		if mlResp.ScorerSource == ScorerSourceMLSidecar {
			scorerSource = ScorerSourceMLSidecar
			if mlResp.ModelVersion != "" {
				modelVersion = mlResp.ModelVersion
			}
			score = mlResp.Score
			pdBps = mlResp.PdBps
			riskTier = tierFromScore(score, p.Status)
			suggested, _ = suggestedLimitWithReason(p, m, score)
			reasons = appendUniqueReasonCodes(mlResp.ReasonCodes, reasons)
		} else if len(mlResp.ReasonCodes) > 0 {
			reasons = appendUniqueReasonCodes(reasons, mlResp.ReasonCodes)
		}
	}

	if scope == ScoreScopeNetwork {
		if consentVerified {
			reasons = appendUniqueReasonCodes(reasons, []ReasonCode{{
				Code:      "NETWORK_CONSENT_VERIFIED",
				Label:     "Cross-supplier scoring consent verified",
				ImpactBps: 0,
				Detail:    fmt.Sprintf("consenting_suppliers=%d", consentingSuppliers),
				Source:    "CONSENT",
			}})
		}
		if isolatedDisputes > 0 {
			reasons = appendUniqueReasonCodes(reasons, []ReasonCode{{
				Code:      "SUPPLIER_DISPUTES_ISOLATED",
				Label:     "Supplier-specific disputed invoices isolated from network score",
				ImpactBps: 0,
				Detail:    fmt.Sprintf("isolated_disputes=%d disputed_minor=%d", isolatedDisputes, isolatedAmount),
				Source:    "DISPUTE_ISOLATION",
			}})
		}
	}

	rcBytes, _ := json.Marshal(reasons)
	return CreditScoreRecord{
		ScoreID:               "csh_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		RetailerID:            p.RetailerID,
		SupplierID:            p.SupplierID,
		Scope:                 scope,
		ConsentVerified:       consentVerified,
		ConsentScope:          consentScope,
		ModelVersion:          modelVersion,
		ScorerSource:          scorerSource,
		Score:                 score,
		PdBps:                 pdBps,
		RiskTier:              riskTier,
		SuggestedLimitMinor:   suggested,
		CurrentLimitMinor:     p.CreditLimitMinor,
		ReasonCodes:           reasons,
		ReasonCodesJSON:       string(rcBytes),
		FactorsJSON:           sc.FactorsJSON,
		FeaturesHash:          featuresHash,
		DisputesIsolatedCount: isolatedDisputes,
		WindowStart:           sc.WindowStart,
		WindowEnd:             sc.WindowEnd,
		ComputedAt:            now,
		CreatedAt:             now,
	}, nil
}

// ComputeRelationshipScore scores a retailer x supplier relationship, invokes optional/required ML scorer,
// persists CreditScoreHistory, and emits credit.score.computed.
func (s *Service) ComputeRelationshipScore(ctx context.Context, retailerID, supplierID string) (CreditScoreRecord, error) {
	if s == nil || s.repo == nil {
		return CreditScoreRecord{}, fmt.Errorf("credit service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if retailerID == "" || supplierID == "" {
		return CreditScoreRecord{}, ErrInvalidInput
	}
	now := s.now()
	p, found, err := s.repo.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return CreditScoreRecord{}, err
	}
	if !found {
		p = Profile{
			RetailerID: retailerID,
			SupplierID: supplierID,
			Status:     StatusActive,
		}
	}
	m, consent := s.loadRelMetricsAndConsent(ctx, retailerID, supplierID)
	consentScope := "NONE"
	if consent {
		consentScope = "SUPPLIER_RELATIONSHIP"
	}
	rec, err := s.evaluateScoreWithML(
		ctx, p, m, ScoreScopeRelationship,
		consent, consentScope, 0,
		m.DisputedInvoicesCount, m.DisputedAmountMinor,
		now,
	)
	if err != nil {
		return CreditScoreRecord{}, err
	}
	if sr := s.getScoreRepo(); sr != nil {
		if err := sr.SaveScoreHistory(ctx, rec, false); err != nil {
			return CreditScoreRecord{}, err
		}
	}
	return rec, nil
}

// ComputeNetworkScore aggregates consenting supplier relationships (ConsentVerified=true) for a retailer,
// isolates supplier-specific disputed invoices, persists CreditScoreHistory + RetailerCreditScores,
// and emits credit.score.computed.
func (s *Service) ComputeNetworkScore(ctx context.Context, retailerID string) (CreditScoreRecord, error) {
	if s == nil || s.repo == nil {
		return CreditScoreRecord{}, fmt.Errorf("credit service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return CreditScoreRecord{}, ErrInvalidInput
	}
	now := s.now()
	sr := s.getScoreRepo()
	var inputs NetworkScoreInputs
	if sr != nil {
		loaded, err := sr.LoadNetworkInputs(ctx, retailerID)
		if err != nil {
			return CreditScoreRecord{}, err
		}
		inputs = loaded
	}

	if len(inputs.ConsentingProfiles) == 0 {
		reasons := []ReasonCode{
			{
				Code:      "NETWORK_CONSENT_UNVERIFIED",
				Label:     "No supplier relationships with ConsentVerified=true for network scoring",
				ImpactBps: 0,
				Detail:    "cross_supplier_consent_count=0",
				Source:    "CONSENT",
			},
			{
				Code:      "THIN_FILE_NO_HISTORY_OR_PROGRAM_LIMIT",
				Label:     "Thin credit file: no profile limit, starter limit, or settled history",
				ImpactBps: 0,
				Detail:    "suggested_limit_minor=0",
				Source:    "SCORECARD",
			},
		}
		rcBytes, _ := json.Marshal(reasons)
		features := buildMLFeatures(Profile{RetailerID: retailerID, Status: StatusInactive}, ScoreMetrics{}, ScoreScopeNetwork, 0)
		rec := CreditScoreRecord{
			ScoreID:               "csh_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
			RetailerID:            retailerID,
			Scope:                 ScoreScopeNetwork,
			ConsentVerified:       false,
			ConsentScope:          "NONE",
			ModelVersion:          "scorecard_v2",
			ScorerSource:          ScorerSourceScorecard,
			Score:                 0,
			PdBps:                 calibratedPdBpsFromScore(0, StatusInactive),
			RiskTier:              RiskTierBlock,
			SuggestedLimitMinor:   0,
			CurrentLimitMinor:     0,
			ReasonCodes:           reasons,
			ReasonCodesJSON:       string(rcBytes),
			FeaturesHash:          ComputeFeaturesHash(features),
			DisputesIsolatedCount: inputs.DisputesIsolatedCount,
			WindowStart:           now.Add(-90 * 24 * time.Hour),
			WindowEnd:             now,
			ComputedAt:            now,
			CreatedAt:             now,
		}
		if sr != nil {
			if err := sr.SaveScoreHistory(ctx, rec, true); err != nil {
				return CreditScoreRecord{}, err
			}
		}
		return rec, nil
	}

	// Aggregate consenting suppliers without propagating supplier-specific disputes into delinquency/DPD.
	aggProfile := Profile{
		RetailerID: retailerID,
		Status:     StatusInactive,
	}
	var aggMetrics ScoreMetrics
	var activeCount int64
	var frozenCount int64
	for _, csp := range inputs.ConsentingProfiles {
		aggProfile.CreditLimitMinor += csp.Profile.CreditLimitMinor
		aggProfile.CurrentBalanceMinor += csp.Profile.CurrentBalanceMinor
		aggProfile.ReservedMinor += csp.Profile.ReservedMinor
		aggProfile.DelinquencyCount += csp.Profile.DelinquencyCount
		switch csp.Profile.Status {
		case StatusActive:
			activeCount++
		case StatusFrozen:
			frozenCount++
		}
		if csp.Metrics.StarterLimitMinor > aggMetrics.StarterLimitMinor {
			aggMetrics.StarterLimitMinor = csp.Metrics.StarterLimitMinor
		}
		if csp.Terms.CreditLimitMinor > aggMetrics.StarterLimitMinor {
			aggMetrics.StarterLimitMinor = csp.Terms.CreditLimitMinor
		}
		if csp.Metrics.MaxDaysPastDue > aggMetrics.MaxDaysPastDue {
			aggMetrics.MaxDaysPastDue = csp.Metrics.MaxDaysPastDue
		}
		aggMetrics.PaymentsLast90d += csp.Metrics.PaymentsLast90d
		aggMetrics.ExpectedPayments90 += csp.Metrics.ExpectedPayments90
		aggMetrics.PaidVolume90dMinor += csp.Metrics.PaidVolume90dMinor
		aggMetrics.OrderCount90d += csp.Metrics.OrderCount90d
		if csp.Metrics.TenureDays > aggMetrics.TenureDays {
			aggMetrics.TenureDays = csp.Metrics.TenureDays
		}
	}
	if activeCount > 0 {
		aggProfile.Status = StatusActive
	} else if frozenCount > 0 {
		aggProfile.Status = StatusFrozen
	}
	if inputs.TotalConsentingCount > 0 && aggMetrics.OrderCount90d > 0 {
		var orderSum int64
		for _, csp := range inputs.ConsentingProfiles {
			orderSum += csp.Metrics.AvgOrderMinor * maxInt64(csp.Metrics.OrderCount90d, 1)
		}
		aggMetrics.AvgOrderMinor = orderSum / maxInt64(aggMetrics.OrderCount90d, 1)
	}
	// Note: aggMetrics.DisputedInvoicesCount stays 0 in the aggregate scoring input so
	// disputed invoices from one supplier stay isolated and recorded separately in DisputesIsolatedCount.
	rec, err := s.evaluateScoreWithML(
		ctx, aggProfile, aggMetrics, ScoreScopeNetwork,
		true, "NETWORK_CONSENTING_SUPPLIERS", inputs.TotalConsentingCount,
		inputs.DisputesIsolatedCount, inputs.DisputedAmountMinor,
		now,
	)
	if err != nil {
		return CreditScoreRecord{}, err
	}
	if sr != nil {
		if err := sr.SaveScoreHistory(ctx, rec, true); err != nil {
			return CreditScoreRecord{}, err
		}
	}
	return rec, nil
}

// GetRetailerScoreView returns relationship and/or network score plus CreditScoreHistory.
// Supplier callers see ONLY their own supplierID's RELATIONSHIP scores plus aggregated consenting NETWORK scores.
func (s *Service) GetRetailerScoreView(ctx context.Context, retailerID, supplierID, scope string, recompute bool) (RetailerScoreView, error) {
	if s == nil || s.repo == nil {
		return RetailerScoreView{}, fmt.Errorf("credit service unavailable")
	}
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if retailerID == "" {
		return RetailerScoreView{}, ErrInvalidInput
	}
	normScope := strings.ToUpper(strings.TrimSpace(scope))
	if normScope == "" {
		normScope = "BOTH"
	}

	view := RetailerScoreView{
		RetailerID: retailerID,
		History:    []CreditScoreRecord{},
	}

	wantRel := normScope == "BOTH" || normScope == ScoreScopeRelationship
	wantNet := normScope == "BOTH" || normScope == ScoreScopeNetwork

	if recompute {
		if wantRel && supplierID != "" {
			relRec, err := s.ComputeRelationshipScore(ctx, retailerID, supplierID)
			if err != nil {
				return RetailerScoreView{}, err
			}
			view.RelationshipScore = &relRec
		}
		if wantNet {
			netRec, err := s.ComputeNetworkScore(ctx, retailerID)
			if err != nil {
				return RetailerScoreView{}, err
			}
			view.NetworkScore = &netRec
		}
	}

	if sr := s.getScoreRepo(); sr != nil {
		hist, err := sr.ListScoreHistory(ctx, retailerID, "", "", 50)
		if err != nil {
			return RetailerScoreView{}, err
		}
		filtered := make([]CreditScoreRecord, 0, len(hist))
		for _, rec := range hist {
			if rec.ReasonCodes == nil {
				rec.ReasonCodes = []ReasonCode{}
			}
			switch rec.Scope {
			case ScoreScopeNetwork:
				if !wantNet {
					continue
				}
				filtered = append(filtered, rec)
				if view.NetworkScore == nil {
					copyRec := rec
					view.NetworkScore = &copyRec
				}
			case ScoreScopeRelationship:
				if !wantRel {
					continue
				}
				if supplierID != "" && rec.SupplierID != supplierID {
					continue
				}
				filtered = append(filtered, rec)
				if view.RelationshipScore == nil && (supplierID == "" || rec.SupplierID == supplierID) {
					copyRec := rec
					view.RelationshipScore = &copyRec
				}
			}
		}
		view.History = filtered
	}

	if wantRel && view.RelationshipScore == nil && supplierID != "" {
		relRec, err := s.ComputeRelationshipScore(ctx, retailerID, supplierID)
		if err != nil {
			return RetailerScoreView{}, err
		}
		view.RelationshipScore = &relRec
		view.History = append([]CreditScoreRecord{relRec}, view.History...)
	}
	if wantNet && view.NetworkScore == nil {
		netRec, err := s.ComputeNetworkScore(ctx, retailerID)
		if err != nil {
			return RetailerScoreView{}, err
		}
		view.NetworkScore = &netRec
		view.History = append(view.History, netRec)
	}
	return view, nil
}

// GetSupplierPortfolio returns portfolio exposure, utilization, risk scores, consent flags, and reason codes.
func (s *Service) GetSupplierPortfolio(ctx context.Context, supplierID, statusFilter, riskTierFilter string, limit int) (PortfolioResponse, error) {
	if s == nil || s.repo == nil {
		return PortfolioResponse{}, fmt.Errorf("credit service unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return PortfolioResponse{}, ErrInvalidInput
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	profiles, err := s.repo.ListBySupplier(ctx, supplierID, statusFilter, limit)
	if err != nil {
		return PortfolioResponse{}, err
	}
	rids := make([]string, 0, len(profiles))
	for _, p := range profiles {
		rids = append(rids, p.RetailerID)
	}
	metaByRetailer, _, _ := s.loadDeskMetadata(ctx, supplierID, rids)

	now := s.now()
	wantTier := strings.ToUpper(strings.TrimSpace(riskTierFilter))
	items := make([]PortfolioItem, 0, len(profiles))
	var summary PortfolioSummary
	var scoreSum int64

	for _, p := range profiles {
		metrics, consent := s.loadRelMetricsAndConsent(ctx, p.RetailerID, supplierID)
		sc := ComputeRiskScore(p, metrics, now)
		if wantTier != "" && string(sc.RiskTier) != wantTier {
			continue
		}
		exposure := p.CurrentBalanceMinor + p.ReservedMinor
		var utilBps int64
		if p.CreditLimitMinor > 0 && exposure > 0 {
			utilBps = (exposure * 10000) / p.CreditLimitMinor
		}
		reasons := sc.ReasonCodes
		if reasons == nil {
			reasons = []ReasonCode{}
		}
		var name string
		if meta, ok := metaByRetailer[p.RetailerID]; ok {
			name = meta.RetailerName
			if name == "" {
				name = meta.DisplayName
			}
		}
		items = append(items, PortfolioItem{
			RetailerID:           p.RetailerID,
			RetailerName:         name,
			CreditLimitMinor:     p.CreditLimitMinor,
			CurrentBalanceMinor:  p.CurrentBalanceMinor,
			ReservedMinor:        p.ReservedMinor,
			AvailableCreditMinor: p.Available(),
			UtilizationBps:       utilBps,
			RiskScore:            sc.Score,
			PdBps:                sc.PdBps,
			RiskTier:             sc.RiskTier,
			Status:               p.Status,
			PrepayOnlyLocked:     p.PrepayOnlyLocked,
			ConsentVerified:      consent,
			SuggestedLimitMinor:  sc.SuggestedLimitMinor,
			ReasonCodes:          reasons,
			LastEvaluatedAt:      sc.ComputedAt.UTC().Format(time.RFC3339),
		})
		summary.TotalRetailers++
		summary.TotalExposureMinor += exposure
		summary.TotalLimitMinor += p.CreditLimitMinor
		if p.Status == StatusFrozen {
			summary.FrozenCount++
		}
		if sc.RiskTier == RiskTierHigh || sc.RiskTier == RiskTierBlock {
			summary.HighRiskCount++
		}
		if consent {
			summary.ConsentVerifiedCount++
		}
		scoreSum += sc.Score
	}

	if summary.TotalLimitMinor > 0 && summary.TotalExposureMinor > 0 {
		summary.UtilizationBps = (summary.TotalExposureMinor * 10000) / summary.TotalLimitMinor
	}
	if summary.TotalRetailers > 0 {
		summary.AvgScore = scoreSum / summary.TotalRetailers
	}
	slices.SortStableFunc(items, func(a, b PortfolioItem) int {
		if a.RiskScore != b.RiskScore {
			return cmp.Compare(a.RiskScore, b.RiskScore)
		}
		return cmp.Compare(b.CurrentBalanceMinor, a.CurrentBalanceMinor)
	})

	return PortfolioResponse{
		SupplierID: supplierID,
		Summary:    summary,
		Items:      items,
	}, nil
}

// RequestOrApplyLimitChange enforces actor permissions, optional M1.2 CEL approval + SoD bridge,
// persists credit score history + policy audit + limit change, and emits credit.limit.changed.
func (s *Service) RequestOrApplyLimitChange(ctx context.Context, in LimitChangeInput) (LimitChangeOutput, error) {
	if s == nil || s.repo == nil {
		return LimitChangeOutput{}, fmt.Errorf("credit service unavailable")
	}
	in.RetailerID = strings.TrimSpace(in.RetailerID)
	in.SupplierID = strings.TrimSpace(in.SupplierID)
	in.Reason = strings.TrimSpace(in.Reason)
	in.ApprovalID = strings.TrimSpace(in.ApprovalID)
	if in.RetailerID == "" || in.SupplierID == "" || in.NewLimitMinor < 0 {
		return LimitChangeOutput{}, ErrInvalidInput
	}
	if in.IsAuditor {
		return LimitChangeOutput{}, ErrAuditorReadOnly
	}
	if strings.EqualFold(strings.TrimSpace(in.ActorType), "agent") {
		return LimitChangeOutput{}, ErrAgentApprovalForbidden
	}
	if in.Reason == "" {
		in.Reason = "credit_limit_adjustment"
	}

	now := s.now()
	existing, found, err := s.repo.GetProfile(ctx, in.RetailerID, in.SupplierID)
	if err != nil {
		return LimitChangeOutput{}, err
	}
	if !found {
		existing = Profile{
			RetailerID: in.RetailerID,
			SupplierID: in.SupplierID,
			Status:     StatusActive,
			CreatedAt:  now,
		}
	}
	oldLimitMinor := existing.CreditLimitMinor

	metrics, currentConsent := s.loadRelMetricsAndConsent(ctx, in.RetailerID, in.SupplierID)
	effectiveConsent := currentConsent
	if in.ConsentVerified != nil {
		effectiveConsent = *in.ConsentVerified
	}
	consentScope := "NONE"
	if effectiveConsent {
		consentScope = "SUPPLIER_RELATIONSHIP"
	}

	scoreRec, err := s.evaluateScoreWithML(
		ctx, existing, metrics, ScoreScopeRelationship,
		effectiveConsent, consentScope, 0,
		metrics.DisputedInvoicesCount, metrics.DisputedAmountMinor,
		now,
	)
	if err != nil {
		return LimitChangeOutput{}, err
	}

	if bridge := s.getLimitApprovalBridge(); bridge != nil {
		deltaMinor := in.NewLimitMinor - oldLimitMinor
		absDeltaMinor := deltaMinor
		if absDeltaMinor < 0 {
			absDeltaMinor = -absDeltaMinor
		}
		var deltaPctBps int64
		if oldLimitMinor > 0 {
			deltaPctBps = (absDeltaMinor * 10000) / oldLimitMinor
		}
		payload := map[string]any{
			"retailer_id":     in.RetailerID,
			"supplier_id":     in.SupplierID,
			"old_limit_minor": oldLimitMinor,
			"new_limit_minor": in.NewLimitMinor,
			"reason":          in.Reason,
		}
		if in.ConsentVerified != nil {
			payload["consent_verified"] = *in.ConsentVerified
		}
		celCtx := map[string]any{
			"retailer_id":      in.RetailerID,
			"supplier_id":      in.SupplierID,
			"old_limit_minor":  oldLimitMinor,
			"new_limit_minor":  in.NewLimitMinor,
			"delta_minor":      deltaMinor,
			"abs_delta_minor":  absDeltaMinor,
			"delta_pct_bps":    deltaPctBps,
			"risk_score":       scoreRec.Score,
			"pd_bps":           scoreRec.PdBps,
			"risk_tier":        string(scoreRec.RiskTier),
			"consent_verified": effectiveConsent,
		}
		res, bErr := bridge.EvaluateOrVerifyLimitChange(ctx, LimitChangeApprovalInput{
			TenantID:       in.SupplierID,
			RetailerID:     in.RetailerID,
			SupplierID:     in.SupplierID,
			OldLimitMinor:  oldLimitMinor,
			NewLimitMinor:  in.NewLimitMinor,
			Reason:         in.Reason,
			ReasonCodes:    scoreRec.ReasonCodes,
			ActorID:        in.ActorID,
			ActorType:      in.ActorType,
			ActorRole:      in.ActorRole,
			ApprovalID:     in.ApprovalID,
			IdempotencyKey: in.IdempotencyKey,
			Context:        celCtx,
			Payload:        payload,
		})
		if bErr != nil {
			return LimitChangeOutput{}, bErr
		}
		if res.Required && !res.Approved {
			return LimitChangeOutput{
				Status:            "APPROVAL_REQUIRED",
				ApprovalRequired:  true,
				ApprovalRequestID: res.ApprovalRequestID,
				PayloadHash:       res.PayloadHash,
				OldLimitMinor:     oldLimitMinor,
				NewLimitMinor:     in.NewLimitMinor,
				ScoreRecord:       &scoreRec,
			}, nil
		}
	}

	scoreRec.CurrentLimitMinor = in.NewLimitMinor
	auditID := "cpa_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	var updated Profile

	if sr := s.getScoreRepo(); sr != nil {
		updated, err = sr.ApplyGovernedLimitChange(ctx, ApplyLimitRepoInput{
			RetailerID:      in.RetailerID,
			SupplierID:      in.SupplierID,
			NewLimitMinor:   in.NewLimitMinor,
			OldLimitMinor:   oldLimitMinor,
			ConsentVerified: in.ConsentVerified,
			ApprovalID:      in.ApprovalID,
			ActorID:         in.ActorID,
			ActorRole:       in.ActorRole,
			Reason:          in.Reason,
			ReasonCodesJSON: scoreRec.ReasonCodesJSON,
			ScoreRecord:     &scoreRec,
			Now:             now,
			AuditID:         auditID,
		})
		if err != nil {
			return LimitChangeOutput{}, err
		}
	} else {
		updated = existing
		updated.CreditLimitMinor = in.NewLimitMinor
		updated.RiskScore = scoreRec.Score
		updated.LastEvaluatedAt = now
		updated.UpdatedAt = now
		updated.AvailableCreditMinor = updated.Available()
		if err := s.repo.UpsertProfile(ctx, updated, func(txn outbox.TxnBuffer) error {
			tsStr := now.UTC().Format(time.RFC3339Nano)
			if err := outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, in.RetailerID, events.TopicMain, events.CreditLimitChangedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventCreditLimitChanged,
					Version:   1,
					Timestamp: tsStr,
				},
				RetailerID:         in.RetailerID,
				SupplierID:         in.SupplierID,
				PreviousLimitMinor: oldLimitMinor,
				NewLimitMinor:      in.NewLimitMinor,
				ApprovalID:         in.ApprovalID,
				ActorID:            in.ActorID,
				Reason:             in.Reason,
				ReasonCodesJSON:    scoreRec.ReasonCodesJSON,
				OccurredAt:         tsStr,
			}); err != nil {
				return err
			}
			return s.emitProfileChanged(ctx, txn, updated, "limit_changed:"+in.Reason)
		}); err != nil {
			return LimitChangeOutput{}, err
		}
		if ps := s.getPolicyService(); ps != nil && ps.repo != nil {
			if terms, ok, err := ps.repo.GetTerms(ctx, in.RetailerID, in.SupplierID); err == nil && ok {
				terms.CreditLimitMinor = in.NewLimitMinor
				if in.ConsentVerified != nil {
					terms.ConsentVerified = *in.ConsentVerified
				}
				terms.Version++
				terms.UpdatedAt = now
				_ = ps.repo.UpsertTerms(ctx, terms, nil)
			}
		}
	}

	return LimitChangeOutput{
		Status:            "APPLIED",
		ApprovalRequired:  false,
		ApprovalRequestID: in.ApprovalID,
		OldLimitMinor:     oldLimitMinor,
		NewLimitMinor:     updated.CreditLimitMinor,
		Profile:           &updated,
		ScoreRecord:       &scoreRec,
	}, nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
