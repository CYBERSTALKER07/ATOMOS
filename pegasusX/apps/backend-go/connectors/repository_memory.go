package connectors

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is a thread-safe in-memory Repository for unit and handler tests.
type MemoryRepository struct {
	mu        sync.RWMutex
	creds     map[string]map[string]ExternalCredential // tenantID -> credID -> cred
	runs      map[string]map[string]ConnectorSyncRun   // tenantID -> runID -> run
	byIdem    map[string]ConnectorSyncRun              // tenantID + "|" + idem -> run
	maps      map[string]ConnectorEntityMap            // tenantID + "|" + provider + "|" + objType + "|" + extID -> map
	snapshots map[string]InternalSnapshot              // tenantID -> snapshot
	emitted   []OutboxEmission
}

// NewMemoryRepository constructs an empty thread-safe in-memory connector repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		creds:     make(map[string]map[string]ExternalCredential),
		runs:      make(map[string]map[string]ConnectorSyncRun),
		byIdem:    make(map[string]ConnectorSyncRun),
		maps:      make(map[string]ConnectorEntityMap),
		snapshots: make(map[string]InternalSnapshot),
	}
}

// SeedInternalSnapshot seeds internal Spanner fixtures for ShadowCompare unit tests.
func (m *MemoryRepository) SeedInternalSnapshot(tenantID string, snap InternalSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[strings.TrimSpace(tenantID)] = snap
}

// EmittedOutbox returns a copy of outbox emissions recorded during tests.
func (m *MemoryRepository) EmittedOutbox() []OutboxEmission {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]OutboxEmission, len(m.emitted))
	copy(out, m.emitted)
	return out
}

func (m *MemoryRepository) UpsertCredential(_ context.Context, cred ExternalCredential) (ExternalCredential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(cred.TenantID)
	if m.creds[tid] == nil {
		m.creds[tid] = make(map[string]ExternalCredential)
	}
	now := time.Now().UTC()
	if cred.CredentialID == "" {
		// Check for existing credential by (provider, account_label) when credential_id was omitted.
		for id, ex := range m.creds[tid] {
			if strings.EqualFold(ex.Provider, cred.Provider) && strings.EqualFold(ex.AccountLabel, cred.AccountLabel) {
				cred.CredentialID = id
				cred.CreatedAt = ex.CreatedAt
				if cred.CreatedBy == "" {
					cred.CreatedBy = ex.CreatedBy
				}
				break
			}
		}
		if cred.CredentialID == "" {
			cred.CredentialID = "cred_" + normalizeProvider(cred.Provider) + "_" + sanitizeIDToken(cred.AccountLabel)
		}
	} else if existing, ok := m.creds[tid][cred.CredentialID]; ok {
		cred.CreatedAt = existing.CreatedAt
		if cred.CreatedBy == "" {
			cred.CreatedBy = existing.CreatedBy
		}
	}
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = now
	}
	cred.UpdatedAt = now
	m.creds[tid][cred.CredentialID] = cloneCredential(cred)
	return cloneCredential(cred), nil
}

func (m *MemoryRepository) GetCredential(_ context.Context, tenantID, provider, accountLabel, credentialID string) (ExternalCredential, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	byTenant := m.creds[tid]
	if len(byTenant) == 0 {
		return ExternalCredential{}, false, nil
	}
	if strings.TrimSpace(credentialID) != "" {
		if c, ok := byTenant[strings.TrimSpace(credentialID)]; ok && strings.EqualFold(c.Provider, normalizeProvider(provider)) {
			return cloneCredential(c), true, nil
		}
		return ExternalCredential{}, false, nil
	}
	prov := normalizeProvider(provider)
	wantLabel := strings.TrimSpace(accountLabel)
	var fallback *ExternalCredential
	for _, c := range byTenant {
		if normalizeProvider(c.Provider) != prov || strings.EqualFold(c.Status, CredStatusDisabled) {
			continue
		}
		if wantLabel != "" && strings.EqualFold(c.AccountLabel, wantLabel) {
			cp := cloneCredential(c)
			return cp, true, nil
		}
		if wantLabel == "" && strings.EqualFold(c.AccountLabel, "default") {
			cp := cloneCredential(c)
			return cp, true, nil
		}
		if wantLabel == "" && fallback == nil {
			cp := cloneCredential(c)
			fallback = &cp
		}
	}
	if fallback != nil {
		return *fallback, true, nil
	}
	return ExternalCredential{}, false, nil
}

func (m *MemoryRepository) ListCredentials(_ context.Context, tenantID, provider string) ([]ExternalCredential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	out := make([]ExternalCredential, 0)
	for _, c := range m.creds[tid] {
		if prov != "" && normalizeProvider(c.Provider) != prov {
			continue
		}
		out = append(out, cloneCredential(c))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		if out[i].AccountLabel != out[j].AccountLabel {
			return out[i].AccountLabel < out[j].AccountLabel
		}
		return out[i].CredentialID < out[j].CredentialID
	})
	return out, nil
}

func (m *MemoryRepository) FindRunByIdempotency(_ context.Context, tenantID, idempotencyKey string) (ConnectorSyncRun, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if strings.TrimSpace(idempotencyKey) == "" {
		return ConnectorSyncRun{}, false, nil
	}
	r, ok := m.byIdem[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(idempotencyKey)]
	if !ok {
		return ConnectorSyncRun{}, false, nil
	}
	return cloneSyncRun(r), true, nil
}

func (m *MemoryRepository) SaveSyncRunAndMaps(_ context.Context, run ConnectorSyncRun, maps []ConnectorEntityMap, emissions []OutboxEmission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tid := strings.TrimSpace(run.TenantID)
	if m.runs[tid] == nil {
		m.runs[tid] = make(map[string]ConnectorSyncRun)
	}
	if run.IdempotencyKey != "" {
		idemKey := tid + "|" + strings.TrimSpace(run.IdempotencyKey)
		if _, exists := m.byIdem[idemKey]; exists {
			return nil
		}
		m.byIdem[idemKey] = cloneSyncRun(run)
	}
	m.runs[tid][run.RunID] = cloneSyncRun(run)
	for _, em := range maps {
		k := tid + "|" + em.Provider + "|" + em.ObjectType + "|" + em.ExternalID
		m.maps[k] = em
	}
	m.emitted = append(m.emitted, emissions...)
	return nil
}

func (m *MemoryRepository) GetSyncRun(_ context.Context, tenantID, runID string) (ConnectorSyncRun, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[strings.TrimSpace(tenantID)][strings.TrimSpace(runID)]
	if !ok {
		return ConnectorSyncRun{}, false, nil
	}
	return cloneSyncRun(r), true, nil
}

func (m *MemoryRepository) ListSyncRuns(_ context.Context, tenantID, provider string, limit int) ([]ConnectorSyncRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	out := make([]ConnectorSyncRun, 0)
	for _, r := range m.runs[tid] {
		if prov != "" && normalizeProvider(r.Provider) != prov {
			continue
		}
		out = append(out, cloneSyncRun(r))
	}
	sortRunsDesc(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) ListLatestRunsByProvider(_ context.Context, tenantID string) ([]ConnectorSyncRun, error) {
	runs, err := m.ListSyncRuns(context.Background(), tenantID, "", 0)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	latest := make([]ConnectorSyncRun, 0)
	for _, r := range runs {
		key := r.Provider + "|" + r.AccountLabel
		if seen[key] {
			continue
		}
		seen[key] = true
		latest = append(latest, r)
	}
	return latest, nil
}

func (m *MemoryRepository) LoadInternalSnapshot(_ context.Context, tenantID, provider string) (InternalSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	snap := m.snapshots[tid]
	for _, em := range m.maps {
		if em.TenantID == tid && normalizeProvider(em.Provider) == prov {
			snap.EntityMaps = append(snap.EntityMaps, em)
		}
	}
	return snap, nil
}

func sortRunsDesc(runs []ConnectorSyncRun) {
	sort.Slice(runs, func(i, j int) bool {
		if !runs[i].StartedAt.Equal(runs[j].StartedAt) {
			return runs[i].StartedAt.After(runs[j].StartedAt)
		}
		return runs[i].RunID > runs[j].RunID
	})
}

func cloneCredential(c ExternalCredential) ExternalCredential {
	cp := c
	if c.Metadata != nil {
		raw, _ := json.Marshal(c.Metadata)
		_ = decodeJSONWithNumbers(raw, &cp.Metadata)
	}
	return cp
}

func cloneSyncRun(r ConnectorSyncRun) ConnectorSyncRun {
	cp := r
	if r.CompletedAt != nil {
		t := *r.CompletedAt
		cp.CompletedAt = &t
	}
	if r.Summary != nil {
		s := *r.Summary
		cp.Summary = &s
	}
	if r.Mismatches != nil {
		cp.Mismatches = append([]ShadowCompareMismatch(nil), r.Mismatches...)
	}
	return cp
}
