package innlookup

import (
	"context"
	"encoding/json"
	"sync"
)

// StubAdapter provides deterministic taxpayer responses for offline tests and
// explicit sandbox drills. Never silently installed as a fake-green production adapter.
type StubAdapter struct {
	mu               sync.RWMutex
	entries          map[string]LookupResult
	forceUnavailable bool
}

// NewStubAdapter returns an empty deterministic StubAdapter.
func NewStubAdapter() *StubAdapter {
	return &StubAdapter{
		entries: make(map[string]LookupResult),
	}
}

// Register adds a FOUND taxpayer record keyed by normalized 9-digit INN or 14-digit PINFL.
func (s *StubAdapter) Register(taxID, legalName, address string) {
	norm, entityType, err := NormalizeTaxID(taxID)
	if err != nil {
		return
	}
	raw, _ := json.Marshal(map[string]any{
		"source":         "stub",
		"normalized_inn": norm,
		"entity_type":    entityType,
		"legal_name":     legalName,
		"address":        address,
		"registry_state": "ACTIVE",
	})
	s.RegisterResult(norm, LookupResult{
		Status:        StatusFound,
		NormalizedINN: norm,
		EntityType:    entityType,
		LegalName:     legalName,
		Address:       address,
		RegistryState: "ACTIVE",
		RawJSON:       raw,
	})
}

// RegisterResult stores an explicit LookupResult for a tax ID.
func (s *StubAdapter) RegisterResult(taxID string, res LookupResult) {
	norm, entityType, err := NormalizeTaxID(taxID)
	if err != nil {
		return
	}
	res.NormalizedINN = norm
	if res.EntityType == "" {
		res.EntityType = entityType
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]LookupResult)
	}
	s.entries[norm] = res
}

// SetForceUnavailable toggles simulated registry outage for fail-closed tests.
func (s *StubAdapter) SetForceUnavailable(unavailable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forceUnavailable = unavailable
}

// Lookup satisfies LookupAdapter.
func (s *StubAdapter) Lookup(_ context.Context, _ string, normalizedINN string) (LookupResult, error) {
	if s == nil {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, ErrRegistryUnavailable
	}
	s.mu.RLock()
	unavailable := s.forceUnavailable
	res, ok := s.entries[normalizedINN]
	s.mu.RUnlock()
	if unavailable {
		return LookupResult{Status: StatusUnavailable, NormalizedINN: normalizedINN}, ErrRegistryUnavailable
	}
	if !ok {
		return LookupResult{Status: StatusNotFound, NormalizedINN: normalizedINN}, nil
	}
	if res.Status == StatusUnavailable {
		return res, ErrRegistryUnavailable
	}
	return res, nil
}
