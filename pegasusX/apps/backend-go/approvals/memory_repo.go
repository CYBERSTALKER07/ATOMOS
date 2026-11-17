package approvals

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository implements Repository in memory for deterministic unit tests and fallback mode.
type MemoryRepository struct {
	mu          sync.RWMutex
	policies    map[string]Policy             // tenantID|policyID -> Policy
	requests    map[string]ApprovalRequest    // tenantID|requestID -> ApprovalRequest
	byID        map[string]string             // requestID -> tenantID|requestID
	idemIndex   map[string]string             // tenantID|idempotencyKey -> requestID
	decisions   map[string][]ApprovalDecision // tenantID|requestID -> []ApprovalDecision
	delegations map[string]ApprovalDelegation // tenantID|delegationID -> ApprovalDelegation
}

// NewMemoryRepository constructs an isolated in-memory approval store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		policies:    make(map[string]Policy),
		requests:    make(map[string]ApprovalRequest),
		byID:        make(map[string]string),
		idemIndex:   make(map[string]string),
		decisions:   make(map[string][]ApprovalDecision),
		delegations: make(map[string]ApprovalDelegation),
	}
}

func memKey(tenantID, id string) string {
	return strings.TrimSpace(tenantID) + "\x00" + strings.TrimSpace(id)
}

func clonePolicy(p Policy) Policy {
	cp := p
	if len(p.Steps) > 0 {
		cp.Steps = make([]ApprovalStep, len(p.Steps))
		for i, st := range p.Steps {
			cp.Steps[i] = st
			cp.Steps[i].AllowedRoles = append([]string(nil), st.AllowedRoles...)
		}
	}
	return cp
}

func cloneRequest(r ApprovalRequest) ApprovalRequest {
	cp := r
	if r.ExpiresAt != nil {
		t := r.ExpiresAt.UTC()
		cp.ExpiresAt = &t
	}
	if r.ResolvedAt != nil {
		t := r.ResolvedAt.UTC()
		cp.ResolvedAt = &t
	}
	return cp
}

func (m *MemoryRepository) CreatePolicy(_ context.Context, p Policy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := memKey(p.TenantID, p.PolicyID)
	if _, exists := m.policies[k]; exists {
		return fmt.Errorf("%w: policy %s already exists", ErrInvalidInput, p.PolicyID)
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	m.policies[k] = clonePolicy(p)
	return nil
}

func (m *MemoryRepository) UpdatePolicy(_ context.Context, p Policy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := memKey(p.TenantID, p.PolicyID)
	if _, exists := m.policies[k]; !exists {
		return ErrPolicyNotFound
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	m.policies[k] = clonePolicy(p)
	return nil
}

func (m *MemoryRepository) GetPolicy(_ context.Context, tenantID, policyID string) (Policy, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.policies[memKey(tenantID, policyID)]
	if !ok {
		return Policy{}, false, nil
	}
	return clonePolicy(p), true, nil
}

func (m *MemoryRepository) ListPolicies(_ context.Context, tenantID, objectType, action string, enabledOnly bool) ([]Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	ot := strings.TrimSpace(objectType)
	act := strings.TrimSpace(action)
	out := make([]Policy, 0)
	for _, p := range m.policies {
		if tid != "" && p.TenantID != tid {
			continue
		}
		if ot != "" && !strings.EqualFold(p.ObjectType, ot) {
			continue
		}
		if act != "" && !strings.EqualFold(p.Action, act) {
			continue
		}
		if enabledOnly && !p.Enabled {
			continue
		}
		out = append(out, clonePolicy(p))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].PolicyID < out[j].PolicyID
	})
	return out, nil
}

func (m *MemoryRepository) FindRequestByIdempotency(_ context.Context, tenantID, idempotencyKey string) (ApprovalRequest, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return ApprovalRequest{}, false, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	reqID, ok := m.idemIndex[memKey(tenantID, idempotencyKey)]
	if !ok {
		return ApprovalRequest{}, false, nil
	}
	req, exists := m.requests[memKey(tenantID, reqID)]
	if !exists {
		return ApprovalRequest{}, false, nil
	}
	return cloneRequest(req), true, nil
}

func (m *MemoryRepository) CreateRequest(ctx context.Context, req ApprovalRequest, auditRec AuditRecorder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.IdempotencyKey != "" {
		if _, dup := m.idemIndex[memKey(req.TenantID, req.IdempotencyKey)]; dup {
			return nil
		}
	}
	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = now
	}
	k := memKey(req.TenantID, req.RequestID)
	m.requests[k] = cloneRequest(req)
	m.byID[req.RequestID] = k
	if req.IdempotencyKey != "" {
		m.idemIndex[memKey(req.TenantID, req.IdempotencyKey)] = req.RequestID
	}
	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, AuditRecordInput{
			TenantID:   req.TenantID,
			ActorID:    req.RequestedBy,
			ActorType:  req.ActorType,
			Action:     "approval.requested",
			ObjectType: "ApprovalRequest",
			ObjectID:   req.RequestID,
			AfterJSON:  req.PayloadJSON,
			ApprovalID: req.RequestID,
		})
	}
	return nil
}

func (m *MemoryRepository) GetRequest(_ context.Context, tenantID, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k := memKey(tenantID, requestID)
	req, ok := m.requests[k]
	if !ok {
		return ApprovalRequest{}, nil, false, nil
	}
	decs := append([]ApprovalDecision(nil), m.decisions[k]...)
	sort.Slice(decs, func(i, j int) bool {
		return decs[i].CreatedAt.Before(decs[j].CreatedAt)
	})
	return cloneRequest(req), decs, true, nil
}

func (m *MemoryRepository) GetRequestByID(_ context.Context, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.byID[strings.TrimSpace(requestID)]
	if !ok {
		return ApprovalRequest{}, nil, false, nil
	}
	req, exists := m.requests[k]
	if !exists {
		return ApprovalRequest{}, nil, false, nil
	}
	decs := append([]ApprovalDecision(nil), m.decisions[k]...)
	sort.Slice(decs, func(i, j int) bool {
		return decs[i].CreatedAt.Before(decs[j].CreatedAt)
	})
	return cloneRequest(req), decs, true, nil
}

func (m *MemoryRepository) ListInbox(_ context.Context, f InboxFilter) ([]ApprovalRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tid := strings.TrimSpace(f.TenantID)
	st := strings.ToUpper(strings.TrimSpace(f.Status))
	ot := strings.TrimSpace(f.ObjectType)
	act := strings.TrimSpace(f.Action)
	out := make([]ApprovalRequest, 0)
	for _, req := range m.requests {
		if tid != "" && req.TenantID != tid {
			continue
		}
		if st != "" && !strings.EqualFold(req.Status, st) {
			continue
		}
		if ot != "" && !strings.EqualFold(req.ObjectType, ot) {
			continue
		}
		if act != "" && !strings.EqualFold(req.Action, act) {
			continue
		}
		if f.BreakGlass != nil && req.BreakGlass != *f.BreakGlass {
			continue
		}
		out = append(out, cloneRequest(req))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].RequestID > out[j].RequestID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) ApplyDecision(
	ctx context.Context,
	tenantID, requestID string,
	mutateFn func(req ApprovalRequest, existingDecisions []ApprovalDecision, policy *Policy) (ApprovalRequest, ApprovalDecision, error),
	auditRec AuditRecorder,
	traceID string,
) (ApprovalRequest, ApprovalDecision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := memKey(tenantID, requestID)
	req, ok := m.requests[k]
	if !ok {
		return ApprovalRequest{}, ApprovalDecision{}, ErrRequestNotFound
	}
	var polPtr *Policy
	if req.PolicyID != "" {
		if pol, pOK := m.policies[memKey(req.TenantID, req.PolicyID)]; pOK {
			cp := clonePolicy(pol)
			polPtr = &cp
		}
	}
	existing := append([]ApprovalDecision(nil), m.decisions[k]...)
	updated, dec, err := mutateFn(cloneRequest(req), existing, polPtr)
	if err != nil {
		return ApprovalRequest{}, ApprovalDecision{}, err
	}
	now := time.Now().UTC()
	if updated.UpdatedAt.IsZero() {
		updated.UpdatedAt = now
	}
	if dec.CreatedAt.IsZero() {
		dec.CreatedAt = now
	}
	m.requests[k] = cloneRequest(updated)
	m.decisions[k] = append(m.decisions[k], dec)
	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, AuditRecordInput{
			TenantID:   updated.TenantID,
			ActorID:    dec.ApproverID,
			ActorType:  "human",
			Action:     "approval." + strings.ToLower(dec.Decision),
			ObjectType: "ApprovalRequest",
			ObjectID:   updated.RequestID,
			AfterJSON:  updated.PayloadJSON,
			ReasonCode: dec.Decision,
			ApprovalID: updated.RequestID,
			TraceID:    traceID,
		})
	}
	return cloneRequest(updated), dec, nil
}

func (m *MemoryRepository) CreateDelegation(_ context.Context, d ApprovalDelegation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	m.delegations[memKey(d.TenantID, d.DelegationID)] = d
	return nil
}

func (m *MemoryRepository) GetDelegation(_ context.Context, tenantID, delegationID string) (ApprovalDelegation, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.delegations[memKey(tenantID, delegationID)]
	return d, ok, nil
}

func (m *MemoryRepository) ListDelegations(_ context.Context, tenantID, actorID string) ([]ApprovalDelegation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	aid := strings.TrimSpace(actorID)
	out := make([]ApprovalDelegation, 0)
	for _, d := range m.delegations {
		if tid != "" && d.TenantID != tid {
			continue
		}
		if aid != "" && d.DelegatorID != aid && d.DelegateID != aid {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryRepository) FindActiveDelegationsForDelegate(_ context.Context, tenantID, delegateID string, at time.Time) ([]ApprovalDelegation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	did := strings.TrimSpace(delegateID)
	out := make([]ApprovalDelegation, 0)
	for _, d := range m.delegations {
		if d.TenantID != tid || d.DelegateID != did || !d.Enabled {
			continue
		}
		if !d.StartsAt.IsZero() && at.Before(d.StartsAt) {
			continue
		}
		if !d.EndsAt.IsZero() && !at.Before(d.EndsAt) {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *MemoryRepository) SetDelegationEnabled(_ context.Context, tenantID, delegationID string, enabled bool) (ApprovalDelegation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := memKey(tenantID, delegationID)
	d, ok := m.delegations[k]
	if !ok {
		return ApprovalDelegation{}, ErrDelegationNotFound
	}
	d.Enabled = enabled
	m.delegations[k] = d
	return d, nil
}
