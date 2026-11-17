package proposals

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository provides thread-safe in-memory storage for Proposals and optional
// seeded L1 agent candidates in unit tests without Spanner.
type MemoryRepository struct {
	mu           sync.RWMutex
	proposals    map[string]map[string]Proposal // tenantID -> proposalID -> Proposal
	byIdem       map[string]Proposal            // tenantID + ":" + idempotencyKey -> Proposal
	reorderCands map[string][]ReorderCandidate
	overdueInvs  map[string][]OverdueInvoiceCandidate
	expiryCands  map[string][]ExpiryRiskCandidate

	// Precondition state overrides for testing stale preconditions.
	closedInsights map[string]bool // tenantID + ":" + insightID
	paidInvoices   map[string]bool // tenantID + ":" + invoiceID
	depletedLots   map[string]bool // tenantID + ":" + lotID
}

// NewMemoryRepository constructs an empty in-memory proposals repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		proposals:      make(map[string]map[string]Proposal),
		byIdem:         make(map[string]Proposal),
		reorderCands:   make(map[string][]ReorderCandidate),
		overdueInvs:    make(map[string][]OverdueInvoiceCandidate),
		expiryCands:    make(map[string][]ExpiryRiskCandidate),
		closedInsights: make(map[string]bool),
		paidInvoices:   make(map[string]bool),
		depletedLots:   make(map[string]bool),
	}
}

func (m *MemoryRepository) SeedReorderCandidates(tenantID string, cands ...ReorderCandidate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(tenantID)
	m.reorderCands[tid] = append(m.reorderCands[tid], cands...)
}

func (m *MemoryRepository) SeedOverdueInvoices(tenantID string, invs ...OverdueInvoiceCandidate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(tenantID)
	m.overdueInvs[tid] = append(m.overdueInvs[tid], invs...)
}

func (m *MemoryRepository) SeedExpiryCandidates(tenantID string, cands ...ExpiryRiskCandidate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(tenantID)
	m.expiryCands[tid] = append(m.expiryCands[tid], cands...)
}

func (m *MemoryRepository) MarkInsightClosed(tenantID, insightID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closedInsights[strings.TrimSpace(tenantID)+":"+strings.TrimSpace(insightID)] = true
}

func (m *MemoryRepository) MarkInvoicePaid(tenantID, invoiceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paidInvoices[strings.TrimSpace(tenantID)+":"+strings.TrimSpace(invoiceID)] = true
}

func (m *MemoryRepository) MarkLotDepleted(tenantID, lotID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.depletedLots[strings.TrimSpace(tenantID)+":"+strings.TrimSpace(lotID)] = true
}

func (m *MemoryRepository) FindByIdempotency(_ context.Context, tenantID, idempotencyKey string) (Proposal, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.byIdem[strings.TrimSpace(tenantID)+":"+strings.TrimSpace(idempotencyKey)]
	return cloneProposal(p), ok, nil
}

func (m *MemoryRepository) Create(_ context.Context, p Proposal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(p.TenantID)
	if m.proposals[tid] == nil {
		m.proposals[tid] = make(map[string]Proposal)
	}
	if _, exists := m.proposals[tid][p.ProposalID]; exists {
		return fmt.Errorf("%w: duplicate proposal_id %s", ErrInvalidInput, p.ProposalID)
	}
	if idem := strings.TrimSpace(p.IdempotencyKey); idem != "" {
		key := tid + ":" + idem
		if _, exists := m.byIdem[key]; exists {
			return nil
		}
		m.byIdem[key] = cloneProposal(p)
	}
	m.proposals[tid][p.ProposalID] = cloneProposal(p)
	return nil
}

func (m *MemoryRepository) Get(_ context.Context, tenantID, proposalID string) (Proposal, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.proposals[strings.TrimSpace(tenantID)][strings.TrimSpace(proposalID)]
	if !ok {
		return Proposal{}, false, nil
	}
	return cloneProposal(p), true, nil
}

func (m *MemoryRepository) List(_ context.Context, f ListFilter) ([]Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(f.TenantID)
	out := make([]Proposal, 0)
	for _, p := range m.proposals[tid] {
		if f.AgentID != "" && !strings.EqualFold(p.AgentID, strings.TrimSpace(f.AgentID)) {
			continue
		}
		if f.Status != "" && !strings.EqualFold(p.Status, strings.TrimSpace(f.Status)) {
			continue
		}
		if f.TargetType != "" && !strings.EqualFold(p.TargetObjectType, strings.TrimSpace(f.TargetType)) {
			continue
		}
		if f.TargetID != "" && p.TargetObjectID != strings.TrimSpace(f.TargetID) {
			continue
		}
		out = append(out, cloneProposal(p))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ProposalID > out[j].ProposalID
	})
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (m *MemoryRepository) Update(_ context.Context, p Proposal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(p.TenantID)
	if m.proposals[tid] == nil {
		return ErrProposalNotFound
	}
	if _, ok := m.proposals[tid][p.ProposalID]; !ok {
		return ErrProposalNotFound
	}
	cp := cloneProposal(p)
	m.proposals[tid][p.ProposalID] = cp
	if idem := strings.TrimSpace(cp.IdempotencyKey); idem != "" {
		m.byIdem[tid+":"+idem] = cp
	}
	return nil
}

func (m *MemoryRepository) HasOverlappingExecuted(_ context.Context, tenantID, targetType, targetID, proposalType, excludeProposalID string, createdBeforeOrAt time.Time) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.proposals[strings.TrimSpace(tenantID)] {
		if p.ProposalID == excludeProposalID {
			continue
		}
		if p.Status == StatusExecuted &&
			strings.EqualFold(p.TargetObjectType, targetType) &&
			p.TargetObjectID == targetID &&
			strings.EqualFold(p.ProposalType, proposalType) &&
			p.ExecutedAt != nil && !createdBeforeOrAt.After(*p.ExecutedAt) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryRepository) HasActiveForTarget(_ context.Context, tenantID, targetType, targetID, proposalType string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.proposals[strings.TrimSpace(tenantID)] {
		if (p.Status == StatusPending || p.Status == StatusApproved) &&
			strings.EqualFold(p.TargetObjectType, targetType) &&
			p.TargetObjectID == targetID &&
			strings.EqualFold(p.ProposalType, proposalType) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryRepository) ListReorderCandidates(_ context.Context, tenantID string, limit int) ([]ReorderCandidate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.reorderCands[strings.TrimSpace(tenantID)]
	if limit > 0 && len(src) > limit {
		src = src[:limit]
	}
	out := make([]ReorderCandidate, len(src))
	copy(out, src)
	return out, nil
}

func (m *MemoryRepository) ListOverdueInvoices(_ context.Context, tenantID string, _ time.Time, limit int) ([]OverdueInvoiceCandidate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.overdueInvs[strings.TrimSpace(tenantID)]
	if limit > 0 && len(src) > limit {
		src = src[:limit]
	}
	out := make([]OverdueInvoiceCandidate, len(src))
	copy(out, src)
	return out, nil
}

func (m *MemoryRepository) ListExpiryRiskCandidates(_ context.Context, tenantID string, _ time.Time, _ int, limit int) ([]ExpiryRiskCandidate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.expiryCands[strings.TrimSpace(tenantID)]
	if limit > 0 && len(src) > limit {
		src = src[:limit]
	}
	out := make([]ExpiryRiskCandidate, len(src))
	copy(out, src)
	return out, nil
}

func (m *MemoryRepository) VerifyReorderPrecondition(_ context.Context, tenantID string, p Proposal) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	payload := decodePayloadMap(p.PayloadJSON)
	if iid, _ := payload["insight_id"].(string); strings.TrimSpace(iid) != "" {
		if m.closedInsights[tid+":"+strings.TrimSpace(iid)] {
			return fmt.Errorf("%w: insight %s is no longer pending", ErrPreconditionFailed, iid)
		}
	}
	if m.closedInsights[tid+":"+p.TargetObjectID] {
		return fmt.Errorf("%w: target %s is no longer pending", ErrPreconditionFailed, p.TargetObjectID)
	}
	return nil
}

func (m *MemoryRepository) VerifyCollectionsPrecondition(_ context.Context, tenantID string, p Proposal) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	payload := decodePayloadMap(p.PayloadJSON)
	invID, _ := payload["invoice_id"].(string)
	if strings.TrimSpace(invID) == "" {
		invID = p.TargetObjectID
	}
	if m.paidInvoices[tid+":"+strings.TrimSpace(invID)] {
		return fmt.Errorf("%w: invoice %s is already paid or settled", ErrPreconditionFailed, invID)
	}
	return nil
}

func (m *MemoryRepository) VerifyExpiryRebalancePrecondition(_ context.Context, tenantID string, p Proposal) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	payload := decodePayloadMap(p.PayloadJSON)
	lotID, _ := payload["lot_id"].(string)
	if strings.TrimSpace(lotID) == "" {
		lotID = p.TargetObjectID
	}
	if m.depletedLots[tid+":"+strings.TrimSpace(lotID)] {
		return fmt.Errorf("%w: stock lot %s is no longer available", ErrPreconditionFailed, lotID)
	}
	return nil
}

func cloneProposal(p Proposal) Proposal {
	cp := p
	if p.Payload != nil {
		m := make(map[string]any, len(p.Payload))
		for k, v := range p.Payload {
			m[k] = v
		}
		cp.Payload = m
	} else {
		cp.Payload = decodePayloadMap(p.PayloadJSON)
	}
	if p.EvidenceRefs != nil {
		cp.EvidenceRefs = append([]EvidenceRef(nil), p.EvidenceRefs...)
	}
	if p.ExecutionResult != nil {
		m := make(map[string]any, len(p.ExecutionResult))
		for k, v := range p.ExecutionResult {
			m[k] = v
		}
		cp.ExecutionResult = m
	}
	return cp
}

// SeedHistoricalProposal inserts a pre-built proposal directly (for soak-window unit tests).
func (m *MemoryRepository) SeedHistoricalProposal(p Proposal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(p.TenantID)
	if m.proposals[tid] == nil {
		m.proposals[tid] = make(map[string]Proposal)
	}
	m.proposals[tid][p.ProposalID] = cloneProposal(p)
}

func (m *MemoryRepository) GetL2SoakStats(_ context.Context, tenantID string, now time.Time) (L2SoakMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var out L2SoakMetrics
	var earliest *time.Time
	for _, p := range m.proposals[strings.TrimSpace(tenantID)] {
		st := strings.ToUpper(strings.TrimSpace(p.Status))
		switch st {
		case StatusExecuted:
			out.ExecutedCount++
			ref := p.CreatedAt
			if p.ExecutedAt != nil && !p.ExecutedAt.IsZero() {
				ref = p.ExecutedAt.UTC()
			}
			if !ref.IsZero() && (earliest == nil || ref.Before(*earliest)) {
				t := ref
				earliest = &t
			}
		case "ROLLED_BACK", "REVERSED":
			out.RollbackCount++
		case StatusFailed:
			if (p.ExecutedAt != nil && !p.ExecutedAt.IsZero()) || strings.TrimSpace(p.ExecutionError) != "" {
				out.RollbackCount++
			}
		}
	}
	out.EarliestExecuteAt = earliest
	if earliest != nil && now.After(*earliest) {
		out.SoakDays = int64(now.Sub(*earliest) / (24 * time.Hour))
	}
	total := out.ExecutedCount + out.RollbackCount
	if total > 0 {
		out.RollbackRateBps = (out.RollbackCount * 10000) / total
	}
	return out, nil
}
